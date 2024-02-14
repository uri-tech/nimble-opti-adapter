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
  echo "Running Go test files..."
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
# clean up after the initialization process and ignore errors (with || true)
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
# defaultIssuerName=letsencrypt-prod: sets the default issuer to be used when creating new certificate resources.
# defaultIssuerKind=ClusterIssuer: allowing for a broader scope of certificate management.

# Apply letsencrypt cluster issuer
echo "Configuring LetsEncrypt Cluster Issuer..."
kubectl apply -f - <<EOF
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: clusterissuer-letsencrypt-http01
spec:
  acme:
    email: smart.apartment.uri@gmail.com
    privateKeySecretRef:
      name: acme-letsencrypt-prod
    server: https://acme-v02.api.letsencrypt.org/directory
    solvers:
    - http01:
        ingress:
          class: nginx
EOF

# ACME Account Registration: When you create a ClusterIssuer and it is used for the first time to obtain a certificate, cert-manager will register an ACME account with the email specified in the ClusterIssuer definition and generate a private key for that account.
# Secret Creation: cert-manager will then create a Kubernetes secret (in this case, acme-letsencrypt-prod) in the same namespace as the cert-manager pod. This secret will contain the generated private key.
# Certificate Issuance: The private key in the acme-letsencrypt-prod secret is used for subsequent communications with the ACME server (Let's Encrypt) for operations like proving domains ownership and requesting certificates.
# http01 is one of the challenge types used by the Automated Certificate Management Environment (ACME) protocol to verify domain ownership.
# http01:
#   1. Challenge Initiation: The ACME client (cert-manager) requests a certificate for a domain from the ACME server (like Let's Encrypt) and agrees to perform an http01 challenge to prove control over the domain.
#   2. Token and File Creation: The ACME server provides a token to the client. The client then creates a file containing a specific value derived from this token and its account key.
#   3. File Placement: The client places this file on the web server at a specific URL under the /.well-known/acme-challenge/ directory. For example, for example.com domain, the file be accessible at http://example.com/.well-known/acme-challenge/<token>.
#   4. Verification by ACME Server: The ACME server then makes an HTTP request to the URL where the file was placed. If the server finds the file and its contents match what's expected, the domain ownership is considered verified.
#   5. Certificate Issuance: Once all requested domains are verified and the challenge's status is updated to indicate success.
#   6. Polling for Validation Status: The ACME client periodically sends requests to the ACME server to check the status of the challenge.
#   7. Requesting the Certificate: After all challenges are successfully validated, the ACME client sends a final request to the ACME server to issue the certificate for the validated domains.
#   8. Certificate Issuance: The ACME server issues the certificate and provides it to the client in the response to the final request. The ACME client then retrieves the certificate and stores them.
#   9. Automatic Renewal: The ACME client is also responsible for monitoring the certificate's expiration and will automatically repeat the process to renew the certificate when it's nearing expiration.
# ingress.class: nginx - it will manage the challenge process through the Ingress resources.

echo "Making manifests files of the operator according to the makefile..."
make manifests

echo "Installing the operator..."
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

echo "Deploying the image to docker hub..."
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

# Create ingress for testing
cat <<EOF | kubectl apply -f -
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ingress-example
  namespace: default
  labels:
    nimble.opti.adapter/enabled: "true"
  annotations:
    cert-manager.io/cluster-issuer: clusterissuer-letsencrypt-http01 # Use the cluster issuer created earlier for automatic certificate management
    nginx.ingress.kubernetes.io/backend-protocol: "HTTPS" # passthrough the encripted HTTPS traffic as is to the backend
    # acme.cert-manager.io/http01-edit-in-place: 'true' # This annotation is not required for cert-manager v1.11.0
spec:
  ingressClassName: nginx
  tls:
    - hosts:
        - example.127.0.0.1.nip.io # Replace with the domain you want to expose
      secretName: tls-letsencrypt-example
  rules:
    - host: example.127.0.0.1.nip.io # Replace with the domain you want to expose
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: service-example # Replace with the name of the service you want to expose
                port:
                  number: 443
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
