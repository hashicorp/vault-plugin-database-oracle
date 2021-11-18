#!/usr/bin/env bats

# Prerequisites
#
# 1. Install Bats Core: https://bats-core.readthedocs.io/en/stable/installation.html
# 2. Docker

# Setup
#
# 1. Oracle plugin is built and registered in vault
# 2. Oracle db docker image has been built
# 3. Oracle db data path is set in DOCKER_VOLUME_MNT
# 4. Export VAULT_LICENSE. This test will only work for enterprise images.
# 5. Export PLUGIN_DIR containing the path to the oracle plugin binary.

# Logs
#
# Vault logs will be written to VAULT_OUTFILE.
# BATs test logs will be written to SETUP_TEARDOWN_OUTFILE.

# vault
export VAULT_ADDR='http://127.0.0.1:8200'
VAULT_TOKEN='root'

SETUP_TEARDOWN_OUTFILE=/tmp/bats-test.log
VAULT_OUTFILE=/tmp/vault.log
VAULT_STARTUP_TIMEOUT=15
DB_NAME="my-oracle-db"
VPASS="myreallysecurepassword"
VROLE="my-role"
VAULT_USER="vaultadmin"
STATIC_USER="staticuser1"
STATIC_PASSWORD="staticpassword1"
DOCKER_VOLUME_MNT=${DOCKER_VOLUME_MNT:-~/dev/oracle/data/}

# error if these are not set
[ ${VAULT_LICENSE?} ]
[ ${PLUGIN_DIR?} ]

# jq is required
command -v jq >/dev/null 2>&1 || { log "jq is required for this test."; exit 1; }

# assert_status evaluates if `status` is equal to $1. If they are not equal a
# log is written to the output file. This makes use of the BATs `status` and
# `output` globals.
#
# Parameters:
#   expect
# Globals:
#   status
#   output
assert_status() {
  local expect
  expect="$1"

  [ "${status}" -eq "${expect}" ] || \
    log_err "bad status: expect: ${expect}, got: ${status}, output:\n${output}"
}

log() {
  echo "INFO: $(date): $@" >> $SETUP_TEARDOWN_OUTFILE
}

log_err() {
  echo -e "ERROR: $(date): [$BATS_TEST_NAME]: $@" >> $SETUP_TEARDOWN_OUTFILE
  exit 1
}

# assert_sqlplus_success asserts that the sqlplus query results in a successful
# response
#
# Globals:
#   output
assert_sqlplus_success() {
  local -r expected_output=$(cat - <<EOF

	 1
----------
	 1

EOF
)
  [[ "${output}" =~ "${expected_output}" ]] || \
    log_err "bad output: expect:\n${expected_output}\ngot:\n${output}"

}

# docker_sqlplus executes an Oracle sqlplus query.
#
# Parameters:
#   cmd
#   username
#   password
docker_sqlplus() {
  local -r cmd="$1"
  local -r username="$2"
  local -r password="$3"

  echo "${cmd}" | docker exec -i oracle sqlplus -S "${username}/${password}@ORCLPDB1"
}

# docker_sqlplus_admin executes an Oracle sqlplus query as sysdba. This should
# only be used for DB admin tasks such as creating a new static user.
#
# Parameters:
#   cmd
docker_sqlplus_admin() {
  local -r cmd="$1"
  echo "${cmd}" | docker exec -i oracle sqlplus -S "sys/$VPASS@ORCLPDB1" as sysdba
}

