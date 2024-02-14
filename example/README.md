## Install and Test nimble-opti-adapter Operator Locally Using Minikube

### Prerequisites

- Install [Minikube](https://minikube.sigs.k8s.io/docs/start/)
- Install [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- Install [Helm](https://helm.sh/docs/intro/install/)

### Step 1: Create cluster with Minikube

Start Minikube as <b>non-root</b> with the following command:

```bash
minikube start
```

List the available contexts using the following command:

```bash
kubectl config get-contexts
```

Change the context to work with Minikube command:

```bash
kubectl config use-context minikube
```

Create a shortcut and check if the cluster was created successfully:

```bash
alias k="minikube kubectl --"
k get all -A
```

Enable dashboard:

```bash
minikube dashboard &
```

### Step 2: Install Cert-Manager operator using Helm

cert-manager provides Helm charts as a first-class method of installation on both Kubernetes and OpenShift.

Be sure never to embed cert-manager as a sub-chart of other Helm charts; cert-manager manages non-namespaced resources in your cluster and care must be taken to ensure that it is installed exactly once.

<i><b>Add the Helm repository:</i></b>
This repository is the only supported source of cert-manager charts. There are some other mirrors and copies across the internet, but those are entirely unofficial and could present a security risk.

Notably, the "Helm stable repository" version of cert-manager is deprecated and should not be used.

```bash
helm repo add jetstack https://charts.jetstack.io
```

<i><b>Update your local Helm chart repository cache:</i></b>

```bash
helm repo update
```

<i><b> Install cert-manager with CustomResourceDefinitions:</i></b>

cert-manager requires a number of CRD resources, which can be installed manually using kubectl, or using the installCRDs option when installing the Helm chart.

To automatically install and manage the CRDs as part of your Helm release, you must add the --set installCRDs=true flag to your Helm installation command.

Uncomment the relevant line in the next steps to enable this.

Note that if you're using a helm version based on Kubernetes v1.18 or below (Helm v3.2), installCRDs will not work with cert-manager v0.16. See the v0.16 upgrade notes for more details.

```bash
helm install \
  cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --create-namespace \
  --version v1.11.0 \
  --set installCRDs=true
```

### Step 3: Install Ingress NGINX Controller

The ingress controller can be installed through minikube's addons system:

```bash
minikube addons enable ingress
```

When you are not using minikube

```bash
helm upgrade --install ingress-nginx ingress-nginx \
  --repo https://kubernetes.github.io/ingress-nginx \
  --namespace ingress-nginx --create-namespace
# or using YAML manifest
k apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.8.2/deploy/static/provider/cloud/deploy.yaml
```

## Step 4: Create letsencrypt cluster issuer

```bash
k apply -f - <<EOF
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: clusterissuer-letsencrypt-http01
spec:
  acme:
    email: <YOUR EMAIL> # replace with your email
    privateKeySecretRef:
      name: acme-letsencrypt-prod
    server: https://acme-v02.api.letsencrypt.org/directory
    solvers:
    - http01:
        ingress:
          class: nginx
EOF
```

## Step 5: Clone the nimble-opti-adapter repository

Clone the nimble-opti-adapter repository to your local machine:

```bash
# git clone -b uri-tech/diagrams-example https://github.com/uri-tech/nimble-opti-adapter.git
git clone -b main https://github.com/uri-tech/nimble-opti-adapter.git
cd nimble-opti-adapter
```

## Step 6: Render the templates and install the operator

### Using makefile with Kustomize (recomended)

```bash
make manifests # Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
make install # Install CRDs into the K8s cluster specified in ~/.kube/config.
make deploy IMG=nimbleopti/nimble-opti-adapter:latest # Deploy controller to the K8s cluster specified in ~/.kube/config.
```

### Using Helm

Create an output directory for saving the rendered templates:

```bash
mkdir output
```

Run the helm template command to render the templates and save them in the output directory:

```bash
helm template nimble-opti-adapter ./helm/nimble-opti-adapter --output-dir ./output
```

Inspect the generated files in the output directory:

```bash
tree -hapugD --dirsfirst --charset=utf-8 ./output
nano ./output/nimble-opti-adapter/templates/deployment.yaml
```

Install the nimble-opti-adapter operator using Helm:

```bash
helm install nimble-opti-adapter ./helm/nimble-opti-adapter --create-namespace --namespace nimble-opti-adapter
```

Inspect the container using kubectl:

```bash
k -n nimble-opti-adapter describe deployment nimble-opti-adapter
k -n nimble-opti-adapter describe pod <pod_name>
k -n nimble-opti-adapter logs <pod_name> <container_name>
```

## Step 7: Label the namespace

Label the default namespace so that the operator will manage certificates in it:

```bash
k label namespace default nimble.opti.adapter/enabled=true
```

## Step 8: Create a nimble-opti-adapterConfig custom resource

Create a `nimble-opti-adapter.yaml` file with the following content:

```yml
apiVersion: nimble-opti-adapter.example.com/v1alpha1
kind: NimbleOptiConfig
metadata:
  name: example-config
spec:
  certificateRenewalThreshold: 30
  annotationRemovalDelay: 10
```

Apply the configuration:

```bash
k apply -f nimble-opti-adapter.yaml
```

## Step 9: Test the operator

To test the operator, you can create an example ingress resource that requires TLS communication. Save the following content in a file named `example-ingress.yaml`:

```yml
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
      secretName: tls-letsencrypt-example # create secret in the same namespace as the ingress that contain the tls.crt and tls.key of the domains
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
```

Apply the ingress resource:

```bash
k apply -f example-ingress.yaml
```

Apply the ingress resource:

```bash
k logs -f -l app.kubernetes.io/name=nimble-opti-adapter
```

## Step 10: Cleanup

Once you've finished testing, you can delete the resources and stop Minikube:

```bash
k delete -f example-ingress.yaml
k delete -f nimble-opti-adapter.yaml
helm uninstall nimble-opti-adapter --namespace nimble-opti-adapter
minikube stop
```
