package ingresswatcher

import (
	"context"
	"errors"
	"fmt"

	"github.com/uri-tech/nimble-opti-adapter/cronjob/internal/utils"
	// _ "github.com/uri-tech/nimble-opti-adapter/cronjob/loggerpkg"
	networkingv1 "k8s.io/api/networking/v1"
)

// removeHTTPSAnnotation removes the "nginx.ingress.kubernetes.io/backend-protocol: HTTPS" annotation from an Ingress.
func (iw *IngressWatcher) removeHTTPSAnnotation(ctx context.Context, ing *networkingv1.Ingress) error {
	// debug
	logger.Debug("removeHTTPSAnnotation")

	delete(ing.Annotations, "nginx.ingress.kubernetes.io/backend-protocol")

	key := utils.IngressKey(ing)

	if isLock := iw.auditMutex.TryLock(key); isLock {
		defer iw.auditMutex.Unlock(key)
		logger.Debug("removeHTTPSAnnotation - key is locked")

		if err := iw.ClientObj.Update(ctx, ing); err != nil {
			logger.Error("Unable to remove HTTPS annotation: ", err)
			return err
		}
	} else {
		errMassage := fmt.Sprintf("key %s is locked, and it should be unlocked", key)
		logger.Errorf(errMassage)
		return errors.New(errMassage)
	}

	return nil
}

// addHTTPSAnnotation adds the "nginx.ingress.kubernetes.io/backend-protocol: HTTPS" annotation to an Ingress.
func (iw *IngressWatcher) addHTTPSAnnotation(ctx context.Context, ing *networkingv1.Ingress) error {
	logger.Debug("addHTTPSAnnotation")

	if ing.Annotations == nil {
		ing.Annotations = make(map[string]string)
	}
	ing.Annotations["nginx.ingress.kubernetes.io/backend-protocol"] = "HTTPS"

	key := utils.IngressKey(ing)

	if isLock := iw.auditMutex.TryLock(key); isLock {
		defer iw.auditMutex.Unlock(key)
		logger.Debug("addHTTPSAnnotation - key is locked")

		if err := iw.ClientObj.Update(ctx, ing); err != nil {
			logger.Error("Unable to add HTTPS annotation: ", err)
			return err
		}
	} else {
		errMassage := fmt.Sprintf("key %s is locked, and should be unlocked", key)
		logger.Errorf(errMassage)
		return errors.New(errMassage)
	}

	return nil
}
