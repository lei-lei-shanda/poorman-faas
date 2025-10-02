// Package faas provides a Python Faas to be deployed on a k8s cluster.
package faas

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

// PythonFaas represents a Python Function as a Service (Faas) to be deployed on a k8s cluster.
type PythonFaas struct {
	// needed by selector
	appName string
	// needed by k8s
	namespace string
	// three kinds of k8s resources
	configMapUUID  string
	deploymentUUID string
	serviceUUID    string
	// user supplied python script
	script string
}

func New(script string) *PythonFaas {
	uuid := uuid.New().String()
	appName := fmt.Sprintf("app-%s", uuid)
	configMapUUID := fmt.Sprintf("configmap-%s", uuid)
	deploymentUUID := fmt.Sprintf("deployment-%s", uuid)
	serviceUUID := fmt.Sprintf("service-%s", uuid)
	return &PythonFaas{
		appName:        appName,
		configMapUUID:  configMapUUID,
		deploymentUUID: deploymentUUID,
		serviceUUID:    serviceUUID,
		script:         script,
	}
}

func (s *PythonFaas) Selector() map[string]string {
	return map[string]string{
		"app": s.appName,
	}
}

// ConfigMap returns a ConfigMap object that contains the Python script.
//
// A ConfigMap is an API object used to store non-confidential data in key-value pairs.
// Pods can consume ConfigMaps as environment variables, command-line arguments, or
// as configuration files in a volume. For more, see:
// https://kubernetes.io/docs/concepts/configuration/configmap/
func (s *PythonFaas) ConfigMap() *apiv1.ConfigMap {
	return &apiv1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: s.configMapUUID,
		},
		Data: map[string]string{"main.py": s.script},
	}
}

// Deployment returns a Deployment object that contains the Python script.
//
// A Deployment manages a set of Pods to run an application workload,
// usually one that doesn't maintain state. For more, see:
// https://kubernetes.io/docs/concepts/workloads/controllers/deployment/
func (s *PythonFaas) Deployment() *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: s.deploymentUUID,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: s.Selector(),
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: s.Selector(),
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{{
						Name:    s.appName,
						Image:   "ghcr.io/astral-sh/uv:python3.12-alpine",
						Command: []string{"uv", "run", "--script", "/scripts/main.py"},
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
									Name: s.configMapUUID,
								},
							},
						},
					}},
				},
			},
		},
	}
}

// Service returns a Service object that exposes the Python Faas.
//
// A Service is a method for exposing a network application
// that is running as one or more Pods in your cluster.
// https://kubernetes.io/docs/concepts/services-networking/service/
func (s *PythonFaas) Service() *apiv1.Service {
	return &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: s.serviceUUID,
		},
		Spec: apiv1.ServiceSpec{
			// https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types
			Type:     apiv1.ServiceTypeClusterIP,
			Selector: s.Selector(),
			Ports: []apiv1.ServicePort{{
				Port:       80,
				Protocol:   apiv1.ProtocolTCP,
				TargetPort: intstr.FromInt(8000),
			}},
		},
	}
}

// Deploy creates the Python Faas on the k8s cluster.
//
// creates in order: configmap -> deployment -> service
func (s *PythonFaas) Deploy(ctx context.Context, clientset *kubernetes.Clientset) error {
	ns := s.namespace
	configMapClient := clientset.CoreV1().ConfigMaps(ns)
	_, err := configMapClient.Create(ctx, s.ConfigMap(), metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("configMapClient.Create(): %w", err)
	}
	deploymentClient := clientset.AppsV1().Deployments(ns)
	_, err = deploymentClient.Create(ctx, s.Deployment(), metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("deploymentClient.Create(): %w", err)
	}
	serviceClient := clientset.CoreV1().Services(ns)
	_, err = serviceClient.Create(ctx, s.Service(), metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("serviceClient.Create(): %w", err)
	}
	return nil
}

// Teardown removes the Python Faas from the k8s cluster.
//
// destroys in reverse order: service -> deployment -> configmap
func (s *PythonFaas) Teardown(ctx context.Context, clientset *kubernetes.Clientset) error {
	ns := s.namespace
	serviceClient := clientset.CoreV1().Services(ns)
	err := serviceClient.Delete(ctx, s.serviceUUID, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("serviceClient.Delete(): %w", err)
	}
	deploymentClient := clientset.AppsV1().Deployments(ns)
	err = deploymentClient.Delete(ctx, s.deploymentUUID, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("deploymentClient.Delete(): %w", err)
	}
	configMapClient := clientset.CoreV1().ConfigMaps(ns)
	err = configMapClient.Delete(ctx, s.configMapUUID, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("configMapClient.Delete(): %w", err)
	}
	return nil
}
