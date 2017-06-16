# vault-oracle
A [Vault](https://www.vaultproject.io) plugin for Oracle

This project uses the database plugin interface introduced in Vault version 0.7.1.

## Build

There is not currently a pre-built binary available.

Before building, you will need to download the Oracle Instant Client library. On macOS, you can use `brew tap mikeclarke/oracle`, then `brew install oracle-client` and `brew install oracle-headers`. Otherwise, the packages are available from [Oracle](http://www.oracle.com/technetwork/database/features/instant-client/index-097480.html). Download the SDK package to get both libraries and headers.

Next, create a [`pkg-config`](https://www.freedesktop.org/wiki/Software/pkg-config/) file to point to the library. Create the file `oci8.pc` on your `PKG_CONFIG_PATH`.

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

Then, `git clone` this repository into your `$GOPATH` and `go build -o oracle-database-plugin ./plugin` from the project directory.

## Installation

The Vault plugin system is documented on the [Vault documentation site](https://www.vaultproject.io/docs/internals/plugins.html).

You will need to define a plugin directory using the `plugin_directory` configuration directive, then place the `oracle-database-plugin` executable generated above in the directory.

Register the plugin using

```
vault write sys/plugins/catalog/oracle-database-plugin \ 
    sha_256=<expected SHA256 Hex value of the plugin binary> \
    command="oracle-database-plugin"
```
