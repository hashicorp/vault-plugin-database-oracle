package oracle

import (
	"strings"

	"github.com/hashicorp/vault/builtin/logical/database/dbplugin"
	"github.com/hashicorp/vault/plugins/helper/database/credsutil"
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
