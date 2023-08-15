#!/bin/bash

# Set the shell to exit if any invoked command fails
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

# Configuration variables with default values
defaultImageTag="${IMAGE_TAG:-v1.0.0}"
certManagerVersion="${CERT_MANAGER_VERSION:-v1.11.0}"
defaultSleepTime="1"
defaultDockerUsername="nimbleopti"
defaultImageName="${DOCKER_IMAGE_NAME:-${defaultDockerUsername}/cronjob-n-o-a}"
defaultBuildPlatform="local" # Choices: local or all
isAdminConfig="${ADMIN_CONFIG:-false}"
testCode="${TEST_CODE:-true}" # true or false

# Run Go tests
if [ "$testCode" = "true" ]; then
  echo "Running Go tests..."
  go test ./cronjob/... -count=1 || {
    echo "Go tests failed"
    exit 1
  }
fi

# Check for existing Minikube and delete if it exists
if command -v minikube >/dev/null 2>&1; then
  echo "Deleting existing Minikube instance..."
  minikube delete
else
  echo "No existing Minikube instance found."
fi

# Initialize Minikube
echo "Starting Minikube..."
minikube start --extra-config=apiserver.enable-admission-plugins="MutatingAdmissionWebhook,ValidatingAdmissionWebhook"

# Set kubectl context to Minikube
echo "Configuring kubectl to use Minikube context..."
kubectl config use-context minikube

# Add Helm repo for cert-manager
echo "Setting up Helm for cert-manager..."
helm repo add jetstack https://charts.jetstack.io
helm repo update

# Enable Minikube ingress
echo "Enabling Minikube ingress..."
minikube addons enable ingress
kubectl delete job -n ingress-nginx ingress-nginx-admission-create || true
kubectl delete job -n ingress-nginx ingress-nginx-admission-patch || true

# Install cert-manager using Helm
echo "Installing cert-manager with Helm..."
helm install cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --create-namespace \
  --version "$certManagerVersion" \
  --set installCRDs=true \
  --set defaultIssuerName=letsencrypt-prod \
  --set defaultIssuerKind=ClusterIssuer \
  --wait

# Configure LetsEncrypt as a cluster issuer
echo "Configuring LetsEncrypt Cluster Issuer..."
kubectl apply -f - <<EOF
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    email: smart.apartment.uri@gmail.com
    privateKeySecretRef:
      name: letsencrypt-prod
    server: https://acme-v02.api.letsencrypt.org/directory
    solvers:
    - http01:
        ingress:
          class: nginx
EOF

# Build and push the Docker image
if [[ "$defaultBuildPlatform" == "local" ]]; then
  echo "Building Docker image..."
  docker build -t "$defaultImageName:latest" -f cronjob/Dockerfile .
  echo "Pushing Docker image to registry..."
  docker push "$defaultImageName"
elif [[ "$defaultBuildPlatform" == "all" ]]; then
  echo "Building Docker image for all platforms..."
  targetPlatforms="linux/arm64,linux/amd64"
  docker buildx build cronjob/ \
    --platform "$targetPlatforms" \
    --tag $defaultImageName:$defaultImageTag --tag $defaultImageName:latest \
    --file cronjob/Dockerfile \
    --output type=image,push=true
else
  echo "Invalid BUILD_PLATFORM value. Choose either 'all' or 'local'." >&2
  exit 1
fi

# Apply Kubernetes configurations
echo "Applying k8s manifests..."
if [[ "$isAdminConfig" == "false" ]]; then
  kubectl apply -f cronjob/deploy/default_rbac.yaml
elif [[ "$isAdminConfig" == "true" ]]; then
  kubectl apply -f cronjob/deploy/admin_rbac.yaml
else
  echo "Invalid ADMIN_CONFIG value. Choose either 'true' or 'false'." >&2
  exit 1
fi

kubectl apply -f cronjob/deploy/configmap.yaml
kubectl apply -f cronjob/deploy/cronjob.yaml

# Restart Minikube dashboard
echo "Restarting Minikube dashboard..."
minikube addons disable dashboard
minikube addons enable dashboard
minikube dashboard --url &

echo "Setup complete."
