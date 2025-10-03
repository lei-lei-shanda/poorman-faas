# Some thoughts

> Do we use OpenAPI codegen or not?

Learning towards Yes, but there are some issues.

There are four endpoints planned currently:
- (A) `GET /faas/python {"user": "foo"}` returns a list of python function service for that user.
- (B) `POST /faas/python --form user=foo --form data=@main.py` uploads the file `main.py` for user `foo`.
- (C) `/gateway/{service-uuid}` reverse proxy request to underlying python function server.

Result with `oapi-codegen` gives the following interface:

```go
// ServerInterface represents all server handlers.
type ServerInterface interface {
	// List MCP services for a user
	// (GET /mcp-service)
	ListMCPService(w http.ResponseWriter, r *http.Request, params ListMCPServiceParams)
	// Create MCP service
	// (POST /mcp-service)
	CreateMCPService(w http.ResponseWriter, r *http.Request)
	// Proxy GET request to underlying service
	// (GET /{service-uuid}/mcp)
	ProxyGetMCP(w http.ResponseWriter, r *http.Request, serviceUUID openapi_types.UUID)
	// Proxy POST request to underlying service
	// (POST /{service-uuid}/mcp)
	ProxyPostMCP(w http.ResponseWriter, r *http.Request, serviceUUID openapi_types.UUID)
}

// The interface specification for the client above.
type ClientInterface interface {
	// ListMCPService request
	ListMCPService(ctx context.Context, params *ListMCPServiceParams, reqEditors ...RequestEditorFn) (*http.Response, error)

	// CreateMCPServiceWithBody request with any body
	CreateMCPServiceWithBody(ctx context.Context, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*http.Response, error)

	// ProxyGetMCP request
	ProxyGetMCP(ctx context.Context, serviceUUID openapi_types.UUID, reqEditors ...RequestEditorFn) (*http.Response, error)

	// ProxyPostMCPWithBody request with any body
	ProxyPostMCPWithBody(ctx context.Context, serviceUUID openapi_types.UUID, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*http.Response, error)

	ProxyPostMCP(ctx context.Context, serviceUUID openapi_types.UUID, body ProxyPostMCPJSONRequestBody, reqEditors ...RequestEditorFn) (*http.Response, error)

	ProxyPostMCPWithTextBody(ctx context.Context, serviceUUID openapi_types.UUID, body ProxyPostMCPTextRequestBody, reqEditors ...RequestEditorFn) (*http.Response, error)
}
```


Problems with them:
1. for (B), it does not generate form value `{user}` or `{data}` parsing and validation code in either server or client code.
2. for (C), the path value `{service-uuid}` is explicit in the server signature, make it incompatible with `net.Handler`
3. client code for proxy is not needed.

> Do we use form submission?

Now leaning towards No. Rather `POST json-body` over `POST form-data`, because:
1. can read code from other sources, i.e. github gist, s3 buckets etc. this makes sharing slightly easy.
2. still can use `oapi-codegen` (form value is not really supported here).
3. integrate a cli in front is still easy.

If we look over to github gist, they use a JSON body too. [See this StackOverflow answer](https://stackoverflow.com/questions/34048241/how-to-create-a-gist-on-command-line)

> How to load user uploaded code into container image? 

A few options:
1. add user code when build container image:
	1. imitate E2B api, make CI process a shell command away. [see guide here](https://e2b.dev/docs/sandbox-template)
	2. have a dedicated build server, like in github action or gitlab workflow.
2. don't touch container image, load it other ways:
	1. via custom API: build a custom image that support `/api/file/upload`
	2. via ConfigMap: build a custom image that loads entry point script with ConfigMap
	3. via mounted volume: ???

We currently went with option 2 (with ConfigMap), with the following drawbacks:
- No multiple files, or zip archive. Just a single python script. (we additionally requires `PEP-723` compliance)
- No additional dependencies. `uv` may install some on node startup, this maybe an issue, so a dedicated image with lots of dependencies may be needed.
- service is stateless. This is true for no, but may not be true in the future.

In the future, we lean towards option 1 or 4, where we accept a container image instead.
1. image building step happens locally,
2. image building happens on a dedicated server.

> How to expose Faas to outside of K8s?

A few options (as I am aware of currently):
1. Directly. 
	1. On Deployment of Faas, creates a Service of `LoadBalancer` kind, and then 
	2. (optional) modify the cluster Ingress kind to route traffic to it.
	3. return `{faas-ip}` to user.
2. Through a gateway. 
	1. On bootstrap, create a dedicated deployment for `gateway` and expose it via Ingress.
	2. On deployment of faas, create a service of `ClusterIP` or `NodePort` kind.
	3. (optional) use DNS addon to let `gateway` dicover it?
	4. return `{gateway-ip}/api/{fass-uuid}` to user.

Leaning towards option 2 (gateway) because:
1. recycling a service is easy. Add a middleware to monitor last access time, and periodically reclaim them. Otherwise we copy `miro-sandbox` approach, modify pod annotation every time it is been accessed.
2. Adding observability is easy. Again, just add a middleware. Otherwise we need to use sidecar pattern to enhance each Faas pod.

> Gateway inside or outside k8s cluster?

Considering k8s cluster has its own DNS and etc, it **has to be** inside the same k8s cluster that spawns Faas pods.

the more interesting question is how to update this gateway service, when code changes.
1. If the gateway service stateless, can just replace it.
2. No need to tear down the entire cluster, just update this single deployment + service.