# vault-oracle
A [Vault](https://www.vaultproject.io) plugin for Oracle

This project uses the database plugin interface introduced in Vault version 0.7.1.

## Build

There is not currently a pre-built binary available.

To build this project, run the `make` command. It has the following targets of interest:

* `quickdev` - builds using the local toolchain
* `dev` - builds for the host platform using the build container
* `bin` - builds for all supported platforms

### Prerequisites

Before building, you will need to download the Oracle Instant Client library, which is available from [Oracle](http://www.oracle.com/technetwork/database/features/instant-client/index-097480.html). Download the SDK package to get the headers and download the Basic package to get the libraries for each platform that you will be building.

If you are building using `quickdev`, you will need to install the Oracle headers and libraries for your host platform, then create a  [`pkg-config`](https://www.freedesktop.org/wiki/Software/pkg-config/) file to point to them. The file should be named `oci8.pc` and be located on your `PKG_CONFIG_PATH`.

An example `oci8.pc` for macOS is:

```
prefix=/usr/local

version=11.2

libdir=${prefix}/lib
includedir=${prefix}/include

Name: oci8
Description: Oracle database engine
Version: ${version}
Libs: -L${libdir} -lclntsh
Libs.private:
Cflags: -I${includedir}
```

## Installation

The Vault plugin system is documented on the [Vault documentation site](https://www.vaultproject.io/docs/internals/plugins.html).

You will need to define a plugin directory using the `plugin_directory` configuration directive, then place the `oracle-database-plugin` executable generated above in the directory.

Register the plugin using

```
vault write sys/plugins/catalog/oracle-database-plugin \ 
    sha_256=<expected SHA256 Hex value of the plugin binary> \
    command="oracle-database-plugin"
```

The SHA-256 can be calculated using `shasum -a 256 <filename>`.
