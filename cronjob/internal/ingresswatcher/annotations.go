package ingresswatcher

import (
	"context"
	"errors"
	"fmt"

	"github.com/uri-tech/nimble-opti-adapter/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"

	// _ "github.com/uri-tech/nimble-opti-adapter/cronjob/loggerpkg"
	networkingv1 "k8s.io/api/networking/v1"
)

// removeHTTPSAnnotation removes the "nginx.ingress.kubernetes.io/backend-protocol: HTTPS" annotation from an Ingress.
func (iw *IngressWatcher) removeHTTPSAnnotation(ctx context.Context, ing *networkingv1.Ingress) error {
	logger.Debugf("starting removeHTTPSAnnotation, ing: %v", ing.Name)

	key := utils.IngressKey(ing)

	if isLock := iw.auditMutex.TryLock(key); isLock {
		// Fetch the ingress again to get the last version
		if err := iw.ClientObj.Get(ctx, client.ObjectKey{Name: ing.Name, Namespace: ing.Namespace}, ing); err != nil {
			logger.Errorf("Failed to get ingress: %v", err)
			return err
		}
		delete(ing.Annotations, "nginx.ingress.kubernetes.io/backend-protocol")

		defer iw.auditMutex.Unlock(key)
		logger.Debug("removeHTTPSAnnotation - key is locked")

		if err := iw.ClientObj.Update(ctx, ing); err != nil {
			logger.Error("Unable to remove HTTPS annotation: ", err)
			return err
		}
		logger.Info("remove HTTPS annotation.")

	} else {
		errMassage := fmt.Sprintf("key %s is locked, and it should be unlocked", key)
		logger.Errorf(errMassage)
		return errors.New(errMassage)
	}

	return nil
}

// addHTTPSAnnotation adds the "nginx.ingress.kubernetes.io/backend-protocol: HTTPS" annotation to an Ingress.
func (iw *IngressWatcher) addHTTPSAnnotation(ctx context.Context, ing *networkingv1.Ingress) error {
	logger.Debugf("starting addHTTPSAnnotation, ing: %v", ing.Name)

	key := utils.IngressKey(ing)

	if isLock := iw.auditMutex.TryLock(key); isLock {
		// Fetch the ingress again to get the last version
		if err := iw.ClientObj.Get(ctx, client.ObjectKey{Name: ing.Name, Namespace: ing.Namespace}, ing); err != nil {
			logger.Errorf("Failed to get ingress: %v", err)
			return err
		}
		if ing.Annotations == nil {
			ing.Annotations = make(map[string]string)
		}
		ing.Annotations["nginx.ingress.kubernetes.io/backend-protocol"] = "HTTPS"

		defer iw.auditMutex.Unlock(key)
		logger.Debug("addHTTPSAnnotation - key is locked")

		if err := iw.ClientObj.Update(ctx, ing); err != nil {
			logger.Error("Unable to add HTTPS annotation: ", err)
			return err
		}

		logger.Info("add HTTPS annotation.")
	} else {
		errMassage := fmt.Sprintf("key %s is locked, and should be unlocked", key)
		logger.Errorf(errMassage)
		return errors.New(errMassage)
	}

	return nil
}
