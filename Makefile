SHELL := /usr/bin/env bash -euo pipefail -c

# Determine this makefile's path.
# Be sure to place this BEFORE `include` directives, if any.
THIS_FILE := $(lastword $(MAKEFILE_LIST))

TEST?=$$(go list ./... | grep -v /vendor/ | grep -v /integ)
GOFMT_FILES?=$$(find . -name '*.go' | grep -v vendor)
EXTERNAL_TOOLS=\
	github.com/mitchellh/gox 

default: dev

# bin generates the releaseable binaries for vault-plugin-database-oracl0e
bin: fmtcheck generate
	@CGO_ENABLED=1 BUILD_TAGS='$(BUILD_TAGS)' XC_ARCH="amd64" XC_OS="linux" XC_OSARCH="linux/amd64" sh -c "'$(CURDIR)/scripts/build.sh'"

dev: fmtcheck generate
	@CGO_ENABLED=1 BUILD_TAGS='$(BUILD_TAGS)' VAULT_DEV_BUILD=1 sh -c "'$(CURDIR)/scripts/build.sh'"

# test runs the unit tests and vets the code
test: fmtcheck generate
	CGO_ENABLED=1 go test -tags='$(BUILD_TAGS)' $(TEST) $(TESTARGS) -timeout=20m -parallel=4

# generate runs `go generate` to build the dynamically generated
# source files.
generate:
	go generate $(go list ./... | grep -v /vendor/)

fmtcheck:
	@sh -c "'$(CURDIR)/scripts/gofmtcheck.sh'"

fmt:
	gofmt -w $(GOFMT_FILES)

# bootstrap the build by downloading additional tools
bootstrap:
	@for tool in  $(EXTERNAL_TOOLS) ; do \
		echo "Installing/Updating $$tool" ; \
		go get -u $$tool; \
	done

# the following targets are run in CircleCI as part of the build/test jobs

# build the build image
build-cross-image:
	docker build -t cross-image:latest .

# use the pre-built image (with dependencies set-up) to build the binary
# by default this will result in the linux_amd64 binary being written to - pkg/bin/linux_amd64/
# to build the dev binary to bin/ - export VAULT_DEV_BUILD=1
build-in-container: build-cross-image
	docker run --rm \
	-v /var/run/docker.sock:/var/run/docker.sock \
	-v $(CURDIR)/pkg:/go/src/github.com/hashicorp/vault-plugin-database-oracle/pkg \
	-v $(CURDIR)/bin:/go/src/github.com/hashicorp/vault-plugin-database-oracle/bin \
	-e VAULT_DEV_BUILD=${VAULT_DEV_BUILD} \
	cross-image:latest \
	make bin

# run tests in the build container
test-in-container: build-cross-image
	docker run --rm \
	-v /var/run/docker.sock:/var/run/docker.sock \
	-v $(CURDIR)/test-results/go:/go/src/github.com/hashicorp/vault-plugin-database-oracle/test-results/go \
	-e RUN_IN_CONTAINER=1 cross-image:latest \
    make test-ci

# when running in CirleCI - convert test results to junit xml (for storage)
test-ci: fmtcheck generate
	go get -x github.com/jstemmer/go-junit-report
	CGO_ENABLED=1 go test $(TEST) -timeout=20m -parallel=4 \
	-v | tee test-results/go/go-test-report.raw
	go-junit-report < test-results/go/go-test-report.raw > test-results/go/go-test-report.xml

.PHONY: bin default generate test fmt fmtcheck dev bootstrap
