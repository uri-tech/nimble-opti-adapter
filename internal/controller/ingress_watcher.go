// internal/controller/ingress_watcher.go

package controller

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	v1 "github.com/uri-tech/nimble-opti-adapter/api/v1"
	metrics "github.com/uri-tech/nimble-opti-adapter/metrics"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"

	errorsK8S "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/watch"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// IngressWatcher is a structure that holds the Client for Kubernetes
// API communication and IngressInformer for caching Ingress resources.
type IngressWatcher struct {
	// IngressWatcherClient IngressWatcherInterface
	// Client               kubernetes.Interface
	Client          KubernetesClient
	IngressInformer cache.SharedIndexInformer
	ClientObj       client.Client
	auditMutex      *NamedMutex
	Queue           workqueue.RateLimitingInterface
}

// KubernetesClient defines methods we're interested in mocking.
type KubernetesClient interface {
	Watch(ctx context.Context, namespace, ingressName string) (watch.Interface, error)
}

// RealKubernetesClient is a structure that holds the Client for Kubernetes.
type RealKubernetesClient struct {
	kubernetes.Interface
}

// Watch implements the KubernetesClient interface.
func (r *RealKubernetesClient) Watch(ctx context.Context, namespace, ingressName string) (watch.Interface, error) {
	// debug
	klog.Info("debug - RealKubernetesClient.Watch")

	opts := metav1.SingleObject(metav1.ObjectMeta{Name: ingressName})
	return r.NetworkingV1().Ingresses(namespace).Watch(ctx, opts)
}

// common way to do kreate uniq key in Kubernetes - use the namespace and the name of the resource, joined by a delimiter. it's like cache.MetaNamespaceKeyFunc(obj).
func ingressKey(ing *networkingv1.Ingress) string {
	// debug
	klog.Info("debug - ingressKey")

	return ing.Namespace + "/" + ing.Name
}

// NewIngressWatcher initializes a new IngressWatcher and starts
// an IngressInformer for caching Ingress resources.
func NewIngressWatcher(clientKube kubernetes.Interface, stopCh <-chan struct{}) (*IngressWatcher, error) {
	// debug
	klog.Info("debug - NewIngressWatcher")

	cfg, err := config.GetConfig()
	if err != nil {
		klog.Fatalf("unable to get config %v", err)
		return nil, err
	}

	// Create a new scheme for decoding into.
	scheme := runtime.NewScheme()
	// assuming `v1` package has `AddToScheme` function
	if err := v1.AddToScheme(scheme); err != nil {
		klog.Fatalf("unable to add v1 scheme %v", err)
		return nil, err
	}

	// Add client-go's scheme for core Kubernetes types
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		klog.Fatalf("unable to add client-go scheme %v", err)
		// setupLog.Error(err, "unable to add client-go scheme")
		return nil, err
	}

	// Create a new client to Kubernetes API.
	cl, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		klog.Fatalf("unable to create client %v", err)
		return nil, err
	}

	iw := &IngressWatcher{
		Client:     &RealKubernetesClient{clientKube},
		ClientObj:  cl,
		auditMutex: NewNamedMutex(),
		Queue:      workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "IngressQueue"),
	}

	// Setup informer
	informerFactory := informers.NewSharedInformerFactory(clientKube, 0)
	iw.IngressInformer = informerFactory.Networking().V1().Ingresses().Informer()
	iw.IngressInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj) // it like ingressKey

			// debug
			klog.Infof("debug - AddFunc - key: %s", key)

			if err != nil {
				klog.ErrorS(err, "Failed to get MetaNamespaceKey")
				return
			}
			if iw.auditMutex.IsLocked(key) {
				klog.Info("debug - AddFunc - key is locked, skip the processing")
				// iw.auditMutex.Unlock(key)
				return
			}
			iw.handleIngressAdd(obj)
		},
	})

	// After starting the IngressInformer
	go iw.IngressInformer.Run(stopCh)

	// // List all existing Ingress resources
	// _, err = clientKube.NetworkingV1().Ingresses("").List(context.TODO(), metav1.ListOptions{})
	// if err != nil {
	// 	klog.Errorf("Failed to list existing ingresses: %v", err)
	// }

	// Wait for the cache to be synced.
	if !cache.WaitForCacheSync(stopCh, iw.IngressInformer.HasSynced) {
		return nil, fmt.Errorf("failed to wait for caches to sync")
	}

	return iw, nil
}

