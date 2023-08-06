// internal/controller/ingress_watcher.go

package controller

import (
	"context"
	"strings"
	"time"

	v1 "github.com/uri-tech/nimble-opti-adapter/api/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// IngressWatcher is a structure that holds the Client for Kubernetes
// API communication and IngressInformer for caching Ingress resources.
type IngressWatcher struct {
	Client          kubernetes.Interface
	IngressInformer cache.SharedIndexInformer
	ClientObj       client.Client
}

// NewIngressWatcher initializes a new IngressWatcher and starts
// an IngressInformer for caching Ingress resources.
func NewIngressWatcher(clientKube kubernetes.Interface, stopCh <-chan struct{}) *IngressWatcher {
	// debug
	klog.InfoS("debug - NewIngressWatcher")

	cfg, err := config.GetConfig()
	if err != nil {
		klog.Fatalf("unable to get config %v", err)
	}

	// Create a new scheme for decoding into.
	scheme := runtime.NewScheme()
	_ = v1.AddToScheme(scheme) // assuming `v1` package has `AddToScheme` function

	// Create a new client to Kubernetes API.
	cl, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		klog.Fatalf("unable to create client %v", err)
	}

	iw := &IngressWatcher{
		Client:    clientKube,
		ClientObj: cl,
	}

	informerFactory := informers.NewSharedInformerFactory(clientKube, 0)
	iw.IngressInformer = informerFactory.Networking().V1().Ingresses().Informer()
	iw.IngressInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    iw.handleIngressAdd,
		UpdateFunc: iw.handleIngressUpdate,
	})

	go iw.IngressInformer.Run(stopCh)

	return iw
}

// handleIngressAdd is called when an Ingress resource is added.
func (iw *IngressWatcher) handleIngressAdd(obj interface{}) {
	// debug
	klog.InfoS("debug - handleIngressAdd")

	ctx := context.Background()

	ing, ok := obj.(*networkingv1.Ingress)
	if !ok {
		klog.Error("Expected Ingress in handleIngressAdd")
		return
	}

	// If "nimble.opti.adapter/enabled" label is true, process it.
	if isAdapterEnabled(ctx, ing) {
		iw.processIngressForAdapter(ctx, ing)
	}
}

// handleIngressUpdate is called when an Ingress resource is updated.
// Implement logic to handle Ingress resource updates.
func (iw *IngressWatcher) handleIngressUpdate(oldObj, newObj interface{}) {
	// debug
	klog.InfoS("debug - handleIngressUpdate")

	ctx := context.Background()

	oldIng, ok := oldObj.(*networkingv1.Ingress)
	if !ok {
		klog.Error("Expected Ingress in handleIngressUpdate oldObj")
		return
	}

	newIng, ok := newObj.(*networkingv1.Ingress)
	if !ok {
		klog.Error("Expected Ingress in handleIngressUpdate newObj")
		return
	}

	// If the adapter is enabled on the new Ingress and it wasn't on the old one,
	// or if the Ingress was updated, process it.
	if isAdapterEnabled(ctx, newIng) && (!isAdapterEnabled(ctx, oldIng) || hasIngressChanged(ctx, oldIng, newIng)) {
		iw.processIngressForAdapter(ctx, newIng)
	}
}

// isAdapterEnabled checks if the "nimble.opti.adapter/enabled" label is present and set to "true".
func isAdapterEnabled(ctx context.Context, ing *networkingv1.Ingress) bool {
	// debug
	klog.InfoS("debug - isAdapterEnabled")
	klog.Infof("debug %s", ing)

	val, ok := ing.Labels["nimble.opti.adapter/enabled"]
	return ok && val == "true"
}

// processIngressForAdapter processes an Ingress to be used with v1.NimbleOpti.
func (iw *IngressWatcher) processIngressForAdapter(ctx context.Context, ing *networkingv1.Ingress) {
	// debug
	klog.InfoS("debug  - processIngressForAdapter")

	// Check if there's a v1.NimbleOpti CRD in the same namespace.
	adapter, err := iw.getOrCreateNimbleOpti(ctx, ing.Namespace)
	if err != nil {
		klog.Errorf("Failed to get or create v1.NimbleOpti: %v", err)
		return
	}

	// debug
	klog.Infof("adapter: %s", adapter)

	// Scan for any path in spec.rules[].http.paths[].path containing .well-known/acme-challenge.
	for _, rule := range ing.Spec.Rules {
		for _, path := range rule.IngressRuleValue.HTTP.Paths {
			if strings.Contains(path.Path, ".well-known/acme-challenge") {
				// Trigger the certificate renewal process.
				iw.startCertificateRenewal(ctx, ing, adapter)
			}
		}
	}
}

// getOrCreateNimbleOpti gets or creates a v1.NimbleOpti CRD in the same namespace as the Ingress.
func (iw *IngressWatcher) getOrCreateNimbleOpti(ctx context.Context, namespace string) (*v1.NimbleOpti, error) {
	// debug
	klog.InfoS("debug - getOrCreateNimbleOpti")
	klog.Infof("namespace: %s", namespace)

	nimbleOpti := &v1.NimbleOpti{}
	key := types.NamespacedName{
		Namespace: namespace,
		Name:      "default",
	}

	if err := iw.ClientObj.Get(ctx, key, nimbleOpti); err != nil {
		// debug
		klog.Infof("err: %s", err)

		if errors.IsNotFound(err) {
			// debug
			klog.InfoS("debug - create NimbleOpti")

			nimbleOpti = &v1.NimbleOpti{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "default",
					Namespace: namespace,
				},
				Spec: v1.NimbleOptiSpec{
					TargetNamespace:             "default",
					CertificateRenewalThreshold: 30,
					AnnotationRemovalDelay:      60,
					RenewalCheckInterval:        60,
				},
			}

			if err := iw.ClientObj.Create(ctx, nimbleOpti); err != nil {
				klog.ErrorS(err, "Failed to create NimbleOpti", "namespace", namespace)
				return nil, err
			}
		} else {
			klog.ErrorS(err, "Failed to get NimbleOpti", "namespace", namespace)
			return nil, err
		}
	}

	return nimbleOpti, nil
}

