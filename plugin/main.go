// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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

	if err := flags.Parse(os.Args[1:]); err != nil {
		fatal(err)
	}

	err := Run()
	if err != nil {
		fatal(err)
	}
}

// Run instantiates an Oracle object, and runs the RPC server for the plugin
func Run() error {
	dbplugin.ServeMultiplex(plugin.New)

	return nil
}

func fatal(err error) {
	log.Println(err)
	os.Exit(1)
}
