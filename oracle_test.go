package oracle

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/tgulacsi/go/orahlp"

	"github.com/hashicorp/vault/builtin/logical/database/dbplugin"
	"github.com/hashicorp/vault/plugins/helper/database/connutil"
	dockertest "gopkg.in/ory-am/dockertest.v3"
)

func prepareOracleTestContainer(t *testing.T) (cleanup func(), connString string) {
	if os.Getenv("ORACLE_DSN") != "" {
		return func() {}, os.Getenv("ORACLE_DSN")
	}

	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Fatalf("Failed to connect to docker: %s", err)
	}

	resource, err := pool.Run("wnameless/oracle-xe-11g", "latest", []string{})
	if err != nil {
		t.Fatalf("Could not start local MySQL docker container: %s", err)
	}

	cleanup = func() {
		err := pool.Purge(resource)
		if err != nil {
			t.Fatalf("Failed to cleanup local container: %s", err)
		}
	}

	connString = fmt.Sprintf("system/oracle@localhost:%s/xe", resource.GetPort("1521/tcp"))

	// exponential backoff-retry
	// the oracle container seems to take at least one minute to start, give us two
	pool.MaxWait = time.Minute * 2
	if err = pool.Retry(func() error {
		var err error
		var db *sql.DB
		db, err = sql.Open("oci8", connString)
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		t.Fatalf("Could not connect to Oracle docker container: %s", err)
	}

	return
}

func TestOracle_Initialize(t *testing.T) {
	cleanup, connURL := prepareOracleTestContainer(t)
	defer cleanup()

	connectionDetails := map[string]interface{}{
		"connection_url": connURL,
	}

	db := New()
	connProducer := db.ConnectionProducer.(*connutil.SQLConnectionProducer)

	err := db.Initialize(connectionDetails, true)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !connProducer.Initialized {
		t.Fatal("Database should be initalized")
	}

	err = db.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestOracle_CreateUser(t *testing.T) {
	cleanup, connURL := prepareOracleTestContainer(t)
	defer cleanup()

	connectionDetails := map[string]interface{}{
		"connection_url": connURL,
	}

	db := New()
	err := db.Initialize(connectionDetails, true)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Test with no configured Creation Statememt
	_, _, err = db.CreateUser(dbplugin.Statements{}, "test", time.Now().Add(time.Minute))
	if err == nil {
		t.Fatal("Expected error when no creation statement is provided")
	}

	statements := dbplugin.Statements{
		CreationStatements: testRole,
	}

	username, password, err := db.CreateUser(statements, "test", time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if err = testCredsExist(t, connURL, username, password); err != nil {
		t.Fatalf("Could not connect with new credentials: %s", err)
	}
}

func testCredsExist(t testing.TB, connString, username, password string) error {
	// Log in with the new creds
	_, _, link := orahlp.SplitDSN(connString)
	connURL := fmt.Sprintf("%s/%s@%s", username, password, link)

	db, err := sql.Open("oci8", connURL)
	if err != nil {
		return err
	}
	defer db.Close()
	return db.Ping()
}

const testRole = `
CREATE USER {{name}} IDENTIFIED BY {{password}};
GRANT CONNECT TO {{name}};
GRANT CREATE SESSION TO {{name}};
`
