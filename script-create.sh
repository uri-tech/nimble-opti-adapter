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
# minikube start
minikube start --extra-config=apiserver.enable-admission-plugins="MutatingAdmissionWebhook,ValidatingAdmissionWebhook"

echo "Setting Minikube context..."
kubectl config use-context minikube

echo "Adding helm repo..."
helm repo add jetstack https://charts.jetstack.io

echo "Updating helm repo..."
helm repo update

echo "Enabling Minikube ingress..."
minikube addons enable ingress
kubectl delete job -n ingress-nginx ingress-nginx-admission-create
kubectl delete job -n ingress-nginx ingress-nginx-admission-patch

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

# echo "Adding argocd for ingress check..."
# helm install argocd argo/argo-cd \
#   --namespace argocd \
#   --create-namespace \
#   --set controller.replicas=1 \
#   --set server.config.url=https://argo.localhost.nip.io/ \
#   --set server.ingress.enabled=true \
#   --set server.ingress.annotations.acme\\.cert-manager\\.io/http01-edit-in-place=true \
#   --set server.ingress.annotations.cert-manager\\.io/cluster-issuer=letsencrypt-prod \
#   --set server.ingress.annotations.kubernetes\\.io/tls-acme=true \
#   --set server.ingress.annotations.nginx\\.ingress\\.kubernetes\\.io/backend-protocol=HTTPS \
#   --set server.ingress.annotations.nginx\\.ingress\\.kubernetes\\.io/ssl-passthrough=true \
#   --set server.ingress.ingressClassName=nginx \
#   --set server.ingress.https=true \
#   --wait

# echo connecting metrics service...
# kubectl port-forward svc/nimble-opti-adapter-controller-manager-metrics-service -n nimble-opti-adapter-system 8443:8443

echo "Creating service for metrics..."
cat <<EOF | kubectl apply -f -
kind: Service
apiVersion: v1
metadata:
  name: metrics-svc
  namespace: nimble-opti-adapter-system
  labels:
  annotations:
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

# Sleep for 1 second to allow the dashboard to start
sleep $SLEEP_TIME

echo "Patching deployment..."
kubectl patch deployment nimble-opti-adapter-controller-manager -n nimble-opti-adapter-system -p '{"spec":{"template":{"spec":{"containers":[{"name":"kube-rbac-proxy","imagePullPolicy":"Always"},{"name":"manager","imagePullPolicy":"Always"}]}}}}'

echo "Starting Minikube dashboard..."
# kubectl create serviceaccount kubernetes-dashboard -n kubernetes-dashboard
# Delete the dashboard
minikube addons disable dashboard
minikube addons enable dashboard
minikube dashboard --url &

echo "provide you with the URL for the metrics-svc service..."
minikube service metrics-svc -n nimble-opti-adapter-system --url

echo "Setup complete."
