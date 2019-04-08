FROM store/oracle/database-instantclient:12.2.0.1

RUN yum update -y && yum install -y  \
		gcc \
		make \
		wget \
		tar \
		gzip \
		vim \
		unzip \
		zip \
		git \
	&& rm -rf /var/lib/apt/lists/*

ENV GOLANG_VERSION 1.11.6

RUN set -eux; \
	\
# this "case" statement is generated via "update.sh"
	url="https://golang.org/dl/go${GOLANG_VERSION}.linux-amd64.tar.gz"; \
	wget -O go.tar.gz "$url"; \
	gunzip go.tar.gz; \
	tar -C /usr/local -xf go.tar; \
	rm go.tar; \
	export PATH="/usr/local/go/bin:$PATH"; \
	go version

ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH

RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" "$GOPATH/src/github.com/hashicorp/vault-plugin-database-oracle" && chmod -R 777 "$GOPATH"
WORKDIR $GOPATH/src/github.com/hashicorp/vault-plugin-database-oracle

CMD make bootstrap bin
