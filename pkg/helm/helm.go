// Package helm templates various k8s resouces to deploy a Python Function as a Service (Faas).
//
// This works similiarly like kustomize or helm tools.
package helm

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

// Chart hydrates various k8s resources via template.
// These resources represent a Python Function as a Service (Faas), like a helm chart.
//
// These resources are:
//   - configmap [Chart.ConfigMap]
//   - deployment [Chart.Deployment]
//   - service [Chart.Service]
//
// One can then deploy it with [Chart.Deploy] and [Chart.Teardown].
// Or Dump them with [Chart.ToYAML] and apply them with `kubectl apply -f <yaml-string>`.
type Chart struct {
	// needed by selector
	appName string
	// needed by k8s
	Namespace string
	// K8s resource UUID, should be RFC-1035 compliant:
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#rfc-1035-label-names
	configMapUUID  string
	deploymentUUID string
	serviceUUID    string
	// user supplied python script
	script []byte
	// user supplied dot file
	dotFile []byte
	env     map[string]string
}

func NewChart(namespace string, scriptBase64 string, dotFileBase64 string) (Chart, error) {
	uuid := uuid.New().String()
	// TODO: name should be RFC-1035 compliant
	appName := fmt.Sprintf("app-%s", uuid)
	configMapUUID := fmt.Sprintf("configmap-%s", uuid)
	deploymentUUID := fmt.Sprintf("deployment-%s", uuid)
	serviceUUID := fmt.Sprintf("service-%s", uuid)

	// decode base64 script
	scriptBytes, err := base64.StdEncoding.DecodeString(scriptBase64)
	if err != nil {
		return Chart{}, fmt.Errorf("base64.DecodeString(script): %w", err)
	}

	// decode base64 dotFile
	dotFileBytes, err := base64.StdEncoding.DecodeString(dotFileBase64)
	if err != nil {
		return Chart{}, fmt.Errorf("base64.DecodeString(dotFile): %w", err)
	}

	// validate PEP 723 metadata
	schema, err := NewMetadata(string(scriptBytes))
	if err != nil {
		return Chart{}, fmt.Errorf("NewMetadata(): %w", err)
	}

	if !schema.Validate() {
		return Chart{}, fmt.Errorf("script is not PEP 723 compliant")
	}

	// validate dot file
	env, err := godotenv.Parse(bytes.NewReader(dotFileBytes))
	if err != nil {
		return Chart{}, fmt.Errorf("godotenv.Parse(): %w", err)
	}

	return Chart{
		appName:        appName,
		Namespace:      namespace,
		configMapUUID:  configMapUUID,
		deploymentUUID: deploymentUUID,
		serviceUUID:    serviceUUID,
		script:         scriptBytes,
		dotFile:        dotFileBytes,
		env:            env,
	}, nil
}

func (s Chart) Selector() map[string]string {
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
func (s Chart) ConfigMap() *apiv1.ConfigMap {
	return &apiv1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: s.Namespace,
			Name:      s.configMapUUID,
		},
		Data: map[string]string{"main.py": string(s.script)},
	}
}

// Deployment returns a Deployment object that contains the Python script.
//
// A Deployment manages a set of Pods to run an application workload,
// usually one that doesn't maintain state. For more, see:
// https://kubernetes.io/docs/concepts/workloads/controllers/deployment/
func (s Chart) Deployment() *appsv1.Deployment {

	envVars := make([]apiv1.EnvVar, 0, len(s.env))
	for k, v := range s.env {
		envVars = append(envVars, apiv1.EnvVar{
			Name:  k,
			Value: v,
		})
	}
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: s.Namespace,
			Name:      s.deploymentUUID,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: s.Selector(),
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: s.Selector(),
					Annotations: map[string]string{
						"vke.volcengine.com/burst-to-vci": "enforce",
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{{
						Name:    s.appName,
						Image:   "ghcr.io/astral-sh/uv:python3.12-alpine",
						Command: []string{"uv", "run", "--script", "/scripts/main.py"},
						Ports: []apiv1.ContainerPort{{
							// todo: make port configurable
							ContainerPort: 8000,
							Protocol:      apiv1.ProtocolTCP,
						}},
						Env: envVars,
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
func (s Chart) Service() *apiv1.Service {
	return &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: s.Namespace,
			Name:      s.serviceUUID,
		},
		Spec: apiv1.ServiceSpec{
			// https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types
			Type:     apiv1.ServiceTypeClusterIP,
			Selector: s.Selector(),
			Ports: []apiv1.ServicePort{{
				Port:       80,
				Protocol:   apiv1.ProtocolTCP,
				TargetPort: intstr.FromInt32(8000),
			}},
		},
	}
}

// Deploy creates the Python Faas on the k8s cluster.
//
// creates in order: configmap -> deployment -> service
func (s Chart) Deploy(ctx context.Context, clientset *kubernetes.Clientset) error {
	ns := s.Namespace
	configMapClient := clientset.CoreV1().ConfigMaps(ns)
	// TODO: use Apply instead of Create?
	// _, err := configMapClient.Apply(ctx, s.ConfigMap(), metav1.ApplyOptions{})
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
func (s *Chart) Teardown(ctx context.Context, clientset *kubernetes.Clientset) error {
	ns := s.Namespace
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

// ToYAML dumps the Python Faas as a single YAML string.
//
// one can then uses
//
//	```bash
//	kubectl apply -f <yaml-string>
//	```
//
// to apply the YAML to the k8s cluster.
func (s *Chart) ToYAML() (string, error) {
	cm := s.ConfigMap()
	deployment := s.Deployment()
	service := s.Service()
	cmYaml, err := yaml.Marshal(cm)
	if err != nil {
		return "", fmt.Errorf("yaml.Marshal(cm): %w", err)
	}
	deploymentYaml, err := yaml.Marshal(deployment)
	if err != nil {
		return "", fmt.Errorf("yaml.Marshal(deployment): %w", err)
	}
	serviceYaml, err := yaml.Marshal(service)
	if err != nil {
		return "", fmt.Errorf("yaml.Marshal(service): %w", err)
	}
	// concat all yaml with triple dash to separate them
	return fmt.Sprintf("%s---\n%s---\n%s", string(cmYaml), string(deploymentYaml), string(serviceYaml)), nil
}
