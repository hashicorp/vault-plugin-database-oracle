package oracle

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-oci8"

	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/sdk/database/dbplugin"
	"github.com/hashicorp/vault/sdk/database/helper/connutil"
	"github.com/hashicorp/vault/sdk/database/helper/credsutil"
	"github.com/hashicorp/vault/sdk/database/helper/dbutil"
	"github.com/hashicorp/vault/sdk/helper/dbtxn"
	"github.com/hashicorp/vault/sdk/helper/strutil"
)

const (
	oracleTypeName             = "oci8"
	oracleUsernameLength       = 30
	oracleDisplayNameMaxLength = 8

	revocationSQL = `
REVOKE CONNECT FROM {{name}};
REVOKE CREATE SESSION FROM {{name}};
DROP USER {{name}};
`

	defaultRotateCredsSql = `ALTER USER {{username}} IDENTIFIED BY "{{password}}"`
)

type Oracle struct {
	*connutil.SQLConnectionProducer
	credsutil.CredentialsProducer
}

// New implements builtinplugins.BuiltinFactory
func New() (interface{}, error) {
	db := new()
	// Wrap the plugin with middleware to sanitize errors
	dbType := dbplugin.NewDatabaseErrorSanitizerMiddleware(db, db.SecretValues)
	return dbType, nil
}

func new() *Oracle {
	connProducer := &connutil.SQLConnectionProducer{}
	connProducer.Type = oracleTypeName

	credsProducer := &oracleCredentialsProducer{
		credsutil.SQLCredentialsProducer{
			DisplayNameLen: oracleDisplayNameMaxLength,
			RoleNameLen:    oracleDisplayNameMaxLength,
			UsernameLen:    oracleUsernameLength,
			Separator:      "_",
		},
	}

	dbType := &Oracle{
		SQLConnectionProducer: connProducer,
		CredentialsProducer:   credsProducer,
	}

	return dbType
}

// Run instantiates an Oracle object, and runs the RPC server for the plugin
func Run(apiTLSConfig *api.TLSConfig) error {
	dbType, err := New()
	if err != nil {
		return err
	}

	dbplugin.Serve(dbType.(dbplugin.Database), api.VaultPluginTLSProvider(apiTLSConfig))

	return nil
}

func (o *Oracle) Type() (string, error) {
	return oracleTypeName, nil
}