// hasIngressChanged checks if the important parts of the Ingress have changed.
// For now, it just checks if the host has changed. Expand this as needed.
func hasIngressChanged(ctx context.Context, oldIng, newIng *networkingv1.Ingress) bool {
	// debug
	klog.InfoS("debug - hasIngressChanged")

	// TODO: Implement a more sophisticated change detection if needed.
	return oldIng.Spec.Rules[0].Host != newIng.Spec.Rules[0].Host
}

// StartDailyAudit starts a daily audit of all Ingress resources.
func (iw *IngressWatcher) StartDailyAudit(ctx context.Context) {
	// debug
	klog.InfoS("debug - StartDailyAudit")

	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		for range ticker.C {
			iw.AuditIngressResources(ctx)
		}
	}()
}

// AuditIngressResources performs an audit of all Ingress resources.
func (iw *IngressWatcher) AuditIngressResources(ctx context.Context) {
	// debug
	klog.InfoS("debug - AuditIngressResources")

	// TODO: Fetch all Ingresses and v1.NimbleOpti CRDs from the cache.
	// For each pair, check if the certificate is due to expire and renew if needed.

	// Note: The actual process of fetching resources from the cache and checking
	// the certificates' expiry dates will depend on your specific use case and tools.
	// Make sure to replace the placeholder code with your own implementation.
}

// StartCertificateRenewal starts the certificate renewal process.
func (iw *IngressWatcher) StartCertificateRenewal(ctx context.Context, ing *networkingv1.Ingress) {
	// debug
	klog.InfoS("debug - StartCertificateRenewal")

	// Remove the annotation.
	iw.removeHTTPSAnnotation(ctx, ing)

	// Wait for the absence of the ACME challenge path or for the lapse of the AnnotationRemovalDelay.
	// TODO: Implement this wait using either a time.Sleep or a more complex mechanism if needed.

	// Reinstate the annotation.
	iw.addHTTPSAnnotation(ctx, ing)

	// Increment the certificate renewals counter.
	// TODO: Implement this, e.g., using Prometheus metrics.
}

// removeHTTPSAnnotation removes the "nginx.ingress.kubernetes.io/backend-protocol: HTTPS" annotation from an Ingress.
func (iw *IngressWatcher) removeHTTPSAnnotation(ctx context.Context, ing *networkingv1.Ingress) {
	// debug
	klog.InfoS("debug - removeHTTPSAnnotation")

	delete(ing.Annotations, "nginx.ingress.kubernetes.io/backend-protocol")

	// Update the Ingress.
	_, err := iw.Client.NetworkingV1().Ingresses(ing.Namespace).Update(context.Background(), ing, metav1.UpdateOptions{})
	if err != nil {
		klog.Error("Unable to remove HTTPS annotation: ", err)
	}
}

// addHTTPSAnnotation adds the "nginx.ingress.kubernetes.io/backend-protocol: HTTPS" annotation to an Ingress.
func (iw *IngressWatcher) addHTTPSAnnotation(ctx context.Context, ing *networkingv1.Ingress) {
	// debug
	klog.InfoS("debug - addHTTPSAnnotation")

	if ing.Annotations == nil {
		ing.Annotations = make(map[string]string)
	}
	ing.Annotations["nginx.ingress.kubernetes.io/backend-protocol"] = "HTTPS"

	// Update the Ingress.
	_, err := iw.Client.NetworkingV1().Ingresses(ing.Namespace).Update(context.Background(), ing, metav1.UpdateOptions{})
	if err != nil {
		klog.Error("Unable to add HTTPS annotation: ", err)
	}
}

// getIngressSecret fetches the Secret referenced in spec.tls[].secretName for a given Ingress.
func (iw *IngressWatcher) getIngressSecret(ctx context.Context, ing *networkingv1.Ingress) (*corev1.Secret, error) {
	// debug
	klog.InfoS("debug - getIngressSecret")

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
func calculateCertificateExpiry(ctx context.Context, secret *corev1.Secret) time.Duration {
	// debug
	klog.InfoS("debug - calculateCertificateExpiry")

	// TODO: Extract the certificate from the Secret and calculate the remaining time until its expiry.
	return 0
}

// updateIngressWithRetry updates an Ingress with retry on conflict.
func (iw *IngressWatcher) updateIngressWithRetry(ctx context.Context, ing *networkingv1.Ingress) error {
	// debug
	klog.InfoS("debug - updateIngressWithRetry")

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

func (iw *IngressWatcher) startCertificateRenewal(ctx context.Context, ing *networkingv1.Ingress, adapter *v1.NimbleOpti) {
	// debug
	klog.InfoS("debug - startCertificateRenewal")

	// Remove the annotation.
	iw.removeHTTPSAnnotation(ctx, ing)

	// Wait for the absence of the ACME challenge path or for the lapse of the AnnotationRemovalDelay.
	// TODO: Implement this wait using either a time.Sleep or a more complex mechanism if needed.

	// Reinstate the annotation.
	iw.addHTTPSAnnotation(ctx, ing)

	// Increment the certificate renewals counter.
	// TODO: Implement this, e.g., using Prometheus metrics.
}
