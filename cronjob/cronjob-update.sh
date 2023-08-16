#!/bin/bash

# causes the shell to exit if any invoked command exits with a non-zero status
set -e
set -o pipefail # Fail the script if any command in a pipeline fails

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

# Configuration variables with default values
docker_username="${DOCKER_USERNAME:-nimbleopti}"
image_tag="${IMAGE_TAG:-v1.0.0}"
docker_image_name="${DOCKER_IMAGE_NAME:-${docker_username}/cronjob-n-o-a}"
admin_config="${ADMIN_CONFIG:-false}"
build_platform="${BUILD_PLATFORM:-all}" # local or all
testCode="${TEST_CODE:-false}"          # true or false

# Run Go tests
if [ "$testCode" = "true" ]; then
    echo "Running Go tests..."
    go test ./cronjob/... || {
        echo "Go tests failed"
        exit 1
    }
fi

# Remove the old configuration
kubectl delete -f cronjob/deploy/cronjob.yaml || true
kubectl delete -f cronjob/deploy/configmap.yaml || true
kubectl delete -f cronjob/deploy/default_rbac.yaml || true
kubectl delete -f cronjob/deploy/admin_rbac.yaml || true

# Docker operations
case "$build_platform" in
"local")
    docker build -t "$docker_image_name:latest" -f cronjob/Dockerfile .
    docker push "$docker_image_name"
    ;;
"all")
    docker_target_platform="linux/arm64,linux/amd64"
    docker buildx build . \
        --platform "$docker_target_platform" \
        --tag $docker_image_name:$image_tag --tag $docker_image_name:latest \
        --file cronjob/Dockerfile \
        --output type=image,push=true
    ;;
*)
    echo "Invalid BUILD_PLATFORM value. Choose either 'all' or 'local'." >&2
    exit 1
    ;;
esac

# Apply k8s manifests
case "$admin_config" in
"false")
    kubectl apply -f cronjob/deploy/default_rbac.yaml
    ;;
"true")
    kubectl apply -f cronjob/deploy/admin_rbac.yaml
    ;;
*)
    echo "Invalid ADMIN_CONFIG value. Choose either 'true' or 'false'." >&2
    exit 1
    ;;
esac

kubectl apply -f cronjob/deploy/configmap.yaml
kubectl apply -f cronjob/deploy/cronjob.yaml

echo "Update complete."
