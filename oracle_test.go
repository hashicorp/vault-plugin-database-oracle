package oracle

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"testing"
	"time"

	dbplugin "github.com/hashicorp/vault/sdk/database/dbplugin/v5"
	dbtesting "github.com/hashicorp/vault/sdk/database/dbplugin/v5/testing"
	"github.com/tgulacsi/go/orahlp"
	dockertest "gopkg.in/ory-am/dockertest.v3"
)

const (
	defaultUser     = "system"
	defaultPassword = "oracle"
)

func getRequestTimeout(t *testing.T) time.Duration {
	rawDur := os.Getenv("VAULT_TEST_DATABASE_REQUEST_TIMEOUT")
	if rawDur == "" {
		return 2 * time.Second
	}

	dur, err := time.ParseDuration(rawDur)
	if err != nil {
		t.Fatalf("Failed to parse custom request timeout %q: %s", rawDur, err)
	}
	return dur
}

func prepareOracleTestContainer(t *testing.T) (connString string, cleanup func()) {
	if os.Getenv("ORACLE_DSN") != "" {
		return os.Getenv("ORACLE_DSN"), func() {}
	}

	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Fatalf("Failed to connect to docker: %s", err)
	}

	resource, err := pool.Run("wnameless/oracle-xe-11g-r2", "latest", []string{})
	if err != nil {
		t.Fatalf("Could not start local Oracle docker container: %s", err)
	}

	cleanup = func() {
		err := pool.Purge(resource)
		if err != nil {
			t.Fatalf("Failed to cleanup local container: %s", err)
		}
	}

	// If we are running these tests inside the cross-image build container,
	// then we need to use the ip address and port of the oracle container.
	// We can't use the container ip on Docker for Mac so default to localhost.
	var url string
	switch os.Getenv("RUN_IN_CONTAINER") {
	case "":
		url = resource.GetHostPort("1521/tcp")
	default:
		url = resource.Container.NetworkSettings.Networks["bridge"].IPAddress + ":" + "1521"
	}

	connString = fmt.Sprintf("%s/%s@%s/xe", defaultUser, defaultPassword, url)

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

	return connString, cleanup
}

func TestOracle_Initialize(t *testing.T) {
	connURL, cleanup := prepareOracleTestContainer(t)
	t.Cleanup(cleanup)

	db := new()
	defer dbtesting.AssertClose(t, db)

	expectedConfig := map[string]interface{}{
		"connection_url": connURL,
	}
	req := dbplugin.InitializeRequest{
		Config: map[string]interface{}{
			"connection_url": connURL,
		},
		VerifyConnection: true,
	}
	resp := dbtesting.AssertInitialize(t, db, req)
	if !reflect.DeepEqual(resp.Config, expectedConfig) {
		t.Fatalf("Actual: %#v\nExpected: %#v", resp.Config, expectedConfig)
	}

	connProducer := db.SQLConnectionProducer
	if !connProducer.Initialized {
		t.Fatal("Database should be initialized")
	}
}

