#!/usr/bin/env bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

#
# This script builds the application from source for multiple platforms.
set -e

# Get the parent directory of where this script is.
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$( cd -P "$( dirname "$SOURCE" )/.." && pwd )"

# Change into that directory
cd "$DIR"

# Get the git commit
#GIT_COMMIT="$(git rev-parse HEAD)"
#GIT_DIRTY="$(test -n "`git status --porcelain`" && echo "+CHANGES" || true)"

# Determine the arch/os combos we're building for
#XC_ARCH=${XC_ARCH:-"386 amd64"}
#XC_OS=${XC_OS:-linux darwin windows freebsd openbsd netbsd solaris}
#XC_OSARCH=${XC_OSARCH:-"linux/386 linux/amd64 linux/arm linux/arm64 darwin/386 darwin/amd64 windows/386 windows/amd64 freebsd/386 freebsd/amd64 freebsd/arm openbsd/386 openbsd/amd64 openbsd/arm netbsd/386 netbsd/amd64 netbsd/arm solaris/amd64"}

GOPATH=${GOPATH:-$(go env GOPATH)}
case $(uname) in
    CYGWIN*)
        GOPATH="$(cygpath $GOPATH)"
        ;;
esac

# Delete the old dir
echo "==> Removing old directory..."
rm -f bin/*
rm -rf pkg/*
mkdir -p bin/
mkdir -p pkg/

# If its dev mode, only build for ourself
if [ "${VAULT_DEV_BUILD}x" != "x" ]; then
    XC_OS=$(go env GOOS)
    XC_ARCH=$(go env GOARCH)
    XC_OSARCH=$(go env GOOS)/$(go env GOARCH)
fi

# Build!
PKG_CONFIG_PATH="${DIR}/scripts/${XC_OS}_${XC_ARCH}/"

echo "==> Building..."
gox \
    -osarch="${XC_OSARCH}" \
    -output "pkg/bin/{{.OS}}_{{.Arch}}/vault-plugin-database-oracle" \
    ./plugin/.

# Move all the compiled things to the $GOPATH/bin
OLDIFS=$IFS
IFS=: MAIN_GOPATH=($GOPATH)
IFS=$OLDIFS

# Copy our OS/Arch to the bin/ directory
if [ "${VAULT_DEV_BUILD}x" != "x" ]; then
    DEV_PLATFORM="./pkg/bin/$(go env GOOS)_$(go env GOARCH)"
    for F in $(find ${DEV_PLATFORM} -mindepth 1 -maxdepth 1 -type f); do
        cp ${F} bin/
        cp ${F} ${MAIN_GOPATH}/bin/
    done
fi

# Done!
echo
echo "==> Results:"
ls -ahlR pkg/