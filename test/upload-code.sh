#!/usr/bin/env bash

NAMESPACE="faas"
SERVICE_NAME="faas-gateway"

LB_IP=$(kubectl -n ${NAMESPACE} get svc/${SERVICE_NAME} -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')

echo "LB_IP: ${LB_IP}"

# Base64 encode the script
SCRIPT=$(cat test/echo-web-server.py | base64)

# Base64 encode the dotFile
DOTFILE=$(cat test/echo-dotfile.ini | base64)

# upload code
curl -X POST "http://${LB_IP}:8080/admin/python" \
 -H "Content-Type: application/json" \
 -d "$(jq -n --arg script "$SCRIPT" --arg dotFile "$DOTFILE" '{"script": $script, "dotFile": $dotFile, "option": {"user": "frank", "replica": 1}}')"