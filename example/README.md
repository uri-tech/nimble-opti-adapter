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

<i><b>Install Install cert-manager with CustomResourceDefinitions:</i></b>

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

## Step 4: Clone the nimble-opti-adapter repository

Clone the nimble-opti-adapter repository to your local machine:

```bash
# git clone -b uri-tech/diagrams-example https://github.com/uri-tech/nimble-opti-adapter.git
git clone -b main https://github.com/uri-tech/nimble-opti-adapter.git
cd nimble-opti-adapter
```

## Step 5: Render the templates and install the operator using Helm

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

## Step 6: Label the namespace

Label the default namespace so that the operator will manage certificates in it:

```bash
k label namespace default nimble.opti.adapter/enabled=true
```

## Step 7: Create a nimble-opti-adapterConfig custom resource

Create a `nimble-opti-adapter.yaml` file with the following content:

```ymal
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

## Step 8: Test the operator

To test the operator, you can create an example ingress resource that requires TLS communication. Save the following content in a file named `example-ingress.yaml`:

```ymal
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: example-ingress
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-staging"
    nginx.ingress.kubernetes.io/backend-protocol: HTTPS
spec:
  tls:
  - hosts:
    - example.com
    secretName: example-tls
  rules:
  - host: example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: example-service
            port:
              number: 80
```

Apply the ingress resource:

```bash
k apply -f example-ingress.yaml
```

Apply the ingress resource:

```bash
k logs -f -l app.kubernetes.io/name=nimble-opti-adapter
```

## Step 9: Cleanup

Once you've finished testing, you can delete the resources and stop Minikube:

```bash
k delete -f example-ingress.yaml
k delete -f nimble-opti-adapter.yaml
helm uninstall nimble-opti-adapter --namespace nimble-opti-adapter
minikube stop
```
