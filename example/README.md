## Install and Test nimble-opti-adapter Operator Locally Using Minikube

### Prerequisites

- Install [Minikube](https://minikube.sigs.k8s.io/docs/start/)
- Install [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- Install [Helm](https://helm.sh/docs/intro/install/)

### Step 1: Start Minikube

Start Minikube as non-root with the following command:

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

## Step 2: Clone the nimble-opti-adapter repository

Clone the nimble-opti-adapter repository to your local machine:

```bash
# git clone -b uri-tech/diagrams-example https://github.com/uri-tech/nimble-opti-adapter.git
git clone -b main https://github.com/uri-tech/nimble-opti-adapter.git
cd nimble-opti-adapter
```

## Step 3: Render the templates and install the operator using Helm

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
k -n default describe deployment nimble-opti-adapter-nimble-opti-adapter
k -n default describe pod <pod_name>
k -n default logs <pod_name> <container_name>
```

## Step 4: Label the namespace

Label the default namespace so that the operator will manage certificates in it:

```bash
k label namespace default nimble.opti.adapter/enabled=true
```

## Step 5: Create a nimble-opti-adapterConfig custom resource

Create a `nimble-opti-adapter.yaml` file with the following content:

```ymal
apiVersion: nimble-opti-adapter.example.com/v1alpha1
kind: nimble-opti-adapterConfig
metadata:
  name: example-config
spec:
  certificateRenewalThreshold: 30
  annotationRemovalDelay: 60
```

Apply the configuration:

```bash
k apply -f nimble-opti-adapter.yaml
```

## Step 6: Test the operator

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

## Step 7: Cleanup

Once you've finished testing, you can delete the resources and stop Minikube:

```bash
k delete -f example-ingress.yaml
k delete -f nimble-opti-adapter.yaml
helm uninstall nimble-opti-adapter --namespace nimble-opti-adapter
minikube stop
```