func TestOracle_NewUser(t *testing.T) {
	type testCase struct {
		displayName           string
		roleName              string
		creationStmts         []string
		usernameTemplate      string
		expectErr             bool
		expectedUsernameRegex string
	}

	tests := map[string]testCase{
		"name creation": {
			displayName: "token",
			roleName:    "myrolenamewithextracharacters",
			creationStmts: []string{`
				CREATE USER {{name}} IDENTIFIED BY "{{password}}";
				GRANT CONNECT TO {{name}};
				GRANT CREATE SESSION TO {{name}};`,
			},
			expectErr:             false,
			expectedUsernameRegex: `^V_TOKEN_MYROLENA_[A-Z0-9]{13}$`,
		},
		"username creation": {
			displayName: "token",
			roleName:    "myrolenamewithextracharacters",
			creationStmts: []string{`
				CREATE USER {{username}} IDENTIFIED BY "{{password}}";
				GRANT CONNECT TO {{username}};
				GRANT CREATE SESSION TO {{username}};`,
			},
			expectErr:             false,
			expectedUsernameRegex: `^V_TOKEN_MYROLENA_[A-Z0-9]{13}$`,
		},
		"default_username_template": {
			displayName: "token-withadisplayname",
			roleName:    "areallylongrolenamewithmanycharacters",
			creationStmts: []string{`
				CREATE USER {{username}} IDENTIFIED BY "{{password}}";
				GRANT CONNECT TO {{username}};
				GRANT CREATE SESSION TO {{username}};`,
			},
			expectErr:             false,
			expectedUsernameRegex: `^V_TOKEN_WI_AREALLYL_[A-Z0-9]{10}$`,
		},
		"custom username_template": {
			displayName: "token",
			roleName:    "myrolenamewithextracharacters",
			creationStmts: []string{`
				CREATE USER {{username}} IDENTIFIED BY "{{password}}";
				GRANT CONNECT TO {{username}};
				GRANT CREATE SESSION TO {{username}};`,
			},
			usernameTemplate:      "{{random 8 | uppercase}}_{{.RoleName | uppercase | truncate 10}}_{{.DisplayName | sha256 | uppercase | truncate 10}}",
			expectErr:             false,
			expectedUsernameRegex: `^[A-Z0-9]{8}_MYROLENAME_3C469E9D6C$`,
		},
		"empty creation": {
			displayName:           "token",
			roleName:              "myrolenamewithextracharacters",
			creationStmts:         []string{},
			expectErr:             true,
			expectedUsernameRegex: `^$`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			connURL, cleanup := prepareOracleTestContainer(t)
			t.Cleanup(cleanup)

			db := new()
			defer dbtesting.AssertClose(t, db)

			initReq := dbplugin.InitializeRequest{
				Config: map[string]interface{}{
					"connection_url":    connURL,
					"username_template": test.usernameTemplate,
				},
				VerifyConnection: true,
			}
			dbtesting.AssertInitialize(t, db, initReq)

			password := "y8fva_sdVA3rasf"

			createReq := dbplugin.NewUserRequest{
				UsernameConfig: dbplugin.UsernameMetadata{
					DisplayName: test.displayName,
					RoleName:    test.roleName,
				},
				Statements: dbplugin.Statements{
					Commands: test.creationStmts,
				},
				Password:   password,
				Expiration: time.Time{},
			}

			ctx, cancel := context.WithTimeout(context.Background(), getRequestTimeout(t))
			defer cancel()

			createResp, err := db.NewUser(ctx, createReq)
			if test.expectErr && err == nil {
				t.Fatalf("err expected, got nil")
			}
			if !test.expectErr && err != nil {
				t.Fatalf("no error expected, got: %s", err)
			}
			re := regexp.MustCompile(test.expectedUsernameRegex)
			if !re.MatchString(createResp.Username) {
				t.Fatalf("Username [%s] does not match regex [%s]", createResp.Username, test.expectedUsernameRegex)
			}

			err = testCredentialsExist(connURL, createResp.Username, password)
			if test.expectErr && err == nil {
				t.Fatalf("err expected, got nil")
			}
			if !test.expectErr && err != nil {
				t.Fatalf("no error expected, got: %s", err)
			}
		})
	}
}

func TestOracle_RenewUser(t *testing.T) {
	connURL, cleanup := prepareOracleTestContainer(t)
	t.Cleanup(cleanup)

	db := new()
	defer dbtesting.AssertClose(t, db)

	initReq := dbplugin.InitializeRequest{
		Config: map[string]interface{}{
			"connection_url": connURL,
		},
		VerifyConnection: true,
	}
	dbtesting.AssertInitialize(t, db, initReq)

	password := "y8fva_sdVA3rasf"

	createReq := dbplugin.NewUserRequest{
		UsernameConfig: dbplugin.UsernameMetadata{
			DisplayName: "test",
			RoleName:    "test",
		},
		Statements: dbplugin.Statements{
			Commands: []string{`
				CREATE USER {{name}} IDENTIFIED BY {{password}};
				GRANT CONNECT TO {{name}};
				GRANT CREATE SESSION TO {{name}};`,
			},
		},
		Password:   password,
		Expiration: time.Now().Add(2 * time.Second),
	}

	createResp := dbtesting.AssertNewUser(t, db, createReq)

	assertCredentialsExist(t, connURL, createResp.Username, password)

	renewReq := dbplugin.UpdateUserRequest{
		Username: createResp.Username,
		Expiration: &dbplugin.ChangeExpiration{
			NewExpiration: time.Now().Add(time.Minute),
		},
	}

	dbtesting.AssertUpdateUser(t, db, renewReq)

	// Sleep longer than the initial expiration time
	time.Sleep(2 * time.Second)

	assertCredentialsExist(t, connURL, createResp.Username, password)
}

