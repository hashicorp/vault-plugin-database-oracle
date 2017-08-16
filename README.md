# vault-database-plugin-oracle

A [Vault](https://www.vaultproject.io) plugin for Oracle

This project uses the database plugin interface introduced in Vault version 0.7.1.

## Build

For linux/amd64, pre-built binaries can be found at [the releases page](https://releases.hashicorp.com/vault-plugin-database-oracle/)

For other platforms, there are not currently pre-built binaries available.

Before building, you will need to download the Oracle Instant Client library, which is available from [Oracle](http://www.oracle.com/technetwork/database/features/instant-client/index-097480.html). Download the SDK package to get the headers and download the Basic package to get the libraries for your platform.

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

Then, `git clone` this repository into your `$GOPATH` and `go build -o vault-plugin-database-oracle ./plugin` from the project directory.

## Installation

The Vault plugin system is documented on the [Vault documentation site](https://www.vaultproject.io/docs/internals/plugins.html).

You will need to define a plugin directory using the `plugin_directory` configuration directive, then place the `vault-plugin-database-oracle` executable generated above in the directory.

Register the plugin using

```
vault write sys/plugins/catalog/vault-plugin-database-oracle \ 
    sha_256=<expected SHA256 Hex value of the plugin binary> \
    command="vault-plugin-database-oracle"
```
