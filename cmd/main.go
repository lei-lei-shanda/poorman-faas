package cmd

import (
	"context"
	"flag"
	"path/filepath"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var pythonScript = `
import sys

print(sys.version)
print("Hello, World!")
`

func run(clientset *kubernetes.Clientset) error {
	// create configmap from pythonScript
	configMap := &apiv1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "python-script",
		},
		Data: map[string]string{"script.py": pythonScript},
	}

	// update cluster with new configMap
	configMapClient := clientset.CoreV1().ConfigMaps(apiv1.NamespaceDefault)
	_, err := configMapClient.Update(context.Background(), configMap, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	// create a deployment with the configMap
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "python-script",
		},
	}

	// update cluster with new deployment
	deploymentClient := clientset.AppsV1().Deployments(apiv1.NamespaceDefault)
	_, err = deploymentClient.Update(context.Background(), deployment, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

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
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	err = run(clientset)
	if err != nil {
		panic(err)
	}
}
