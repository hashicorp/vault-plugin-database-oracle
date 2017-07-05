# Go cross compiler (xgo): Base cross-compilation layer
# Copyright (c) 2014 Péter Szilágyi. All rights reserved.
# Copyright (c) 2017 Graham Davison
#
# Released under the MIT license.

FROM ubuntu:16.04

# Configure the Go environment, since it's not going to change
ENV PATH   /usr/local/go/bin:$PATH
ENV GOPATH /go


# Inject the remote file fetcher and checksum verifier
COPY fetch.sh /fetch.sh
ENV  FETCH /fetch.sh
RUN  chmod +x $FETCH


# Make sure apt-get is up to date and dependent packages are installed
RUN \
  apt-get update && \
  apt-get install -y automake autogen build-essential ca-certificates \
    clang llvm-dev             \
    pkg-config           \
    make xz-utils wget git mercurial bzr            \
    --no-install-recommends

# Fix any stock package issues
RUN ln -s /usr/include/asm-generic /usr/include/asm

# Configure the container for OSX cross compilation
ENV OSX_SDK     MacOSX10.11.sdk
ENV OSX_NDK_X86 /usr/local/osx-ndk-x86

RUN \
  OSX_SDK_PATH=https://s3.dockerproject.org/darwin/v2/$OSX_SDK.tar.xz && \
  $FETCH $OSX_SDK_PATH dd228a335194e3392f1904ce49aff1b1da26ca62       && \
  \
  git clone https://github.com/tpoechtrager/osxcross.git && \
  mv `basename $OSX_SDK_PATH` /osxcross/tarballs/        && \
  \
  sed -i -e 's|-march=native||g' /osxcross/build_clang.sh /osxcross/wrapper/build.sh && \
  UNATTENDED=yes OSX_VERSION_MIN=10.6 /osxcross/build.sh                             && \
  mv /osxcross/target $OSX_NDK_X86                                                   && \
  \
  rm -rf /osxcross

COPY patch.tar.xz $OSX_NDK_X86/SDK/$OSX_SDK/usr/include/c++
ENV  PATH $OSX_NDK_X86/bin:$PATH

# Inject the new Go root distribution downloader and bootstrapper
COPY bootstrap_pure.sh /bootstrap_pure.sh
ENV  BOOTSTRAP_PURE /bootstrap_pure.sh
RUN  chmod +x $BOOTSTRAP_PURE

ENV GO_VERSION 183

RUN \
  export ROOT_DIST=https://storage.googleapis.com/golang/go1.8.3.linux-amd64.tar.gz     && \
  export ROOT_DIST_SHA=1862f4c3d3907e59b04a757cfda0ea7aa9ef39274af99a784f5be843c80c6772 && \
  \
  $BOOTSTRAP_PURE

## Inject the C dependency cross compiler
#ADD build_deps.sh /build_deps.sh
#ENV BUILD_DEPS /build_deps.sh
#RUN chmod +x $BUILD_DEPS

# Inject the container entry point, the build script
ADD build.sh /build.sh
ENV BUILD /build.sh
RUN chmod +x $BUILD

ENTRYPOINT ["/build.sh"]