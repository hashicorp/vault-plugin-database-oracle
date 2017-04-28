package main

import (
	"fmt"
	"os"

	"github.com/gdavison/vault-oracle"
	"github.com/hashicorp/vault/builtin/logical/database/dbplugin"
)

func main() {
	err := Run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// Run instantiates an Oracle object, and runs the RPC server for the plugin
func Run() error {
	dbType := oracle.New()

	dbplugin.NewPluginServer(dbType)

	return nil
}
