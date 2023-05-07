// internal/controller/ingress_watcher.go

package controller

import (
	"context"
	"crypto/x509"
	"fmt"
	"strings"
	"time"

	nimbleoptiadapterclientset "github.com/uri-tech/nimble-opti-adapter/pkg/generated/clientset/versioned"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

// IngressWatcher watches for Ingress resource events using a shared informer.
type IngressWatcher struct {
	Client                  kubernetes.Interface      // Kubernetes client
	IngressInformer         cache.SharedIndexInformer // Shared informer for Ingress resources
	NimbleOptiAdapterClient nimbleoptiadapterclientset.Interface
}

// NewIngressWatcher initializes a new IngressWatcher with the provided Kubernetes client.
func NewIngressWatcher(client kubernetes.Interface) *IngressWatcher {
	iw := &IngressWatcher{
		Client: client,
	}
	iw.NimbleOptiAdapterClient = nimbleOptiAdapterClient

	// Initialize a new shared informer factory and create an informer for Ingress resources.
	informerFactory := informers.NewSharedInformerFactory(client, 0)
	iw.IngressInformer = informerFactory.Networking().V1().Ingresses().Informer()

	return iw
}

// Start starts the IngressWatcher and begins watching for Ingress resource events.
func (iw *IngressWatcher) Start(stopCh <-chan struct{}) {
	// Add event handlers for the Add and Update events on Ingress resources.
	iw.IngressInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			ing := obj.(*networkingv1.Ingress)
			iw.handleIngressUpdate(ing)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			ing := newObj.(*networkingv1.Ingress)
			iw.handleIngressUpdate(ing)
		},
	})

	klog.Info("Starting Ingress Watcher")

	// Run the informer and wait for it to sync with the Kubernetes API server.
	go iw.IngressInformer.Run(stopCh)

	// Check if the informer cache has synced.
	if !cache.WaitForCacheSync(stopCh, iw.IngressInformer.HasSynced) {
		runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	// Start a separate goroutine to run the daily check for Ingress resources
	go iw.checkIngressesDaily(stopCh)
}

// handleIngressUpdate processes updates to Ingress resources.
func (iw *IngressWatcher) handleIngressUpdate(ing *networkingv1.Ingress) {
	// Handle Ingress resource update logic
}

// checkIngressesDaily checks Ingress resources with the specified label once a day.
func (iw *IngressWatcher) checkIngressesDaily(stopCh <-chan struct{}) {
	ticker := time.NewTicker(24 * time.Hour)

	for {
		select {
		case <-stopCh:
			ticker.Stop()
			return
		case <-ticker.C:
			// Perform the daily check for Ingress resources
			iw.processDailyIngressCheck()
		}
	}
}

func (iw *IngressWatcher) processDailyIngressCheck() {
	// Retrieve all Ingress resources with the specified label
	ingressList, err := iw.Client.NetworkingV1().Ingresses("").List(context.TODO(), metav1.ListOptions{
		LabelSelector: "nimble.opti.adapter/enabled=true",
	})
	if err != nil {
		klog.Errorf("Failed to list Ingress resources: %v", err)
		return
	}
	// for _, ing := range ingressList.Items {
	// 	// Check for the associated `NimbleOptiAdapter` CRD resource in the same namespace
	// 	// and retrieve the Secret specified in `spec.tls[].secretName` for each tls[]

	// 	// Calculate the time remaining until the certificate expires

	// 	// If the certificate expires in equal or fewer days than the `CertificateRenewalThreshold`
	// 	// specified in the `NimbleOptiAdapter` resource in the same namespace,
	// 	// initiate the certificate renewal process
	// 	// If the certificate expires in more days than the `CertificateRenewalThreshold`
	// 	// specified in the `NimbleOptiAdapter` resource in the same namespace,
	// 	// check if any path in `spec.rules[].http.paths[].path` contains `.well-known/acme-challenge`

	// 	// If there is a match, initiate the certificate renewal process
	// }
	for _, ing := range ingressList.Items {
		nimbleOptiAdapter, err := iw.NimbleOptiAdapterClient.YourApiGroupV1().NimbleOptiAdapters(ing.Namespace).Get(context.TODO(), ing.Name, metav1.GetOptions{})
		if err != nil {
			klog.Errorf("Failed to get NimbleOptiAdapter resource: %v", err)
			continue
		}

		for _, tls := range ing.Spec.TLS {
			secret, err := iw.Client.CoreV1().Secrets(ing.Namespace).Get(context.TODO(), tls.SecretName, metav1.GetOptions{})
			if err != nil {
				klog.Errorf("Failed to get Secret: %v", err)
				continue
			}

			// Use the secret to calculate the time remaining until the certificate expires
			// (Assuming the certificate is in the 'tls.crt' data field of the Secret)

			certBytes := secret.Data["tls.crt"]
			cert, err := x509.ParseCertificate(certBytes)
			if err != nil {
				klog.Errorf("Failed to parse certificate: %v", err)
				continue
			}

			remainingDays := cert.NotAfter.Sub(time.Now()).Hours() / 24

			if remainingDays <= float64(nimbleOptiAdapter.Spec.CertificateRenewalThreshold) {
				// Initiate the certificate renewal process
			} else {
				// Check if any path in `spec.rules[].http.paths[].path` contains `.well-known/acme-challenge`
				for _, rule := range ing.Spec.Rules {
					for _, path := range rule.HTTP.Paths {
						if strings.Contains(path.Path, ".well-known/acme-challenge") {
							// Initiate the certificate renewal process
						}
					}
				}
			}
		}
	}
}