# setup_file runs once before all tests
setup_file(){
  # clear log file
  echo "" > $SETUP_TEARDOWN_OUTFILE
  echo "" > $VAULT_OUTFILE

  VAULT_TOKEN='root'

  log "BEGIN SETUP"

  docker rm oracle --force
  docker run \
    -d \
    --rm \
    --name oracle \
    -p 1521:1521 \
    -p 5500:5500 \
    -e ORACLE_SID=myservice \
    -e ORACLE_PWD=$VPASS \
    -v $DOCKER_VOLUME_MNT:/opt/oracle/oradata \
    oracle/database:19.3.0-se2

  log "waiting for oracle db..."
  while ! docker logs oracle | grep "DATABASE IS READY TO USE" > /dev/null; do sleep 1; done

  # setup vault user
  docker_sqlplus_admin "DROP USER $VAULT_USER;"
  docker_sqlplus_admin "CREATE USER $VAULT_USER IDENTIFIED BY \"$VPASS\";"
  docker_sqlplus_admin "GRANT ALL PRIVILEGES TO $VAULT_USER;"
  docker_sqlplus_admin "GRANT SELECT ON gv_\$SESSION TO $VAULT_USER;"

  ./vault server -dev -dev-root-token-id=root -log-level=trace \
    -dev-plugin-dir=${PLUGIN_DIR?} > $VAULT_OUTFILE 2>&1 &

  log "waiting for vault..."
  i=0
  while ! vault status >/dev/null 2>&1; do
    sleep 1
    ((i=i+1))
    [ $i -gt $VAULT_STARTUP_TIMEOUT ] && log_err "timed out waiting for vault to start"
  done

  vault login ${VAULT_TOKEN?}

  run vault status
  assert_status 0
  log "vault started successfully"

  vault namespace create ns1

  vault secrets enable --namespace=ns1 database
  log "vault enabled database secrets engine"

  log "END SETUP"
}

# teardown_file runs once after all tests complete
teardown_file(){
  log "BEGIN TEARDOWN"

  log "dropping the vault user from oracle db"
  docker_sqlplus_admin "DROP USER $VAULT_USER;"

  log "dropping the static user from oracle db"
  docker_sqlplus_admin "DROP USER $STATIC_USER;"

  log "removing the oracle docker container"
  docker rm oracle --force

  log "killing vault process"
  pkill vault

  log "END TEARDOWN"
}

@test "Read license" {
  run vault read -format=json sys/license/status
  assert_status 0
}

@test "POST /database/config/:name - write oracle connection config" {
  log "VAULT_NAMESPACE: $VAULT_NAMESPACE"
  run vault write --namespace=ns1 database/config/$DB_NAME \
    plugin_name=vault-plugin-database-oracle \
    connection_url="{{username}}/{{password}}@localhost:1521/ORCLPDB1" \
    allowed_roles="*" \
    username="$VAULT_USER" \
    password="$VPASS" \
    max_conneciton_lifetime="30s"
  assert_status 0
}

@test "GET /database/config/:name - read oracle connection config" {
  run vault read --namespace=ns1 database/config/$DB_NAME
  assert_status 0
}

@test "LIST /database/config - list configs" {
  run vault list --namespace=ns1 database/config
  assert_status 0
}

@test "DELETE /database/config/:name - delete oracle connection config" {
  run vault write --namespace=ns1 database/config/delete-me \
    plugin_name=vault-plugin-database-oracle \
    connection_url="{{username}}/{{password}}@localhost:1521/ORCLPDB1" \
    allowed_roles="*" \
    username="$VAULT_USER" \
    password="$VPASS" \
    max_conneciton_lifetime="30s"
  assert_status 0

  run vault delete --namespace=ns1 database/config/delete-me
  assert_status 0
}

@test "POST /database/reset/:name - reset oracle connection" {
  run vault write --namespace=ns1 -force database/reset/$DB_NAME
  assert_status 0
}

@test "POST /database/roles/:name - write role" {
  run vault write --namespace=ns1 database/roles/$VROLE \
    db_name=$DB_NAME \
    creation_statements='CREATE USER "{{username}}" IDENTIFIED BY "{{password}}"' \
    creation_statements='GRANT CONNECT TO "{{username}}"' \
    revocation_statements='DROP USER "{{username}}"' \
    default_ttl="1h" \
    max_ttl="1h"
  assert_status 0
}

@test "GET /database/roles/:name - read role" {
  run vault read --namespace=ns1 database/roles/$VROLE
  assert_status 0
}

@test "LIST /database/roles - list roles" {
  run vault list --namespace=ns1 database/roles
  assert_status 0
}

