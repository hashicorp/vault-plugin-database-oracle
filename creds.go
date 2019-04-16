package oracle

import (
	"strings"

	"github.com/hashicorp/vault/sdk/database/dbplugin"
	"github.com/hashicorp/vault/sdk/database/helper/credsutil"
)

// oracleCredentialsProducer implements CredentialsProducer.
type oracleCredentialsProducer struct {
	credsutil.SQLCredentialsProducer
}

func (ocp *oracleCredentialsProducer) GenerateUsername(config dbplugin.UsernameConfig) (string, error) {
	if username, err := ocp.SQLCredentialsProducer.GenerateUsername(config); err != nil {
		return "", err
	} else {
		username = strings.Replace(username, "-", "_", -1)
		username = strings.Replace(username, ".", "_", -1)
		return strings.ToLower(username), nil
	}
}

func (ocp *oracleCredentialsProducer) GeneratePassword() (string, error) {
	if password, err := ocp.SQLCredentialsProducer.GeneratePassword(); err != nil {
		return "", err
	} else {
		password = strings.Replace(password, "-", "_", -1)
		return password, nil
	}
}
