package ingresswatcher

import (
	"context"
	"errors"
	"net/http"
	"time"

	v1 "github.com/uri-tech/nimble-opti-adapter/api/v1"
	"github.com/uri-tech/nimble-opti-adapter/cronjob/configenv"
	"github.com/uri-tech/nimble-opti-adapter/cronjob/loggerpkg"
	"github.com/uri-tech/nimble-opti-adapter/utils"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type IngressWatcher struct {
	Client     KubernetesClient
	ClientObj  client.Client
	auditMutex *utils.NamedMutex
	Config     *configenv.ConfigEnv
}

// logger is the logger for the ingresswatcher package.
var logger = loggerpkg.GetNamedLogger("ingresswatcher").WithOptions()

// func NewIngressWatcher(clientKube *kubernetes.Clientset, ecfg *configenv.ConfigEnv) (*IngressWatcher, error) {
func NewIngressWatcher(clientKube kubernetes.Interface, ecfg *configenv.ConfigEnv) (*IngressWatcher, error) {
	logger.Debugf("starting NewIngressWatcher, ecfg: %v", ecfg)

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

	// Create a new dynamic RESTMapper. The RESTMapper is responsible for converting
	// group, version, and kind information to and from resource paths and scopes.
	// A dynamic RESTMapper can update itself automatically when new API types are registered.
	// 'cfg' provides the connection configuration to the Kubernetes cluster, and
	// 'http.DefaultClient' is the default HTTP client used for making API requests.
	mapper, err := apiutil.NewDynamicRESTMapper(cfg, http.DefaultClient)
	if err != nil {
		logger.Fatalf("unable to create RESTMapper: %v", err)
		return nil, err
	}

	// Create a new client to Kubernetes API.
	cl, err := client.New(cfg, client.Options{
		Cache:  nil,
		Scheme: scheme,
		Mapper: mapper,
	})
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
	logger.Debug("starting AuditIngressResources")

	// initialize the IngressForRenewal struct
	countIngressForRenewal := 0
	// count the ingress that was successfully renewed
	countIngressRenewed := 0

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
			countIngressForRenewal++
			logger.Infof("Found ingress with ACME challenge path, ingress name: %v", ing.Name)
			// start certificate renewal
			isRenew, err := iw.startCertificateRenewalAudit(ctx, &ing)
			if err != nil {
				logger.Errorf("Failed to start certificate renewal: %v", err)
				return err
			}
			if !isRenew {
				oldSecret := ing.Spec.TLS[0].SecretName
				logger.Infof("Certificate was not renewed, trying now change secret %s for renewal.", oldSecret)

				// change the connected ingress secret in ing.Spec.TLS for making cert-manager create new certificate secret.
				if err := iw.changeIngressSecretName(ctx, &ing, oldSecret); err != nil {
					logger.Errorf("Failed to change ingress secret name: %v", err)
					return err
				}
				isSecondTeyRenew, err := iw.startCertificateRenewalAudit(ctx, &ing)
				if err != nil {
					logger.Errorf("Failed to start certificate renewal: %v", err)
					return err
				}
				if !isSecondTeyRenew {
					logger.Infof("Certificate was not renewed also when it secret name was change, old secret: %s, new secret: %s.", oldSecret, ing.Spec.TLS[0].SecretName)
				} else {
					countIngressRenewed++
					logger.Infof("Certificate was renewed when it secret name was change, ingress name: %v", ing.Name)
				}
			} else {
				countIngressRenewed++
				logger.Infof("Certificate was renewed, ingress name: %v", ing.Name)
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

				// delete connected ingress secret
				if err := iw.deleteIngressSecret(ctx, secretName, ing.Namespace); err != nil {
					logger.Errorf("Failed to delete ingress secret: %v", err)
					return err
				}

				// sleep for 5 seconds for make sure the secret was deleted
				time.Sleep(5 * time.Second)

				// start certificate renewal
				isRenew, err := iw.startCertificateRenewalAudit(ctx, &ing)
				if err != nil {
					logger.Errorf("Failed to start certificate renewal: %v", err)
					return err
				}
				if isRenew {
					logger.Infof("Certificate was renewed, ingress name: %v", ing.Name)
				}
			} else {
				logger.Infof("Certificate is not up to renewal, time remaining: %v", timeRemaining)
			}
		}
	}
	logger.Infof("Finished auditing %d Ingress resources. There was %d ingress needed renewal", len(ingresses.Items), countIngressForRenewal)
	logger.Infof("There was %d ingress successfully renewed", countIngressRenewed)

	return nil
}

// startCertificateRenewal get ingress that has "".well-known/acme-challenge" and resolve it. if the resolve was successful - return true, else - return false.
func (iw *IngressWatcher) startCertificateRenewalAudit(ctx context.Context, ing *networkingv1.Ingress) (bool, error) {
	logger.Debugf("starting startCertificateRenewal, ingress: %v", ing.Name)

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

// changeIngressSecretName change the secret name in ing.Spec.TLS to make cert-manager create new certificate secret.
func (iw *IngressWatcher) changeIngressSecretName(ctx context.Context, ing *networkingv1.Ingress, secretName string) error {
	logger.Debugf("starting changeIngressSecretName, ingress: %v", ing.Name)

	// Iterate over spec.tls[] to fetch associated secrets
	idxName := 0
	for i := range ing.Spec.TLS {
		if ing.Spec.TLS[i].SecretName == secretName {
			// check if the name has "-vX" suffix for example (-v1), if not - add it. if it have - change it to "-vX+1".
			newSecretName, err := utils.ChangeSecretName(secretName)
			if err != nil {
				logger.Errorf("Failed to change secret name: %v", err)
				return err
			}

			// Change secret name the it name + "-v(X+1))"
			ing.Spec.TLS[i].SecretName = newSecretName
			idxName = i
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

		logger.Infof("Change ingress secret name to %s", ing.Spec.TLS[idxName].SecretName)
	} else {
		errMassage := "key " + key + " is locked, and it should be unlocked"
		logger.Errorf(errMassage)
		return errors.New(errMassage)
	}

	return nil
}

// delete connected ingress secret
func (iw *IngressWatcher) deleteIngressSecret(ctx context.Context, secretName string, secretNamespace string) error {
	logger.Debugf("starting deleteIngressSecret, secretName: %v", secretName)

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
	logger.Infof("Secret %s was deleted", secretName)

	return nil
}
