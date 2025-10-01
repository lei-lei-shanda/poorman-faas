package cmd

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	_ "embed"
)

//go:embed echo.py
var pythonScript string

func run(clientset *kubernetes.Clientset) error {
	configMapUniqueName := "python-script-config"
	// create configmap from pythonScript
	// ```yaml
	// apiVersion: v1
	// kind: ConfigMap
	// metadata:
	//   name: python-script-config
	// data:
	//   main.py: content of echo.py
	// ```
	configMap := &apiv1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: configMapUniqueName,
			Labels: map[string]string{
				"user": "frank",
				"app":  "echo-service",
			},
		},
		Data: map[string]string{"main.py": pythonScript},
	}

	// create cluster with new configMap
	configMapClient := clientset.CoreV1().ConfigMaps(apiv1.NamespaceDefault)
	_, err := configMapClient.Create(context.Background(), configMap, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("configMapClient.Create(): %w", err)
	}

	// create a deployment with the configMap
	// ```yaml
	// apiVersion: apps/v1
	// kind: Deployment
	// metadata:
	// 		name: python-app
	// spec:
	// template:
	// 	spec:
	// 	containers:
	// 	- name: python-container
	// 		image: python:3.9-slim
	// 		command: ["sh", "-c"]
	// 		args:
	// 		- |
	// 		python -c "$(cat /scripts/main.py)"
	// 		volumeMounts:
	// 		- name: script-volume
	// 		mountPath: /scripts
	// 	volumes:
	// 	- name: script-volume
	// 		configMap:
	// 		name: python-script-config
	// ```
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "python-app",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "python-app",
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "python-app",
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{{
						Name:    "python-container",
						Image:   "python:3.9-slim",
						Command: []string{"sh", "-c"},
						Args:    []string{"python -c $(cat /scripts/main.py)"},
						Ports: []apiv1.ContainerPort{{
							ContainerPort: 8000,
							Protocol:      apiv1.ProtocolTCP,
						}},
						VolumeMounts: []apiv1.VolumeMount{{
							Name:      "script-volume",
							MountPath: "/scripts",
						}},
					}},
					Volumes: []apiv1.Volume{{
						Name: "script-volume",
						VolumeSource: apiv1.VolumeSource{
							ConfigMap: &apiv1.ConfigMapVolumeSource{
								LocalObjectReference: apiv1.LocalObjectReference{
									Name: configMapUniqueName,
								},
							},
						},
					}},
				},
			},
		},
	}

	// create cluster with new deployment
	deploymentClient := clientset.AppsV1().Deployments(apiv1.NamespaceDefault)
	_, err = deploymentClient.Create(context.Background(), deployment, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("deploymentClient.Create(): %w", err)
	}

	// create a service for external access
	// https://kubernetes.io/docs/concepts/services-networking/service/
	// In Kubernetes, a Service is a method for exposing a network application
	// that is running as one or more Pods in your cluster.
	service := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "python-app-service",
		},
		Spec: apiv1.ServiceSpec{
			Selector: map[string]string{
				"app": "python-app",
			},
			Ports: []apiv1.ServicePort{{
				Port:       8000,
				TargetPort: intstr.FromInt(8000),
				Protocol:   apiv1.ProtocolTCP,
			}},
			Type: apiv1.ServiceTypeLoadBalancer,
		},
	}

	// create the service
	serviceClient := clientset.CoreV1().Services(apiv1.NamespaceDefault)
	createdService, err := serviceClient.Create(context.Background(), service, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("serviceClient.Create(): %w", err)
	}

	// wait for service to get external IP
	// In a real implementation, you'd want to poll until the external IP is assigned
	// For now, we'll return the service name and port
	fmt.Printf("Service created: %s:%d\n", createdService.Name, createdService.Spec.Ports[0].Port)
	fmt.Printf("Access via: http://%s:%d/echo/\n", createdService.Name, createdService.Spec.Ports[0].Port)

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
