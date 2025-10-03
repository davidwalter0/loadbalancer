/*

Copyright 2018-2025 David Walter.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

*/

package main

import (
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/davidwalter0/loadbalancer/global"
	"github.com/davidwalter0/loadbalancer/kubeconfig"
	"github.com/davidwalter0/loadbalancer/mgr"
)

var envCfg = global.Cfg()

// retries number of attempts
var retries = 3

// logReloadTimeout in seconds
var logReloadTimeout = time.Duration(600)

// Build info text
var Build string

// Commit git string
var Commit string

// Version semver string
var Version string

// process arg[0]
var me string

var signals = make(chan os.Signal, 1)
var dumpSignals = make(chan os.Signal, 1)

// Message of repo info
var Message = `
Copyright 2018-2025 David Walter.

`

func init() {
	array := strings.Split(os.Args[0], "/")
	me = array[len(array)-1]
	log.SetOutput(os.Stderr)
	log.Printf("%s: %s version build %s commit %s\n\n%s\n", me, Version, Build, Commit, Message)
	log.SetOutput(os.Stdout)
	// Only catch signals that should trigger a graceful shutdown
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	// Notify SIGUSR1 for service dump
	signal.Notify(dumpSignals, syscall.SIGUSR1)
	log.Printf("Signal handlers registered: SIGINT, SIGTERM for shutdown, SIGUSR1 for service dump")
}

// dumpServices prints the current services being managed by the loadbalancer
func dumpServices(manager *mgr.Mgr) {
	log.Printf("=== Current Load Balancer Services [%s] ===", time.Now().Format(time.RFC3339))
	if manager == nil {
		log.Printf("ERROR: Manager is nil in dumpServices")
		return
	}
	serviceInfo := manager.DumpServicesInfo()
	log.Printf("Service info length: %d bytes", len(serviceInfo))
	log.Print(serviceInfo)
	log.Printf("=== End Service Dump ===")
}

func main() {

	// Initialize IP manager with the configured link device
	mgr.Initialize()

	// Create manager with or without Kubernetes support
	var manager *mgr.Mgr

	if envCfg.Kubernetes {
		// Kubernetes mode - create clientset and run full manager
		clientset := kubeconfig.NewClientset(envCfg.Kubeconfig)
		if clientset == nil {
			log.Fatal("Kubernetes connection failed")
		}
		manager = mgr.NewMgr(envCfg, clientset)
		go manager.Run()
	} else {
		log.Printf("Running in non-Kubernetes mode (for testing)")
		// Create manager without clientset, don't run watchers
		manager = mgr.NewMgr(envCfg, nil)
		// In non-Kubernetes mode, we don't call Run() which would panic on nil clientset
	}

	// Start a goroutine to handle SIGUSR1 signals for service dumping
	go func() {
		log.Printf("Starting SIGUSR1 signal handler for service dumps")
		for {
			<-dumpSignals
			log.Printf("SIGUSR1 signal received, dumping services...")
			dumpServices(manager)
		}
	}()

	select {
	case signal := <-signals:
		log.Printf("%s: %s version build %s commit %s\n", me, Version, Build, Commit)
		log.Printf("%s: handling signal %s\n", me, signal)
		manager.Shutdown()
	}
}
