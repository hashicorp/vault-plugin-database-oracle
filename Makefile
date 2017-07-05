default: quickdev

# Dev builds:
# * quickdev - build using the local build chain
# * dev      - build for the host platform using the build container

quickdev:
	go build -o build/`go env GOHOSTOS`_`go env GOHOSTARCH`/vault-plugin-database-oracle ./plugin

dev: docker
	rm -rf build
	docker run --rm -v `pwd`:/go/src/github.com/gdavison/vault-oracle \
		-v `pwd`/cgo:/cgo -v `pwd`/build:/build \
		-w /go/src/github.com/gdavison/vault-oracle \
		-e "TARGETS=`go env GOHOSTOS`/`go env GOHOSTARCH`" \
		-t vault-plugin-oracle-builder

bin: docker
	rm -rf build
	docker run --rm -v `pwd`:/go/src/github.com/gdavison/vault-oracle \
		-v `pwd`/cgo:/cgo -v `pwd`/build:/build \
		-w /go/src/github.com/gdavison/vault-oracle \
		-t vault-plugin-oracle-builder

docker:
	docker build -t vault-plugin-oracle-builder scripts/docker/

.PHONY: quickdev docker
