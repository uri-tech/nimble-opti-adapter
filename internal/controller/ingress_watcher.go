// internal/controller/ingress_watcher.go

package controller

import (
	"time"

	"github.com/golang/glog"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// IngressWatcher is a structure that holds the Client for Kubernetes
// API communication and IngressInformer for caching Ingress resources.
type IngressWatcher struct {
	Client          kubernetes.Interface
	IngressInformer cache.SharedIndexInformer
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
		UpdateFunc: iw.handleIngressUpdate,
	})

	// Starting IngressInformer
	go iw.IngressInformer.Run(make(chan struct{}))

	return iw
}

// handleIngressUpdate is called when an Ingress resource is updated.
// Implement logic to handle Ingress resource updates.
func (iw *IngressWatcher) handleIngressUpdate(oldObj, newObj interface{}) {
	ing, ok := newObj.(*networkingv1.Ingress)
	if !ok {
		glog.Error("Expected Ingress in handleIngressUpdate")
		return
	}

	// TODO: Implement your logic for handling Ingress updates
}

// processDailyIngressCheck checks all Ingress resources daily and performs
// necessary operations if any certificate needs to be renewed.
func (iw *IngressWatcher) processDailyIngressCheck() {
	for {
		// Sleep for a day before processing
		time.Sleep(24 * time.Hour)

		// List all Ingress resources from the cache
		ingressList := iw.IngressInformer.GetStore().List()

		for _, obj := range ingressList {
			ing, ok := obj.(*networkingv1.Ingress)
			if !ok {
				glog.Error("Expected Ingress in processDailyIngressCheck")
				continue
			}

			// TODO: Here you should check the annotations or TLS sections
			// of the Ingress resource and decide whether a certificate needs
			// to be renewed or not. If a certificate needs to be renewed,
			// you can call a function to perform the renewal operation.
			// Note: This function should be implemented by you.
		}
	}
}
