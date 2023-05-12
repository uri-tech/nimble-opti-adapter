# nimble-opti-adapter

<p><i>The Nimble-Opti-Adapter is a dedicated Kubernetes operator engineered to handle specific use cases. It's targeted towards Kubernetes clusters that have already integrated the Cert-Manager operator and Nginx-Ingress controller, and use Let's Encrypt as their certificate authority for acquiring SSL certificates validated through the `acme.cert-manager.io/http01-edit-in-place: true` annotation in the ingress. Moreover, the ingress requires the `nginx.ingress.kubernetes.io/backend-protocol: HTTPS` annotation to ensure service accessibility. The operator proficiently addresses the issue of auto-renewal of certificates in this scenario, given that the HTTP01 Ingress resolver doesn't align with this setup.
</i></p>

<p align="center">
  <!-- <img src="diagrams/main.png" alt="nimble-opti-adapter diagrams" width="300" height="300"> -->
  <img src="diagrams/main.png" alt="nimble-opti-adapter diagrams">
</p>

<p align="center">
  <!-- <a href="https://github.com/uri-tech/nimble-opti-adapter/actions">
    <img alt="Build Status" src="diagrams/main.png">
  </a> -->
  <a href="https://github.com/uri-tech/nimble-opti-adapter/blob/master/LICENSE">
    <img alt="License: Apache 2.0" src="https://img.shields.io/badge/License-Apache%202.0-blue.svg">
  </a>
  <a href="https://github.com/uri-tech/nimble-opti-adapter/releases">
    <img alt="GitHub release" src="https://img.shields.io/github/v/release/uri-tech/nimble-opti-adapter">
  </a>
  <a href="https://github.com/uri-tech/nimble-opti-adapter/issues">
    <img alt="GitHub issues" src="https://img.shields.io/github/issues/uri-tech/nimble-opti-adapter">
  </a>
</p>

nimble-opti-adapter is a Kubernetes operator that automates certificate renewal management when using ingress with the annotation `cert-manager.io/cluster-issuer` for services that require TLS communication. This operator is designed to work seamlessly with the NGINX ingress controller, efficiently handling the `nginx.ingress.kubernetes.io/backend-protocol: HTTPS` annotation.

## ⚙️ Operator Workflow

The operator monitors the creation and modification of both CustomResourceDefinitions (CRDs) of kind `NimbleOptiAdapter` and Ingress resources. The following is a detailed overview of the operator's behavior:

1. If a CRD of kind `nimble-opti-adapter` is created or modified, the operator currently performs no action.

2. When an Ingress resource is created or modified, the operator's controller checks for the presence of the `nimble.opti.adapter/enabled: "true"` label:

   - If the label does not exist, the operator takes no action.
   - If the label exists, the operator checks for the presence of a `NimbleOptiAdapter` CRD in the same namespace:
     - If the CRD does not exist, the operator creates a new `NimbleOptiAdapter` CRD with default values and proceeds as if the CRD already existed.
     - If the CRD exists, the operator checks if any path in `spec.rules[].http.paths[].path` contains `.well-known/acme-challenge`. If so, the operator initiates the certificate renewal process for the Ingress resource.

3. Once a day, the operator checks all Ingress resources with the label `nimble.opti.adapter/enabled: "true"` and associated `NimbleOptiAdapter` CRD resource in the same namespace:

   - If there is no match, the operator takes no action.
   - If there is a match, the operator checks if the Secret name specified in `spec.tls[].secretName` exists for each tls[], retrieves the Secret, and calculates the time remaining until the certificate expires:
     - If the certificate expires in equal or fewer days than the `CertificateRenewalThreshold` specified in the `NimbleOptiAdapter` resource in the same namespace, the operator initiates the certificate renewal process.
     - If the certificate expires in more days than the `CertificateRenewalThreshold` specified in the `NimbleOptiAdapter` resource in the same namespace, the operator checks if any path in `spec.rules[].http.paths[].path` contains `.well-known/acme-challenge`:
       - If there is no match, the operator takes no action.
       - If there is a match, the operator initiates the certificate renewal process.

4. Certificate renewal process:
   - Remove the `nginx.ingress.kubernetes.io/backend-protocol: HTTPS` annotation from the Ingress resource.
   - Start a timer and wait until either there is no `spec.rules[].http.paths[].path` containing `.well-known/acme-challenge`, or the `AnnotationRemovalDelay` time specified in the `NimbleOptiAdapter` CRD resource has passed.
   - Send to prometheus endpoint duration (in seconds) of annotation updates during renewal through `nimble-opti-adapter_annotation_updates_duration_seconds`
   - Re-add the annotation `nginx.ingress.kubernetes.io/backend-protocol: HTTPS` to the Ingress resource.
   - Increment `nimble-opti-adapter_certificate_renewals_total` and send to prometheus endpoint.

