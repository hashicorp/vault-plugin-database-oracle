package oracle

import (
	"fmt"
	"math/rand"
	"reflect"
	"regexp"
	"testing"
	"testing/quick"

	"github.com/hashicorp/vault/builtin/logical/database/dbplugin"
	"github.com/hashicorp/vault/plugins/helper/database/credsutil"
)

type testUsername string

func (tu testUsername) Generate(rand *rand.Rand, size int) reflect.Value {
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
	if username, err := credsProducer.GenerateUsername(usernameConfig); err != nil {
		panic(err)
	} else {
		return reflect.ValueOf(testUsername(username))
	}
}

func TestUsernameShouldMatchOracleRequirements(t *testing.T) {
	assertion := func(username testUsername) bool {
		result, _ := regexp.MatchString(fmt.Sprintf("^[[:lower:]][_[:lower:][:digit:]]{%d}$", oracleUsernameLength-1), string(username))
		return result
	}
	if err := quick.Check(assertion, nil); err != nil {
		t.Error(err)
	}
}

type testPassword string

func (tp testPassword) Generate(rand *rand.Rand, size int) reflect.Value {
	credsProducer := &oracleCredentialsProducer{}
	if password, err := credsProducer.GeneratePassword(); err != nil {
		panic(err)
	} else {
		return reflect.ValueOf(testPassword(password))
	}
}

func TestPasswordShouldMatchOracleRequirements(t *testing.T) {

	assertion := func(password testPassword) bool {
		result, _ := regexp.MatchString(fmt.Sprintf("^[[:lower:]][_[:lower:][:digit:]]{%d}$", oraclePasswordLength-1), string(password))
		return result
	}
	if err := quick.Check(assertion, nil); err != nil {
		t.Error(err)
	}
}
