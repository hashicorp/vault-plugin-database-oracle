FROM docker.mirror.hashicorp.services/oraclelinux:7

RUN yum-config-manager --add-repo http://yum.oracle.com/repo/OracleLinux/OL7/oracle/instantclient/x86_64

RUN yum install -y deltarpm

RUN yum update -y && yum install -y  \
		gcc \
		make \
		wget \
		tar \
		gzip \
		vim \
		unzip \
		zip \
		git

ENV GOLANG_VERSION 1.14.10

RUN set -eux; \
	url="https://golang.org/dl/go${GOLANG_VERSION}.linux-amd64.tar.gz"; \
	wget -O go.tar.gz "$url"; \
	gunzip go.tar.gz; \
	tar -C /usr/local -xf go.tar; \
	rm go.tar; \
	export PATH="/usr/local/go/bin:$PATH"; \
	go version

ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH
ENV PKG_CONFIG_PATH $GOPATH/src/github.com/hashicorp/vault-plugin-database-oracle/scripts/linux_amd64

RUN yum install -y \
		oracle-instantclient19.6-basic \
		oracle-instantclient19.6-devel

RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" "$GOPATH/src/github.com/hashicorp/vault-plugin-database-oracle" && chmod -R 777 "$GOPATH"

WORKDIR $GOPATH/src/github.com/hashicorp/vault-plugin-database-oracle/

COPY . .

RUN mkdir -p test-results/go

RUN make bootstrap

CMD echo "Please specify a command, e.g. 'make bin' or 'make test-ci'"
