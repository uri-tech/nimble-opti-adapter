#!/bin/bash

# causes the shell to exit if any invoked command exits with a non-zero status
set -e
set -o pipefail # Fail the script if any command in a pipeline fails

# Configuration variables with default values
docker_username="${DOCKER_USERNAME:-nimbleopti}"
image_tag="${IMAGE_TAG:-v1.0.0}"
docker_image_name="${DOCKER_IMAGE_NAME:-${docker_username}/nimble-opti-adapter}"
build_platform="${BUILD_PLATFORM:-local}" # local or all
testCode="${TEST_CODE:-true}"             # true or false

# Parse command line options
while getopts ":e:" opt; do
    case $opt in
    e)
        export $OPTARG
        ;;
    \?)
        echo "Invalid option: -$OPTARG" >&2
        exit 1
        ;;
    :)
        echo "Option -$OPTARG requires an argument." >&2
        exit 1
        ;;
    esac
done

# Run Go tests
if [ "$testCode" = "true" ]; then
    echo "Running Go tests..."
    go test ./... || {
        echo "Go tests failed"
        exit 1
    }
fi

echo "Making manifests..."
make manifests

echo "Installing..."
make install

# Docker operations
case "$build_platform" in
"local")
    docker build -t "$docker_image_name:latest" -f Dockerfile .
    docker push "$docker_image_name"
    ;;
"all")
    docker_target_platform="linux/arm64,linux/amd64"
    docker buildx build . \
        --platform "$docker_target_platform" \
        --tag $docker_image_name:$image_tag --tag $docker_image_name:latest \
        --file Dockerfile \
        --output type=image,push=true
    ;;
*)
    echo "Invalid BUILD_PLATFORM value. Choose either 'all' or 'local'." >&2
    exit 1
    ;;
esac

echo "Deploying..."
make deploy IMG=$docker_image_name:latest

echo "Rolling out updates..."
kubectl rollout restart deployment/nimble-opti-adapter-controller-manager -n nimble-opti-adapter-system
kubectl rollout status deployment/nimble-opti-adapter-controller-manager -n nimble-opti-adapter-system

echo "Patching deployment..."
kubectl patch deployment nimble-opti-adapter-controller-manager -n nimble-opti-adapter-system -p '{"spec":{"template":{"spec":{"containers":[{"name":"kube-rbac-proxy","imagePullPolicy":"Always"},{"name":"manager","imagePullPolicy":"Always"}]}}}}'

# delete the nimble-opti-adapter-controller-manager pod to force a restart
kubectl delete pod -n nimble-opti-adapter-system -l control-plane=controller-manager

echo "Update complete."
