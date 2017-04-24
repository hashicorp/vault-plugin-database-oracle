package oracle

import (
	"database/sql"
	"strings"
	"time"

	_ "github.com/mattn/go-oci8"

	"github.com/hashicorp/vault/builtin/logical/database/dbplugin"
	"github.com/hashicorp/vault/helper/strutil"
	"github.com/hashicorp/vault/plugins/helper/database/connutil"
	"github.com/hashicorp/vault/plugins/helper/database/credsutil"
	"github.com/hashicorp/vault/plugins/helper/database/dbutil"
)

const oracleTypeName string = "oci8"

type Oracle struct {
	connutil.ConnectionProducer
	credsutil.CredentialsProducer
}

func New() *Oracle {
	connProducer := &connutil.SQLConnectionProducer{}
	connProducer.Type = oracleTypeName

	credsProducer := &oracleCredentialsProducer{}

	dbType := &Oracle{
		ConnectionProducer:  connProducer,
		CredentialsProducer: credsProducer,
	}

	return dbType
}

func (p *Oracle) CreateUser(statements dbplugin.Statements, usernamePrefix string, expiration time.Time) (username string, password string, err error) {
	if statements.CreationStatements == "" {
		return "", "", dbutil.ErrEmptyCreationStatement
	}

	// Grab the lock
	p.Lock()
	defer p.Unlock()

	username, err = p.GenerateUsername(usernamePrefix)
	if err != nil {
		return "", "", err
	}

	password, err = p.GeneratePassword()
	if err != nil {
		return "", "", err
	}

	expirationStr, err := p.GenerateExpiration(expiration)
	if err != nil {
		return "", "", err
	}

	// Get the connection
	db, err := p.getConnection()
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

func (p *Oracle) getConnection() (*sql.DB, error) {
	db, err := p.Connection()
	if err != nil {
		return nil, err
	}

	return db.(*sql.DB), nil
}
