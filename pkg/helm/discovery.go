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

	// List all Services and filter by annotation
	// Note: annotations don't support selectors directly, so we need to list all and filter manually
	serviceClient := clientset.CoreV1().Services(namespace)
	services, err := serviceClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		logger.Error("failed to list services during discovery", "error", err)
		return discovered
	}

	logger.Info("discovering charts from cluster", "namespace", namespace, "total_services", len(services.Items))

	// Filter services by annotation and process each one
	for _, service := range services.Items {
		// Check if this service is managed by poorman-faas
		if service.Annotations == nil || service.Annotations[AnnotationManagedBy] != "true" {
			continue
		}

		logger.Debug("found managed service", "service", service.Name)

		// Get the service ID from annotations
		serviceID, exists := service.Annotations[AnnotationServiceID]
		if !exists {
			discovered = append(discovered, DiscoveredChart{
				Error: fmt.Errorf("service %s missing %s annotation", service.Name, AnnotationServiceID),
			})
			continue
		}

		// Find the corresponding Deployment
		deploymentClient := clientset.AppsV1().Deployments(namespace)
		deployments, err := deploymentClient.List(ctx, metav1.ListOptions{})
		if err != nil {
			discovered = append(discovered, DiscoveredChart{
				Error: fmt.Errorf("failed to list deployments for service %s: %w", service.Name, err),
			})
			continue
		}

		var matchingDeployment *appsv1.Deployment
		for i := range deployments.Items {
			d := &deployments.Items[i]
			if d.Annotations != nil && d.Annotations[AnnotationServiceID] == serviceID {
				matchingDeployment = d
				break
			}
		}

		if matchingDeployment == nil {
			discovered = append(discovered, DiscoveredChart{
				Error: fmt.Errorf("no deployment found for service %s with service-id %s", service.Name, serviceID),
			})
			continue
		}

		// Find the corresponding ConfigMap
		configMapClient := clientset.CoreV1().ConfigMaps(namespace)
		configMaps, err := configMapClient.List(ctx, metav1.ListOptions{})
		if err != nil {
			discovered = append(discovered, DiscoveredChart{
				Error: fmt.Errorf("failed to list configmaps for service %s: %w", service.Name, err),
			})
			continue
		}

		var matchingConfigMap *apiv1.ConfigMap
		for i := range configMaps.Items {
			cm := &configMaps.Items[i]
			if cm.Annotations != nil && cm.Annotations[AnnotationServiceID] == serviceID {
				matchingConfigMap = cm
				break
			}
		}

		if matchingConfigMap == nil {
			discovered = append(discovered, DiscoveredChart{
				Error: fmt.Errorf("no configmap found for service %s with service-id %s", service.Name, serviceID),
			})
			continue
		}

		// Reconstruct the Chart from the k8s resources
		chart, err := NewChartFromK8sResources(matchingConfigMap, matchingDeployment, &service)
		if err != nil {
			discovered = append(discovered, DiscoveredChart{
				Error: fmt.Errorf("failed to reconstruct chart for service %s: %w", service.Name, err),
			})
			continue
		}

		logger.Debug("successfully reconstructed chart", "service", service.Name, "configmap", matchingConfigMap.Name, "deployment", matchingDeployment.Name)

		discovered = append(discovered, DiscoveredChart{
			Chart: chart,
			Error: nil,
		})
	}

	logger.Info("discovery complete", "total_discovered", len(discovered))

	return discovered
}
