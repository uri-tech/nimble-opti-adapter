# Nimble Opti Adapter Helm Chart

This Helm chart deploys the nimble-opti-adapter, a Kubernetes operator that automates certificate renewal management when using ingress with the annotation `cert-manager.io/cluster-issuer` for services requiring TLS communication. It is designed to work seamlessly with the NGINX ingress controller, efficiently handling the `nginx.ingress.kubernetes.io/backend-protocol: HTTPS` annotation.

## Prerequisites

- Kubernetes cluster (v1.16+)
- Helm (v3+)

## Installation

To install the chart with the release name `nimble-opti-adapter`:

```bash
git clone https://github.com/uri-tech/nimble-opti-adapter.git
cd ./nimble-opti-adapter
helm install nimble-opti-adapter ./helm/nimble-opti-adapter
```

## Configuration

The following table lists the configurable parameters of the nimble-opti-adapter chart and their default values.

| Parameter                     | Description                                                | Default                               |
| ----------------------------- | ---------------------------------------------------------- | ------------------------------------- |
| `labelSelector`               | Label selector for namespaces the operator will manage     | `nimble.opti.adapter/enabled: 'true'` |
| `certificateRenewalThreshold` | Waiting time (in days) before certificate expires to renew | `30`                                  |
| `annotationRemovalDelay`      | Delay (in seconds) after removing HTTPS annotation         | `60`                                  |

To customize these parameters, edit the `values.yaml` file in the `helm/nimble-opti-adapter` directory.

## Usage

1. Label the namespaces where the operator should manage certificates:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: your-target-namespace
  labels:
    nimble.opti.adapter/enabled: "true"
```

2. Create a nimble-opti-adapterConfig custom resource in any namespace:

```yaml
apiVersion: nimble-opti-adapter.example.com/v1alpha1
kind: NimbleOptiConfig
metadata:
  name: example-config
spec:
  certificateRenewalThreshold: 30
  annotationRemovalDelay: 10
  
```

## Upgrading

To upgrade the nimble-opti-adapter Helm chart:

```bash
helm upgrade nimble-opti-adapter ./helm/nimble-opti-adapter
```

## Uninstalling

To uninstall the nimble-opti-adapter Helm chart:

```bash
helm uninstall nimble-opti-adapter
```
