## Install and Test NimbleOpticAdapter Operator Locally Using Minikube

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
minikube dashboard
```

## Step 2: Clone the NimbleOpticAdapter repository

Clone the NimbleOpticAdapter repository to your local machine:

```bash
git clone -b main https://github.com/uri-tech/NimbleOpticAdapter.git
cd NimbleOpticAdapter
```

## Step 3: Render the templates and install the operator using Helm

Create an output directory for saving the rendered templates:

```bash
mkdir output
```

Run the helm template command to render the templates and save them in the output directory:

```bash
helm template nimbleopticadapter ./helm/nimbleopticadapterconfig --output-dir ./output
```

Inspect the generated files in the output directory:

```bash
ls -l ./output/nimbleopticadapter/templates
```

Install the NimbleOpticAdapter operator using Helm:

```bash
helm install nimbleopticadapter ./helm/nimbleopticadapterconfig
```

## Step 4: Label the namespace

Label the default namespace so that the operator will manage certificates in it:

```bash
k label namespace default nimble.optic.adapter/enabled=true
```

## Step 5: Create a NimbleOpticAdapterConfig custom resource

Create a `nimbleopticadapterconfig.yaml` file with the following content:

```ymal
apiVersion: nimbleopticadapter.example.com/v1alpha1
kind: NimbleOpticAdapterConfig
metadata:
  name: example-config
spec:
  certificateRenewalThreshold: 30
  annotationRemovalDelay: 60
```

Apply the configuration:

```bash
k apply -f nimbleopticadapterconfig.yaml
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
k logs -f -l app.kubernetes.io/name=nimbleopticadapter
```

## Step 7: Cleanup

Once you've finished testing, you can delete the resources and stop Minikube:

```bash
k delete -f example-ingress.yaml
k delete -f nimbleopticadapterconfig.yaml
helm uninstall nimbleopticadapter
minikube stop
```
