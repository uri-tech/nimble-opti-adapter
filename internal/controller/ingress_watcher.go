// internal/controller/ingress_watcher.go

package controller

import (
	"context"
	"strings"
	"time"

	"github.com/golang/glog"
	v1 "github.com/uri-tech/nimble-opti-adapter/api/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// IngressWatcher is a structure that holds the Client for Kubernetes
// API communication and IngressInformer for caching Ingress resources.
type IngressWatcher struct {
	Client          kubernetes.Interface
	IngressInformer cache.SharedIndexInformer
	// Add a NimbleOptiAdapterClient here
	// This client is responsible for communicating with v1.NimbleOptiAdapter API
	// NimbleOptiAdapterClient *nimbleoptiadapterclient.Client
	NimbleOptiAdapterClient NimbleOptiAdapterClient // Add the NimbleOptiAdapterClient here
}

// NimbleOptiAdapterClient represents the client for communicating with the v1.NimbleOptiAdapter API.
type NimbleOptiAdapterClient interface {
	Get(namespace, name string) (*v1.NimbleOptiAdapter, error)
	Create(namespace string, adapter *v1.NimbleOptiAdapter) (*v1.NimbleOptiAdapter, error)
}


// NewIngressWatcher initializes a new IngressWatcher and starts
// an IngressInformer for caching Ingress resources.
func NewIngressWatcher(client kubernetes.Interface) *IngressWatcher {
	iw := &IngressWatcher{
		Client: client,
	}

	// Using SharedIndexInformer to cache Ingress resources
	informerFactory := informers.NewSharedInformerFactory(client, 0)
	iw.IngressInformer = informerFactory.Networking().V1().Ingresses().Informer()
	iw.IngressInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    iw.handleIngressAdd,
		UpdateFunc: iw.handleIngressUpdate,
	})

	// Starting IngressInformer
	go iw.IngressInformer.Run(make(chan struct{}))

	return iw
}

// handleIngressAdd is called when an Ingress resource is added.
func (iw *IngressWatcher) handleIngressAdd(obj interface{}) {
	ing, ok := obj.(*networkingv1.Ingress)
	if !ok {
		glog.Error("Expected Ingress in handleIngressAdd")
		return
	}

	// TODO: Implement your logic for handling Ingress addition
	if isAdapterEnabled(ing) {
		iw.processIngressForAdapter(ing)
	}
}

// handleIngressUpdate is called when an Ingress resource is updated.
// Implement logic to handle Ingress resource updates.
func (iw *IngressWatcher) handleIngressUpdate(oldObj, newObj interface{}) {
	oldIng, ok := oldObj.(*networkingv1.Ingress)
	if !ok {
		glog.Error("Expected Ingress in handleIngressUpdate oldObj")
		return
	}

	newIng, ok := newObj.(*networkingv1.Ingress)
	if !ok {
		glog.Error("Expected Ingress in handleIngressUpdate newObj")
		return
	}

	// If the adapter is enabled on the new Ingress and it wasn't on the old one,
	// or if the Ingress was updated, process it.
	if isAdapterEnabled(newIng) && (!isAdapterEnabled(oldIng) || hasIngressChanged(oldIng, newIng)) {
		iw.processIngressForAdapter(newIng)
	}
}

// isAdapterEnabled checks if the "nimble.opti.adapter/enabled" label is present and set to "true".
func isAdapterEnabled(ing *networkingv1.Ingress) bool {
	val, ok := ing.Labels["nimble.opti.adapter/enabled"]
	return ok && val == "true"
}

// hasIngressChanged checks if the important parts of the Ingress have changed.
// For now, it just checks if the host has changed. Expand this as needed.
func hasIngressChanged(oldIng, newIng *networkingv1.Ingress) bool {
	// TODO: Implement a more sophisticated change detection if needed.
	return oldIng.Spec.Rules[0].Host != newIng.Spec.Rules[0].Host
}

// StartDailyAudit starts a daily audit of all Ingress resources.
func (iw *IngressWatcher) StartDailyAudit() {
	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		for range ticker.C {
			iw.AuditIngressResources()
		}
	}()
}

// AuditIngressResources performs an audit of all Ingress resources.
func (iw *IngressWatcher) AuditIngressResources() {
	// TODO: Fetch all Ingresses and v1.NimbleOptiAdapter CRDs from the cache.
	// For each pair, check if the certificate is due to expire and renew if needed.

	// Note: The actual process of fetching resources from the cache and checking
	// the certificates' expiry dates will depend on your specific use case and tools.
	// Make sure to replace the placeholder code with your own implementation.
}

// StartCertificateRenewal starts the certificate renewal process.
func (iw *IngressWatcher) StartCertificateRenewal(ing *networkingv1.Ingress) {
	// Remove the annotation.
	iw.removeHTTPSAnnotation(ing)

	// Wait for the absence of the ACME challenge path or for the lapse of the AnnotationRemovalDelay.
	// TODO: Implement this wait using either a time.Sleep or a more complex mechanism if needed.

	// Reinstate the annotation.
	iw.addHTTPSAnnotation(ing)

	// Increment the certificate renewals counter.
	// TODO: Implement this, e.g., using Prometheus metrics.
}

