#!/bin/bash

# causes the shell to exit if any invoked command exits with a non-zero status
set -e

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

docker_username="${DOCKER_USERNAME:-nimbleopti}"
image_tag="${IMAGE_TAG:-v1.0.0}"
docker_image_name="${DOCKER_IMAGE_NAME:-${docker_username}/cronjob-n-o-a}"
build_platform="${BUILD_PLATFORM:-local}" # local or all
admin_config="${ADMIN_CONFIG:-false}"

# Remove the old configuration
echo "Removing old configuration..."
kubectl delete -f cronjob/deploy/cronjob.yaml || true
kubectl delete -f cronjob/deploy/configmap.yaml || true
kubectl delete -f cronjob/deploy/default_rbac.yaml || true
kubectl delete -f cronjob/deploy/admin_rbac.yaml || true

# Docker operations
if [[ "$build_platform" == "local" ]]; then
    echo "Building Docker image..."
    docker build -t "$docker_image_name:latest" -f cronjob/Dockerfile .
    echo "Pushing Docker image to registry..."
    docker push "$docker_image_name"
elif [[ "$build_platform" == "all" ]]; then
    echo "Building Docker image for all platforms..."
    docker_target_platform="linux/arm64,linux/amd64"
    docker buildx build cronjob/ \
        --platform "$docker_target_platform" \
        --tag $docker_image_name:$image_tag --tag $docker_image_name:latest \
        --file cronjob/Dockerfile \
        --output type=image,push=true
else
    echo "Invalid BUILD_PLATFORM value. Choose either 'all' or 'local'." >&2
    exit 1
fi

# Apply k8s manifests
echo "Applying k8s manifests..."
if [[ "$admin_config" == "false" ]]; then
    kubectl apply -f cronjob/deploy/default_rbac.yaml
elif [[ "$admin_config" == "true" ]]; then
    kubectl apply -f cronjob/deploy/admin_rbac.yaml
else
    echo "Invalid ADMIN_CONFIG value. Choose either 'true' or 'false'." >&2
    exit 1
fi

kubectl apply -f cronjob/deploy/configmap.yaml
kubectl apply -f cronjob/deploy/cronjob.yaml

echo "Update complete."
