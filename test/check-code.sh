#!/usr/bin/env bash
NAMESPACE="faas"
SERVICE_NAME="faas-gateway"

LB_IP=$(kubectl -n ${NAMESPACE} get svc/${SERVICE_NAME} -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')

echo "LB_IP: ${LB_IP}"

# USER_SERVICE_NAME="service-b1c5fff5-f338-4747-b92c-7948848825d3"
# USER_SERVICE_NAME="service-5421c2a1-8137-4fd0-be7e-87a35d06c4dc"
USER_SERVICE_NAME="service-518483cb-2919-4d16-999a-ae5718645eb8"
# for simple echo service
# curl -X POST "http://${LB_IP}:8080/gateway/${USER_SERVICE_NAME}/echo/" \
#  -H "Content-Type: application/json" \
#  -d '{"message": "Hello, World!"}'

# for mcp service
# first initialize it
echo "=== Initialize Request ==="
curl -X POST "http://${LB_IP}:8080/gateway/${USER_SERVICE_NAME}/mcp" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json,text/event-stream" \
  -w "\nHTTP Status: %{http_code}\n" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "initialize",
    "params": {
      "protocolVersion": "2025-03-26",
      "capabilities": {
        "tools": {},
        "resources": {},
        "prompts": {}
      },
      "clientInfo": {
        "name": "test-client",
        "version": "1.0.0"
      }
    }
  }'

echo -e "\n=== Initialized ===\n"

echo "=== Tools List Request ==="
curl -X POST "http://${LB_IP}:8080/gateway/${USER_SERVICE_NAME}/mcp" \
    -H "Content-Type: application/json" \
    -H "Accept: application/json,text/event-stream" \
    -w "\nHTTP Status: %{http_code}\n" \
    -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/list",
    "params": {}
  }'

echo -e "\n=== Tools Listed ===\n"

echo "=== Tools Call Request ==="
curl -X POST "http://${LB_IP}:8080/gateway/${USER_SERVICE_NAME}/mcp" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json,text/event-stream" \
  -w "\nHTTP Status: %{http_code}\n" \
  -d '{
    "jsonrpc": "2.0",
    "id": 3,
    "method": "tools/call",
    "params": {
      "name": "echo",
      "arguments": {
        "message": {
          "message": "Hello, World!"
        }
      }
    }
  }'

echo -e "\n=== Tools Called ===\n"
