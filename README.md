# NimbleOpticAdapter

<!-- <p align="center">
  <img src="https://example.com/icon.png" alt="NimbleOpticAdapter Icon" width="80" height="80">
</p> -->

NimbleOpticAdapter is a Kubernetes operator that automates certificate renewal management when using ingress with the annotation `cert-manager.io/cluster-issuer` for services that require TLS communication. This operator is designed to work seamlessly with the NGINX ingress controller, efficiently handling the "nginx.ingress.kubernetes.io/backend-protocol: HTTPS" annotation.

<!-- ![NimbleOpticAdapter Diagram](diagram.png) -->

## Features

- Automatic certificate renewal based on certificate validity and user-defined waiting times
- Supports multi-namespace operation with a configurable label selector
- Prometheus metrics collection for certificate renewals and annotation updates
- Easy installation using Helm
- Extensible architecture for future enhancements

## Future Enhancements

- Customizable alerting and notification system for certificate renewals and errors
- Integration with external certificate issuers or other certificate management systems
- Enhanced Prometheus metrics for deeper insights into certificate management
- Support for other ingress controllers besides NGINX
- Automatic handling of additional ingress annotations as needed

## Prerequisites

- Kubernetes cluster (v1.16+)
- Helm (v3+)
- Go (v1.13+)
- Kubebuilder (v2+)

## Quick Start

### Step 1: Clone the repository

```bash
git clone https://github.com/uri-tech/NimbleOpticAdapter.git
cd NimbleOpticAdapter
```

### Step 2: Install the operator using Helm

```bash
helm install nimbleopticadapter ./helm/nimbleopticadapterconfig
```

### Step 3: Modify the operator

To modify the operator, edit the Helm chart templates or values.yaml file in the helm/nimbleopticadapterconfig directory.

### Step 4: Update the operator using Helm

Repackage the Helm chart and upgrade the release with the following commands:

```bash
cd NimbleOpticAdapter/helm/
helm package nimbleopticadapterconfig
helm upgrade nimbleopticadapter ./nimbleopticadapterconfig-0.1.0.tgz
```

## Configuration

Edit the `values.yaml` file in the `helm/nimbleopticadapterconfig` directory to customize the following parameters:

- `labelSelector`: The label selector for namespaces the operator will manage certificates in (default: `nimble.optic.adapter/enabled: 'true'`)
- `certificateRenewalThreshold`: The waiting time (in days) before the certificate expires to trigger renewal
- `annotationRemovalDelay`: The delay (in seconds) after removing the "nginx.ingress.kubernetes.io/backend-protocol: HTTPS" annotation before re-adding it

## Usage

Label the namespaces where the operator should manage certificates:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: your-target-namespace
  labels:
    nimble.optic.adapter/enabled: "true"
```

Create a NimbleOpticAdapterConfig custom resource in any namespace:

```yaml
apiVersion: nimbleopticadapter.example.com/v1alpha1
kind: NimbleOpticAdapterConfig
metadata:
  name: example-config
spec:
  certificateRenewalThreshold: 30
  annotationRemovalDelay: 60
```

## Metrics

NimbleOpticAdapter exposes the following Prometheus metrics:

- `nimbleopticadapter_certificate_renewals_total`: Total number of certificate renewals
- `nimbleopticadapter_annotation_updates_duration_seconds`: Duration (in seconds) of annotation updates during each renewal

## Contributing

We welcome contributions to the NimbleOpticAdapter project! Please see the CONTRIBUTING.md file for more information on how to contribute.

## License

NimbleOpticAdapter is licensed under the Apache License, Version 2.0.

## Support

For any questions, bug reports, or feature requests, please open an issue on our [GitHub repository](https://github.com/uri-tech/NimbleOpticAdapter/issues)

<!-- ## Attribution

### Images

Diagram: [Unsplash](https://unsplash.com/photos/U9s5m5L2Gn0) (License: CC0) -->

<!-- git pull --allow-unrelated-histories https://github.com/uri-tech/NimbleOpticAdapter main -->

<!-- kubebuilder init --domain nimbleopticadapter.tech-ua.com --repo github.com/uri-tech/NimbleOpticAdapter -->
