// internal/controller/ingress_watcher.go

package controller

import (
	"context"
	"crypto/x509"
	"reflect"
	"strings"
	"time"

	v1 "github.com/uri-tech/nimble-opti-adapter/api/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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
	auditMutex      *NamedMutex
}

// StartAudit audits daily all Ingress resources with the label "nimble.opti.adapter/enabled=true" in the cluster
func (iw *IngressWatcher) StartAudit(stopCh <-chan struct{}) {
	// debug
	klog.InfoS("debug - StartAudit")

	go func() {
		// debug
		klog.InfoS("debug - StartAudit - go func")

		// Start immediate audit
		if err := iw.auditIngressResources(context.TODO()); err != nil {
			klog.ErrorS(err, "error auditing ingress resources")
		}

		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := iw.auditIngressResources(context.TODO()); err != nil {
					klog.ErrorS(err, "error auditing ingress resources")
				}
			case <-stopCh:
				return
			}
		}
	}()
}

// NewIngressWatcher initializes a new IngressWatcher and starts
// an IngressInformer for caching Ingress resources.
func NewIngressWatcher(clientKube kubernetes.Interface, stopCh <-chan struct{}) (*IngressWatcher, error) {
	// debug
	klog.InfoS("debug - NewIngressWatcher")

	cfg, err := config.GetConfig()
	if err != nil {
		klog.Fatalf("unable to get config %v", err)
		return nil, err
	}

	// Create a new scheme for decoding into.
	scheme := runtime.NewScheme()
	_ = v1.AddToScheme(scheme) // assuming `v1` package has `AddToScheme` function

	// Create a new client to Kubernetes API.
	cl, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		klog.Fatalf("unable to create client %v", err)
		return nil, err
	}

	iw := &IngressWatcher{
		Client:     clientKube,
		ClientObj:  cl,
		auditMutex: NewNamedMutex(),
	}

	informerFactory := informers.NewSharedInformerFactory(clientKube, 0)
	iw.IngressInformer = informerFactory.Networking().V1().Ingresses().Informer()
	iw.IngressInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    iw.handleIngressAdd,
		UpdateFunc: iw.handleIngressUpdate,
	})

	go iw.IngressInformer.Run(stopCh)

	return iw, nil
}

