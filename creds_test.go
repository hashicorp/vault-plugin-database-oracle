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

func TestShouldReplaceHyphenInDisplayNameAndRoleName(t *testing.T) {
	credsProducer := &oracleCredentialsProducer{
		credsutil.SQLCredentialsProducer{
			DisplayNameLen: oracleDisplayNameMaxLength,
			RoleNameLen:    oracleDisplayNameMaxLength,
			UsernameLen:    oracleUsernameLength,
			Separator:      "_",
		},
	}
	usernameConfig := dbplugin.UsernameConfig{
		DisplayName: "a-b-c",
		RoleName:    "d-e-f",
	}
	if username, err := credsProducer.GenerateUsername(usernameConfig); err != nil {
		t.Errorf("err: %s", err)
	} else {
		if match, err := regexp.MatchString("a_b_c_d_e_f", username); err != nil {
			t.Errorf("err: %s", err)
		} else if !match {
			t.Errorf("does not match expected name. was '%s', expected to match '%s'", username, "a_b_c_d_e_f")
		}
	}
}

func TestShouldReplaceDotInDisplayNameAndRoleName(t *testing.T) {
	credsProducer := &oracleCredentialsProducer{
		credsutil.SQLCredentialsProducer{
			DisplayNameLen: oracleDisplayNameMaxLength,
			RoleNameLen:    oracleDisplayNameMaxLength,
			UsernameLen:    oracleUsernameLength,
			Separator:      "_",
		},
	}
	usernameConfig := dbplugin.UsernameConfig{
		DisplayName: "a.b.c",
		RoleName:    "d.e.f",
	}
	if username, err := credsProducer.GenerateUsername(usernameConfig); err != nil {
		t.Errorf("err: %s", err)
	} else {
		if match, err := regexp.MatchString("a_b_c_d_e_f", username); err != nil {
			t.Errorf("err: %s", err)
		} else if !match {
			t.Errorf("does not match expected name. was '%s', expected to match '%s'", username, "a_b_c_d_e_f")
		}
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
		result, _ := regexp.MatchString(fmt.Sprintf("^[[:upper:]][_[:lower:][:upper:][:digit:]]{%d}$", 19), string(password))
		return result
	}
	if err := quick.Check(assertion, nil); err != nil {
		t.Error(err)
	}
}
