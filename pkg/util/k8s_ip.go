package util

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// K8SInternalDNSName returns the internal domain name (without schema).
//
// {svc-name}.{ns}.svc.cluster.local
func K8SInternalDNSName(namespace string, serviceName string) string {
	// TODO: use url.Parse to check
	return fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, namespace)
}

// K8sExternalDomainName returns the gateway url (with schema).
//
// https://{lb-ip}:{lb-port}/{gateway-prefix}/{svc-name}
func K8sExternalDomainName(ctx context.Context, clientset *kubernetes.Clientset, loadBalancerPort int, gatewayPrefix string, namespace string, serviceName string) (string, error) {
	svc, err := clientset.CoreV1().Services(namespace).Get(context.Background(), serviceName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("clientset.CoreV1().Services(%s).Get(%s): %w", namespace, serviceName, err)
	}

	// TODO: use url.Parse to check
	// TODO: already checked that service is read when creating helm.Chart
	LoadBalancerIP := svc.Status.LoadBalancer.Ingress[0].IP
	if LoadBalancerIP == "" || LoadBalancerIP == "<pending>" {
		return "", fmt.Errorf("svc.Status.LoadBalancer.Ingress[0].IP is empty or pending")
	}
	return fmt.Sprintf("https://%s:%d%s/%s", LoadBalancerIP, loadBalancerPort, gatewayPrefix, serviceName), nil
}
