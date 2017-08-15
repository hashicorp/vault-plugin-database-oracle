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

.PHONY: bin default generate test fmt fmtcheck dev bootstrap
