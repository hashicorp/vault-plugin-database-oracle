package main

import (
	"log"
	"os"

	plugin "github.com/hashicorp/vault-plugin-database-oracle"
	"github.com/hashicorp/vault/api"
	dbplugin "github.com/hashicorp/vault/sdk/database/dbplugin/v5"
)

func main() {
	apiClientMeta := &api.PluginAPIClientMeta{}
	flags := apiClientMeta.FlagSet()
	flags.Parse(os.Args[1:])

	err := Run()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

// Run instantiates an Oracle object, and runs the RPC server for the plugin
func Run() error {
	db, err := plugin.New()
	if err != nil {
		return err
	}

	dbplugin.Serve(db.(dbplugin.Database))

	return nil
}
