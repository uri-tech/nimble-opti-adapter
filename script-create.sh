#!/bin/bash
set -e

DOCKER_USERNAME="${DOCKER_USERNAME:-nimbleopti}"
IMAGE_TAG="${IMAGE_TAG:-v1.0.0}"
DOCKER_IMAGE_NAME="${DOCKER_IMAGE_NAME:-${DOCKER_USERNAME}/nimble-opti-adapter}"
CERT_MANAGER_VERSION="${CERT_MANAGER_VERSION:-v1.11.0}"
SLEEP_TIME="${SLEEP_TIME:-1}"
BUILD_PLATFORM="${BUILD_PLATFORM:-local}" # local or all

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
  --version "$CERT_MANAGER_VERSION" \
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

# Make manifests and install
echo "Generating manifests..."
make manifests

echo "Installing application..."
make install

# Docker operations
if [[ "$BUILD_PLATFORM" == "local" ]]; then
  echo "Building Docker image..."
  docker build -t "$DOCKER_IMAGE_NAME:latest" .
  echo "Pushing Docker image to registry..."
  docker push "$DOCKER_IMAGE_NAME"
elif [[ "$BUILD_PLATFORM" == "all" ]]; then
  echo "Building Docker image for all platforms..."
  DOCKER_TARGET_PLATFORM="linux/arm64,linux/amd64"
  docker buildx build . \
    --platform $DOCKER_TARGET_PLATFORM \
    --tag $DOCKER_IMAGE_NAME:$IMAGE_TAG --tag $DOCKER_IMAGE_NAME:latest \
    --file ./Dockerfile \
    --output type=image,push=true
else
  echo "Invalid BUILD_PLATFORM value. Choose either 'all' or 'local'."
fi

echo "Deploying..."
make deploy IMG=$DOCKER_IMAGE_NAME

# Allow the system a moment to process the previous command
echo "Waiting for ${SLEEP_TIME}s..."
sleep "$SLEEP_TIME"

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
