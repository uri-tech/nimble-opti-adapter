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

// internal/controller/nimbleoptiadapter_controller.go

package controller

import (
	"context"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	nimbleoptiadapterv1 "github.com/uri-tech/nimble-opti-adapter/api/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NimbleOptiAdapterReconciler reconciles a NimbleOptiAdapter object
type NimbleOptiAdapterReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	// Client          client.Client
	// Scheme          *runtime.Scheme
	KubernetesClient kubernetes.Interface
	IngressWatcher   *IngressWatcher
}

//+kubebuilder:rbac:groups=nimbleoptiadapter.nimble-opti-adapter.example.com,resources=nimbleoptiadapters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nimbleoptiadapter.nimble-opti-adapter.example.com,resources=nimbleoptiadapters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nimbleoptiadapter.nimble-opti-adapter.example.com,resources=nimbleoptiadapters/finalizers,verbs=update


// SetupWithManager sets up the controller with the Manager
func (r *NimbleOptiAdapterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nimbleoptiadapterv1.NimbleOptiAdapter{}).
		Owns(&networkingv1.Ingress{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}

// Modify your Reconcile function to handle events for Ingress crd "NimbleOptiAdapter" objects.
func (r *NimbleOptiAdapterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Fetch Ingress resource
	var ingress networkingv1.Ingress
	if err := r.Get(ctx, req.NamespacedName, &ingress); err != nil {
		// Error fetching Ingress resource
		log.Error(err, "Failed to get Ingress resource")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check for the nimble.opti.adapter/enabled: "true" label
	if enabledLabel, ok := ingress.Labels["nimble.opti.adapter/enabled"]; !ok || enabledLabel != "true" {
		// Label does not exist or is not set to "true", take no action
		return ctrl.Result{}, nil
	}

	// Get or create the corresponding NimbleOptiAdapter CRD
	nimbleOptiAdapterInstance, err := r.getOrCreateNimbleOptiAdapter(ctx, req.Namespace)
	if err != nil {
		log.Error(err, "Failed to get or create NimbleOptiAdapter CRD")
		return ctrl.Result{}, err
	}

	// Check for the presence of .well-known/acme-challenge in the path
	if r.checkForAcmeChallengePath(&ingress) {
		// If the path exists, initiate the certificate renewal process
		err = r.renewCertificate(ctx, &ingress, nimbleOptiAdapterInstance)
		if err != nil {
			log.Error(err, "Failed to renew certificate")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// getOrCreateNimbleOptiAdapter fetches an existing NimbleOptiAdapter CRD in the specified namespace or creates a new one if not found.
func (r *NimbleOptiAdapterReconciler) getOrCreateNimbleOptiAdapter(ctx context.Context, namespace string) (*nimbleoptiadapterv1.NimbleOptiAdapter, error) {
	nimbleOptiAdapterList := &nimbleoptiadapterv1.NimbleOptiAdapterList{}
	if err := r.List(ctx, nimbleOptiAdapterList, client.InNamespace(namespace)); err != nil {
		return nil, err
	}

	if len(nimbleOptiAdapterList.Items) == 0 {
		// No NimbleOptiAdapter CRD found, create a new one with default values
		newNimbleOptiAdapter := &nimbleoptiadapterv1.NimbleOptiAdapter{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "default",
				Namespace: namespace,
			},
			Spec: nimbleoptiadapterv1.NimbleOptiAdapterSpec{
				CertificateRenewalThreshold: 30,
				AnnotationRemovalDelay:      5 * 60, // 5 minutes in seconds
			},
		}
		if err := r.Create(ctx, newNimbleOptiAdapter); err != nil {
			return nil, err
		}
		return newNimbleOptiAdapter, nil
	}

	return &nimbleOptiAdapterList.Items[0], nil
}

// checkForAcmeChallengePath checks if any path in the Ingress resource contains .well-known/acme-challenge
func (r *NimbleOptiAdapterReconciler) checkForAcmeChallengePath(ingress *networkingv1.Ingress) bool {
	for _, rule := range ingress.Spec.Rules {
		for _, path := range rule.HTTP.Paths {
			if strings.Contains(path.Path, ".well-known/acme-challenge") {
				return true
			}
		}
	}
	return false
}

// renewCertificate initiates the certificate renewal process for the Ingress resource
func (r *NimbleOptiAdapterReconciler) renewCertificate(ctx context.Context, ingress *networkingv1.Ingress, nimbleOptiAdapter *nimbleoptiadapterv1.NimbleOptiAdapter) error {
	// Remove the nginx.ingress.kubernetes.io/backend-protocol: HTTPS annotation from the Ingress resource
	ingressCopy := ingress.DeepCopy()
	delete(ingressCopy.Annotations, "nginx.ingress.kubernetes.io/backend-protocol")

	if err := r.Update(ctx, ingressCopy); err != nil {
		return err
	}

	// Start a timer and wait until either there is no spec.rules[].http.paths[].path containing .well-known/acme-challenge,
	// or the AnnotationRemovalDelay time specified in the nimble-opti-adapter resource has passed
	// TODO: Implement the check for the presence of .well-known/acme-challenge in the path
	time.Sleep(time.Duration(nimbleOptiAdapter.Spec.AnnotationRemovalDelay) * time.Second)

	// Re-add the annotation nginx.ingress.kubernetes.io/backend-protocol: HTTPS to the Ingress resource
	ingressCopy.Annotations["nginx.ingress.kubernetes.io/backend-protocol"] = "HTTPS"

	if err := r.Update(ctx, ingressCopy); err != nil {
		return err
	}

	return nil
}
