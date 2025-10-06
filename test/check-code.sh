#!/usr/bin/env bash
NAMESPACE="faas"
SERVICE_NAME="faas-gateway"

LB_IP=$(kubectl -n ${NAMESPACE} get svc/${SERVICE_NAME} -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')

USER_SERVICE_NAME="service-8c8e00fc-f913-4536-8106-557aa3b954ad"

curl -X POST "http://${LB_IP}:8080/gateway/${USER_SERVICE_NAME}/echo/" \
 -H "Content-Type: application/json" \
 -d '{"message": "Hello, World!"}'