func (o *Oracle) CreateUser(ctx context.Context, statements dbplugin.Statements, usernameConfig dbplugin.UsernameConfig, expiration time.Time) (username string, password string, err error) {
	statements = dbutil.StatementCompatibilityHelper(statements)

	if len(statements.Creation) == 0 {
		return "", "", dbutil.ErrEmptyCreationStatement
	}

	// Grab the lock
	o.Lock()
	defer o.Unlock()

	username, err = o.GenerateUsername(usernameConfig)
	if err != nil {
		return "", "", err
	}

	password, err = o.GeneratePassword()
	if err != nil {
		return "", "", err
	}

	expirationStr, err := o.GenerateExpiration(expiration)
	if err != nil {
		return "", "", err
	}

	// Get the connection
	db, err := o.getConnection(ctx)
	if err != nil {
		return "", "", err

	}

	// Start a transaction
	tx, err := db.Begin()
	if err != nil {
		return "", "", err

	}
	// Effectively a no-op if the transaction commits successfully
	defer tx.Rollback()

	// Execute each query
	for _, stmt := range statements.Creation {
		for _, query := range strutil.ParseArbitraryStringSlice(stmt, ";") {
			query = strings.TrimSpace(query)
			if len(query) == 0 {
				continue
			}

			m := map[string]string{
				"name":       username,
				"password":   password,
				"expiration": expirationStr,
			}

			if err := dbtxn.ExecuteTxQuery(ctx, tx, m, query); err != nil {
				return "", "", err
			}
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return "", "", err

	}

	// Return the secret
	return username, password, nil
}

func (o *Oracle) RenewUser(ctx context.Context, statements dbplugin.Statements, username string, expiration time.Time) error {
	return nil // NOOP
}

func (o *Oracle) RevokeUser(ctx context.Context, statements dbplugin.Statements, username string) error {
	// Grab the lock
	o.Lock()
	defer o.Unlock()

	// Get the connection
	db, err := o.getConnection(ctx)
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	// Effectively a no-op if the transaction commits successfully
	defer tx.Rollback()

	if err := o.disconnectSession(db, username); err != nil {
		return err
	}

	statements = dbutil.StatementCompatibilityHelper(statements)
	revocationStatements := statements.Revocation
	if len(revocationStatements) == 0 {
		revocationStatements = []string{revocationSQL}
	}

	// We can't use a transaction here, because Oracle treats DROP USER as a DDL statement, which commits immediately.
	// Execute each query
	for _, stmt := range revocationStatements {
		for _, query := range strutil.ParseArbitraryStringSlice(stmt, ";") {
			query = strings.TrimSpace(query)
			if len(query) == 0 {
				continue
			}

			m := map[string]string{
				"name": username,
			}

			if err := dbtxn.ExecuteTxQuery(ctx, tx, m, query); err != nil {
				return err
			}
		}
	}

	return nil
}

func (o *Oracle) RotateRootCredentials(ctx context.Context, statements []string) (map[string]interface{}, error) {
	o.Lock()
	defer o.Unlock()

	if len(o.Username) == 0 || len(o.Password) == 0 {
		return nil, errors.New("username and password are required to rotate")
	}

	rotateStatements := statements
	if len(rotateStatements) == 0 {
		rotateStatements = []string{defaultRotateCredsSql}
	}

	db, err := o.getConnection(ctx)
	if err != nil {
		return nil, err
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	// Effectively a no-op if the transaction commits successfully
	defer tx.Rollback()

	password, err := o.GeneratePassword()
	if err != nil {
		return nil, err
	}

	for _, stmt := range rotateStatements {
		for _, query := range strutil.ParseArbitraryStringSlice(stmt, ";") {
			query = strings.TrimSpace(query)
			if len(query) == 0 {
				continue
			}

			m := map[string]string{
				"username": o.Username,
				"password": password,
			}

			if err := dbtxn.ExecuteTxQuery(ctx, tx, m, query); err != nil {
				return nil, err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	if err := db.Close(); err != nil {
		return nil, err
	}

	o.RawConfig["password"] = password
	return o.RawConfig, nil
}

func (o *Oracle) SetCredentials(ctx context.Context, statements dbplugin.Statements, staticUser dbplugin.StaticUserConfig) (username, password string, err error) {
	rotateStatements := statements.Rotation
	if len(rotateStatements) == 0 {
		rotateStatements = []string{defaultRotateCredsSql}
	}

	username = staticUser.Username
	password = staticUser.Password
	if username == "" || password == "" {
		return "", "", errors.New("must provide both username and password")
	}

	variables := map[string]string{
		"username": username,
		"password": password,
	}

	queries := splitQueries(rotateStatements)
	if len(queries) == 0 { // Extra check to protect against future changes
		return "", "", errors.New("no rotation queries found")
	}

	// Lock the SQL connection
	o.Lock()
	defer o.Unlock()

	db, err := o.getConnection(ctx)
	if err != nil {
		return "", "", fmt.Errorf("unable to get database connection: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return "", "", fmt.Errorf("unable to create database transaction: %w", err)
	}
	// Effectively a no-op if the transaction commits successfully
	defer tx.Rollback()

	for _, rawQuery := range queries {
		parsedQuery := dbutil.QueryHelper(rawQuery, variables)
		err := dbtxn.ExecuteTxQuery(ctx, tx, nil, parsedQuery)
		if err != nil {
			return "", "", fmt.Errorf("unable to execute rotation query [%s]: %w", parsedQuery, err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return "", "", fmt.Errorf("unable to commit rotation queries: %w", err)
	}

	return username, password, nil
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
		"name": username,
	}
	disconnectQuery := dbutil.QueryHelper(`SELECT sid, serial#, username FROM v$session WHERE username = UPPER('{{name}}')`, disconnectVars)
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
