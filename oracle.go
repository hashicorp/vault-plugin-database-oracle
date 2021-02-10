package oracle

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	dbplugin "github.com/hashicorp/vault/sdk/database/dbplugin/v5"
	"github.com/hashicorp/vault/sdk/database/helper/connutil"
	"github.com/hashicorp/vault/sdk/database/helper/dbutil"
	"github.com/hashicorp/vault/sdk/helper/dbtxn"
	"github.com/hashicorp/vault/sdk/helper/strutil"
	"github.com/hashicorp/vault/sdk/helper/template"
	_ "github.com/mattn/go-oci8"
)

const (
	oracleTypeName = "oci8"

	revocationSQL = `
REVOKE CONNECT FROM {{username}};
REVOKE CREATE SESSION FROM {{username}};
DROP USER {{username}};
`

	defaultRotateCredsSql = `ALTER USER {{username}} IDENTIFIED BY "{{password}}"`

	defaultUsernameTemplate = `{{ printf "V_%s_%s_%s_%s" (.DisplayName | truncate 8) (.RoleName | truncate 8) (random 20) (unix_time) | truncate 30 | uppercase | replace "-" "_" | replace "." "_" }}`
)

var _ dbplugin.Database = (*Oracle)(nil)

type Oracle struct {
	*connutil.SQLConnectionProducer
	usernameProducer template.StringTemplate
}

func New() (interface{}, error) {
	db := new()
	// Wrap the plugin with middleware to sanitize errors
	dbType := dbplugin.NewDatabaseErrorSanitizerMiddleware(db, db.secretValues)
	return dbType, nil
}

func new() *Oracle {
	connProducer := &connutil.SQLConnectionProducer{}
	connProducer.Type = oracleTypeName

	dbType := &Oracle{
		SQLConnectionProducer: connProducer,
	}

	return dbType
}

func (o *Oracle) Initialize(ctx context.Context, req dbplugin.InitializeRequest) (dbplugin.InitializeResponse, error) {
	usernameTemplate, err := strutil.GetString(req.Config, "username_template")
	if err != nil {
		return dbplugin.InitializeResponse{}, fmt.Errorf("failed to retrieve username_template: %w", err)
	}
	if usernameTemplate == "" {
		usernameTemplate = defaultUsernameTemplate
	}

	up, err := template.NewTemplate(template.Template(usernameTemplate))
	if err != nil {
		return dbplugin.InitializeResponse{}, fmt.Errorf("unable to initialize username template: %w", err)
	}
	o.usernameProducer = up

	_, err = o.usernameProducer.Generate(dbplugin.UsernameMetadata{})
	if err != nil {
		return dbplugin.InitializeResponse{}, fmt.Errorf("invalid username template: %w", err)
	}

	err = o.SQLConnectionProducer.Initialize(ctx, req.Config, req.VerifyConnection)
	if err != nil {
		return dbplugin.InitializeResponse{}, err
	}
	resp := dbplugin.InitializeResponse{
		Config: req.Config,
	}
	return resp, nil
}

func (o *Oracle) NewUser(ctx context.Context, req dbplugin.NewUserRequest) (dbplugin.NewUserResponse, error) {
	statements := removeEmpty(req.Statements.Commands)
	if len(statements) == 0 {
		return dbplugin.NewUserResponse{}, dbutil.ErrEmptyCreationStatement
	}

	o.Lock()
	defer o.Unlock()

	username, err := o.usernameProducer.Generate(req.UsernameConfig)
	if err != nil {
		return dbplugin.NewUserResponse{}, fmt.Errorf("failed to generate username: %w", err)
	}

	db, err := o.getConnection(ctx)
	if err != nil {
		return dbplugin.NewUserResponse{}, fmt.Errorf("failed to get connection: %w", err)
	}

	err = newUser(ctx, db, username, req.Password, req.Expiration, req.Statements.Commands)
	if err != nil {
		return dbplugin.NewUserResponse{}, err
	}

	resp := dbplugin.NewUserResponse{
		Username: username,
	}
	return resp, nil
}

func removeEmpty(strs []string) []string {
	newStrs := []string{}
	for _, str := range strs {
		str = strings.TrimSpace(str)
		if str == "" {
			continue
		}
		newStrs = append(newStrs, str)
	}
	return newStrs
}

