#!/usr/bin/env bash

set -ex

if ! command -v sqlplus >/dev/null 2>&1; then
    echo "Error: sqlplus is not installed or not in your PATH." >&2
    echo "Please install the sqlplus command and try again." >&2
    echo "https://download.oracle.com/otn_software/mac/instantclient/instantclient-sqlplus-macos-arm64.dmg" >&2
    exit 1
fi

# All of these environment variables are required or an error will be returned.
[ "${CONN_URL:?}" ]
[ "${QUERY_FILE_PATH:?}" ]
[ "${VAULT_ADMIN:?}" ]
[ "${VAULT_ADMIN_PASSWORD:?}" ]
[ "${NUM_STATIC_USERS:?}" ]

# call sqlplus with the path to the query script and additional arguments that
# will be passed to the query
sqlplus -S "${CONN_URL}" @"${QUERY_FILE_PATH}" "${VAULT_ADMIN}" "${VAULT_ADMIN_PASSWORD}" "${NUM_STATIC_USERS}"
