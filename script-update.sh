#!/bin/bash

# causes the shell to exit if any invoked command exits with a non-zero status
set -e

DOCKER_USERNAME=${DOCKER_USERNAME:-nimbleopti}
DOCKER_IMAGE_NAME=${DOCKER_IMAGE_NAME:-${DOCKER_USERNAME}/nimble-opti-adapter:latest}

# echo "Patching deployment..."
# kubectl patch deployment nimble-opti-adapter-controller-manager -n nimble-opti-adapter-system -p '{"spec":{"template":{"spec":{"containers":[{"name":"kube-rbac-proxy","imagePullPolicy":"Always"},{"name":"manager","imagePullPolicy":"Always"}]}}}}'

echo "Making manifests..."
make manifests

echo "Installing..."
make install

echo "Building Docker image..."
docker build -t $DOCKER_IMAGE_NAME .

echo "Pushing Docker image..."
docker push $DOCKER_IMAGE_NAME

echo "Deploying..."
make deploy IMG=$DOCKER_IMAGE_NAME

echo "Rolling out updates..."
kubectl rollout restart deployment/nimble-opti-adapter-controller-manager -n nimble-opti-adapter-system
kubectl rollout status deployment/nimble-opti-adapter-controller-manager -n nimble-opti-adapter-system

echo "Patching deployment..."
kubectl patch deployment nimble-opti-adapter-controller-manager -n nimble-opti-adapter-system -p '{"spec":{"template":{"spec":{"containers":[{"name":"kube-rbac-proxy","imagePullPolicy":"Always"},{"name":"manager","imagePullPolicy":"Always"}]}}}}'

# delete the nimble-opti-adapter-controller-manager pod to force a restart
kubectl delete pod -n nimble-opti-adapter-system -l control-plane=controller-manager

echo "Update complete."
