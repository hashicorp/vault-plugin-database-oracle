#!/bin/bash
#
# Copyright (c) 2014 Péter Szilágyi. All rights reserved.
# Copyright (c) 2017 Graham Davison
# Released under the MIT license.
#
# Contains the main cross compiler, that individually sets up each target build
# platform, compiles all the C dependencies, then build the requested executable
# itself.
#
# Usage: build.sh <import path>
#
# Needed environment variables:
#   TARGETS        - Comma separated list of build targets to compile for
#   EXT_GOPATH     - GOPATH elements mounted from the host filesystem

export GOPATH=$GOPATH:`pwd`

echo "GOPATH=$GOPATH"

# If no build targets were specified, inject a catch all wildcard
if [ "$TARGETS" == "" ]; then
  TARGETS="./."
fi

# Build for each requested platform individually
for TARGET in $TARGETS; do
  # Split the target into platform and architecture
  XGOOS=`echo $TARGET | cut -d '/' -f 1`
  XGOARCH=`echo $TARGET | cut -d '/' -f 2`

  # Check and build for Linux targets
  if ([ $XGOOS == "." ] || [ $XGOOS == "linux" ]) && ([ $XGOARCH == "." ] || [ $XGOARCH == "amd64" ]); then
    echo "Compiling for linux/amd64..."
    export PKG_CONFIG_PATH=/cgo/linux_amd64
    GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -o "/build/linux_amd64/vault-plugin-database-oracle" ./plugin
  fi
  if ([ $XGOOS == "." ] || [ $XGOOS == "linux" ]) && ([ $XGOARCH == "." ] || [ $XGOARCH == "386" ]); then
    echo "Compiling for linux/386..."
    export PKG_CONFIG_PATH=/cgo/linux_386
    GOOS=linux GOARCH=386 CGO_ENABLED=1 go build -o "/build/linux_386/vault-plugin-database-oracle" ./plugin
  fi

  # Check and build for Windows targets
  if [ $XGOOS == "." ] || [[ $XGOOS == windows* ]]; then
    # Split the platform version and configure the Windows NT version
    PLATFORM=`echo $XGOOS | cut -d '-' -f 2`
    if [ "$PLATFORM" == "" ] || [ "$PLATFORM" == "." ] || [ "$PLATFORM" == "windows" ]; then
      # Minimum supported version for Go 1.8 is Windows 7/Windows Server 2008R2: https://github.com/golang/go/wiki/MinimumRequirements
      # Which is 6.1 (https://msdn.microsoft.com/en-ca/library/windows/desktop/ms724832(v=vs.85).aspx)
      PLATFORM=6.1 
    fi
  
    MAJOR=`echo $PLATFORM | cut -d '.' -f 1`
    if [ "${PLATFORM/.}" != "$PLATFORM" ] ; then
      MINOR=`echo $PLATFORM | cut -d '.' -f 2`
    fi
    CGO_NTDEF="-D_WIN32_WINNT=0x`printf "%02d" $MAJOR``printf "%02d" $MINOR`"
  
    # Build the requested windows binaries
    if [ $XGOARCH == "." ] || [ $XGOARCH == "amd64" ]; then
      echo "Compiling for windows/amd64..."
      export PKG_CONFIG_PATH=/cgo/windows_amd64:/usr/x86_64-w64-mingw32/lib/pkgconfig
  
      CC=x86_64-w64-mingw32-gcc-posix CXX=x86_64-w64-mingw32-g++-posix GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CGO_CFLAGS="$CGO_NTDEF" CGO_CXXFLAGS="$CGO_NTDEF" go build -o "/build/windows_amd64/vault-plugin-database-oracle.exe" ./plugin
    fi
    if [ $XGOARCH == "." ] || [ $XGOARCH == "386" ]; then
      echo "Compiling for windows/386..."
      export PKG_CONFIG_PATH=/cgo/windows_386:/usr/i686-w64-mingw32/lib/pkgconfig
  
      CC=i686-w64-mingw32-gcc-posix CXX=i686-w64-mingw32-g++-posix GOOS=windows GOARCH=386 CGO_ENABLED=1 CGO_CFLAGS="$CGO_NTDEF" CGO_CXXFLAGS="$CGO_NTDEF" go build -o "/build/windows_386/vault-plugin-database-oracle.exe" ./plugin
    fi
  fi
  
  # Check and build for OSX targets
  if [ $XGOOS == "." ] || [[ $XGOOS == darwin* ]]; then
    # Split the platform version and configure the deployment target
    PLATFORM=`echo $XGOOS | cut -d '-' -f 2`
    if [ "$PLATFORM" == "" ] || [ "$PLATFORM" == "." ] || [ "$PLATFORM" == "darwin" ]; then
      PLATFORM=10.8 # Minimum supported version for Go 1.8: https://github.com/golang/go/wiki/Darwin
    fi
    export MACOSX_DEPLOYMENT_TARGET=$PLATFORM

    # Build the requested darwin binaries
    if [ $XGOARCH == "." ] || [ $XGOARCH == "amd64" ]; then
      echo "Compiling for darwin/amd64..."
      export PKG_CONFIG_PATH=/cgo/darwin_amd64
      CC=o64-clang CXX=o64-clang++ GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -o "/build/darwin_amd64/vault-plugin-database-oracle" ./plugin
    fi
    if [ $XGOARCH == "." ] || [ $XGOARCH == "386" ]; then
      echo "Compiling for darwin/386..."
      export PKG_CONFIG_PATH=/cgo/darwin_386
      CC=o32-clang CXX=o32-clang++ GOOS=darwin GOARCH=386 CGO_ENABLED=1 go build -o "/build/darwin_386/vault-plugin-database-oracle" ./plugin
    fi
    # Remove any automatically injected deployment target vars
    unset MACOSX_DEPLOYMENT_TARGET
  fi
done
