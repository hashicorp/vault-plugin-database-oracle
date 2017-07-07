package oracle

import (
	"testing"
	"testing/quick"

	"fmt"
	"github.com/hashicorp/vault/builtin/logical/database/dbplugin"
	"github.com/hashicorp/vault/plugins/helper/database/credsutil"
	"regexp"
)

func TestUsernameShouldMatchOracleRequirements(t *testing.T) {
	credsProducer := &oracleCredentialsProducer{
		credsutil.SQLCredentialsProducer{
			DisplayNameLen: oracleDisplayNameMaxLength,
			RoleNameLen:    oracleDisplayNameMaxLength,
			UsernameLen:    oracleUsernameLength,
			Separator:      "_",
		},
	}
	usernameConfig := dbplugin.UsernameConfig{
		DisplayName: "display",
		RoleName:    "role",
	}

	assertion := func() bool {
		username, _ := credsProducer.GenerateUsername(usernameConfig)
		result, _ := regexp.MatchString(fmt.Sprintf("[[:lower:]][_[:lower:][:digit:]]{%d}", oracleUsernameLength-1), username)
		return result
	}
	if err := quick.Check(assertion, nil); err != nil {
		t.Error(err)
	}

}
