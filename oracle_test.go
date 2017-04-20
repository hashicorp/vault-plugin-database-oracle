package oracle

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

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