func TestOracle_RevokeUser(t *testing.T) {
	connURL, cleanup := prepareOracleTestContainer(t)
	t.Cleanup(cleanup)

	type testCase struct {
		deleteStatements []string
	}

	tests := map[string]testCase{
		"name revoke": {
			deleteStatements: []string{`
				REVOKE CONNECT FROM {{name}};
				REVOKE CREATE SESSION FROM {{name}};
				DROP USER {{name}};`,
			},
		},
		"username revoke": {
			deleteStatements: []string{`
				REVOKE CONNECT FROM {{username}};
				REVOKE CREATE SESSION FROM {{username}};
				DROP USER {{username}};`,
			},
		},
		"default revoke": {},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			db := new()
			defer dbtesting.AssertClose(t, db)

			initReq := dbplugin.InitializeRequest{
				Config: map[string]interface{}{
					"connection_url": connURL,
				},
				VerifyConnection: true,
			}
			dbtesting.AssertInitialize(t, db, initReq)

			password := "y8fva_sdVA3rasf"

			createReq := dbplugin.NewUserRequest{
				UsernameConfig: dbplugin.UsernameMetadata{
					DisplayName: "test",
					RoleName:    "test",
				},
				Statements: dbplugin.Statements{
					Commands: []string{`
						CREATE USER {{name}} IDENTIFIED BY {{password}};
						GRANT CONNECT TO {{name}};
						GRANT CREATE SESSION TO {{name}};`,
					},
				},
				Password:   password,
				Expiration: time.Now().Add(2 * time.Second),
			}

			createResp := dbtesting.AssertNewUser(t, db, createReq)

			assertCredentialsExist(t, connURL, createResp.Username, password)

			deleteReq := dbplugin.DeleteUserRequest{
				Username: createResp.Username,
				Statements: dbplugin.Statements{
					Commands: test.deleteStatements,
				},
			}
			dbtesting.AssertDeleteUser(t, db, deleteReq)
			assertCredentialsDoNotExist(t, connURL, createResp.Username, password)
		})
	}
}

func TestSplitQueries(t *testing.T) {
	type testCase struct {
		input    []string
		expected []string
	}

	tests := map[string]testCase{
		"nil input": {
			input:    nil,
			expected: nil,
		},
		"empty input": {
			input:    []string{},
			expected: nil,
		},
		"empty string": {
			input:    []string{""},
			expected: nil,
		},
		"string with only semicolon": {
			input:    []string{";"},
			expected: nil,
		},
		"only semicolons": {
			input:    []string{";;;;"},
			expected: nil,
		},
		"single input": {
			input: []string{
				"alter user {{username}} identified by {{password}}",
			},
			expected: []string{
				"alter user {{username}} identified by {{password}}",
			},
		},
		"single input with trailing semicolon": {
			input: []string{
				"alter user {{username}} identified by {{password}};",
			},
			expected: []string{
				"alter user {{username}} identified by {{password}}",
			},
		},
		"single input with leading semicolon": {
			input: []string{
				";alter user {{username}} identified by {{password}}",
			},
			expected: []string{
				"alter user {{username}} identified by {{password}}",
			},
		},
		"multiple queries in single line": {
			input: []string{
				"alter user {{username}} identified by {{password}};do something with {{username}} {{password}};",
			},
			expected: []string{
				"alter user {{username}} identified by {{password}}",
				"do something with {{username}} {{password}}",
			},
		},
		"multiple queries in multiple lines": {
			input: []string{
				"foo;bar;baz",
				"qux ; quux ; quuz",
			},
			expected: []string{
				"foo",
				"bar",
				"baz",
				"qux",
				"quux",
				"quuz",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			actual := splitQueries(test.input)

			if !reflect.DeepEqual(actual, test.expected) {
				t.FailNow()
			}
		})
	}
}