// handleIngressAdd is called when an Ingress resource is added.
func (iw *IngressWatcher) handleIngressAdd(obj interface{}) {
	// debug
	klog.InfoS("debug - handleIngressAdd")

	ctx := context.Background()

	ing, ok := obj.(*networkingv1.Ingress)
	if !ok {
		klog.Error("Expected Ingress in handleIngressAdd")
	}

	// If "nimble.opti.adapter/enabled" label is true, process it.
	if isAdapterEnabled(ctx, ing) {
		_, err := iw.processIngressForRenewal(ctx, ing)
		if err != nil {
			klog.Errorf("error processing ingress. %v", err)
		}
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

	// If the adapter is enabled on the new Ingress and the Ingress has changed, process it.
	if isAdapterEnabled(ctx, newIng) && hasIngressChanged(ctx, oldIng, newIng) {
		// debug
		klog.Infof("Ingress %s/%s has changed, processing", newIng.Namespace, newIng.Name)

		// Process the new Ingress.
		_, err := iw.processIngressForRenewal(ctx, newIng)
		if err != nil {
			klog.ErrorS(err, "error processing ingress")
		}
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

// processIngressForRenewal processes an Ingress resource for the adapter.
func (iw *IngressWatcher) processIngressForRenewal(ctx context.Context, ing *networkingv1.Ingress) (bool, error) {
	// debug
	klog.InfoS("debug  - processIngressForRenewal")

	// indicate if make ceartificate renewal process
	makeRenewal := false

	// Check if there's a v1.NimbleOpti CRD in the same namespace.
	adapter, err := iw.getOrCreateNimbleOpti(ctx, ing.Namespace)
	if err != nil {
		klog.Errorf("Failed to get or create v1.NimbleOpti: %v", err)
		return makeRenewal, err
	}

	// debug
	klog.Infof("adapter: %s", adapter)

	// Scan for any path in spec.rules[].http.paths[].path containing .well-known/acme-challenge.
	for _, rule := range ing.Spec.Rules {
		for _, path := range rule.IngressRuleValue.HTTP.Paths {
			if strings.Contains(path.Path, ".well-known/acme-challenge") {
				// debug
				klog.Infof("Found .well-known/acme-challenge in path %s", path.Path)
				// Trigger the certificate renewal process.
				if err := iw.startCertificateRenewal(ctx, ing, adapter); err != nil {
					klog.Errorf("Failed to start certificate renewal: %v", err)
					return makeRenewal, err
				}

				makeRenewal = true
			}
		}
	}
	return makeRenewal, nil
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
		if errors.IsNotFound(err) {
			// debug
			klog.InfoS("debug - create NimbleOpti")

			nimbleOpti = &v1.NimbleOpti{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "default",
					Namespace: namespace,
				},
				Spec: v1.NimbleOptiSpec{
					TargetNamespace:             namespace,
					CertificateRenewalThreshold: 30,
					AnnotationRemovalDelay:      10,
				},
			}

			if err := iw.ClientObj.Create(ctx, nimbleOpti); err != nil {
				klog.ErrorS(err, "Failed to create NimbleOpti", "namespace", namespace)
				return nil, err
			}
			// debug
			klog.InfoS("debug - create NimbleOpti done")
		} else {
			klog.ErrorS(err, "Failed to get NimbleOpti", "namespace", namespace)
			return nil, err
		}
	}

	return nimbleOpti, nil
}

// hasIngressChanged checks if the important parts of the Ingress have changed.
func hasIngressChanged(ctx context.Context, oldIng *networkingv1.Ingress, newIng *networkingv1.Ingress) bool {
	// debug
	klog.InfoS("debug - hasIngressChanged")

	// Check for changes in the spec
	if !reflect.DeepEqual(oldIng.Spec, newIng.Spec) {
		klog.InfoS("Ingress spec has changed")
		return true
	}

	oldLabelValue, oldLabelExists := oldIng.Labels["nimble.opti.adapter/enabled"]
	newLabelValue, newLabelExists := newIng.Labels["nimble.opti.adapter/enabled"]

	// Check for changes in the important labels
	if oldLabelExists != newLabelExists || (oldLabelExists && newLabelExists && oldLabelValue != newLabelValue) {
		klog.InfoS("Ingress nimble.opti.adapter/enabled label has changed")
		return true
	}

	// Check for changes in annotations
	if !reflect.DeepEqual(oldIng.Annotations, newIng.Annotations) {
		klog.InfoS("Ingress annotations have changed")
		return true
	}

	// Check for changes in the default backend
	if !reflect.DeepEqual(oldIng.Spec.DefaultBackend, newIng.Spec.DefaultBackend) {
		klog.InfoS("Ingress default backend has changed")
		return true
	}

	// Check for changes in the TLS configurations
	if !reflect.DeepEqual(oldIng.Spec.TLS, newIng.Spec.TLS) {
		klog.InfoS("Ingress TLS configuration has changed")
		return true
	}

	return false
}

// startCertificateRenewal get ingress that has "".well-known/acme-challenge" and resolve it.
func (iw *IngressWatcher) startCertificateRenewal(ctx context.Context, ing *networkingv1.Ingress, adapter *v1.NimbleOpti) error {
	// debug
	klog.InfoS("debug - startCertificateRenewal")

	// Remove the annotation.
	if err := iw.removeHTTPSAnnotation(ctx, ing); err != nil {
		klog.Errorf("Failed to remove HTTPS annotation: %v", err)
		return err
	}

	// Wait for the absence of the ACME challenge path or for the timeout.
	timeout := time.Duration(adapter.Spec.AnnotationRemovalDelay) * time.Second
	success, err := iw.waitForChallengeAbsence(ctx, timeout, ing.Namespace, ing.Name)
	if err != nil {
		klog.Errorf("Failed to wait for the absence of ACME challenge path: %v", err)
		return err
	}
	if !success {
		klog.Warningln("Failed to confirm the absence of ACME challenge path before timeout.")
	}

	// Reinstate the annotation.
	if err := iw.addHTTPSAnnotation(ctx, ing); err != nil {
		klog.Errorf("Failed to add HTTPS annotation: %v", err)
		return err
	}

	// Increment the certificate renewals counter.
	// TODO: implement incrementing nimble-opti-adapter_certificate_renewals_total  and sent to a Prometheus endpoint.

	return nil
}

// removeHTTPSAnnotation removes the "nginx.ingress.kubernetes.io/backend-protocol: HTTPS" annotation from an Ingress.
func (iw *IngressWatcher) removeHTTPSAnnotation(ctx context.Context, ing *networkingv1.Ingress) error {
	klog.InfoS("debug - removeHTTPSAnnotation")

	delete(ing.Annotations, "nginx.ingress.kubernetes.io/backend-protocol")

	// Update the Ingress.
	if err := iw.ClientObj.Update(ctx, ing); err != nil {
		klog.Error("Unable to remove HTTPS annotation: ", err)
		return err
	}

	return nil
}

// waitForChallengeAbsence waits for the absence of the ACME challenge path in the Ingress or until a timeout is reached.
// Returns false when the timeout has passed or there is an error.
func (iw *IngressWatcher) waitForChallengeAbsence(ctx context.Context, timeout time.Duration, ingNamespace, ingName string) (bool, error) {
	// debug
	klog.InfoS("Starting waitForACMEPathAbsence")

	// Create a child context with the specified timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel() // Ensure resources are cleaned up after timeout or successful completion

	for {
		select {
		case <-timeoutCtx.Done():
			klog.InfoS("Timeout reached or context cancelled. Stopping.")
			return false, nil
		default:
			// debug
			klog.InfoS("debug - Checking Ingress")

			// Get the Ingress
			ingress := &networkingv1.Ingress{}
			err := iw.ClientObj.Get(timeoutCtx, client.ObjectKey{Name: ingName, Namespace: ingNamespace}, ingress)
			if err != nil {
				klog.ErrorS(err, "Error fetching ingress")
				return false, err
			}

			// Check all paths of the Ingress for the ACME challenge path
			pathFound := false
			for _, rule := range ingress.Spec.Rules {
				for _, pathType := range rule.HTTP.Paths {
					if strings.Contains(pathType.Path, ".well-known/acme-challenge") {
						// debug
						klog.InfoS("debug - ACME challenge path found")
						pathFound = true
						break
					}
				}
				if pathFound {
					break
				}
			}

			if !pathFound {
				// If we reach here, the ACME challenge path was not found in any rule
				klog.InfoS("ACME challenge path not found. Stopping.")
				return true, nil
			}

			// Introduce a short delay to prevent high CPU usage
			time.Sleep(1 * time.Second)
		}
	}
}

// addHTTPSAnnotation adds the "nginx.ingress.kubernetes.io/backend-protocol: HTTPS" annotation to an Ingress.
func (iw *IngressWatcher) addHTTPSAnnotation(ctx context.Context, ing *networkingv1.Ingress) error {
	// debug
	klog.InfoS("debug - addHTTPSAnnotation")

	if ing.Annotations == nil {
		ing.Annotations = make(map[string]string)
	}
	ing.Annotations["nginx.ingress.kubernetes.io/backend-protocol"] = "HTTPS"

	// Update the Ingress.
	if err := iw.ClientObj.Update(ctx, ing); err != nil {
		klog.Error("Unable to add HTTPS annotation: ", err)
		return err
	}

	return nil
}

// // getIngressSecret fetches the Secret referenced in spec.tls[].secretName for a given Ingress.
// func (iw *IngressWatcher) getIngressSecret(ctx context.Context, ing *networkingv1.Ingress) (*corev1.Secret, error) {
// 	// debug
// 	klog.InfoS("debug - getIngressSecret")

// 	// Assuming the first TLS entry is the one to be used
// 	if len(ing.Spec.TLS) > 0 {
// 		secretName := ing.Spec.TLS[0].SecretName
// 		secret, err := iw.Client.CoreV1().Secrets(ing.Namespace).Get(context.Background(), secretName, metav1.GetOptions{})
// 		if err != nil {
// 			return nil, err
// 		}
// 		return secret, nil
// 	}
// 	return nil, nil
// }

// // updateIngressWithRetry updates an Ingress with retry on conflict.
// func (iw *IngressWatcher) updateIngressWithRetry(ctx context.Context, ing *networkingv1.Ingress) error {
// 	// debug
// 	klog.InfoS("debug - updateIngressWithRetry")

// 	// Creating a context with a timeout
// 	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
// 	defer cancel()

// 	// Define retry policy
// 	backoff := wait.Backoff{
// 		Steps:    5,
// 		Duration: 500 * time.Millisecond,
// 		Factor:   1.5,
// 	}

// 	// Implement the update operation with retry
// 	err := wait.ExponentialBackoff(backoff, func() (bool, error) {
// 		_, err := iw.Client.NetworkingV1().Ingresses(ing.Namespace).Update(ctx, ing, metav1.UpdateOptions{})
// 		if err != nil {
// 			return false, err
// 		}
// 		return true, nil
// 	})

// 	return err
// }

// auditIngressResources audits all Ingress with the label "nimble.opti.adapter/enabled:true".
func (iw *IngressWatcher) auditIngressResources(ctx context.Context) error {
	// debug
	klog.InfoS("debug - auditIngressResources")

	// Fetch all Ingress resources
	ingresses := &networkingv1.IngressList{}
	if err := iw.ClientObj.List(ctx, ingresses); err != nil {
		klog.Errorf("Failed to list ingresses: %v", err)
		return err
	}

	for _, ing := range ingresses.Items {
		// check if the ingress is labeled with the label "nimble.opti.adapter/enabled:true"
		if isAdapterEnabled(ctx, &ing) {
			// try to lock the namespace
			if iw.auditMutex.TryLock(ing.Namespace) {
				// process the ingress
				makeRenewal, err := iw.processIngressForRenewal(ctx, &ing)
				if err != nil {
					klog.Errorf("Failed to process ingress: %v", err)
					return err
				}
				// unlock the namespace
				iw.auditMutex.Unlock(ing.Namespace)

				if makeRenewal {
					// The operator fetches the associated Secret referenced in `spec.tls[].secretName` for each tls[],
					//  calculates the remaining time until certificate expiry and checks it against the `CertificateRenewalThreshold` specified in the `NimbleOpti` CRD.
					// If the certificate is due to expire within or on the threshold, certificate renewal is initiated.
					if err := iw.renewCertificateIfNecessary(ctx, &ing); err != nil {
						klog.Errorf("Error renewing certificate for ingress %s: %v", ing.Name, err)
						return err
					}
				}

			} else {
				klog.Infof("Failed to acquire lock for namespace: %s", ing.Namespace)
			}
		}
	}
	return nil
}

// move on all the secret connected to the ingress and renew the certificate if necessary
func (iw *IngressWatcher) renewCertificateIfNecessary(ctx context.Context, ing *networkingv1.Ingress) error {
	// Iterate over spec.tls[] to fetch associated secrets
	for _, tlsSpec := range ing.Spec.TLS {
		secretName := tlsSpec.SecretName

		// Fetch the secret
		secret := &corev1.Secret{}
		err := iw.ClientObj.Get(ctx, client.ObjectKey{Name: secretName, Namespace: ing.Namespace}, secret)
		if err != nil {
			klog.Errorf("Failed to fetch secret %s: %v", secretName, err)
			continue
		}

		// Extract the certificate from the secret. Assuming it's stored under the key "tls.crt"
		certBytes, ok := secret.Data["tls.crt"]
		if !ok {
			klog.Errorf("Secret %s does not have tls.crt", secretName)
			continue
		}
		cert, err := x509.ParseCertificate(certBytes)
		if err != nil {
			klog.Errorf("Failed to parse certificate from secret %s: %v", secretName, err)
			continue
		}

		// Calculate remaining duration until certificate expiry
		timeRemaining := cert.NotAfter.Sub(time.Now())

		// Fetch the associated NimbleOpti CRD
		adapter := &v1.NimbleOpti{}
		err = iw.ClientObj.Get(ctx, client.ObjectKey{Name: "default", Namespace: ing.Namespace}, adapter)
		if err != nil {
			klog.Errorf("Failed to fetch NimbleOpti CRD: %v", err)
			continue
		}

		// Check against CertificateRenewalThreshold
		if timeRemaining <= time.Duration(adapter.Spec.CertificateRenewalThreshold)*time.Hour {
			klog.Infof("Initiating certificate renewal for secret %s", secretName)
			iw.startCertificateRenewal(ctx, ing, adapter)
		}
	}
	return nil
}