func newUser(ctx context.Context, db *sql.DB, username, password string, expiration time.Time, commands []string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start a transaction: %w", err)
	}
	// Effectively a no-op if the transaction commits successfully
	defer tx.Rollback()

	for _, stmt := range commands {
		for _, query := range strutil.ParseArbitraryStringSlice(stmt, ";") {
			query = strings.TrimSpace(query)
			if len(query) == 0 {
				continue
			}

			m := map[string]string{
				"username":   username,
				"name":       username, // backwards compatibility
				"password":   password,
				"expiration": expiration.Format("2006-01-02 15:04:05-0700"),
			}

			err = dbtxn.ExecuteTxQuery(ctx, tx, m, query)
			if err != nil {
				return fmt.Errorf("failed to execute query: %w", err)
			}
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (o *Oracle) UpdateUser(ctx context.Context, req dbplugin.UpdateUserRequest) (dbplugin.UpdateUserResponse, error) {
	if req.Password == nil && req.Expiration == nil {
		return dbplugin.UpdateUserResponse{}, fmt.Errorf("no change requested")
	}

	if req.Password != nil {
		err := o.changeUserPassword(ctx, req.Username, req.Password.NewPassword, req.Password.Statements.Commands)
		if err != nil {
			return dbplugin.UpdateUserResponse{}, fmt.Errorf("failed to change password: %w", err)
		}
		return dbplugin.UpdateUserResponse{}, nil
	}
	// Expiration change is a no-op
	return dbplugin.UpdateUserResponse{}, nil
}

func (o *Oracle) changeUserPassword(ctx context.Context, username string, newPassword string, rotateStatements []string) error {
	if len(rotateStatements) == 0 {
		rotateStatements = []string{defaultRotateCredsSql}
	}

	if username == "" || newPassword == "" {
		return errors.New("must provide both username and password")
	}

	variables := map[string]string{
		"username": username,
		"name":     username, // backwards compatibility
		"password": newPassword,
	}

	queries := splitQueries(rotateStatements)
	if len(queries) == 0 { // Extra check to protect against future changes
		return errors.New("no rotation queries found")
	}

	o.Lock()
	defer o.Unlock()

	db, err := o.getConnection(ctx)
	if err != nil {
		return fmt.Errorf("unable to get database connection: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("unable to create database transaction: %w", err)
	}
	// Effectively a no-op if the transaction commits successfully
	defer tx.Rollback()

	for _, rawQuery := range queries {
		parsedQuery := dbutil.QueryHelper(rawQuery, variables)
		err := dbtxn.ExecuteTxQuery(ctx, tx, nil, parsedQuery)
		if err != nil {
			return fmt.Errorf("unable to execute query [%s]: %w", rawQuery, err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("unable to commit queries: %w", err)
	}

	return nil
}

func (o *Oracle) DeleteUser(ctx context.Context, req dbplugin.DeleteUserRequest) (dbplugin.DeleteUserResponse, error) {
	o.Lock()
	defer o.Unlock()

	db, err := o.getConnection(ctx)
	if err != nil {
		return dbplugin.DeleteUserResponse{}, fmt.Errorf("failed to make connection: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return dbplugin.DeleteUserResponse{}, fmt.Errorf("failed to start transaction: %w", err)
	}
	// Effectively a no-op if the transaction commits successfully
	defer tx.Rollback()

	err = o.disconnectSession(db, req.Username)
	if err != nil {
		return dbplugin.DeleteUserResponse{}, fmt.Errorf("failed to disconnect user %s: %w", req.Username, err)
	}

	revocationStatements := req.Statements.Commands
	if len(revocationStatements) == 0 {
		revocationStatements = []string{revocationSQL}
	}

	// We can't use a transaction here, because Oracle treats DROP USER as a DDL statement, which commits immediately.
	for _, stmt := range revocationStatements {
		for _, query := range strutil.ParseArbitraryStringSlice(stmt, ";") {
			query = strings.TrimSpace(query)
			if len(query) == 0 {
				continue
			}

			m := map[string]string{
				"username": req.Username,
				"name":     req.Username, // backwards compatibility
			}

			if err := dbtxn.ExecuteTxQuery(ctx, tx, m, query); err != nil {
				return dbplugin.DeleteUserResponse{}, fmt.Errorf("failed to execute query: %w", err)
			}
		}
	}

	return dbplugin.DeleteUserResponse{}, nil
}

func (o *Oracle) Type() (string, error) {
	return oracleTypeName, nil
}

func (o *Oracle) secretValues() map[string]string {
	return map[string]string{
		o.Password: "[password]",
	}
}

func splitQueries(rawQueries []string) (queries []string) {
	for _, rawQ := range rawQueries {
		split := strutil.ParseArbitraryStringSlice(rawQ, ";")
		for _, newQ := range split {
			newQ = strings.TrimSpace(newQ)
			if newQ == "" {
				continue
			}
			queries = append(queries, newQ)
		}
	}
	return queries
}

func (o *Oracle) disconnectSession(db *sql.DB, username string) error {
	disconnectVars := map[string]string{
		"username": username,
	}
	disconnectQuery := dbutil.QueryHelper(`SELECT sid, serial#, username FROM gv$session WHERE username = UPPER('{{username}}')`, disconnectVars)
	disconnectStmt, err := db.Prepare(disconnectQuery)
	if err != nil {
		return err
	}
	defer disconnectStmt.Close()
	if rows, err := disconnectStmt.Query(); err != nil {
		return err
	} else {
		defer rows.Close()
		for rows.Next() {
			var sessionID, serialNumber int
			var username sql.NullString
			err = rows.Scan(&sessionID, &serialNumber, &username)
			if err != nil {
				return err
			}

			killStatement := fmt.Sprintf(`ALTER SYSTEM KILL SESSION '%d,%d' IMMEDIATE`, sessionID, serialNumber)
			_, err = db.Exec(killStatement)
			if err != nil {
				return err
			}
		}
		err = rows.Err()
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *Oracle) getConnection(ctx context.Context) (*sql.DB, error) {
	db, err := o.Connection(ctx)
	if err != nil {
		return nil, err
	}

	return db.(*sql.DB), nil
}
