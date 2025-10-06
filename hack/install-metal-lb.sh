#!/usr/bin/env bash

# oct-06: this scriptdoes not work (line 16). keep this for legacy reason.
# 
# follow https://kind.sigs.k8s.io/docs/user/loadbalancer/ instead.

# Install MetalLB v0.14.8 (compatible with Kubernetes 1.25+)
kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/v0.14.8/config/manifests/metallb-native.yaml

# Wait for MetalLB to be ready
echo "Waiting for MetalLB controller to be ready..."
kubectl wait --namespace metallb-system \
  --for=condition=ready pod \
  --selector=app=metallb \
  --timeout=90s

KIND_ADDRESS_RANGE=$(podman network inspect kind | jq '.[0].IPAM.Config[0].Subnet' -r)
if [[ "$KIND_ADDRESS_RANGE" == *.0.0/16 ]]; then
  PREFIX=${KIND_ADDRESS_RANGE%".0.0/16"}
  LB_ADDRESS_RANGE=$PREFIX.255.200-$PREFIX.255.250
else
  # TODO: add support for other CIDR blocks
  echo "Only x.y.0.0/16 subnets are supported by this script. Your kind subnet is $KIND_ADDRESS_RANGE."
  exit 1
fi

# Configure MetalLB using the new CRD-based configuration (v0.13+)
kubectl apply -f - <<EOF
apiVersion: metallb.io/v1beta1
kind: IPAddressPool
metadata:
  name: default-pool
  namespace: metallb-system
spec:
  addresses:
  - $LB_ADDRESS_RANGE
---
apiVersion: metallb.io/v1beta1
kind: L2Advertisement
metadata:
  name: default-l2-advert
  namespace: metallb-system
spec:
  ipAddressPools:
  - default-pool
EOF