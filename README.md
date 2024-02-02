# vault-database-plugin-oracle

A [Vault](https://www.vaultproject.io) plugin for Oracle.

For more information on this plugin, see the [Oracle Database Secrets Engine](https://developer.hashicorp.com/vault/docs/secrets/databases/oracle) page.

This project uses the database plugin interface introduced in Vault version 0.7.1.

This plugin is not compatible with Alpine Linux out of the box. Oracle's libraries are glibc dependant, and Alpine has musl as its default C library.

## Releases

For linux/amd64, pre-built binaries can be found at
[the releases page](https://releases.hashicorp.com/vault-plugin-database-oracle/).
See the following table to determine what version of the Oracle Instant Client
SDK the plugin was built with:

|Plugin Release|Instant Client Version|
|---|---|
|v0.9.0|19.22|
|v0.8.3|19.20|
|v0.8.2|19.18|
|v0.8.1|19.18|
|v0.8.0|19.18|
|v0.7.0|19.6 |
|v0.6.1|19.6 |
|v0.6.0|19.6 |
|v0.5.0|19.6 |
|v0.4.0|19.6 |
|v0.3.0|19.6 |
|v0.2.0|19.3 |


## Build

For platforms other than linux/amd64, there are not currently pre-built
binaries available.

Before building, you will need to download the Oracle Instant Client library, which is available from
[Oracle](http://www.oracle.com/technetwork/database/features/instant-client/index-097480.html). Download the SDK package to get the headers and
download the Basic package to get the libraries for your platform. Inside the SDK package's subfolder: `instantclient_<version>/sdk/include/` are a
number of header files. Similarly, inside the Basic package's subfolder: `instantclient_<version>/` are a number of library files. These will need to
be placed into the standard locations for your platform.

For instance, if you are using MacOS, place the header files from the SDK package into either `/usr/local/include/` or `~/include/`.
Similarly, place the library files from the Basic package into either `/usr/local/lib/` or `~/lib/`

Next, ensure that you have [`pkg-config`](https://www.freedesktop.org/wiki/Software/pkg-config/) installed on your system. For MacOS, you can install
it using `brew install pkg-config`.

Create a `pkg-config` file to point to the library. Create the file `oci8.pc` on your `PKG_CONFIG_PATH`.

An example `oci8.pc` for macOS is:

```
prefix=/usr/local

version=11.2
build=client64

libdir=${prefix}/lib
includedir=${prefix}/include

Name: oci8
Description: Oracle database engine
Version: ${version}
Libs: -L${libdir} -lclntsh
Libs.private:
Cflags: -I${includedir}
```

Then, `git clone` this repository into your `$GOPATH` and `go build -o vault-plugin-database-oracle ./plugin` from the project directory.

## Tests

`make test` will run a basic test suite against a Docker version of Oracle.

Additionally, there are some [Bats](https://github.com/bats-core/bats-core) tests in the `tests` directory.

#### Prerequisites

- [Install Bats Core](https://bats-core.readthedocs.io/en/stable/installation.html)
- Docker
- A vault binary in the `tests` directory.

#### Setup

- Oracle plugin is built and saved in `PLUGIN_DIR`
    - Export `PLUGIN_DIR` containing the path to the oracle plugin binary.
- Oracle db docker image has been built
- Oracle db data path is set in `DOCKER_VOLUME_MNT`. i.e. `~/dev/oracle/data`
    - If you do not use a persistent store for Oracle data, the amount of time
      the container will need to start up will be dramatically longer. Using
      the volume mount skips a lot of first-time setup steps.
- Export `VAULT_LICENSE`. This test will only work for enterprise images.

#### Logs

Vault logs will be written to `VAULT_OUTFILE`. Bats test logs will be written to
`SETUP_TEARDOWN_OUTFILE`.

#### Run Bats tests

```
# export env vars
export VAULT_LICENSE="12345"
export PLUGIN_DIR="~/dev/plugins"
export DOCKER_VOLUME_MNT="~/dev/plugins/oracle/data"

# run tests
cd tests/
./test.bats
```


## Installation

**See [Case Sensitivity](#case-sensitivity) for important information about custom creation & rotation statements.**

Before running the plugin you will need to have the the Oracle Instant Client library installed. These can be downloaded from Oracle. The libraries will need to be placed in the default library search path or defined in the ld.so.conf configuration files.

If you are running Vault with mlock enabled, you will need to enable ipc_lock capabilities for the plugin binary.

The Vault plugin system is documented on the [Vault documentation site](https://www.vaultproject.io/docs/internals/plugins.html).

You will need to define a plugin directory using the `plugin_directory` configuration directive, then place the
`vault-plugin-database-oracle` executable generated above in the directory.

**Please note:** Versions v0.3.0 onwards of this plugin are incompatible with Vault versions before 1.6.0 due to an update of the database plugin interface.

Sample commands for plugin registration in current versions of Vault and starting to use the plugin:

```shell-session
$ vault plugin register -sha256=<SHA256 Hex value of the plugin binary> \
    database \                  # type
    vault-plugin-database-oracle
Success! Registered plugin: vault-plugin-database-oracle
```

Vault versions prior to v0.10.4 lacked the `vault plugin` operator and the 
registration step for them is:

```shell-session
$ shasum -a 256 vault-plugin-database-oracle > /tmp/oracle-plugin.sha256

$ vault write sys/plugins/catalog/database/vault-plugin-database-oracle \
    sha256=$(cat /tmp/oracle-plugin.sha256) \
    command="vault-plugin-database-oracle"
```

```shell-session
$ vault secrets enable database

$ vault write database/config/oracle \
    plugin_name=vault-plugin-database-oracle \
    allowed_roles="*" \
    connection_url='{{username}}/{{password}}@//url.to.oracle.db:1521/oracle_service' \
    username='vaultadmin' \
    password='reallysecurepassword'

# You should consider rotating the admin password. Note that if you do, the new password will never be made available
# through Vault, so you should create a vault-specific database admin user for this.
$ vault write -force database/rotate-root/oracle
```

If running the plugin on MacOS you may run into an issue where the OS prevents the Oracle libraries from being executed.
See [How to open an app that hasn't been notarized or is from an unidentified developer](https://support.apple.com/en-us/HT202491)
on Apple's support website to be able to run this.

## Usage

### Case Sensitivity

It is important that you do NOT specify double quotes around the username in any of the SQL statements.
Otherwise Oracle may create/look up a user with the incorrect name (`foo_bar` instead of `FOO_BAR`).

### Default statements

The [rotation statements](https://www.vaultproject.io/api/secret/databases/index.html#rotation_statements) are optional
and will default to `ALTER USER {{username}} IDENTIFIED BY "{{password}}"`

The [disconnect statements](https://developer.hashicorp.com/vault/api-docs/secret/databases/oracle#statements) are optional and will default to the sql below. Setting `disconnect_statements` to `false` will disable the disconnect functionality, but should be disabled with caution since it may limit the effectiveness of revocation.

```sql
ALTER USER {{username}} ACCOUNT LOCK;
begin
  for x in ( select inst_id, sid, serial# from gv$session where username="{{username}}" )
  loop
   execute immediate ( 'alter system kill session '''|| x.Sid || ',' || x.Serial# || '@' || x.inst_id ''' immediate' );
  end loop;
  dbms_lock.sleep(1);
end;
DROP USER {{username}};
```
