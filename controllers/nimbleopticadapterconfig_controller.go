// nimbleopticadapterconfig_controller.go

/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"time"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/client-go/tools/cache"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
	"github.com/uri-tech/NimbleOpticAdapter/api/v1alpha1"

	// configv1alpha1 "github.com/uri-tech/NimbleOpticAdapter/api/v1alpha1"
	corev1informers "k8s.io/client-go/informers/core/v1"
)

// NimbleOpticAdapterConfigReconciler reconciles a NimbleOpticAdapterConfig object
type NimbleOpticAdapterConfigReconciler struct {
	client.Client
	Log            logr.Logger
	Scheme         *runtime.Scheme
	SecretInformer corev1informers.SecretInformer
}

// type NimbleOpticAdapterConfigReconciler struct {
// 	client.Client
// 	Log               logr.Logger
// 	Scheme            *runtime.Scheme
// 	SecretInformer    cache.SharedIndexInformer
// 	IngressInformer   cache.SharedIndexInformer
// 	ConfigMapInformer cache.SharedIndexInformer
// }

//+kubebuilder:rbac:groups=config.nimbleopticadapter.tech-ua.com,resources=nimbleopticadapterconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=config.nimbleopticadapter.tech-ua.com,resources=nimbleopticadapterconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=config.nimbleopticadapter.tech-ua.com,resources=nimbleopticadapterconfigs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NimbleOpticAdapterConfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile

// Modify the SetupWithManager function to include watches on Ingress and Secret resources
func (r *NimbleOpticAdapterConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.NimbleOpticAdapterConfig{}).
		// Watch Ingress resources and enqueue reconcile requests
		Watches(&source.Kind{Type: &networkingv1.Ingress{}}, &handler.EnqueueRequestForObject{}).
		// Watch Secret resources and enqueue reconcile requests
		Watches(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForObject{}).
		Complete(r)
}

// Modify your Reconcile function to handle events for Ingress and Secret resources
func (r *NimbleOpticAdapterConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Fetch the NimbleOpticAdapterConfig instance
	nimbleOpticAdapterConfig := &v1alpha1.NimbleOpticAdapterConfig{}
	err := r.Client.Get(ctx, req.NamespacedName, nimbleOpticAdapterConfig)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	// Watch for Ingress events
	ingress := &networkingv1.Ingress{}

	err = r.Client.Get(ctx, req.NamespacedName, ingress)
	if err == nil {
		return r.handleIngressEvent(ctx, req, nimbleOpticAdapterConfig)
	}
	return ctrl.Result{}, nil
}

// handleIngressEvent is the event handler for the Ingress object. It fetches the Ingress object, checks if the Namespace
// has the required label, and checks if the Ingress has the required annotations. If the required label and annotations
// are present, it starts watching for the certificate expiration of each TLS in the Ingress.
// This function returns a ctrl.Result and an error.
func (r *NimbleOpticAdapterConfigReconciler) handleIngressEvent(ctx context.Context, req ctrl.Request, nimbleOpticAdapterConfig *v1alpha1.NimbleOpticAdapterConfig) (ctrl.Result, error) {
	// Fetch the Ingress instance
	ingress := &networkingv1.Ingress{}
	err := r.Client.Get(ctx, req.NamespacedName, ingress)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Check if the Namespace has the required label
	namespace := &corev1.Namespace{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: ingress.Namespace}, namespace)
	if err != nil {
		return ctrl.Result{}, err
	}

	enabled, ok := namespace.Labels["nimble.optic.adapter/enabled"]
	if !ok || enabled != "true" {
		return ctrl.Result{}, nil
	}

	// Check if Ingress has the required annotations
	hasClusterIssuerAnnotation := false
	hasBackendProtocolAnnotation := false

	for key, value := range ingress.Annotations {
		if key == "cert-manager.io/cluster-issuer" {
			hasClusterIssuerAnnotation = true
		}
		if key == "nginx.ingress.kubernetes.io/backend-protocol" && value == "HTTPS" {
			hasBackendProtocolAnnotation = true
		}
		if hasClusterIssuerAnnotation && hasBackendProtocolAnnotation {
			break
		}
	}

	if !hasClusterIssuerAnnotation || !hasBackendProtocolAnnotation {
		return ctrl.Result{}, nil
	}

	// Iterate through Ingress TLS and start watching for the Secrets
	for _, tls := range ingress.Spec.TLS {
		secretName := tls.SecretName

		// Start a goroutine to watch for the certificate expiration of each Secret
		go r.watchCertificateExpiration(ctx, ingress.Namespace, secretName, nimbleOpticAdapterConfig.Spec.CertificateRenewalThreshold, nimbleOpticAdapterConfig)
	}

	return ctrl.Result{}, nil
}

