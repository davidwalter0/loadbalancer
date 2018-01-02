package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang/glog"

	"github.com/davidwalter0/llb/global"
	"github.com/davidwalter0/llb/kubeconfig"
	mgmt "github.com/davidwalter0/llb/manager"
	"github.com/davidwalter0/llb/watch"
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

func main() {
	array := strings.Split(os.Args[0], "/")
	me := array[len(array)-1]
	fmt.Printf("%s: Version %s version build %s commit %s\n", me, Version, Build, Commit)

	fmt.Println(*envCfg)
	// creates the clientset

	clientset := kubeconfig.NewClientset(envCfg)
	if clientset == nil {
		glog.Fatal("Kubernetes connection failed")
	}
	var mgr *mgmt.Mgr = mgmt.NewMgr(envCfg, clientset)
	watch.SetConfig(envCfg)
	go watch.RunWatcher(clientset)
	go mgr.Run()
	select {}
}
