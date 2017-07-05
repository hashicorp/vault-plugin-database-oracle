# Build System

This build system is an adaptation of [`xgo`](https://github.com/karalabe/xgo), a Go [CGO](https://blog.golang.org/c-go-cgo) cross compiler.

As of [Go 1.5](https://blog.golang.org/go1.5), released in August 2015, cross-compiling Go code has been straight forward. Tools like [`gox`](https://github.com/mitchellh/gox) make it even easier by running the builds in parallel and managing the platforms.

Adding CGO into the mix, however, is still challenging, as it requires the C toolchain for each platform. `xgo` provides a [Docker](https://www.docker.com) container loaded with the toolchains needed for general purpose CGO cross-compilation. I have adapted this to work specifically with with project.