// watchCertificateExpiration watches for certificate expiration in the specified Secret
func (r *NimbleOpticAdapterConfigReconciler) watchCertificateExpiration(ctx context.Context, namespace, secretName string, renewalThreshold int, config *v1alpha1.NimbleOpticAdapterConfig) {
	// Create an informer for the Secret
	secretInformer := r.SecretInformer.Informer()
	secretInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			newSecret := newObj.(*corev1.Secret)
			if newSecret.Namespace == namespace && newSecret.Name == secretName {
				r.checkCertificateExpiration(ctx, newSecret, config, namespace, secretName)
			}
		},
	})

	// Start the informer
	secretInformer.Run(ctx.Done())
}

// checkCertificateExpiration checks if the certificate in the specified Secret has expired, and renews the certificates if necessary.
// It temporarily makes the services unavailable during the renewal process, updates the "nginx.ingress.kubernetes.io/backend-protocol: HTTPS" annotation to make the services available again,
// and sets the status and conditions of the NimbleOpticAdapterConfig custom resource.
//
// Parameters:
//
//	ctx: The context of the request.
//	secret: The Secret containing the certificate to check.
//	config: The NimbleOpticAdapterConfig custom resource containing the configurations.
func (r *NimbleOpticAdapterConfigReconciler) checkCertificateExpiration(ctx context.Context, secret *corev1.Secret, config *v1alpha1.NimbleOpticAdapterConfig, namespace string, secretName string) {
	// Extract the certificate bytes from the Secret
	certBytes, ok := secret.Data["tls.crt"]
	if !ok {
		return
	}

	// Parse the certificate
	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return
	}

	// Calculate the time remaining until the certificate expires
	timeRemaining := time.Until(cert.NotAfter)

	// Check if the remaining time is less than the threshold
	thresholdDuration := time.Duration(config.Spec.CertificateRenewalThreshold) * 24 * time.Hour
	if timeRemaining < thresholdDuration {
		// Renew the certificates if necessary - Temporarily make the services unavailable
		ingressPathsForRenewal := []string{} // Use the appropriate ingress paths for renewal
		err = r.removeIngressAnnotationsAndWait(ctx, ingressPathsForRenewal, config, namespace, secretName)
		if err != nil {
			// Log the error and return if failed to remove annotations and wait for certificate renewal
			log.Log.Error(err, "failed to remove annotations and wait for certificate renewal")
			return
		}

		// Update the "nginx.ingress.kubernetes.io/backend-protocol: HTTPS" annotation - make the services available again.
		err = r.restoreIngressAnnotations(ctx, ingressPathsForRenewal, config)
		if err != nil {
			// Log the error and return if failed to restore annotations
			log.Log.Error(err, "failed to restore annotations")
			return
		}

		// Set the status and conditions of the NimbleOpticAdapterConfig custom resource
		// Update the status of the custom resource to reflect the current state
		err = r.updateCustomResourceStatus(ctx, config, ingressPathsForRenewal)
		if err != nil {
			// Log the error and return if failed to update custom resource status
			log.Log.Error(err, "failed to update custom resource status")
			return
		}
	}
}