// removeHTTPSAnnotation removes the "nginx.ingress.kubernetes.io/backend-protocol: HTTPS" annotation from an Ingress.
func (iw *IngressWatcher) removeHTTPSAnnotation(ing *networkingv1.Ingress) {
	delete(ing.Annotations, "nginx.ingress.kubernetes.io/backend-protocol")

	// Update the Ingress.
	_, err := iw.Client.NetworkingV1().Ingresses(ing.Namespace).Update(context.Background(), ing, metav1.UpdateOptions{})
	if err != nil {
		glog.Error("Unable to remove HTTPS annotation: ", err)
	}
}

// addHTTPSAnnotation adds the "nginx.ingress.kubernetes.io/backend-protocol: HTTPS" annotation to an Ingress.
func (iw *IngressWatcher) addHTTPSAnnotation(ing *networkingv1.Ingress) {
	if ing.Annotations == nil {
		ing.Annotations = make(map[string]string)
	}
	ing.Annotations["nginx.ingress.kubernetes.io/backend-protocol"] = "HTTPS"

	// Update the Ingress.
	_, err := iw.Client.NetworkingV1().Ingresses(ing.Namespace).Update(context.Background(), ing, metav1.UpdateOptions{})
	if err != nil {
		glog.Error("Unable to add HTTPS annotation: ", err)
	}
}

// getIngressSecret fetches the Secret referenced in spec.tls[].secretName for a given Ingress.
func (iw *IngressWatcher) getIngressSecret(ing *networkingv1.Ingress) (*corev1.Secret, error) {
	// Assuming the first TLS entry is the one to be used
	if len(ing.Spec.TLS) > 0 {
		secretName := ing.Spec.TLS[0].SecretName
		secret, err := iw.Client.CoreV1().Secrets(ing.Namespace).Get(context.Background(), secretName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return secret, nil
	}
	return nil, nil
}

// calculateCertificateExpiry checks the remaining time until certificate expiry.
// This is a placeholder and should be replaced with actual implementation.
func calculateCertificateExpiry(secret *corev1.Secret) time.Duration {
	// TODO: Extract the certificate from the Secret and calculate the remaining time until its expiry.
	return 0
}

// updateIngressWithRetry updates an Ingress with retry on conflict.
func (iw *IngressWatcher) updateIngressWithRetry(ing *networkingv1.Ingress) error {
	// Creating a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Define retry policy
	backoff := wait.Backoff{
		Steps:    5,
		Duration: 500 * time.Millisecond,
		Factor:   1.5,
	}

	// Implement the update operation with retry
	err := wait.ExponentialBackoff(backoff, func() (bool, error) {
		_, err := iw.Client.NetworkingV1().Ingresses(ing.Namespace).Update(ctx, ing, metav1.UpdateOptions{})
		if err != nil {
			return false, err
		}
		return true, nil
	})

	return err
}

// getOrCreateNimbleOptiAdapter gets or creates a v1.NimbleOptiAdapter CRD in the same namespace as the Ingress.
func (iw *IngressWatcher) getOrCreateNimbleOptiAdapter(namespace string) (*v1.NimbleOptiAdapter, error) {
	// Assuming a Get function on your NimbleOptiAdapterClient
	adapter, err := iw.NimbleOptiAdapterClient.Get(namespace, "default")
	if err != nil {
		if errors.IsNotFound(err) {
			// The v1.NimbleOptiAdapter does not exist, create it
			// Assuming a Create function on your NimbleOptiAdapterClient
			adapter, err = iw.NimbleOptiAdapterClient.Create(namespace, &v1.NimbleOptiAdapter{
				// Set your default values here
			})
			if err != nil {
				return nil, err
			}
		} else {
			// An unexpected error occurred
			return nil, err
		}
	}

	return adapter, nil
}

// processIngressForAdapter processes an Ingress to be used with v1.NimbleOptiAdapter.
func (iw *IngressWatcher) processIngressForAdapter(ing *networkingv1.Ingress) {
	// Check if there's a v1.NimbleOptiAdapter CRD in the same namespace.
	adapter, err := iw.getOrCreateNimbleOptiAdapter(ing.Namespace)
	if err != nil {
		glog.Errorf("Failed to get or create v1.NimbleOptiAdapter: %v", err)
		return
	}

	// Scan for any path in spec.rules[].http.paths[].path containing .well-known/acme-challenge.
	for _, rule := range ing.Spec.Rules {
		for _, path := range rule.IngressRuleValue.HTTP.Paths {
			if strings.Contains(path.Path, ".well-known/acme-challenge") {
				// Trigger the certificate renewal process.
				iw.startCertificateRenewal(ing, adapter)
			}
		}
	}
}

func (iw *IngressWatcher) startCertificateRenewal(ing *networkingv1.Ingress, adapter *v1.NimbleOptiAdapter) {
	// Remove the annotation.
	iw.removeHTTPSAnnotation(ing)

	// Wait for the absence of the ACME challenge path or for the lapse of the AnnotationRemovalDelay.
	// TODO: Implement this wait using either a time.Sleep or a more complex mechanism if needed.

	// Reinstate the annotation.
	iw.addHTTPSAnnotation(ing)

	// Increment the certificate renewals counter.
	// TODO: Implement this, e.g., using Prometheus metrics.
}