func TestUpdateUser_ChangePassword(t *testing.T) {
	username := "TESTUSER"
	initialPassword := "myreallysecurepassword"

	type testCase struct {
		req dbplugin.UpdateUserRequest

		expectedPassword string
		expectErr        bool
	}

	tests := map[string]testCase{
		"missing username": {
			req: dbplugin.UpdateUserRequest{
				Username: "",
				Password: &dbplugin.ChangePassword{
					NewPassword: "newpassword",
				},
			},
			expectedPassword: initialPassword,
			expectErr:        true,
		},
		"missing password": {
			req: dbplugin.UpdateUserRequest{
				Username: username,
			},
			expectedPassword: initialPassword,
			expectErr:        true,
		},
		"missing username and password": {
			req:              dbplugin.UpdateUserRequest{},
			expectedPassword: initialPassword,
			expectErr:        true,
		},
		"happy path": {
			req: dbplugin.UpdateUserRequest{
				Username: username,
				Password: &dbplugin.ChangePassword{
					NewPassword: "somenewpassword",
				},
			},
			expectedPassword: "somenewpassword",
			expectErr:        false,
		},
		"bad statements": {
			req: dbplugin.UpdateUserRequest{
				Username: username,
				Password: &dbplugin.ChangePassword{
					NewPassword: "somenewpassword",
					Statements: dbplugin.Statements{
						Commands: []string{
							"foo bar",
						},
					},
				},
			},
			expectedPassword: initialPassword,
			expectErr:        true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			connURL, cleanup := prepareOracleTestContainer(t)
			t.Cleanup(cleanup)

			db := new()

			initReq := dbplugin.InitializeRequest{
				Config: map[string]interface{}{
					"connection_url": connURL,
				},
				VerifyConnection: true,
			}
			dbtesting.AssertInitialize(t, db, initReq)

			// Manually create a user since we need to know the username ahead of time when we
			// declare the test cases above
			ctx, cancel := context.WithTimeout(context.Background(), getRequestTimeout(t))
			defer cancel()

			sqlDB, err := db.getConnection(ctx)
			if err != nil {
				t.Fatalf("unable to get connection to database: %s", err)
			}

			// Create the user manually so we can manipulate it
			createCommands := []string{
				`CREATE USER "{{username}}" IDENTIFIED BY "{{password}}"`,
				`GRANT ALL PRIVILEGES TO {{username}}`,
			}
			err = newUser(ctx, sqlDB, username, initialPassword, time.Now().Add(1*time.Minute), createCommands)
			if err != nil {
				t.Fatalf("failed to create user: %s", err)
			}

			assertCredentialsExist(t, connURL, username, initialPassword)

			ctx, cancel = context.WithTimeout(context.Background(), getRequestTimeout(t))
			defer cancel()

			_, err = db.UpdateUser(ctx, test.req)
			if test.expectErr && err == nil {
				t.Fatalf("err expected, got nil")
			}
			if !test.expectErr && err != nil {
				t.Fatalf("no error expected, got: %s", err)
			}

			assertCredentialsExist(t, connURL, username, test.expectedPassword)
		})
	}
}

func testCredentialsExist(connString, username, password string) error {
	// Log in with the new credentials
	_, _, link := orahlp.SplitDSN(connString)
	connURL := fmt.Sprintf("%s/%s@%s", username, password, link)

	db, err := sql.Open("oci8", connURL)
	if err != nil {
		return err
	}
	defer db.Close()
	return db.Ping()
}

func assertCredentialsExist(t *testing.T, connString, username, password string) {
	t.Helper()
	err := testCredentialsExist(connString, username, password)
	if err != nil {
		t.Fatalf("failed to login: %s", err)
	}
}

func assertCredentialsDoNotExist(t *testing.T, connString, username, password string) {
	t.Helper()
	err := testCredentialsExist(connString, username, password)
	if err == nil {
		t.Fatalf("logged in when it shouldn't have been able to")
	}
}
