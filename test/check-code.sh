#!/usr/bin/env bash
NAMESPACE="faas"
SERVICE_NAME="faas-gateway"

LB_IP=$(kubectl -n ${NAMESPACE} get svc/${SERVICE_NAME} -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')

echo "LB_IP: ${LB_IP}"

# USER_SERVICE_NAME="service-b1c5fff5-f338-4747-b92c-7948848825d3"
USER_SERVICE_NAME="service-5421c2a1-8137-4fd0-be7e-87a35d06c4dc"

curl -X POST "http://${LB_IP}:8080/gateway/${USER_SERVICE_NAME}/echo/" \
 -H "Content-Type: application/json" \
 -d '{"message": "Hello, World!"}'