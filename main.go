/*

Copyright 2018 David Walter.

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

	"github.com/davidwalter0/llb/global"
	"github.com/davidwalter0/llb/kubeconfig"
	mgmt "github.com/davidwalter0/llb/manager"
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

func init() {
	array := strings.Split(os.Args[0], "/")
	me = array[len(array)-1]
	log.SetOutput(os.Stderr)
	log.Printf("%s: %s version build %s commit %s\n", me, Version, Build, Commit)
	log.SetOutput(os.Stdout)
}

func main() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// creates the clientset
	clientset := kubeconfig.NewClientset(envCfg.Kubeconfig)
	if clientset == nil {
		log.Fatal("Kubernetes connection failed")
	}
	var mgr *mgmt.Mgr = mgmt.NewMgr(envCfg, clientset)
	go mgr.Run()
	select {
	case signal := <-signals:
		log.Printf("%s: %s version build %s commit %s\n", me, Version, Build, Commit)
		log.Printf("%s: handling signal %s\n", me, signal)
		mgr.Shutdown()
	}
}
