# vault-database-plugin-oracle

A [Vault](https://www.vaultproject.io) plugin for Oracle

This project uses the database plugin interface introduced in Vault version 0.7.1.

This plugin is not compatible with Alpine Linux out of the box. Oracle's libraries are glibc dependant, and Alpine has musl as its default C library.

## Build

For linux/amd64, pre-built binaries can be found at [the releases page](https://releases.hashicorp.com/vault-plugin-database-oracle/) (built with Oracle Instant Client SDK 19.3)

For other platforms, there are not currently pre-built binaries available.

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

`make test` will run a basic test suite against a Docker version of Oracle.

## Installation

**See [Case Sensitivity](#case-sensitivity) for important information about custom creation & rotation statements.**

The Vault plugin system is documented on the [Vault documentation site](https://www.vaultproject.io/docs/internals/plugins.html).

You will need to define a plugin directory using the `plugin_directory` configuration directive, then place the
`vault-plugin-database-oracle` executable generated above in the directory.

**Please note:** Versions v0.3.0 onwards of this plugin are incompatible with Vault versions before 1.6.0 due to an update of the database plugin interface.

Sample commands for registering and starting to use the plugin:

```
$ shasum -a 256 vault-plugin-database-oracle > /tmp/oracle-plugin.sha256

$ vault write sys/plugins/catalog/database/vault-plugin-database-oracle \
    sha256=$(cat /tmp/oracle-plugin.sha256) \
    command="vault-plugin-database-oracle"

$ vault secrets enable database

$ vault write database/config/oracle plugin_name \
    vault-plugin-database-oracle \
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
