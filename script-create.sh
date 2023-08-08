#!/bin/bash
set -e

DOCKER_USERNAME=${DOCKER_USERNAME:-nimbleopti}
DOCKER_IMAGE_NAME=${DOCKER_IMAGE_NAME:-${DOCKER_USERNAME}/nimble-opti-adapter:latest}
CERT_MANAGER_VERSION=${CERT_MANAGER_VERSION:-v1.11.0}
SLEEP_TIME=${SLEEP_TIME:-1}

# echo "Login to docker..."
# echo $DOCKER_PASSWORD | docker login -u $DOCKER_USERNAME --password-stdin

echo "Deleting Minikube..."
minikube delete

echo "Starting Minikube..."
minikube start

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

echo "Patching deployment..."
kubectl patch deployment nimble-opti-adapter-controller-manager -n nimble-opti-adapter-system -p '{"spec":{"template":{"spec":{"containers":[{"name":"kube-rbac-proxy","imagePullPolicy":"Always"},{"name":"manager","imagePullPolicy":"Always"}]}}}}'

echo "Setting Minikube context..."
kubectl config use-context minikube

echo "Adding helm repo..."
helm repo add jetstack https://charts.jetstack.io

echo "Updating helm repo..."
helm repo update

echo "Installing cert-manager..."
helm install \
    cert-manager jetstack/cert-manager \
    --namespace cert-manager \
    --create-namespace \
    --version $CERT_MANAGER_VERSION \
    --set installCRDs=true \
    --set defaultIssuerName=letsencrypt-prod \
    --set defaultIssuerKind=ClusterIssuer \
    --wait

echo "Applying letsencrypt cluster issuer..."
cat <<EOF | kubectl apply -f -
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

echo "Enabling Minikube ingress..."
minikube addons enable ingress

# echo "Adding https ingress..."
# helm install argocd argo/argo-cd \
#     --namespace argocd \
#     --create-namespace \
#     --set controller.replicas=1 \
#     --set server.config.url=https://argo.localhost.nip.io/ \
#     --set server.ingress.enabled=true \
#     --set server.ingress.annotations.acme\\.cert-manager\\.io/http01-edit-in-place=true \
#     --set server.ingress.annotations.cert-manager\\.io/cluster-issuer=letsencrypt-prod \
#     --set server.ingress.annotations.kubernetes\\.io/tls-acme=true \
#     --set server.ingress.annotations.nginx\\.ingress\\.kubernetes\\.io/backend-protocol=HTTPS \
#     --set server.ingress.annotations.nginx\\.ingress\\.kubernetes\\.io/ssl-passthrough=true \
#     --set server.ingress.ingressClassName=nginx \
#     --set server.ingress.https=true \
#     --wait

echo "Starting Minikube dashboard..."
minikube dashboard --url &

# Sleep for 1 second to allow the dashboard to start
sleep $SLEEP_TIME

echo "Setup complete."
