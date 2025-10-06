#!/usr/bin/env bash

NAMESPACE="faas"
SERVICE_NAME="faas-gateway"

LB_IP=$(kubectl -n ${NAMESPACE} get svc/${SERVICE_NAME} -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')

echo "LB_IP: ${LB_IP}"

SCRIPT=$(cat test/echo-web-server.py)

# upload code
curl -X POST "http://${LB_IP}:8080/admin/python" \
 -H "Content-Type: application/json" \
 -d "$(jq -n --arg script "$SCRIPT" '{"script": $script, "option": {"user": "frank", "replica": 1}}')"