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

func init() {
	array := strings.Split(os.Args[0], "/")
	me := array[len(array)-1]
	log.SetOutput(os.Stderr)
	log.Printf("%s: Version %s version build %s commit %s\n", me, Version, Build, Commit)
	log.SetOutput(os.Stdout)
}

func main() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, os.Kill)

	// creates the clientset
	clientset := kubeconfig.NewClientset(envCfg.Kubeconfig)
	if clientset == nil {
		log.Fatal("Kubernetes connection failed")
	}
	var mgr *mgmt.Mgr = mgmt.NewMgr(envCfg, clientset)
	go mgr.Run()
	select {
	case <-signals:
		mgr.Shutdown()
	}
}
