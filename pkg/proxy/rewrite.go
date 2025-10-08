package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"poorman-faas/pkg/util"
	"strings"
)

// RewriteURL rewrites request URL to lb to request URL to internal service.
//
// Incoming: https://{lb-ip}/api/{svc-name}/{path-suffix}
// Outgoing: http://{svc-name}.{ns}.svc.cluster.local/{path-suffix}
func RewriteURL(pathPrefix string, namespace string, getServiceName func(*http.Request) string) func(*httputil.ProxyRequest) {
	return func(req *httputil.ProxyRequest) {
		serviceName := getServiceName(req.In)
		newPath := strings.TrimPrefix(req.In.URL.Path, fmt.Sprintf("%s/%s", pathPrefix, serviceName))
		newHost := util.K8SInternalDNSName(namespace, serviceName)

		req.Out.URL = &url.URL{
			Scheme: "http",
			Host:   newHost,
			Path:   newPath,
		}
	}
}
