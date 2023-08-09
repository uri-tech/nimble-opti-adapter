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

package controller

import (
	"context"

	// networkingv1 "k8s.io/api/networking/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	// Required for Watching

	adapterv1 "github.com/uri-tech/nimble-opti-adapter/api/v1"
)

// NimbleOptiReconciler reconciles a NimbleOpti object
type NimbleOptiReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	KubernetesClient kubernetes.Interface
	IngressWatcher   *IngressWatcher
}

//+kubebuilder:rbac:groups=adapter.uri-tech.github.io,resources=nimbleoptis,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=adapter.uri-tech.github.io,resources=nimbleoptis/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=adapter.uri-tech.github.io,resources=nimbleoptis/finalizers,verbs=update

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *NimbleOptiReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// debug
	klog.InfoS("debug - Reconcile")

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func (r *NimbleOptiReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// debug
	klog.InfoS("debug - SetupWithManager")

	// NewControllerManagedBy returns a builder for a controller
	// managed by mgr that will be started with mgr.Start.
	b := ctrl.NewControllerManagedBy(mgr)

	// For the primary resource type that this controller watches
	b = b.For(&adapterv1.NimbleOpti{})

	// Owns specifies objects that are owned by the primary resource
	// The argument here must be a runtime object that will have its
	// Group, Version, and Kind filled in.
	// b = b.Owns(&networkingv1.Ingress{})

	// WithEventFilter specifies a Predicate that will be used to filter
	// events before they are sent to event handlers.
	// The GenerationChangedPredicate filters out objects that have not changed their .metadata.generation field.
	// b = b.WithEventFilter(predicate.GenerationChangedPredicate{})

	// Watch Ingress objects - do not need it - do it from the IngressWatcher
	// if err := b.Watches(&networkingv1.Ingress{}, &handler.EnqueueRequestForObject{}); err != nil {
	// 	klog.Error(err, "unable to watch Ingress")
	// }

	// Watch for changes to Pods
	// if err := b.Watches(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForObject{}); err != nil {
	// 	klog.Error(err, "unable to watch Pods")
	// }

	// Call Complete to create the NimbleOptiReconciler. This step comes at the end
	// as it finalizes the controller's configuration.
	return b.Complete(r)
}
