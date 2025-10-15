package helm

import (
	"context"
	"fmt"
	"log/slog"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// DiscoveredChart represents a chart found in the cluster along with any discovery errors
type DiscoveredChart struct {
	Chart Chart
	Error error
}

// DiscoverCharts finds all managed poorman-faas resources in the cluster and reconstructs Charts from them.
// It returns a slice of DiscoveredChart, where each entry may contain a valid Chart or an Error.
// This allows partial discovery - some charts may fail to reconstruct while others succeed.
func DiscoverCharts(ctx context.Context, clientset *kubernetes.Clientset, namespace string, logger *slog.Logger) []DiscoveredChart {
	var discovered []DiscoveredChart

	// Use label selector to filter managed resources at the API level
	labelSelector := LabelManagedBy + "=true"
	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	// List only managed resources using label selector
	serviceClient := clientset.CoreV1().Services(namespace)
	services, err := serviceClient.List(ctx, listOptions)
	if err != nil {
		logger.Error("failed to list services during discovery", "error", err)
		return discovered
	}

	deploymentClient := clientset.AppsV1().Deployments(namespace)
	deployments, err := deploymentClient.List(ctx, listOptions)
	if err != nil {
		logger.Error("failed to list deployments during discovery", "error", err)
		return discovered
	}

	configMapClient := clientset.CoreV1().ConfigMaps(namespace)
	configMaps, err := configMapClient.List(ctx, listOptions)
	if err != nil {
		logger.Error("failed to list configmaps during discovery", "error", err)
		return discovered
	}

	logger.Info("discovering charts from cluster", "namespace", namespace, "total_services", len(services.Items), "total_deployments", len(deployments.Items), "total_configmaps", len(configMaps.Items))

	// Build maps: uuid -> service, uuid -> deployment, uuid -> configmap
	serviceByUUID := make(map[string]*apiv1.Service)
	deploymentByUUID := make(map[string]*appsv1.Deployment)
	configMapByUUID := make(map[string]*apiv1.ConfigMap)

	// Collect all services by UUID (already filtered by label selector)
	for i := range services.Items {
		service := &services.Items[i]

		serviceID, exists := service.Labels[LabelServiceID]
		if !exists {
			discovered = append(discovered, DiscoveredChart{
				Error: fmt.Errorf("service %s missing %s label", service.Name, LabelServiceID),
			})
			continue
		}

		logger.Debug("found managed service", "service", service.Name, "uuid", serviceID)
		serviceByUUID[serviceID] = service
	}

	// Collect all deployments by UUID (already filtered by label selector)
	for i := range deployments.Items {
		deployment := &deployments.Items[i]
		if serviceID, exists := deployment.Labels[LabelServiceID]; exists {
			deploymentByUUID[serviceID] = deployment
		}
	}

	// Collect all configmaps by UUID (already filtered by label selector)
	for i := range configMaps.Items {
		configMap := &configMaps.Items[i]
		if serviceID, exists := configMap.Labels[LabelServiceID]; exists {
			configMapByUUID[serviceID] = configMap
		}
	}

	// Link them up by UUID and reconstruct charts
	for serviceID, service := range serviceByUUID {
		deployment, hasDeployment := deploymentByUUID[serviceID]
		if !hasDeployment {
			discovered = append(discovered, DiscoveredChart{
				Error: fmt.Errorf("no deployment found for service %s with service-id %s", service.Name, serviceID),
			})
			continue
		}

		configMap, hasConfigMap := configMapByUUID[serviceID]
		if !hasConfigMap {
			discovered = append(discovered, DiscoveredChart{
				Error: fmt.Errorf("no configmap found for service %s with service-id %s", service.Name, serviceID),
			})
			continue
		}

		// Reconstruct the Chart from the k8s resources
		chart, err := NewChartFromK8sResources(configMap, deployment, service)
		if err != nil {
			discovered = append(discovered, DiscoveredChart{
				Error: fmt.Errorf("failed to reconstruct chart for service %s: %w", service.Name, err),
			})
			continue
		}

		logger.Debug("successfully reconstructed chart", "service", service.Name, "configmap", configMap.Name, "deployment", deployment.Name)

		discovered = append(discovered, DiscoveredChart{
			Chart: chart,
			Error: nil,
		})
	}

	logger.Info("discovery complete", "total_discovered", len(discovered))

	return discovered
}
