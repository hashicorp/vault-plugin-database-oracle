package main

import (
	"log"
	"os"

	"github.com/gdavison/vault-oracle"
	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/helper/pluginutil"
	"github.com/hashicorp/vault/plugins"
)

func main() {
	apiClientMeta := &pluginutil.APIClientMeta{}
	flags := apiClientMeta.FlagSet()
	flags.Parse(os.Args)

	err := Run(apiClientMeta.GetTLSConfig())
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

// Run instantiates an Oracle object, and runs the RPC server for the plugin
func Run(apiTLSConfig *api.TLSConfig) error {
	dbType := oracle.New()

	plugins.Serve(dbType, apiTLSConfig)

	return nil
}