// removeIngressAnnotationsAndWait removes the required annotations from the Ingress resources and waits for the certificate renewal
// before making the services available again. This function accepts a slice of Ingress paths to be updated and the NimbleOpticAdapterConfig custom resource.
// It iterates through the Ingress resources, removes the "nginx.ingress.kubernetes.io/backend-protocol: HTTPS" annotation, and updates the Ingress resources.
// After updating the Ingress resources, it waits for the certificate renewal or until the AnnotationRemovalDelay expires.
func (r *NimbleOpticAdapterConfigReconciler) removeIngressAnnotationsAndWait(
	ctx context.Context,
	ingressPathsForRenewal []string,
	config *v1alpha1.NimbleOpticAdapterConfig,
	namespace string,
	secretName string,
) error {
	// Iterate through the Ingress resources that need to be updated
	for _, ingressPath := range ingressPathsForRenewal {

		// Get the Ingress resource
		ingress := &networkingv1.Ingress{}
		err := r.Client.Get(ctx, types.NamespacedName{Name: ingressPath, Namespace: config.Spec.TargetNamespace}, ingress)
		if err != nil {
			return err
		}

		// Remove the "nginx.ingress.kubernetes.io/backend-protocol: HTTPS" annotation
		delete(ingress.Annotations, "nginx.ingress.kubernetes.io/backend-protocol")

		// Update the Ingress resource
		err = r.Client.Update(ctx, ingress)
		if err != nil {
			return err
		}
	}

	// Wait for the certificate renewal or until the AnnotationRemovalDelay expires
	// Get the current time
	currentTime := time.Now()

	// Calculate the expiration threshold based on the CertificateRenewalThreshold
	expirationThreshold := currentTime.Add(time.Hour * 24 * time.Duration(config.Spec.CertificateRenewalThreshold))

	// Calculate the end time based on the AnnotationRemovalDelay
	endTime := currentTime.Add(time.Minute * time.Duration(config.Spec.AnnotationRemovalDelay))

	for {
		// Check if the current time is after the end time
		if currentTime.After(endTime) {
			log.Log.Info("Annotation removal delay has passed, returning")
			break
		}

		// Check if there is a new certificate secret with the same name and
		// namespace that has more than CertificateRenewalThreshold time left until it expires
		durationUntilExpiration := time.Until(expirationThreshold)
		flagNotExpiring, err := isSecretNotExpiring(r.Client, namespace, secretName, durationUntilExpiration)
		if err != nil {
			return err
		}
		if flagNotExpiring {
			log.Log.Info("New certificate has been found and it is new, returning")
			break
		}

		currentTime = time.Now()
	}

	return nil
}

func (r *NimbleOpticAdapterConfigReconciler) restoreIngressAnnotations(ctx context.Context, ingressPathsForRenewal []string, config *v1alpha1.NimbleOpticAdapterConfig) error {
	// Iterate through the Ingress resources that need to be updated
	for _, ingressPath := range ingressPathsForRenewal {
		// Get the Ingress resource
		ingress := &networkingv1.Ingress{}
		err := r.Client.Get(ctx, types.NamespacedName{Name: ingressPath, Namespace: config.Spec.TargetNamespace}, ingress)
		if err != nil {
			return err
		}

		// Restore the annotation
		if ingress.Annotations == nil {
			ingress.Annotations = make(map[string]string)
		}
		ingress.Annotations["nginx.ingress.kubernetes.io/backend-protocol"] = "HTTPS"

		// Update the Ingress resource
		err = r.Client.Update(ctx, ingress)
		if err != nil {
			return err
		}
	}

	return nil
}

// updateCustomResourceStatus updates the status of the NimbleOpticAdapterConfig custom resource
// with the list of ingress paths that need certificate renewal and a condition representing the
// successful renewal of certificates.
func (r *NimbleOpticAdapterConfigReconciler) updateCustomResourceStatus(ctx context.Context, config *v1alpha1.NimbleOpticAdapterConfig, ingressPathsForRenewal []string) error {
	// Set the IngressPathsForRenewal field
	config.Status.IngressPathsForRenewal = ingressPathsForRenewal

	// Set the conditions for the custom resource
	// You can customize the conditions based on your requirements
	condition := metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             "CertificateRenewed",
		Message:            "Certificate renewed successfully",
		LastTransitionTime: metav1.Now(),
	}
	config.Status.Conditions = []metav1.Condition{condition}

	// Update the status of the custom resource
	err := r.Status().Update(ctx, config)
	if err != nil {
		return err
	}

	return nil
}

// Define a function to check if the secret exists and if it is expiring soon
func isSecretNotExpiring(coreClient client.Client, namespace, secretName string, threshold time.Duration) (bool, error) {
	secret := &corev1.Secret{}
	err := coreClient.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: secretName}, secret)
	if err != nil {
		log.Log.Error(err, "failed to get secret")
		return false, err
	}
	cert, err := tls.X509KeyPair(secret.Data["tls.crt"], secret.Data["tls.key"])
	if err != nil {
		log.Log.Error(err, "failed to parse secret data")
		return false, err
	}
	chain, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		log.Log.Error(err, "failed to parse certificate")
		return false, err
	}
	durationUntilExpiration := time.Until(chain.NotAfter)
	daysLeft := int(durationUntilExpiration.Hours() / 24)
	if daysLeft < int(threshold.Hours()/24) {
		log.Log.Info("certificate is not expiring soon", "days left", daysLeft)
		return false, nil
	}
	return true, nil
}

