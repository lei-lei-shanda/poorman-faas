package proxy

import (
	"log/slog"
	"net/http/httputil"
	"net/url"

	apiv1 "k8s.io/api/core/v1"
)

// RewriteMCPPath rewrites the path of the request to the MCP service.
//
// Incoming: https://{k8s_ip}/v1/{pod_uuid}/mcp
// Outgoing: http://{internal_pod_ip}/mcp
func RewriteMCPPath(lookup func(string) (*apiv1.Pod, error)) func(*httputil.ProxyRequest) {
	return func(req *httputil.ProxyRequest) {
		podID := req.In.PathValue("pod_uuid")
		pod, err := lookup(podID)
		if err != nil {
			// TODO: rewrite is assumed to never fail. how to handle error here when `lookup` fails?
			slog.Error("failed to lookup pod", "error", err)
			return
		}

		req.Out.URL = &url.URL{
			Scheme: "http",
			Host:   pod.Status.PodIP,
			Path:   "/mcp",
		}
	}
}
