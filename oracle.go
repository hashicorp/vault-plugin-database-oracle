package oracle

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
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

	defaultRotateCredsSql = `ALTER USER {{username}} IDENTIFIED BY "{{password}}"`

	defaultUsernameTemplate = `{{ printf "V_%s_%s_%s_%s" (.DisplayName | truncate 8) (.RoleName | truncate 8) (random 20) (unix_time) | truncate 30 | uppercase | replace "-" "_" | replace "." "_" }}`
)

var (
	defaultRevocationStatements = []string{
		`REVOKE CONNECT FROM {{username}}`,
		`REVOKE CREATE SESSION FROM {{username}}`,
		`DROP USER {{username}}`,
	}

	defaultSessionRevocationStatements = []string{
		`ALTER USER {{username}} ACCOUNT LOCK`,
		`begin
		  for x in ( select inst_id, sid, serial# from gv$session where username="{{username}}" )
		  loop
		   execute immediate ( 'alter system kill session '''|| x.Sid || ',' || x.Serial# || '@' || x.inst_id ''' immediate' );
		  end loop;
		  dbms_lock.sleep(1);
		end;`,
		`DROP USER {{username}}`,
	}
)

var _ dbplugin.Database = (*Oracle)(nil)

type Oracle struct {
	*connutil.SQLConnectionProducer
	usernameProducer template.StringTemplate

	splitStatements    bool
	disconnectSessions bool
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

	splitStatements, err := coerceToBool(req.Config, "split_statements", true)
	if err != nil {
		return dbplugin.InitializeResponse{}, fmt.Errorf("failed to parse 'split_statements' field: %w", err)
	}
	o.splitStatements = splitStatements

	disconnectSessions, err := coerceToBool(req.Config, "disconnect_sessions", true)
	if err != nil {
		return dbplugin.InitializeResponse{}, fmt.Errorf("failed to parse 'disconnect_sessions' field: %w", err)
	}
	o.disconnectSessions = disconnectSessions

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

func coerceToBool(m map[string]interface{}, key string, def bool) (bool, error) {
	rawVal, ok := m[key]
	if !ok {
		return def, nil
	}

	switch val := rawVal.(type) {
	case bool:
		return val, nil
	case string:
		return strconv.ParseBool(val)
	}

	return false, fmt.Errorf("invalid type for key [%s]", key)
}

func (o *Oracle) NewUser(ctx context.Context, req dbplugin.NewUserRequest) (dbplugin.NewUserResponse, error) {
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

	err = o.newUser(ctx, db, username, req.Password, req.Expiration, req.Statements.Commands)
	if err != nil {
		return dbplugin.NewUserResponse{}, err
	}

	resp := dbplugin.NewUserResponse{
		Username: username,
	}
	return resp, nil
}

func (o *Oracle) newUser(ctx context.Context, db *sql.DB, username, password string, expiration time.Time, commands []string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start a transaction: %w", err)
	}
	// Effectively a no-op if the transaction commits successfully
	defer tx.Rollback()

	statements := o.parseStatements(commands)
	if len(statements) == 0 {
		return dbutil.ErrEmptyCreationStatement
	}

	for _, query := range statements {
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

	statements := o.parseStatements(rotateStatements)
	if len(statements) == 0 { // Extra check to protect against future changes
		return errors.New("no rotation statements found")
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

	for _, query := range statements {
		parsedQuery := dbutil.QueryHelper(query, variables)
		err := dbtxn.ExecuteTxQuery(ctx, tx, nil, parsedQuery)
		if err != nil {
			return fmt.Errorf("unable to execute query [%s]: %w", query, err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("unable to commit statements: %w", err)
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

	if o.disconnectSessions {
		err = o.disconnectSession(db, req.Username)
		if err != nil {
			return dbplugin.DeleteUserResponse{}, fmt.Errorf("failed to disconnect user %s: %w", req.Username, err)
		}
	}

	revocationStatements := o.getRevocationStatements(req.Statements.Commands)
	if len(revocationStatements) == 0 {
		return dbplugin.DeleteUserResponse{}, fmt.Errorf("empty revocation statements")
	}

	// We can't use a transaction here, because Oracle treats DROP USER as a DDL statement, which commits immediately.
	for _, query := range revocationStatements {
		m := map[string]string{
			"username": req.Username,
			"name":     req.Username, // backwards compatibility
		}

		if err := dbtxn.ExecuteTxQuery(ctx, tx, m, query); err != nil {
			return dbplugin.DeleteUserResponse{}, fmt.Errorf("failed to execute query: %w", err)
		}
	}

	return dbplugin.DeleteUserResponse{}, nil
}

func (o *Oracle) getRevocationStatements(statements []string) []string {
	if len(statements) > 0 {
		statements = o.parseStatements(statements)
		return statements
	}

	if !o.splitStatements || !o.disconnectSessions {
		return defaultSessionRevocationStatements
	} else {
		return defaultRevocationStatements
	}
}

func (o *Oracle) Type() (string, error) {
	return oracleTypeName, nil
}

func (o *Oracle) secretValues() map[string]string {
	return map[string]string{
		o.Password: "[password]",
	}
}

// parseStatements conditionally splits the list of commands on semi-colons. If `split_statements` is true, this
// will return the provided slice of commands without altering them
func (o *Oracle) parseStatements(rawStatements []string) []string {
	if !o.splitStatements {
		statements := []string{}
		for _, rawQ := range rawStatements {
			newQ := strings.TrimSpace(rawQ)
			if newQ == "" {
				continue
			}
			statements = append(statements, newQ)
		}
		return statements
	}

	statements := []string{}
	for _, rawQ := range rawStatements {
		split := strutil.ParseArbitraryStringSlice(rawQ, ";")
		for _, newQ := range split {
			newQ = strings.TrimSpace(newQ)
			if newQ == "" {
				continue
			}
			statements = append(statements, newQ)
		}
	}
	return statements
}

func (o *Oracle) disconnectSession(db *sql.DB, username string) error {
	err := o.disconnectFromCluster(db, username)
	if err == nil {
		return nil
	}

	return o.disconnectLocal(db, username)
}

func (o *Oracle) disconnectFromCluster(db *sql.DB, username string) error {
	disconnectVars := map[string]string{
		"username": username,
	}
	query := dbutil.QueryHelper(`SELECT inst_id, sid, serial#, username FROM gv$session WHERE username = UPPER('{{username}}')`, disconnectVars)

	disconnectStmt, err := db.Prepare(query)
	if err != nil {
		return err
	}
	defer disconnectStmt.Close()
	rows, err := disconnectStmt.Query()
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var instID, sessionID, serialNumber int
		var username sql.NullString
		err = rows.Scan(&instID, &sessionID, &serialNumber, &username)
		if err != nil {
			return err
		}

		killStatement := fmt.Sprintf(`ALTER SYSTEM KILL SESSION '%d,%d,@%d' IMMEDIATE`, sessionID, serialNumber, instID)
		_, err = db.Exec(killStatement)
		if err != nil {
			return err
		}
	}
	err = rows.Err()
	if err != nil {
		return err
	}
	return nil
}

func (o *Oracle) disconnectLocal(db *sql.DB, username string) error {
	disconnectVars := map[string]string{
		"username": username,
	}
	query := dbutil.QueryHelper(`SELECT sid, serial#, username FROM v$session WHERE username = UPPER('{{username}}')`, disconnectVars)

	disconnectStmt, err := db.Prepare(query)
	if err != nil {
		return err
	}
	defer disconnectStmt.Close()
	rows, err := disconnectStmt.Query()
	if err != nil {
		return err
	}
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
	return nil
}

func (o *Oracle) getConnection(ctx context.Context) (*sql.DB, error) {
	db, err := o.Connection(ctx)
	if err != nil {
		return nil, err
	}

	return db.(*sql.DB), nil
}
