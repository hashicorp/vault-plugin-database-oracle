// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"strconv"

	plugin "github.com/hashicorp/vault-plugin-database-oracle"
	"github.com/hashicorp/vault/api"
	dbplugin "github.com/hashicorp/vault/sdk/database/dbplugin/v5"
)

func main() {
	apiClientMeta := &api.PluginAPIClientMeta{}
	flags := apiClientMeta.FlagSet()
	flags.Parse(os.Args[1:])

	// Set up pprof for the plugin
	RunPProf()

	err := Run()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

// Run instantiates an Oracle object, and runs the RPC server for the plugin
func Run() error {
	dbplugin.ServeMultiplex(plugin.New)

	return nil
}

func RunPProf() {
	// Create a TCP listener on a random available port
	var err error
	var listener net.Listener
	// Try to listen on the default pprof port first
	listener, err = net.Listen("tcp", "127.0.0.1:6060")
	if err != nil {
		// If we can't listen on the default port, try to find an available port
		listener, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			log.Printf("Failed to start pprof listener: %v", err)
		}
		return
	}

	// write the port to a file for reference
	port := listener.Addr().(*net.TCPAddr).Port
	file, err := os.Create("/tmp/pprof_port.txt")
	if err != nil {
		log.Printf("Failed to create port file: %v", err)
		return
	}
	defer file.Close()
	_, err = file.WriteString(strconv.Itoa(port))
	if err != nil {
		log.Printf("Failed to write port to file: %v", err)
		return
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/debug/pprof/", http.DefaultServeMux.ServeHTTP)
	mux.HandleFunc("/debug/index", pprof.Index)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.HandleFunc("/debug/pprof/threadcreate", pprof.Handler("threadcreate").ServeHTTP)
	mux.HandleFunc("/debug/pprof/mutex", pprof.Handler("mutex").ServeHTTP)
	mux.HandleFunc("/debug/pprof/block", pprof.Handler("block").ServeHTTP)
	mux.HandleFunc("/debug/pprof/allocs", pprof.Handler("allocs").ServeHTTP)
	mux.HandleFunc("/debug/pprof/goroutine", pprof.Handler("goroutine").ServeHTTP)
	mux.HandleFunc("/debug/pprof/heap", pprof.Handler("heap").ServeHTTP)

	go func() {
		defer listener.Close()
		if err := http.Serve(listener, mux); err != nil {
			log.Printf("pprof server error: %v", err)
		}
	}()
}
