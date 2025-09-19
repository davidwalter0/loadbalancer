/*

------------------------------------------------------------------------

Modified from the original example code to first connect with
incluster configuration then if unsuccessful external kubeconfig
format file

------------------------------------------------------------------------

Copyright 2016 The Kubernetes Authors.

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

// Note: the example only works with the code within the same
// release/branch.

package kubeconfig

import (
	"os"

	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"

	// Uncomment the following line to load the gcp plugin (only
	// required to authenticate against GKE clusters).

	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"k8s.io/client-go/tools/clientcmd"
)

// InCluster true when endpoints are accessible, when the service is
// running with the cluster's network namespaces
var InCluster bool

// CheckInCluster reports if the env variable is set for cluster
func CheckInCluster() bool {
	return len(os.Getenv("KUBERNETES_PORT")) > 0
}

func init() {
	InCluster = CheckInCluster()
}

// getKubeConfigPath returns the best kubeconfig path to use
// Order of precedence:
// 1. Command line flag (--kubeconfig)
// 2. KUBECONFIG environment variable
// 3. $HOME/.kube/config
// 4. /root/.kube/config
// 5. In-cluster config (handled elsewhere)
func getKubeConfigPath(explicitPath string) string {
	// Log information about which config sources are being considered
	logConfigSources()

	// 1. Use explicitly provided path if provided and not default value
	if explicitPath != "" && explicitPath != "cluster/auth/kubeconfig" {
		if _, err := os.Stat(explicitPath); err == nil {
			os.Stderr.WriteString("Using kubeconfig from flag: " + explicitPath + "\n")
			return explicitPath
		} else {
			os.Stderr.WriteString("Warning: Kubeconfig file specified by flag does not exist: " + explicitPath + "\n")
		}
	}

	// 2. Use KUBECONFIG environment variable if set
	if envPath := os.Getenv("KUBECONFIG"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			os.Stderr.WriteString("Using kubeconfig from KUBECONFIG environment variable: " + envPath + "\n")
			return envPath
		} else {
			os.Stderr.WriteString("Warning: Kubeconfig file specified by KUBECONFIG environment variable does not exist: " + envPath + "\n")
		}
	}

	// 3. Use default $HOME/.kube/config
	homeDir, err := os.UserHomeDir()
	if err == nil {
		homePath := homeDir + "/.kube/config"
		if _, err := os.Stat(homePath); err == nil {
			os.Stderr.WriteString("Using default kubeconfig from home directory: " + homePath + "\n")
			return homePath
		}
	}

	// 4. Check in root's home directory
	rootPath := "/root/.kube/config"
	if _, err := os.Stat(rootPath); err == nil {
		os.Stderr.WriteString("Using kubeconfig from root directory: " + rootPath + "\n")
		return rootPath
	}

	// 5. Fall back to specified path if it exists (which might be the default)
	if explicitPath != "" {
		os.Stderr.WriteString("Falling back to specified kubeconfig: " + explicitPath + "\n")
		return explicitPath
	}

	// No kubeconfig found
	os.Stderr.WriteString("No valid kubeconfig found, will attempt to use in-cluster config\n")
	return ""
}

// logConfigSources logs information about which config sources are being considered
func logConfigSources() {
	os.Stderr.WriteString("Kubeconfig lookup order:\n")
	os.Stderr.WriteString("1. Command line flag (--kubeconfig)\n")
	os.Stderr.WriteString("2. KUBECONFIG environment variable: " + os.Getenv("KUBECONFIG") + "\n")

	homeDir, err := os.UserHomeDir()
	if err == nil {
		os.Stderr.WriteString("3. $HOME/.kube/config: " + homeDir + "/.kube/config\n")
	} else {
		os.Stderr.WriteString("3. $HOME/.kube/config: Unable to determine home directory\n")
	}

	os.Stderr.WriteString("4. /root/.kube/config\n")
	os.Stderr.WriteString("5. In-cluster config\n")
}

// NewClientset returns a new handle to a kubernetes client
// Order of precedence:
// 1. In-cluster configuration (if running in a Kubernetes pod)
// 2. Command line flag (--kubeconfig)
// 3. KUBECONFIG environment variable
// 4. $HOME/.kube/config
// 5. /root/.kube/config
func NewClientset(kubeconfig string) *kubernetes.Clientset {
	var kubeRestConfig *restclient.Config
	var clientset *kubernetes.Clientset
	var err error

	// First, try in-cluster configuration
	if InCluster {
		os.Stderr.WriteString("Detected running inside Kubernetes cluster, trying in-cluster configuration\n")
		kubeRestConfig, err = restclient.InClusterConfig()
		if err == nil {
			os.Stderr.WriteString("Successfully loaded in-cluster configuration\n")
			clientset, err = kubernetes.NewForConfig(kubeRestConfig)
			if err == nil {
				return clientset
			}
			os.Stderr.WriteString("Error creating clientset from in-cluster config: " + err.Error() + "\n")
		} else {
			os.Stderr.WriteString("Failed to load in-cluster configuration: " + err.Error() + "\n")
		}
	}

	// If in-cluster config didn't work or we're not in a cluster, try kubeconfig file
	kubeconfigPath := getKubeConfigPath(kubeconfig)
	if kubeconfigPath != "" {
		kubeRestConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			os.Stderr.WriteString("Error building kubeconfig from " + kubeconfigPath + ": " + err.Error() + "\n")
		} else {
			os.Stderr.WriteString("Successfully loaded kubeconfig from " + kubeconfigPath + "\n")
			clientset, err = kubernetes.NewForConfig(kubeRestConfig)
			if err != nil {
				os.Stderr.WriteString("Error creating clientset from kubeconfig: " + err.Error() + "\n")
			}
		}
	}

	// If we still don't have a clientset, attempt one more try with in-cluster config (even if InCluster is false)
	if clientset == nil && !InCluster {
		os.Stderr.WriteString("No kubeconfig worked, trying in-cluster config as last resort\n")
		kubeRestConfig, err = restclient.InClusterConfig()
		if err == nil {
			clientset, err = kubernetes.NewForConfig(kubeRestConfig)
			if err == nil {
				os.Stderr.WriteString("Successfully created clientset from in-cluster config (last resort)\n")
			} else {
				os.Stderr.WriteString("Error creating clientset from in-cluster config: " + err.Error() + "\n")
			}
		} else {
			os.Stderr.WriteString("Failed to load in-cluster configuration: " + err.Error() + "\n")
		}
	}

	if clientset == nil {
		os.Stderr.WriteString("CRITICAL: Failed to create Kubernetes clientset. Check your kubeconfig or cluster connectivity.\n")
	}

	return clientset
}