// handleIngressAdd is called when an Ingress resource is added.
func (iw *IngressWatcher) handleIngressAdd(obj interface{}) {
	// debug
	klog.Info("debug - handleIngressAdd")

	ctx := context.Background()

	ing, ok := obj.(*networkingv1.Ingress)
	if !ok {
		klog.Error("Expected Ingress in handleIngressAdd")
	}

	// If "nimble.opti.adapter/enabled" label is true, process it.
	if isAdapterEnabledLabel(ctx, ing) && isBackendHttpsAnnotations(ctx, ing) {
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
	klog.Info("debug - handleIngressUpdate")

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
	if isAdapterEnabledLabel(ctx, newIng) && hasIngressChanged(ctx, oldIng, newIng) {
		// debug
		klog.Infof("Ingress %s/%s has changed, processing", newIng.Namespace, newIng.Name)

		// Process the new Ingress.
		_, err := iw.processIngressForRenewal(ctx, newIng)
		if err != nil {
			klog.ErrorS(err, "error processing ingress")
		}
	}
}

// removeHTTPSAnnotation removes the "nginx.ingress.kubernetes.io/backend-protocol: HTTPS" annotation from an Ingress.
func (iw *IngressWatcher) removeHTTPSAnnotation(ctx context.Context, ing *networkingv1.Ingress) error {
	// debug
	klog.Info("debug - removeHTTPSAnnotation")

	delete(ing.Annotations, "nginx.ingress.kubernetes.io/backend-protocol")

	key := ingressKey(ing)

	// debug
	klog.Infof("debug - removeHTTPSAnnotation - key: %s", key)

	if isLock := iw.auditMutex.TryLock(key); isLock {
		defer iw.auditMutex.Unlock(key)
		// debug
		klog.Infof("debug - removeHTTPSAnnotation - key is locked")

		if err := iw.ClientObj.Update(ctx, ing); err != nil {
			klog.Error("Unable to remove HTTPS annotation: ", err)
			return err
		}
	} else {
		errMassage := fmt.Sprintf("key %s is locked, and should be unlocked", key)
		klog.Errorf(errMassage)
		return errors.New(errMassage)
	}

	return nil
}

// addHTTPSAnnotation adds the "nginx.ingress.kubernetes.io/backend-protocol: HTTPS" annotation to an Ingress.
func (iw *IngressWatcher) addHTTPSAnnotation(ctx context.Context, ing *networkingv1.Ingress) error {
	// debug
	klog.Info("debug - addHTTPSAnnotation")

	if ing.Annotations == nil {
		ing.Annotations = make(map[string]string)
	}
	ing.Annotations["nginx.ingress.kubernetes.io/backend-protocol"] = "HTTPS"

	key := ingressKey(ing)

	// debug
	klog.Infof("debug - addHTTPSAnnotation - key: %s", key)

	if isLock := iw.auditMutex.TryLock(key); isLock {
		defer iw.auditMutex.Unlock(key)
		// debug
		klog.Infof("debug - addHTTPSAnnotation - key is locked")

		if err := iw.ClientObj.Update(ctx, ing); err != nil {
			klog.Error("Unable to add HTTPS annotation: ", err)
			return err
		}
	} else {
		errMassage := fmt.Sprintf("key %s is locked, and should be unlocked", key)
		klog.Errorf(errMassage)
		return errors.New(errMassage)
	}

	return nil
}

// StartAudit audits daily all Ingress resources with the label "nimble.opti.adapter/enabled=true" in the cluster
func (iw *IngressWatcher) StartAudit(stopCh <-chan struct{}) {
	// debug
	klog.Info("debug - StartAudit")

	go func() {
		// debug
		klog.Info("debug - StartAudit - go func")

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

// section 2

// isAdapterEnabledLabel checks if the "nimble.opti.adapter/enabled" label is present and set to "true".
func isAdapterEnabledLabel(ctx context.Context, ing *networkingv1.Ingress) bool {
	// debug
	klog.Info("debug - isAdapterEnabledLabel")

	val, ok := ing.Labels["nimble.opti.adapter/enabled"]

	return ok && val == "true"
}

// isBackendHttpsAnnotations checks if the "nginx.ingress.kubernetes.io/backend-protocol" annotation is present and set to "HTTPS".
func isBackendHttpsAnnotations(ctx context.Context, ing *networkingv1.Ingress) bool {
	// debug
	klog.Info("debug - isBackendHttpsAnnotations")

	val, ok := ing.Annotations["nginx.ingress.kubernetes.io/backend-protocol"]

	return ok && val == "HTTPS"
}

// processIngressForRenewal return true if it renew the certificate.
func (iw *IngressWatcher) processIngressForRenewal(ctx context.Context, ing *networkingv1.Ingress) (bool, error) {
	// debug
	klog.Info("debug  - processIngressForRenewal")

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
	if containsAcmeChallenge(ctx, ing) {
		// Trigger the certificate renewal process.
		isRenew, err := iw.startCertificateRenewal(ctx, ing, adapter)
		if err != nil {
			klog.Errorf("Failed to start certificate renewal: %v", err)
			return false, err
		}
		makeRenewal = isRenew
	}

	return makeRenewal, nil
}

// isAcmeChallengePath checks if the given path contains the ACME challenge string.
func isAcmeChallengePath(ctx context.Context, p string) bool {
	// debug
	klog.Info("debug - isAcmeChallengePath")

	const acmeChallengePath = ".well-known/acme-challenge"
	return strings.Contains(p, acmeChallengePath)
}

// containsAcmeChallenge checks if the given ingress contains any ACME challenge paths.
func containsAcmeChallenge(ctx context.Context, ing *networkingv1.Ingress) bool {
	// debug
	klog.Info("debug - containsAcmeChallenge")

	for _, rule := range ing.Spec.Rules {
		for _, path := range rule.IngressRuleValue.HTTP.Paths {
			if isAcmeChallengePath(ctx, path.Path) {
				klog.Infof("Found %s in path %s", ".well-known/acme-challenge", path.Path)
				return true
			}
		}
	}
	return false
}

// getOrCreateNimbleOpti gets or creates a v1.NimbleOpti CRD in the same namespace as the Ingress.
func (iw *IngressWatcher) getOrCreateNimbleOpti(ctx context.Context, namespace string) (*v1.NimbleOpti, error) {
	// debug
	klog.Info("debug - getOrCreateNimbleOpti")

	nimbleOpti := &v1.NimbleOpti{}
	key := types.NamespacedName{
		Namespace: namespace,
		Name:      namespace,
	}

	if err := iw.ClientObj.Get(ctx, key, nimbleOpti); err != nil {
		if errorsK8S.IsNotFound(err) {
			// debug
			klog.Info("debug - create NimbleOpti")

			nimbleOpti = &v1.NimbleOpti{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namespace,
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
			klog.Info("debug - create NimbleOpti done")

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
	klog.Info("debug - hasIngressChanged")

	// Check for changes in the spec.rules configurations
	if !reflect.DeepEqual(oldIng.Spec.Rules, newIng.Spec.Rules) {
		klog.Info("Ingress spec.rules configuration has changed")
		return true
	}

	return false
}

// startCertificateRenewal get ingress that has "".well-known/acme-challenge" and resolve it.
func (iw *IngressWatcher) startCertificateRenewal(ctx context.Context, ing *networkingv1.Ingress, adapter *v1.NimbleOpti) (bool, error) {
	// debug
	klog.Info("debug - startCertificateRenewal")

	var isRenew = false

	// Remove the annotation.
	if err := iw.removeHTTPSAnnotation(ctx, ing); err != nil {
		klog.Errorf("Failed to remove HTTPS annotation: %v", err)
		return false, err
	}

	// Wait for the absence of the ACME challenge path or for the timeout.
	timeout := time.Duration(adapter.Spec.AnnotationRemovalDelay) * time.Second
	successTime, err := iw.waitForChallengeAbsence(ctx, timeout, ing.Namespace, ing.Name)
	if err != nil {
		klog.Errorf("Failed to wait for the absence of ACME challenge path: %v", err)
		return false, err
	}
	if successTime > timeout {
		klog.Warningln("Failed to confirm the absence of ACME challenge path before timeout.")
	}

	// log the duration (in seconds) of annotation updates during each renewal
	if successTime == timeout*2 {
		klog.Infof("Annotation update duration: %v", timeout)
		metrics.RecordAnnotationUpdateDuration(timeout.Seconds())
	} else {
		klog.Infof("Annotation update duration: %v", successTime)
		metrics.RecordAnnotationUpdateDuration(successTime.Seconds())
		isRenew = true
	}

	// Reinstate the annotation.
	if err := iw.addHTTPSAnnotation(ctx, ing); err != nil {
		klog.Errorf("Failed to add HTTPS annotation: %v", err)
		return isRenew, err
	}

	// Increment the certificate renewals counter.
	if successTime <= timeout {
		metrics.IncrementCertificateRenewals()
	}

	return isRenew, nil
}

// waitForChallengeAbsence waits for the absence of the ACME challenge path in the Ingress or until a timeout is reached.
// Returns the time it took to renew(timeout*2 when it failed) or there is an error.
func (iw *IngressWatcher) waitForChallengeAbsence(ctx context.Context, timeout time.Duration, ingNamespace, ingName string) (time.Duration, error) {
	// debug
	klog.Info("Starting waitForChallengeAbsence")

	// Capture the start time
	startTime := time.Now()

	// Create a child context with the specified timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel() // Ensure resources are cleaned up after timeout or successful completion

	for {
		select {
		case <-timeoutCtx.Done():
			klog.Info("Timeout reached or context cancelled. Stopping.")
			return timeout * 2, nil
		default:
			// debug
			klog.Info("debug - Checking Ingress")

			// Get the Ingress
			ingress := &networkingv1.Ingress{}
			if err := iw.ClientObj.Get(timeoutCtx, client.ObjectKey{Name: ingName, Namespace: ingNamespace}, ingress); err != nil {
				klog.ErrorS(err, "Error fetching ingress")
				elapsedTime := time.Since(startTime)
				return elapsedTime, err
			}

			// Check all paths of the Ingress for the ACME challenge path
			pathFound := false
			for _, rule := range ingress.Spec.Rules {
				for _, pathType := range rule.HTTP.Paths {
					if strings.Contains(pathType.Path, ".well-known/acme-challenge") {
						// debug
						klog.Info("debug - ACME challenge path found")

						pathFound = true
						break
					}
				}
				if pathFound {
					break
				}
			}

			if !pathFound {
				// debug
				klog.Info("ACME challenge path not found. Stopping.")

				// If we reach here, the ACME challenge path was not found in any rule
				elapsedTime := time.Since(startTime)
				return elapsedTime, nil // Return the elapsed time on success
			}

			// Introduce a short delay to prevent high CPU usage
			time.Sleep(1 * time.Second)
		}
	}
}

// auditIngressResources audits all Ingress with the label "nimble.opti.adapter/enabled:true".
func (iw *IngressWatcher) auditIngressResources(ctx context.Context) error {
	// debug
	klog.Info("debug - auditIngressResources")

	// Fetch all Ingress resources
	ingresses := &networkingv1.IngressList{}

	// Fetch all Ingress resources using the standard Kubernetes client
	// ingresses, err := iw.Client.NetworkingV1().Ingresses("").List(ctx, metav1.ListOptions{})
	err := iw.ClientObj.List(ctx, ingresses, &client.ListOptions{})
	if err != nil {
		klog.Errorf("Failed to list ingresses: %v", err)
		return err
	}

	// Iterate through all Ingress resources
	for _, ing := range ingresses.Items {
		// check if the ingress is labeled with the label "nimble.opti.adapter/enabled:true"
		if isAdapterEnabledLabel(ctx, &ing) && isBackendHttpsAnnotations(ctx, &ing) {
			// process the ingress
			isRenew, err := iw.processIngressForRenewal(ctx, &ing)
			if err != nil {
				klog.Errorf("Failed to process ingress: %v", err)
				return err
			}

			if !isRenew {
				// The operator fetches the associated Secret referenced in `spec.tls[].secretName` for each tls[],
				//  calculates the remaining time until certificate expiry and checks it against the `CertificateRenewalThreshold` specified in the `NimbleOpti` CRD.
				// If the certificate is due to expire within or on the threshold, certificate renewal is initiated.
				if err := iw.renewValidCertificateIfNecessary(ctx, &ing); err != nil {
					klog.Errorf("Error renewing certificate for ingress %s: %v", ing.Name, err)
					return err
				}
			}
		}
	}
	return nil
}

// move on all the secret connected to the ingress and renew the certificate if necessary
func (iw *IngressWatcher) renewValidCertificateIfNecessary(ctx context.Context, ing *networkingv1.Ingress) error {
	// debug
	klog.Info("debug - renewValidCertificateIfNecessary")

	// Iterate over spec.tls[] to fetch associated secrets
	for _, tlsSpec := range ing.Spec.TLS {
		secretName := tlsSpec.SecretName

		// Fetch the secret
		secret := &corev1.Secret{}
		err := iw.ClientObj.Get(ctx, client.ObjectKey{Name: secretName, Namespace: ing.Namespace}, secret)
		if err != nil {
			klog.Errorf("Failed to fetch secret %s: %v", secretName, err)
			// continue
			return err
		}

		// Extract the certificate from the secret. Assuming it's stored under the key "tls.crt"
		certData, ok := secret.Data["tls.crt"]
		if !ok {
			klog.Errorf("Secret %s does not have tls.crt", secretName)
			return errors.New("missing tls.crt in secret")
		}

		// Check if the certificate is in PEM or DER format
		var certDER []byte
		if strings.Contains(string(certData), "-----BEGIN CERTIFICATE-----") {
			// debug
			klog.Info("debug - renewValidCertificateIfNecessary - PEM format")

			// Decode PEM to get the DER-encoded certificate
			block, _ := pem.Decode(certData)
			if block == nil || block.Type != "CERTIFICATE" {
				klog.Errorf("Failed to decode PEM block from secret %s", secretName)
				return errors.New("failed to decode PEM block")
			}
			certDER = block.Bytes
		} else {
			// debug
			klog.Info("debug - renewValidCertificateIfNecessary - DER format")

			// Assume it's DER format
			certDER = certData
		}

		cert, err := x509.ParseCertificate(certDER)
		if err != nil {
			klog.Errorf("Failed to parse certificate from secret %s: %v", secretName, err)
			return err
		}

		// Calculate remaining duration until certificate expiry
		timeRemaining := cert.NotAfter.Sub(time.Now())

		// debug
		klog.Infof("debug - timeRemaining: %v", timeRemaining)

		// Fetch the associated NimbleOpti CRD
		adapter := &v1.NimbleOpti{}
		err = iw.ClientObj.Get(ctx, client.ObjectKey{Name: ing.Namespace, Namespace: ing.Namespace}, adapter)
		if err != nil {
			klog.Errorf("Failed to fetch NimbleOpti CRD: %v", err)
			// continue
			return err
		}

		// debug
		klog.Infof("debug - adapter.Spec.CertificateRenewalThreshold: %v", adapter.Spec.CertificateRenewalThreshold)
		klog.Infof("debug - time.Duration(adapter.Spec.CertificateRenewalThreshold*24)*time.Hour: %s", time.Duration(adapter.Spec.CertificateRenewalThreshold*24)*time.Hour)

		// Check against CertificateRenewalThreshold
		if timeRemaining <= time.Duration(adapter.Spec.CertificateRenewalThreshold*24)*time.Hour {
			// debug
			klog.Infof("Initiating certificate renewal for secret %s", secretName)

			// Create a Secret object with only Name and Namespace populated.
			deleteSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: ing.Namespace,
				},
			}

			// Delete the secret.
			if err := iw.ClientObj.Delete(ctx, deleteSecret); err != nil {
				klog.Errorf("Failed to remove secret: %v", err)
				return err
			}

			// Wait until ".well-known/acme-challenge" appears in the path of the associate ingress.
			if err := iw.waitForAcmeChallenge(ctx, ing.Namespace, ing.Name); err != nil {
				return err
			}

			// Start certificate renewal
			_, err := iw.startCertificateRenewal(ctx, ing, adapter)
			if err != nil {
				klog.Errorf("Failed to start certificate renewal: %v", err)
				continue
			}
		}
	}
	return nil
}

// waitForAcmeChallenge waits for the ".well-known/acme-challenge" to appear in the specified ingress's paths.
// It uses a Kubernetes watcher to efficiently detect changes to the ingress resource.
//
// Parameters:
// - ctx: context for cancellation and timeout.
// - client: Kubernetes clientset to interact with the cluster.
// - namespace: The namespace where the ingress is located.
// - ingressName: The name of the ingress resource to watch.
//
// Returns:
// - nil if the acme challenge appears in the ingress paths.
// - error if the ingress gets deleted, if there's a watcher error, or if the function times out.
func (iw *IngressWatcher) waitForAcmeChallenge(ctx context.Context, namespace string, ingressName string) error {
	// debug
	klog.Info("debug - waitForAcmeChallenge")

	// Start watching the specified ingress for changes.
	// watcher, err := iw.Client.Client.NetworkingV1().Ingresses(namespace).Watch(ctx, metav1.SingleObject(metav1.ObjectMeta{Name: ingressName}))
	watcher, err := iw.Client.Watch(ctx, namespace, ingressName)
	if err != nil {
		return err
	}
	defer watcher.Stop()

	// Set a timeout for safety, for example, to exit after 10 minutes if the condition doesn't become true.
	timeoutCh := time.After(10 * time.Second)

	for {
		select {
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return fmt.Errorf("watch channel closed")
			}

			// debug
			klog.Infof("debug - event.Type: %v", event.Type)
			klog.Infof("debug - event.Object: %v", event.Object)

			// Handle different types of watch events.
			switch event.Type {
			case watch.Added, watch.Modified:
				// Check if the updated ingress contains the ACME challenge.
				ing, ok := event.Object.(*networkingv1.Ingress)
				if ok && containsAcmeChallenge(ctx, ing) {
					return nil
				}
			case watch.Deleted:
				return fmt.Errorf("ingress deleted before acme challenge appeared")
			case watch.Error:
				return fmt.Errorf("error watching ingress")
			}
		case <-timeoutCh:
			// Handle the case where the function times out.
			return fmt.Errorf("timed out waiting for acme challenge")
		case <-ctx.Done():
			// Handle context cancellation or deadline exceed.
			return ctx.Err()
		}
	}
}
