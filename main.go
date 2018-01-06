package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/golang/glog"

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

func init() {
	array := strings.Split(os.Args[0], "/")
	me := array[len(array)-1]
	fmt.Printf("%s %s: Version %s version build %s commit %s\n", log.Prefix(), me, Version, Build, Commit)
}

func main() {
	// creates the clientset
	clientset := kubeconfig.NewClientset(envCfg.Kubeconfig)
	if clientset == nil {
		glog.Fatal("Kubernetes connection failed")
	}
	var mgr *mgmt.Mgr = mgmt.NewMgr(envCfg, clientset)
	go mgr.Run()
	select {}
}
