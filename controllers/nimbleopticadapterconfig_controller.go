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
	// This will depend on how you store and manage your certificates

	// Step 3: Determine if any certificate needs renewal
	// Compare the expiration dates with the CertificateRenewalThreshold from the config

	// Step 4: Renew the certificates if necessary
	// You'll need to implement the actual renewal logic based on your certificate management system

	// Step 5: Update the "nginx.ingress.kubernetes.io/backend-protocol: HTTPS" annotation
	// This will involve updating the Ingress resources with the specified annotation based on the AnnotationRemovalDelay from the config

	// Step 6: Set the status and conditions of the NimbleOpticAdapterConfig custom resource
	// Update the status of the custom resource to reflect the current state

	// Finally, return the ctrl.Result{} and any error if applicable
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NimbleOpticAdapterConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&configv1alpha1.NimbleOpticAdapterConfig{}).
		Complete(r)
}
