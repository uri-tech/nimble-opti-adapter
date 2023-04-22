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
	"crypto/x509"
	"fmt"
	"time"

	"context"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	configv1alpha1 "github.com/uri-tech/NimbleOpticAdapter/api/v1alpha1"
)

// NimbleOpticAdapterConfigReconciler reconciles a NimbleOpticAdapterConfig object
type NimbleOpticAdapterConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

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
func (r *NimbleOpticAdapterConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Step 1: Get the NimbleOpticAdapterConfig custom resource
	config := &configv1alpha1.NimbleOpticAdapterConfig{}
	if err := r.Get(ctx, req.NamespacedName, config); err != nil {
		log.Error(err, "unable to fetch NimbleOpticAdapterConfig")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Step 2: Retrieve the relevant certificates and their expiration dates
	// move on all the ingress resources in the allowed namespace and that have the
	// annotations "cert-manager.io/cluster-issuer" and "nginx.ingress.kubernetes.io/backend-protocol: HTTPS",
	// take the secret name from it spec.tls.secretName, and return map object which contained the ingress path as
	// the key and a list which containd  "spec.tls.secretName" value, and it certificate expiration date. as the value.
	certificatesInfo, err := r.retrieveCertificates(ctx, config)
	if err != nil {
		log.Error(err, "unable to fetch certificates and their information")
		return ctrl.Result{}, err
	}

	// Step 3: Determine if any certificate needs renewal
	// Move on all the certificate expiration date from "Step 2", compare the expiration dates with
	// the CertificateRenewalThreshold from the config and return the ingress path that connected to the
	// secret which it certificates needed to be renewed.
	ingressPathsForRenewal := r.findIngressPathsForRenewal(certificatesInfo, config)

	// Step 4: Renew the certificates if necessary - Temporarily make the services unavailable
	// remove the "nginx.ingress.kubernetes.io/backend-protocol: HTTPS" annotation from the relevant Ingress resources and wait up to AnnotationRemovalDelay from the config or until the certificate is renewed (The first of them that takes place).
	err = r.removeIngressAnnotationsAndWait(ctx, ingressPathsForRenewal, config)
	if err != nil {
		log.Error(err, "failed to remove annotations and wait for certificate renewal")
		return ctrl.Result{}, err
	}

	// Step 5: Update the "nginx.ingress.kubernetes.io/backend-protocol: HTTPS" annotation - make the services available again.
	err = r.restoreIngressAnnotations(ctx, ingressPathsForRenewal, config)
	if err != nil {
		log.Error(err, "failed to restore annotations")
		return ctrl.Result{}, err
	}

	// Step 6: Set the status and conditions of the NimbleOpticAdapterConfig custom resource
	// Update the status of the custom resource to reflect the current state
	err = r.updateCustomResourceStatus(ctx, config, ingressPathsForRenewal)
	if err != nil {
		log.Error(err, "failed to update custom resource status")
		return ctrl.Result{}, err
	}

	// Finally, return the ctrl.Result{} and any error if applicable
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NimbleOpticAdapterConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&configv1alpha1.NimbleOpticAdapterConfig{}).
		Complete(r)
}

// move on all the ingress resources in the allowed namespace and that have the
// annotations "cert-manager.io/cluster-issuer" and "nginx.ingress.kubernetes.io/backend-protocol: HTTPS",
// take the secret name from it spec.tls.secretName, and return map object which contained the
// ingress path as the key and a list which containd  "spec.tls.secretName" value, and it certificate expiration date.
// as the value.
func (r *NimbleOpticAdapterConfigReconciler) retrieveCertificates(ctx context.Context, config *configv1alpha1.NimbleOpticAdapterConfig) (map[string][]interface{}, error) {
	ingressList := &networkingv1.IngressList{}
	err := r.List(ctx, ingressList, &client.ListOptions{Namespace: config.Namespace})
	if err != nil {
		return nil, err
	}

	certificatesInfo := make(map[string][]interface{})

	for _, ingress := range ingressList.Items {
		// Check if the ingress resource has the required annotations
		annotations := ingress.GetAnnotations()
		if annotations["cert-manager.io/cluster-issuer"] != "" && annotations["nginx.ingress.kubernetes.io/backend-protocol"] == "HTTPS" {
			// Iterate through the ingress TLS configuration
			for _, tls := range ingress.Spec.TLS {
				secretName := tls.SecretName
				secret := &corev1.Secret{}
				err := r.Get(ctx, client.ObjectKey{Name: secretName, Namespace: ingress.Namespace}, secret)
				if err != nil {
					return nil, err
				}

				// Extract the expiration date from the secret
				certBytes := secret.Data["tls.crt"]
				cert, err := x509.ParseCertificate(certBytes)
				if err != nil {
					return nil, err
				}
				expirationDate := cert.NotAfter

				// Iterate through the ingress rules and add their path to the map with the secret name and expiration date
				for _, rule := range ingress.Spec.Rules {
					for _, path := range rule.HTTP.Paths {
						certificatesInfo[path.Path] = []interface{}{secretName, expirationDate}
					}
				}
			}
		}
	}
	return certificatesInfo, nil
}