<!-- ![nimble-opti-adapter Diagram](diagram.png) -->

## 🌟 Features

- 🔄 Automatic certificate renewal based on certificate validity and user-defined waiting times
- 🏷️ Supports multi-namespace operation with a configurable label selector
- 📊 Prometheus metrics collection for certificate renewals and annotation updates
- 🚀 Easy installation using Helm
- 🔌 Extensible architecture for future enhancements

## ⏳ Future Enhancements

- 🔔 Customizable alerting and notification system for certificate renewals and errors
- 🔗 Integration with external certificate issuers or other certificate management systems
- 📈 Enhanced Prometheus metrics for deeper insights into certificate management
- 🚦 Support for other ingress controllers besides NGINX
- 📝 Automatic handling of additional ingress annotations as needed

## 📚 Prerequisites

- Kubernetes cluster (v1.16+)
- [Helm (v3+)](https://helm.sh/docs/intro/install)
- [Cert-Manager operator](https://github.com/cert-manager/cert-manager)
- [Ingress NGINX Controller](https://github.com/kubernetes/ingress-nginx)

## 🚀 Quick Start

This configuration assumes you are working from a Linux Shell or macOS.

### Step 1: Clone the repository

```bash
git clone https://github.com/uri-tech/nimble-opti-adapter.git
cd nimble-opti-adapter
```

### Step 2: Install the operator using Helm

```bash
helm install nimble-opti-adapter ./helm/nimble-opti-adapterconfig
```

### Step 3: Modify the operator

To modify the operator, edit the Helm chart templates or values.yaml file in the helm/nimble-opti-adapterconfig directory.

### Step 4: Update the operator using Helm

Repackage the Helm chart and upgrade the release with the following commands:

```bash
cd nimble-opti-adapter/helm/
helm package nimble-opti-adapterconfig
helm upgrade nimble-opti-adapter ./nimble-opti-adapterconfig-0.1.0.tgz
```

## ⚙️ Configuration

Edit the `values.yaml` file in the `helm/nimble-opti-adapterconfig` directory to customize the following parameters:

- `labelSelector`: The label selector for namespaces the operator will manage certificates in (default: `nimble.opti.adapter/enabled: 'true'`)
- `certificateRenewalThreshold`: The waiting time (in days) before the certificate expires to trigger renewal
- `annotationRemovalDelay`: The delay (in seconds) after removing the `nginx.ingress.kubernetes.io/backend-protocol: HTTPS` annotation before re-adding it

## 📝 Usage

Label the Ingress where the operator should manage certificates:

```yaml
apiVersion: v1
kind: Ingress
metadata:
  name: your-target-ingress
  labels:
    nimble.opti.adapter/enabled: "true"
```

Create a nimble-opti-adapter custom resource in any namespace:

```yaml
apiVersion: nimble-opti-adapter.example.com/v1alpha1
kind: nimble-opti-adapter
metadata:
  name: example-config
spec:
  certificateRenewalThreshold: 30
  annotationRemovalDelay: 60
```

## 📊 Metrics

nimble-opti-adapter exposes the following Prometheus metrics:

- `nimble-opti-adapter_certificate_renewals_total`: Total number of certificate renewals
- `nimble-opti-adapter_annotation_updates_duration_seconds`: Duration (in seconds) of annotation updates during each renewal

## 🤝 Contributing

We welcome contributions to the nimble-opti-adapter project! Please see the [CONTRIBUTING.md](CONTRIBUTING.md) file for more information on how to contribute.

## 📜 License

nimble-opti-adapter is licensed under the [Apache License, Version 2.0](LICENSE).

## 📞 Support

For any questions, bug reports, or feature requests, please open an issue on our [GitHub repository](https://github.com/uri-tech/nimble-opti-adapter/issues).

<!-- ## Attribution

### Images

Diagram: [Unsplash](https://unsplash.com/photos/U9s5m5L2Gn0) (License: CC0) -->

<!-- git pull --allow-unrelated-histories https://github.com/uri-tech/nimble-opti-adapter main -->

<!-- kubebuilder init --domain nimble-opti-adapter.tech-ua.com --repo github.com/uri-tech/nimble-opti-adapter -->
