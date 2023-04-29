## Install and Test NimbleOpticAdapter Operator Locally Using Minikube

### Prerequisites

- Install [Minikube](https://minikube.sigs.k8s.io/docs/start/)
- Install [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- Install [Helm](https://helm.sh/docs/intro/install/)

### Step 1: Start Minikube

Start Minikube with the following command:

```bash
minikube start
```

## Step 2: Clone the NimbleOpticAdapter repository

Clone the NimbleOpticAdapter repository to your local machine:
```bash
git clone https://github.com/uri-tech/NimbleOpticAdapter.git
cd NimbleOpticAdapter
```

## Step 3: Install the operator using Helm

Install the NimbleOpticAdapter operator using Helm:
```bash
helm install nimbleopticadapter ./helm/nimbleopticadapterconfig
```

## Step 4: Label the namespace

Label the default namespace so that the operator will manage certificates in it:
```bash
kubectl label namespace default nimble.optic.adapter/enabled=true
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
kubectl apply -f nimbleopticadapterconfig.yaml
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
kubectl apply -f example-ingress.yaml
```

Apply the ingress resource:
```bash
kubectl logs -f -l app.kubernetes.io/name=nimbleopticadapter
```

## Step 7: Cleanup

Once you've finished testing, you can delete the resources and stop Minikube:
```bash
kubectl delete -f example-ingress.yaml
kubectl delete -f nimbleopticadapterconfig.yaml
helm uninstall nimbleopticadapter
minikube stop
```
