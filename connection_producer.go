// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package oracle

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/go-secure-stdlib/parseutil"
	"github.com/hashicorp/vault/sdk/database/helper/dbutil"
	go_ora "github.com/sijms/go-ora/v2"
	"log"
	"net/url"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/hashicorp/vault/sdk/database/helper/connutil"
	"github.com/mitchellh/mapstructure"
)

var (
	ErrInvalidOracleURL                 = fmt.Errorf("invalid connection URL format, expect username/password@//{{server}}:{{port}}/{{DBName}}")
	serverPortAndDBNameFromConnURLRegex = regexp.MustCompile(`^.+@\/\/(.+):(.+)\/(.+)$`) // Expected format: username/password@//{{server}}:{{port}}/{{DBName}}
)

type oracleConnectionProducer struct {
	ConnectionURL            string      `json:"connection_url"`
	MaxOpenConnections       int         `json:"max_open_connections"`
	MaxIdleConnections       int         `json:"max_idle_connections"`
	MaxConnectionLifetimeRaw interface{} `json:"max_connection_lifetime"`
	Username                 string      `json:"username"`
	Password                 string      `json:"password"`
	PrivateKey               []byte      `json:"private_key"`
	UsernameTemplate         string      `json:"username_template"`
	DisableEscaping          bool        `json:"disable_escaping"`

	Initialized           bool
	RawConfig             map[string]any
	Type                  string
	maxConnectionLifetime time.Duration
	db                    *sql.DB
	mu                    sync.RWMutex
}

func (c *oracleConnectionProducer) secretValues() map[string]string {
	return map[string]string{
		c.Password: "[password]",
	}
}

func (c *oracleConnectionProducer) Init(ctx context.Context, initConfig map[string]interface{}, verifyConnection bool) (saveConfig map[string]interface{}, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.RawConfig = initConfig

	decoderConfig := &mapstructure.DecoderConfig{
		Result:           c,
		WeaklyTypedInput: true,
		TagName:          "json",
	}

	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return nil, err
	}

	err = decoder.Decode(initConfig)
	if err != nil {
		return nil, err
	}

	if len(c.ConnectionURL) == 0 {
		return nil, fmt.Errorf("connection_url cannot be empty")
	}

	username := c.Username
	password := c.Password

	if !c.DisableEscaping {
		username = url.PathEscape(c.Username)
		password = url.PathEscape(c.Password)
	}

	// Replace templated username and password in connection URL with actual values
	c.ConnectionURL = dbutil.QueryHelper(c.ConnectionURL, map[string]string{
		"username": username,
		"password": password,
	})

	server, port, dbName, err := parseOracleFieldsFromURL(c.ConnectionURL)
	if err != nil {
		return nil, fmt.Errorf("error parsing connection URL: %w", err)
	}

	log.Printf("[ORACLE TEST] info server=%s, port=%s, dbName=%s", server, port, dbName)

	p, err := strconv.Atoi(port)
	if err != nil {
		return nil, fmt.Errorf("error parsing port: %w", err)
	}
	c.ConnectionURL = go_ora.BuildUrl(server, p, dbName, username, password, nil)
	log.Printf("[ORACLE TEST] final connection URL=%s", c.ConnectionURL)

	if c.MaxOpenConnections == 0 {
		c.MaxOpenConnections = 4
	}

	if c.MaxIdleConnections == 0 {
		c.MaxIdleConnections = c.MaxOpenConnections
	}
	if c.MaxIdleConnections > c.MaxOpenConnections {
		c.MaxIdleConnections = c.MaxOpenConnections
	}
	if c.MaxConnectionLifetimeRaw == nil {
		c.MaxConnectionLifetimeRaw = "0s"
	}

	c.maxConnectionLifetime, err = parseutil.ParseDurationSecond(c.MaxConnectionLifetimeRaw)
	if err != nil {
		return nil, errwrap.Wrapf("invalid max_connection_lifetime: {{err}}", err)
	}

	c.Initialized = true

	if verifyConnection {
		if _, err := c.Connection(ctx); err != nil {
			c.close()
			return nil, fmt.Errorf("error verifying connection: %w", err)
		}

		if err := c.db.PingContext(ctx); err != nil {
			return nil, errwrap.Wrapf("error verifying connection: ping failed: {{err}}", err)
		}
	}

	return initConfig, nil
}

func (c *oracleConnectionProducer) Initialize(ctx context.Context, config map[string]any, verifyConnection bool) error {
	_, err := c.Init(ctx, config, verifyConnection)
	return err
}

func (c *oracleConnectionProducer) Connection(ctx context.Context) (interface{}, error) {
	// This is intentionally not grabbing the lock since the calling functions (e.g. CreateUser)
	// are claiming it.

	if !c.Initialized {
		return nil, connutil.ErrNotInitialized
	}

	if c.db != nil {
		log.Printf("[ORACLE TEST] oracle connection already open")
		return c.db, nil
	}

	var db *sql.DB
	var err error
	db, err = sql.Open(oracleTypeName, c.ConnectionURL)
	if err != nil {
		return nil, fmt.Errorf("error opening oracle connection using user-pass auth: %w", err)
	}

	c.db = db
	c.db.SetMaxOpenConns(c.MaxOpenConnections)
	c.db.SetMaxIdleConns(c.MaxIdleConnections)
	c.db.SetConnMaxLifetime(c.maxConnectionLifetime)

	return c.db, nil
}

// close terminates the database connection without locking
func (c *oracleConnectionProducer) close() error {
	if c.db != nil {
		if err := c.db.Close(); err != nil {
			return err
		}
	}

	return nil
}

// Close terminates the database connection with locking
func (c *oracleConnectionProducer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.db = nil

	return c.close()
}

// parseOracleFieldsFromURL uses a regex to extract account and DB
// info from a connectionURL
func parseOracleFieldsFromURL(connectionURL string) (string, string, string, error) {
	if !serverPortAndDBNameFromConnURLRegex.MatchString(connectionURL) {
		return "", "", "", ErrInvalidOracleURL
	}
	res := serverPortAndDBNameFromConnURLRegex.FindStringSubmatch(connectionURL)
	if len(res) != 4 {
		return "", "", "", ErrInvalidOracleURL
	}

	return res[1], res[2], res[3], nil
}