// Define a function to check if the annotation has been removed
// func isAnnotationRemoved(coreClient client.Client, namespace, annotation string) bool {
// 	ingresses := &networkingv1.IngressList{}
// 	err := coreClient.List(context.Background(), ingresses, client.InNamespace(namespace))
// 	if err != nil {
// 		log.Log.Error(err, "failed to list ingresses")
// 		return false
// 	}
// 	for _, ingress := range ingresses.Items {
// 		if _, ok := ingress.Annotations[annotation]; ok {
// 			log.Log.Info("annotation still present", "ingress", ingress.Name)
// 			return false
// 		}
// 	}
// 	return true
// }

// func (r *NimbleOpticAdapterConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
// 	log := log.FromContext(ctx)

// 	// Step 1: Get the NimbleOpticAdapterConfig custom resource
// 	config := &configv1alpha1.NimbleOpticAdapterConfig{}
// 	if err := r.Get(ctx, req.NamespacedName, config); err != nil {
// 		log.Error(err, "unable to fetch NimbleOpticAdapterConfig")
// 		return ctrl.Result{}, client.IgnoreNotFound(err)
// 	}

// 	// Step 2: Retrieve the relevant certificates and their expiration dates
// 	// move on all the ingress resources in the allowed namespace and that have the
// 	// annotations "cert-manager.io/cluster-issuer" and "nginx.ingress.kubernetes.io/backend-protocol: HTTPS",
// 	// take the secret name from it spec.tls.secretName, and return map object which contained the ingress path as
// 	// the key and a list which containd  "spec.tls.secretName" value, and it certificate expiration date. as the value.
// 	certificatesInfo, err := r.retrieveCertificates(ctx, config)
// 	if err != nil {
// 		log.Error(err, "unable to fetch certificates and their information")
// 		return ctrl.Result{}, err
// 	}

// 	// Step 3: Determine if any certificate needs renewal
// 	// Move on all the certificate expiration date from "Step 2", compare the expiration dates with
// 	// the CertificateRenewalThreshold from the config and return the ingress path that connected to the
// 	// secret which it certificates needed to be renewed.
// 	ingressPathsForRenewal := r.findIngressPathsForRenewal(certificatesInfo, config)

// 	// Step 4: Renew the certificates if necessary - Temporarily make the services unavailable
// 	// remove the "nginx.ingress.kubernetes.io/backend-protocol: HTTPS" annotation from the relevant Ingress resources and wait up to AnnotationRemovalDelay from the config or until the certificate is renewed (The first of them that takes place).
// 	err = r.removeIngressAnnotationsAndWait(ctx, ingressPathsForRenewal, config)
// 	if err != nil {
// 		log.Error(err, "failed to remove annotations and wait for certificate renewal")
// 		return ctrl.Result{}, err
// 	}

// 	// Step 5: Update the "nginx.ingress.kubernetes.io/backend-protocol: HTTPS" annotation - make the services available again.
// 	err = r.restoreIngressAnnotations(ctx, ingressPathsForRenewal, config)
// 	if err != nil {
// 		log.Error(err, "failed to restore annotations")
// 		return ctrl.Result{}, err
// 	}

// 	// Step 6: Set the status and conditions of the NimbleOpticAdapterConfig custom resource
// 	// Update the status of the custom resource to reflect the current state
// 	err = r.updateCustomResourceStatus(ctx, config, ingressPathsForRenewal)
// 	if err != nil {
// 		log.Error(err, "failed to update custom resource status")
// 		return ctrl.Result{}, err
// 	}

// 	// Finally, return the ctrl.Result{} and any error if applicable
// 	return ctrl.Result{}, nil
// }

// // SetupWithManager sets up the controller with the Manager.
// func (r *NimbleOpticAdapterConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
// 	return ctrl.NewControllerManagedBy(mgr).
// 		For(&configv1alpha1.NimbleOpticAdapterConfig{}).
// 		Complete(r)
// }