// Move on all the certificate expiration date from "Step 2", compare the expiration dates with the
// CertificateRenewalThreshold from the config and return the ingress path that connected to the secret
// which it certificates needed to be renewed.
func (r *NimbleOpticAdapterConfigReconciler) findIngressPathsForRenewal(certificatesInfo map[string][]interface{}, config *configv1alpha1.NimbleOpticAdapterConfig) []string {
	ingressPathsForRenewal := []string{}

	for ingressPath, certInfo := range certificatesInfo {
		expirationDate := certInfo[1].(time.Time)
		renewalThreshold := time.Duration(config.Spec.CertificateRenewalThreshold) * time.Hour
		thresholdTime := time.Now().Add(renewalThreshold)

		// If the certificate expires within the renewal threshold, add the ingress path to the list for renewal
		if expirationDate.Before(thresholdTime) {
			ingressPathsForRenewal = append(ingressPathsForRenewal, ingressPath)
		}
	}

	return ingressPathsForRenewal
}

// remove the "nginx.ingress.kubernetes.io/backend-protocol: HTTPS" annotation from the relevant Ingress resources
// and wait up to AnnotationRemovalDelay from the config or until the certificate is renewed (The first of them that takes place)
func (r *NimbleOpticAdapterConfigReconciler) removeIngressAnnotationsAndWait(ctx context.Context, ingressPathsForRenewal []string, config *configv1alpha1.NimbleOpticAdapterConfig) error {
	annotationKey := "nginx.ingress.kubernetes.io/backend-protocol"
	annotationValue := "HTTPS"

	for _, ingressPath := range ingressPathsForRenewal {
		// Find the Ingress resource by the ingress path
		ingress := &networkingv1.Ingress{}
		err := r.Get(ctx, client.ObjectKey{Name: ingressPath, Namespace: config.Namespace}, ingress)
		if err != nil {
			return fmt.Errorf("failed to get Ingress resource: %w", err)
		}

		// Remove the annotation if it exists
		if ingress.Annotations[annotationKey] == annotationValue {
			delete(ingress.Annotations, annotationKey)

			// Update the Ingress resource
			err = r.Update(ctx, ingress)
			if err != nil {
				return fmt.Errorf("failed to update Ingress resource: %w", err)
			}
		}

		// Wait for the certificate to be renewed or the AnnotationRemovalDelay to be reached
		renewalTimeout := time.After(time.Duration(config.Spec.AnnotationRemovalDelay) * time.Second)
		ticker := time.NewTicker(10 * time.Second)

		defer ticker.Stop()

		for {
			select {
			case <-renewalTimeout:
				return nil
			case <-ticker.C:
				// Check if the certificate has been renewed
				// Implement your logic to check if the certificate is renewed
				certificateRenewed := false
				if certificateRenewed {
					return nil
				}
			}
		}
	}

	return nil
}

// Update the "nginx.ingress.kubernetes.io/backend-protocol: HTTPS" annotation - make the services available again.
func (r *NimbleOpticAdapterConfigReconciler) restoreIngressAnnotations(ctx context.Context, ingressPathsForRenewal []string, config *configv1alpha1.NimbleOpticAdapterConfig) error {
	annotationKey := "nginx.ingress.kubernetes.io/backend-protocol"
	annotationValue := "HTTPS"

	for _, ingressPath := range ingressPathsForRenewal {
		// Find the Ingress resource by the ingress path
		ingress := &networkingv1.Ingress{}
		err := r.Get(ctx, client.ObjectKey{Name: ingressPath, Namespace: config.Namespace}, ingress)
		if err != nil {
			return fmt.Errorf("failed to get Ingress resource: %w", err)
		}

		// Update the annotation if it does not exist
		if ingress.Annotations[annotationKey] != annotationValue {
			if ingress.Annotations == nil {
				ingress.Annotations = make(map[string]string)
			}
			ingress.Annotations[annotationKey] = annotationValue

			// Update the Ingress resource
			err = r.Update(ctx, ingress)
			if err != nil {
				return fmt.Errorf("failed to update Ingress resource: %w", err)
			}
		}
	}

	return nil
}

func (r *NimbleOpticAdapterConfigReconciler) updateCustomResourceStatus(ctx context.Context, config *configv1alpha1.NimbleOpticAdapterConfig, ingressPathsForRenewal []string) error {
	// Set the status and conditions of the NimbleOpticAdapterConfig custom resource
	config.Status.IngressPathsForRenewal = ingressPathsForRenewal

	// Update the status of the custom resource to reflect the current state
	err := r.Status().Update(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to update NimbleOpticAdapterConfig status: %w", err)
	}

	return nil
}
