package ingresswatcher

import (
	"context"
	"errors"
	"time"

	v1 "github.com/uri-tech/nimble-opti-adapter/api/v1"
	"github.com/uri-tech/nimble-opti-adapter/cronjob/configenv"
	"github.com/uri-tech/nimble-opti-adapter/cronjob/internal/utils"
	"github.com/uri-tech/nimble-opti-adapter/cronjob/loggerpkg"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type IngressWatcher struct {
	Client     KubernetesClient
	ClientObj  client.Client
	auditMutex *utils.NamedMutex
	Config     *configenv.ConfigEnv
}

// logger is the logger for the ingresswatcher package.
var logger = loggerpkg.GetNamedLogger("ingresswatcher")

// func NewIngressWatcher(clientKube *kubernetes.Clientset, ecfg *configenv.ConfigEnv) (*IngressWatcher, error) {
func NewIngressWatcher(clientKube kubernetes.Interface, ecfg *configenv.ConfigEnv) (*IngressWatcher, error) {

	// debug
	logger.Debug("NewIngressWatcher")

	cfg, err := config.GetConfig()
	if err != nil {
		logger.Fatal("unable to get config %v", err)
		return nil, err
	}

	// Create a new scheme for decoding into.
	scheme := runtime.NewScheme()
	// assuming `v1` package has `AddToScheme` function
	if err := v1.AddToScheme(scheme); err != nil {
		logger.Fatalf("unable to add v1 scheme %v", err)
		return nil, err
	}

	// Add client-go's scheme for core Kubernetes types
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		logger.Fatalf("unable to add client-go scheme %v", err)
		return nil, err
	}

	// Create a new client to Kubernetes API.
	cl, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		logger.Fatalf("unable to create client %v", err)
		return nil, err
	}

	return &IngressWatcher{
		Client:     &RealKubernetesClient{clientKube},
		ClientObj:  cl,
		auditMutex: utils.NewNamedMutex(),
		Config:     ecfg,
	}, nil
}

// auditIngressResources audits all Ingress with the label "nimble.opti.adapter/enabled:true".
func (iw *IngressWatcher) AuditIngressResources(ctx context.Context) error {
	// debug
	logger.Debug("AuditIngressResources")

	// Fetch all Ingress resources
	ingresses := &networkingv1.IngressList{}

	// Fetch all Ingress resources using the standard Kubernetes client
	// ingresses, err := iw.Client.NetworkingV1().Ingresses("").List(ctx, metav1.ListOptions{})
	err := iw.ClientObj.List(ctx, ingresses, &client.ListOptions{})
	if err != nil {
		logger.Errorf("Failed to list ingresses: %v", err)
		return err
	}

	// Iterate through all Ingress resources
	for _, ing := range ingresses.Items {
		// check if the ingress is labeled with the label "nimble.opti.adapter/enabled:true"
		if isContainsAcmeChallenge(ctx, &ing) {
			// start certificate renewal
			_, err := iw.startCertificateRenewalAudit(ctx, &ing)
			if err != nil {
				logger.Errorf("Failed to start certificate renewal: %v", err)
				return err
			}
		} else if iw.Config.AdminUserPermission {
			// Calculate the time remaining for renewal
			timeRemaining, secretName, err := iw.timeRemainingCertificateUpToRenewal(ctx, &ing)
			if err != nil {
				logger.Errorf("Failed to check if the certificate is up to renewal: %v", err)
				return err
			}

			// Check if the certificate is up to renewal
			if timeRemaining <= time.Duration(iw.Config.CertificateRenewalThreshold*24)*time.Hour {
				// Change secret name in ing.Spec.TLS for make cert-manager create new certificate secret.

				// delete connected ingress secret
				if err := iw.deleteIngressSecret(ctx, secretName, ing.Namespace); err != nil {
					logger.Errorf("Failed to delete ingress secret: %v", err)
					return err
				}

				// start certificate renewal
				_, err := iw.startCertificateRenewalAudit(ctx, &ing)
				if err != nil {
					logger.Errorf("Failed to start certificate renewal: %v", err)
					return err
				}
			}
		}

	}
	return nil
}

// startCertificateRenewal get ingress that has "".well-known/acme-challenge" and resolve it.
func (iw *IngressWatcher) startCertificateRenewalAudit(ctx context.Context, ing *networkingv1.Ingress) (bool, error) {
	// debug
	logger.Debug("startCertificateRenewal")

	var isRenew = false

	// Remove the annotation.
	if err := iw.removeHTTPSAnnotation(ctx, ing); err != nil {
		// logger.Errorf("Failed to remove HTTPS annotation: %v", err)
		return false, err
	}

	// Wait for the absence of the ACME challenge path or for the timeout.
	timeout := time.Duration(iw.Config.AnnotationRemovalDelay) * time.Second
	successTime, err := iw.waitForChallengeAbsence(ctx, timeout, ing.Namespace, ing.Name)
	if err != nil {
		logger.Errorf("Failed to wait for the absence of ACME challenge path: %v", err)
		return false, err
	}
	if successTime > timeout {
		logger.Warn("Failed to confirm the absence of ACME challenge path before timeout.")
	} else {
		isRenew = true
	}

	// Reinstate the annotation.
	if err := iw.addHTTPSAnnotation(ctx, ing); err != nil {
		logger.Errorf("Failed to add HTTPS annotation: %v", err)
		return isRenew, err
	}

	return isRenew, nil
}

// delete connected ingress secret
func (iw *IngressWatcher) deleteIngressSecret(ctx context.Context, secretName string, secretNamespace string) error {
	// debug
	logger.Debug("deleteIngressSecret")

	// Create a Secret object with only Name and Namespace populated.
	deleteSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: secretNamespace,
		},
	}

	// Delete the secret.
	if err := iw.ClientObj.Delete(ctx, deleteSecret); err != nil {
		klog.Errorf("Failed to remove secret: %v", err)
		return err
	}

	return nil
}

// changeIngressSecretName change the secret name in ing.Spec.TLS to make cert-manager create new certificate secret.
func (iw *IngressWatcher) changeIngressSecretName(ctx context.Context, ing *networkingv1.Ingress, secretName string) error {
	logger.Debug("changeIngressSecretName")

	// Iterate over spec.tls[] to fetch associated secrets
	for _, tlsSpec := range ing.Spec.TLS {
		if tlsSpec.SecretName == secretName {
			// check if the name has "-vX" suffix for example (-v1), if not - add it. if it have - change it to "-vX+1".
			newSecretName, err := utils.ChangeSecretName(secretName)
			if err != nil {
				logger.Errorf("Failed to change secret name: %v", err)
				return err
			}

			// Change secret name the it name + "-v(X+1))"
			tlsSpec.SecretName = newSecretName
			break
		}
	}

	// for lock the specific ingress
	key := utils.IngressKey(ing)

	// lock the specific ingress
	if isLock := iw.auditMutex.TryLock(key); isLock {
		defer iw.auditMutex.Unlock(key)
		logger.Debug("changeIngressSecretName - key is locked")

		// update the ingress
		if err := iw.ClientObj.Update(ctx, ing); err != nil {
			logger.Error("Unable to change ingress secret name: ", err)
			return err
		}
	} else {
		errMassage := "key " + key + " is locked, and it should be unlocked"
		logger.Errorf(errMassage)
		return errors.New(errMassage)
	}

	return nil
}
