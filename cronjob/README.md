# 🚀 Ingress Annotation Modifier CronJob 🚀

This Kubernetes CronJob is designed to scan and modify annotations of `Ingress` resources across all namespaces once a day. Specifically, it targets the annotation `"nginx.ingress.kubernetes.io/backend-protocol: HTTPS"`.

🔍 **Primary Functionality**:
- The job fetches all `Ingress` resources.
- It checks if an `Ingress` resource has a specific ACME challenge.
- If found, it starts the certificate renewal process, which involves:
  - Removing the HTTPS annotation.
  - Waiting for the absence of the ACME challenge path.
  - Re-adding the HTTPS annotation.

## 📂 Project Structure 📂

```
ingress-certificate-renew/
├── Dockerfile                               # Dockerfile to containerize the application
├── README.md                                # Project documentation and overview
├── cmd
│   └── ingress-annotation-modifier          # Main application directory
│       └── main.go                          # Entry point for the application, initiates the process
├── config
│   └── cronjob.yaml                         # Kubernetes CronJob configuration to run the application periodically
└── internal                                 # Internal packages; not meant for external use
    ├── ingresswatcher                       # Code related to the IngressWatcher functionality
    │   ├── annotations.go                   # Functions to handle adding and removing of Ingress annotations
    │   ├── challenge.go                     # Functions related to handling and verifying ACME challenges
    │   ├── ingresswatcher.go                # Core logic for the IngressWatcher
    │   └── kubernetesclient.go              # Kubernetes client interactions and related utility functions
    └── utils                                # Utility functions and shared code
        ├── namedmutex.go                    # Named mutex utility for handling concurrent locks by key
        └── utils.go                         # Miscellaneous utility functions (consider splitting as it grows)

```


## 📂 Contents

1. [Function Descriptions](#function-descriptions)
2. [Deployment](#deployment)
3. [Monitoring & Logging](#monitoring-&-logging)
4. [Best Practices & Tips](#best-practices-&-tips)

## 🛠 Function Descriptions 🛠

### `auditIngressResources`

This is the main function that gets triggered by the CronJob. It fetches all the `Ingress` resources and checks each one to determine if it contains an ACME challenge. If the challenge is present, it starts the certificate renewal process.

### `startCertificateRenewalAudit`

Starts the renewal process for an `Ingress`. The steps involve:
- Removing the HTTPS annotation.
- Waiting for the absence of the ACME challenge path.
- Re-adding the HTTPS annotation.

### `removeHTTPSAnnotation`

Removes the `"nginx.ingress.kubernetes.io/backend-protocol: HTTPS"` annotation from an `Ingress`.

### `addHTTPSAnnotation`

Adds the `"nginx.ingress.kubernetes.io/backend-protocol: HTTPS"` annotation to an `Ingress`.

### `waitForChallengeAbsence`

Waits for the absence of the ACME challenge path in an `Ingress` or until a timeout is reached.

## 🚀 Deployment 🚀

To deploy the CronJob, apply the provided Kubernetes YAML file:

```
kubectl apply -f cronjob.yaml
```

## 📊 Monitoring & Logging 📊

Ensure your application logs meaningful information to STDOUT/STDERR. This will be captured by Kubernetes logging solutions like Fluentd or Loki.

## 🎓 Best Practices & Tips 🎓

1. **Namespacing**: The CronJob and related resources are in the `ingress-modify-ns` namespace. This logically separates our resources, especially if you have multiple applications/workloads in the cluster.
2. **Image Pull Secrets**: If your Docker image is in a private registry, remember to add image pull secrets to the service account.
3. **Resource Limits**: Set resource requests and limits for your container. This ensures that the container has the necessary resources and protects other workloads running in the same namespace or cluster.
4. **Logging and Monitoring**: Monitor the logs regularly for any errors or issues.
5. **Error Handling**: Ensure your Go code gracefully handles errors and retries where necessary, especially given the periodic nature of the CronJob.

---

❤️ Happy Kubernetes-ing! ❤️
