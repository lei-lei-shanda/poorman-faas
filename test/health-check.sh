#!/usr/bin/env bash

NAMESPACE="faas"
SERVICE_NAME="faas-gateway"

LB_IP=$(kubectl -n ${NAMESPACE} get svc/${SERVICE_NAME} -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')

echo "LB_IP: ${LB_IP}"

if [ -z "${LB_IP}" ]; then
    echo "LB_IP is empty, please check if load balancer is ready"
    exit 1
fi

curl -v "http://${LB_IP}:8080/health"