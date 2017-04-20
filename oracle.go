package oracle

import (
	_ "github.com/mattn/go-oci8"

	"github.com/hashicorp/vault/plugins/helper/database/connutil"
	"github.com/hashicorp/vault/plugins/helper/database/credsutil"
)

const oracleTypeName string = "oci8"

type Oracle struct {
	connutil.ConnectionProducer
	credsutil.CredentialsProducer
}

func New() *Oracle {
	connProducer := &connutil.SQLConnectionProducer{}
	connProducer.Type = oracleTypeName

	credsProducer := &credsutil.SQLCredentialsProducer{
		DisplayNameLen: 4,
		UsernameLen:    16,
	}

	dbType := &Oracle{
		ConnectionProducer:  connProducer,
		CredentialsProducer: credsProducer,
	}

	return dbType
}
