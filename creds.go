package oracle

import (
	"math/rand"
	"strings"
	"time"

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
		return strings.ToLower(username), nil
	}
}

func (ocp *oracleCredentialsProducer) GeneratePassword() (string, error) {
	// Oracle passwords: https://asktom.oracle.com/pls/apex/f?p=100:11:0::::P11_QUESTION_ID:595223460734
	// o Passwords must be from 1 to 30 characters long.
	// o Passwords cannot contain quotation marks.
	// o Passwords are not case sensitive.
	// o A Password must begin with an alphabetic character.
	// o Passwords can contain only alphanumeric characters and the
	//   underscore (_), dollar sign ($), and pound sign (#). Oracle
	//   strongly discourages you from using $ and #.
	// o A Password cannot be an Oracle reserved word (eg: SELECT).
	if password, err := ocp.SQLCredentialsProducer.GeneratePassword(); err != nil {
		return "", err
	} else {
		// credsutil.SQLCredentialsProducer.GeneratePassword() uses github.com/hashicorp/go-uuid, which generates
		// cryptographically-random UUIDs. We should be safe replacing the first character with a non-secure
		// random lower-case character
		charOffset := rand.Intn(26)
		char := string(rune('a') + rune(charOffset))
		password = strings.Replace(password, "-", "", -1)
		password = char + password[1:oraclePasswordLength]
		return password, nil
	}
}

func (ocp *oracleCredentialsProducer) GenerateExpiration(ttl time.Time) (string, error) {
	return ttl.Format("2006-01-02 15:04:05-0700"), nil
}
