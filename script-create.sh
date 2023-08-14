#!/bin/bash

# causes the shell to exit if any invoked command exits with a non-zero status
set -e
set -o pipefail # Fail the script if any command in a pipeline fails

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
image_tag="${IMAGE_TAG:-v1.0.0}"
cert_manager_version="${CERT_MANAGER_VERSION:-v1.11.0}"
sleep_time="${SLEEP_TIME:-1}"
docker_username="${DOCKER_USERNAME:-nimbleopti}"
image_name="${DOCKER_IMAGE_NAME:-${docker_username}/cronjob-n-o-a}"
build_platform="${BUILD_PLATFORM:-local}"
admin_config="${ADMIN_CONFIG:-false}"
test_code="${TEST_CODE:-true}"

# Run Go tests
if [ "$test_code" = "true" ]; then
  echo "Running Go tests..."
  go test ./... || {
    echo "Go tests failed"
    exit 1
  }
fi

# Check for minikube and delete if exists
if command -v minikube >/dev/null 2>&1; then
  echo "Deleting existing Minikube instance..."
  minikube delete
else
  echo "No existing Minikube instance found."
fi

# Start minikube
echo "Starting Minikube..."
minikube start --extra-config=apiserver.enable-admission-plugins="MutatingAdmissionWebhook,ValidatingAdmissionWebhook"

echo "Configuring kubectl to use Minikube context..."
kubectl config use-context minikube

# Setup helm for cert-manager
echo "Setting up Helm for cert-manager..."
helm repo add jetstack https://charts.jetstack.io
helm repo update

echo "Enabling Minikube ingress..."
minikube addons enable ingress
kubectl delete job -n ingress-nginx ingress-nginx-admission-create || true
kubectl delete job -n ingress-nginx ingress-nginx-admission-patch || true

echo "Installing cert-manager with Helm..."
helm install cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --create-namespace \
  --version "$cert_manager_version" \
  --set installCRDs=true \
  --set defaultIssuerName=letsencrypt-prod \
  --set defaultIssuerKind=ClusterIssuer \
  --wait

# Apply letsencrypt cluster issuer
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

echo "Making manifests..."
make manifests

echo "Installing..."
make install

# Docker operations
case "$build_platform" in
"local")
  docker build -t "$image_name:latest" -f Dockerfile .
  docker push "$image_name"
  ;;
"all")
  docker_target_platform="linux/arm64,linux/amd64"
  docker buildx build . \
    --platform "$docker_target_platform" \
    --tag $image_name:$image_tag --tag $image_name:latest \
    --file Dockerfile \
    --output type=image,push=true
  ;;
*)
  echo "Invalid BUILD_PLATFORM value. Choose either 'all' or 'local'." >&2
  exit 1
  ;;
esac

echo "Deploying..."
make deploy IMG=$image_name:latest

# Allow the system a moment to process the previous command
echo "Waiting for ${sleep_time}s..."
sleep "$sleep_time"

# Patch the deployment
echo "Patching deployment to ensure images are always pulled..."
kubectl patch deployment nimble-opti-adapter-controller-manager -n nimble-opti-adapter-system -p '{"spec":{"template":{"spec":{"containers":[{"name":"kube-rbac-proxy","imagePullPolicy":"Always"},{"name":"manager","imagePullPolicy":"Always"}]}}}}'

# Create a service for metrics
echo "Creating service for metrics..."
kubectl apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: metrics-svc
  namespace: nimble-opti-adapter-system
spec:
  ports:
  - name: https
    protocol: TCP
    port: 8080
    targetPort: 8080
  selector:
    control-plane: controller-manager
  type: ClusterIP
EOF

# Handle Minikube dashboard
echo "Restarting Minikube dashboard..."
minikube addons disable dashboard
minikube addons enable dashboard
minikube dashboard --url &

# Display the URL for the metrics-svc service
echo "Fetching the URL for the metrics-svc service..."
minikube service metrics-svc -n nimble-opti-adapter-system --url

echo "Setup complete."