@test "DELETE /database/roles/:name - delete role" {
  run vault write --namespace=ns1 database/roles/delete-me \
    db_name=$DB_NAME \
    creation_statements='CREATE USER "{{username}}" IDENTIFIED BY "{{password}}"' \
    creation_statements='GRANT CONNECT TO "{{username}}"' \
    revocation_statements='DROP USER "{{username}}"' \
    default_ttl="1h" \
    max_ttl="1h"
  assert_status 0

  run vault delete --namespace=ns1 database/roles/delete-me
  assert_status 0
}

@test "GET /database/creds/:name - generate dynamic credentials" {
  run vault read --namespace=ns1 database/creds/$VROLE
  assert_status 0
}

@test "Execute sqlplus query with dynamic credentials" {
  # Get some dynamic creds
  local -r CREDS="$(vault read --namespace=ns1 -format=json database/creds/$VROLE)"
  local -r DYNAMIC_USER="$(echo $CREDS | jq -r '.data.username')"
  local -r DYNAMIC_PASSWORD="$(echo $CREDS | jq -r '.data.password')"
  local -r LEASE_ID="$(echo $CREDS | jq -r '.lease_id')"

  # Run a query with the creds
  run docker_sqlplus "select 1 from dual;" "$DYNAMIC_USER" "$DYNAMIC_PASSWORD"
  assert_sqlplus_success
  assert_status 0

  # Lookup and renew lease
  run vault lease lookup --namespace=ns1 "$LEASE_ID"
  assert_status 0

  run vault lease renew --namespace=ns1 "$LEASE_ID"
  assert_status 0

  run vault lease lookup --namespace=ns1 "$LEASE_ID"
  assert_status 0

  # Run a query with the creds
  run docker_sqlplus "select 1 from dual;" "$DYNAMIC_USER" "$DYNAMIC_PASSWORD"
  assert_sqlplus_success
  assert_status 0

  # Revoke credendtials
  run vault lease revoke --namespace=ns1 "$LEASE_ID"
  assert_status 0

  run docker_sqlplus "select 1 from dual;" "$DYNAMIC_USER" "$DYNAMIC_PASSWORD"
  [[ "${output}" =~ "ORA-01017: invalid username/password; logon denied" ]]
}

@test "Execute sqlplus query with static credentials" {
  # create static user
  docker_sqlplus_admin "CREATE USER $STATIC_USER IDENTIFIED BY \"$STATIC_PASSWORD\";"
  docker_sqlplus_admin "GRANT CONNECT TO $STATIC_USER;"
  # docker_sqlplus_admin 'GRANT CREATE SESSION TO $STATICUSER;'

  # Try the static password directly
  run docker_sqlplus "select 1 from dual;" "$STATIC_USER" "$STATIC_PASSWORD"
  assert_sqlplus_success
  assert_status 0

  # Create the static role
  run vault write --namespace=ns1 database/static-roles/my-static-role \
      username=$STATIC_USER \
      rotation_period=10s \
      db_name=$DB_NAME
  assert_status 0

  # Pre-existing creds should no longer work
  run docker_sqlplus "select 1 from dual;" "$STATIC_USER" "$STATIC_PASSWORD"
  [[ "${output}" =~ "ORA-01017: invalid username/password; logon denied" ]]
}

@test "GET /database/static-creds/:name - read the current credentials for the static role" {
  # Use Vault's new static password
  run vault read --namespace=ns1 -format=json database/static-creds/my-static-role
  assert_status 0
  local -r NEW_STATIC_PASSWORD="$(echo "${output}" | jq -r '.data.password')"

  run docker_sqlplus "select 1 from dual;" "$STATIC_USER" "$NEW_STATIC_PASSWORD"
  assert_sqlplus_success
  assert_status 0
}

@test "POST /database/rotate-root/:name - rotate the root user credentials for the database connection" {
  # Check rotate root works
  run docker_sqlplus "select 1 from dual;" "$VAULT_USER" "$VPASS"
  assert_status 0

  run vault write --namespace=ns1 -force database/rotate-root/$DB_NAME
  assert_status 0

  # root password should no longer work
  run docker_sqlplus "select 1 from dual;" "$VAULT_USER" "$VPASS"
  [[ "${output}" =~ "ORA-01017: invalid username/password; logon denied" ]]
}
