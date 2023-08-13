# 🚀 Ingress Annotation Modifier CronJob 🚀

Welcome to the Ingress Annotation Modifier CronJob! This tool is designed to scan and modify annotations of `Ingress` resources across all Kubernetes namespaces daily. Its primary focus is on the annotation `"nginx.ingress.kubernetes.io/backend-protocol: HTTPS"`.

## 📂 Contents

1. [Function Descriptions](#function-descriptions)
2. [Deployment](#deployment)
3. [Monitoring & Logging](#monitoring-&-logging)
4. [Best Practices & Tips](#best-practices-&-tips)
5. [Configuration Insights](#configuration-insights)


## 🛠 Function Descriptions 🛠

### `NewIngressWatcher`
This function is the constructor for the `IngressWatcher` type. It's like the birth of our watcher! 🎉 When invoked, it sets up the necessary configurations, schemes, and clients to interact with the Kubernetes API. It's the starting point, ensuring our watcher is ready to monitor and act upon Ingress resources.

### `AuditIngressResources`
This function is the heart of our watcher. 💓 It's like a diligent detective, scanning through all Ingress resources in the cluster. For each Ingress:

1. If the Ingress is labeled with `"nimble.opti.adapter/enabled:true"` and contains the ACME challenge, it initiates the certificate renewal process.
2. If not, it calculates the time remaining for the certificate's renewal. If the certificate is nearing its expiration:
   - For users with admin permissions (`ADMIN_USER_PERMISSION: "true"`), it deletes the associated Ingress secret.
   - For other users (`ADMIN_USER_PERMISSION: "false"`), it changes the secret's name to prompt the cert-manager to create a new certificate.
   
After all the checks and actions, it logs the number of Ingress resources it audited.

### `startCertificateRenewalAudit`
This function is the certificate's guardian. 🛡️ When an Ingress contains the `.well-known/acme-challenge`, this function steps in to renew the certificate. It:
1. Removes the HTTPS annotation.
2. Waits for the ACME challenge path to disappear or for a timeout.
3. Once confirmed, it reinstates the HTTPS annotation.

The function ensures that the certificate is renewed and up-to-date, keeping the traffic secure.

### `changeIngressSecretName`
Think of this function as a name-changer. 🔄 When the certificate is about to expire, and the user doesn't have admin permissions (`ADMIN_USER_PERMISSION: "false"`), this function alters the secret's name in `ing.Spec.TLS`. By doing so, it prompts the cert-manager to create a new certificate. It checks if the name has a version suffix (like `-v1`). If not, it adds one. If it does, it increments it. It's a clever trick to get a fresh certificate without deleting the old one!

### `deleteIngressSecret`
This function is like a cleaner. 🧹 When the certificate needs renewal and the user has admin permissions (`ADMIN_USER_PERMISSION: "true"`), this function deletes the associated Ingress secret. It ensures that old, soon-to-expire certificates are removed, making way for new ones.


## 🚀 Deployment 🚀

### 🔄 Installing/Updating the CronJob Only
###  Full Installation (including a Minikube cluster)
The `cronjob-create.sh` script provides a comprehensive setup. From initializing a Minikube cluster to applying the Kubernetes configurations, it's got you covered!

🔧 **What it does**:
- Initializes a Minikube cluster.
- Sets up Helm for cert-manager.
- Enables Minikube ingress.
- Installs cert-manager.
- Configures LetsEncrypt as a cluster issuer.
- Builds and pushes the Docker image.
- Applies the Kubernetes configurations.

🏃 **To deploy, simply run**:
```
./cronjob-create.sh
```

### 🔄 Installing/Updating the CronJob Only
The `cronjob-update.sh` script is your go-to for updating the CronJob. It's efficient, removing the old configuration and applying the new one after rebuilding the Docker image.

🔧 **What it does**:
- Removes the old configuration.
- Rebuilds and pushes the Docker image.
- Applies the updated Kubernetes configurations.

🏃 **To update, execute**:
```
./cronjob-update.sh
```

### 🛠️ Script Options:
Both scripts support the `-e` option to set environment variables. Here's how you can use it:

- **Setting the Image Tag**:
```
./cronjob-create.sh -e IMAGE_TAG=v2.0.0
./cronjob-update.sh -e IMAGE_TAG=v2.0.0
```

- **Setting the Cert Manager Version**:
```
./cronjob-create.sh -e CERT_MANAGER_VERSION=v1.12.0
```

- **Other options**:
  - `DOCKER_USERNAME`: Set the Docker username.
  - `DOCKER_IMAGE_NAME`: Specify the Docker image name.
  - `BUILD_PLATFORM`: Choose the build platform (`local` or `all`).
  - `ADMIN_CONFIG`: Set to `true` or `false` to control admin configurations.

For example, to set the Docker username while creating:
```
./cronjob-create.sh -e DOCKER_USERNAME=myusername
```

Remember, you can combine multiple environment variables by separating them with spaces:
```
./cronjob-create.sh -e IMAGE_TAG=v2.0.0 -e DOCKER_USERNAME=myusername -e CERT_MANAGER_VERSION=v1.12.0
```


## 📊 Monitoring & Logging 📊

The container provides detailed logs of its operations. You can adjust the verbosity of the logs by setting the `RUN_MODE`:

- `"dev"`: Provides a comprehensive breakdown of what's happening under the hood.
- `"prod"`: Standard logs suitable for production environments.

## 🎓 Best Practices & Tips 🎓

1. **Namespacing**: All resources are neatly organized under the `ingress-modify-ns` namespace. This ensures a tidy separation from other workloads in your cluster.
2. **Private Docker Registries**: If you're using one, remember to add image pull secrets to the service account.
3. **Resource Management**: The CronJob is configured with resource requests and limits, ensuring it gets the resources it needs without hogging cluster resources.
4. **Stay Informed**: Regularly check the logs for insights and potential issues.
5. **Graceful Error Handling**: The Go code is designed to handle errors gracefully and retries operations when necessary.

## 🌟 Configuration Insights 🌟

The `configmap.yaml` file is your go-to for tweaking the CronJob's behavior. Here are some key configurations you can adjust:

- 🔄 `RUN_MODE`: Adjust the verbosity of logs. Options include `"dev"` for detailed logs and `"prod"` for standard logs.
- 📝 `LOG_OUTPUT`: Choose between `"console"` for human-readable logs or `"json"` for structured logging.
- ⏳ `CERTIFICATE_RENEWAL_THRESHOLD`: Defines the number of days before a certificate's expiration to initiate renewal.
- ⌛ `ANNOTATION_REMOVAL_DELAY`: The delay (in seconds) to wait after removing an annotation.