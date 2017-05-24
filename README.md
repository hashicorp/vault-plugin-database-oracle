# vault-oracle
A [Vault](https://www.vaultproject.io) plugin for Oracle

This project uses the database plugin interface introduced in Vault version 0.7.1.

## Installation

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

Then, `go get github.com/gdavison/vault-oracle` and `go build -o oracle-database-plugin ./plugin` from the project directory.
