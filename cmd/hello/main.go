package main

import (
	"flag"
	"fmt"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func run(clientset *kubernetes.Clientset) error {
	// get k8s server version
	serverVersion, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("clientset.Discovery().ServerVersion(): %w", err)
	}
	fmt.Printf("Server Version: %s\n", serverVersion.String())

	return nil
}

func main() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}

	// fmt.Printf("Kubeconfig: %#v\n", config)
	fmt.Printf("Kubernetes Cluster Host: %s\n", config.Host)

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	err = run(clientset)
	if err != nil {
		panic(err)
	}
}
