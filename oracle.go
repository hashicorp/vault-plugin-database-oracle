package oracle

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-oci8"

	"github.com/hashicorp/vault/builtin/logical/database/dbplugin"
	"github.com/hashicorp/vault/helper/strutil"
	"github.com/hashicorp/vault/plugins/helper/database/connutil"
	"github.com/hashicorp/vault/plugins/helper/database/credsutil"
	"github.com/hashicorp/vault/plugins/helper/database/dbutil"
)

const typeName string = "oci8"

const revocationSQL = `
REVOKE CONNECT FROM {{name}};
REVOKE CREATE SESSION FROM {{name}};
DROP USER {{name}};
`

const sessionQuerySQL = `SELECT sid, serial#, username FROM v$session WHERE username = UPPER('{{name}}')`

const sessionKillSQL = `ALTER SYSTEM KILL SESSION '%d,%d' IMMEDIATE`

type Oracle struct {
	connutil.ConnectionProducer
	credsutil.CredentialsProducer
}

func New() *Oracle {
	connProducer := &connutil.SQLConnectionProducer{}
	connProducer.Type = typeName

	credsProducer := &oracleCredentialsProducer{}

	dbType := &Oracle{
		ConnectionProducer:  connProducer,
		CredentialsProducer: credsProducer,
	}

	return dbType
}

func (o *Oracle) Type() (string, error) {
	return typeName, nil
}

func (o *Oracle) CreateUser(statements dbplugin.Statements, usernamePrefix string, expiration time.Time) (username string, password string, err error) {
	if statements.CreationStatements == "" {
		return "", "", dbutil.ErrEmptyCreationStatement
	}

	// Grab the lock
	o.Lock()
	defer o.Unlock()

	username, err = o.GenerateUsername(usernamePrefix)
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
	db, err := o.getConnection()
	if err != nil {
		return "", "", err

	}

	// Start a transaction
	tx, err := db.Begin()
	if err != nil {
		return "", "", err

	}
	defer func() {
		tx.Rollback()
	}()
	// Return the secret

	// Execute each query
	for _, query := range strutil.ParseArbitraryStringSlice(statements.CreationStatements, ";") {
		query = strings.TrimSpace(query)
		if len(query) == 0 {
			continue
		}

		stmt, err := tx.Prepare(dbutil.QueryHelper(query, map[string]string{
			"name":       username,
			"password":   password,
			"expiration": expirationStr,
		}))
		if err != nil {
			return "", "", err

		}
		defer stmt.Close()
		if _, err := stmt.Exec(); err != nil {
			return "", "", err

		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return "", "", err

	}

	return username, password, nil
}

// NOOP
func (o *Oracle) RenewUser(statements dbplugin.Statements, username string, expiration time.Time) error {
	return nil
}

func (o *Oracle) RevokeUser(statements dbplugin.Statements, username string) error {
	// Grab the lock
	o.Lock()
	defer o.Unlock()

	// Get the connection
	db, err := o.getConnection()
	if err != nil {
		return err
	}

	// Disconnect the session
	disconnectStmt, err := db.Prepare(strings.Replace(sessionQuerySQL, "{{name}}", username, -1))
	if err != nil {
		return err
	}
	defer disconnectStmt.Close()
	if rows, err := disconnectStmt.Query(); err != nil {
		return err
	} else {
		defer rows.Close()
		for rows.Next() {
			var sessionId, serialNumber int
			var username sql.NullString
			err = rows.Scan(&sessionId, &serialNumber, &username)
			if err != nil {
				return err
			}
			killStatement := fmt.Sprintf(sessionKillSQL, sessionId, serialNumber)
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

	// We can't use a transaction here, because Oracle treats DROP USER as a DDL statement, which commits immediately.
	// Execute each query
	for _, query := range strutil.ParseArbitraryStringSlice(revocationSQL, ";") {
		query = strings.TrimSpace(query)
		if len(query) == 0 {
			continue
		}

		stmt, err := db.Prepare(strings.Replace(query, "{{name}}", username, -1))
		if err != nil {
			return err
		}
		defer stmt.Close()
		if _, err := stmt.Exec(); err != nil {
			return err
		}
	}

	return nil
}

func (o *Oracle) getConnection() (*sql.DB, error) {
	db, err := o.Connection()
	if err != nil {
		return nil, err
	}

	return db.(*sql.DB), nil
}
