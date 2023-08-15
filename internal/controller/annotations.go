package controller

import (
	"context"
	"errors"
	"fmt"

	"github.com/uri-tech/nimble-opti-adapter/utils"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// removeHTTPSAnnotation removes the "nginx.ingress.kubernetes.io/backend-protocol: HTTPS" annotation from an Ingress.
func (iw *IngressWatcher) removeHTTPSAnnotation(ctx context.Context, ing *networkingv1.Ingress) error {
	klog.Infof("starting removeHTTPSAnnotation, ing: %v", ing.Name)

	key := utils.IngressKey(ing)

	if isLock := iw.auditMutex.TryLock(key); isLock {
		// Fetch the ingress again to get the last version
		if err := iw.ClientObj.Get(ctx, client.ObjectKey{Name: ing.Name, Namespace: ing.Namespace}, ing); err != nil {
			klog.Errorf("Failed to get ingress: %v", err)
			return err
		}
		delete(ing.Annotations, "nginx.ingress.kubernetes.io/backend-protocol")

		defer iw.auditMutex.Unlock(key)
		klog.Info("removeHTTPSAnnotation - key is locked")

		if err := iw.ClientObj.Update(ctx, ing); err != nil {
			klog.Error("Unable to remove HTTPS annotation: ", err)
			return err
		}
		klog.Info("remove HTTPS annotation.")

	} else {
		errMassage := fmt.Sprintf("key %s is locked, and it should be unlocked", key)
		klog.Errorf(errMassage)
		return errors.New(errMassage)
	}

	return nil
}

// addHTTPSAnnotation adds the "nginx.ingress.kubernetes.io/backend-protocol: HTTPS" annotation to an Ingress.
func (iw *IngressWatcher) addHTTPSAnnotation(ctx context.Context, ing *networkingv1.Ingress) error {
	klog.Infof("starting addHTTPSAnnotation, ing: %v", ing.Name)

	key := utils.IngressKey(ing)

	if isLock := iw.auditMutex.TryLock(key); isLock {
		// Fetch the ingress again to get the last version
		if err := iw.ClientObj.Get(ctx, client.ObjectKey{Name: ing.Name, Namespace: ing.Namespace}, ing); err != nil {
			klog.Errorf("Failed to get ingress: %v", err)
			return err
		}
		if ing.Annotations == nil {
			ing.Annotations = make(map[string]string)
		}
		ing.Annotations["nginx.ingress.kubernetes.io/backend-protocol"] = "HTTPS"

		defer iw.auditMutex.Unlock(key)
		klog.Info("addHTTPSAnnotation - key is locked")

		if err := iw.ClientObj.Update(ctx, ing); err != nil {
			klog.Error("Unable to add HTTPS annotation: ", err)
			return err
		}

		klog.Info("add HTTPS annotation.")
	} else {
		errMassage := fmt.Sprintf("key %s is locked, and should be unlocked", key)
		klog.Errorf(errMassage)
		return errors.New(errMassage)
	}

	return nil
}
