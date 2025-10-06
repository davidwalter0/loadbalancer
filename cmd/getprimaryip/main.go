/*
Copyright 2025 David Walter.

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
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/davidwalter0/loadbalancer/ipmgr"
	"github.com/davidwalter0/loadbalancer/pkg/interfaces"
)

func main() {
	var showIP bool
	var showInterface bool
	var showHelp bool

	flag.BoolVar(&showIP, "ip", false, "Show only the IP address")
	flag.BoolVar(&showInterface, "interface", false, "Show only the interface name")
	flag.BoolVar(&showHelp, "help", false, "Show help message")
	flag.Parse()

	if showHelp {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Detect and display the primary network interface and IP address.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		os.Exit(0)
	}

	// Auto-detect the primary interface
	ifaceName, err := ipmgr.AutoDetectInterface()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Get the IP address for the interface
	ipAddress, err := interfaces.GetInterfaceAddress(ifaceName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting IP for interface %s: %v\n", ifaceName, err)
		os.Exit(1)
	}

	// Output based on flags
	if showIP {
		fmt.Println(ipAddress)
		return
	}

	if showInterface {
		fmt.Println(ifaceName)
		return
	}

	// Default: show table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "INTERFACE\tIP ADDRESS")
	fmt.Fprintf(w, "%s\t%s\n", ifaceName, ipAddress)
	w.Flush()
}
