package ingresswatcher

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"testing"
	"time"

	// v1 "github.com/uri-tech/nimble-opti-adapter/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/uri-tech/nimble-opti-adapter/cronjob/configenv"
	"github.com/uri-tech/nimble-opti-adapter/utils"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakec "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// func newIngressWatcherForTesting(clientKube *kubernetes.Clientset, ecfg *configenv.ConfigEnv) (*IngressWatcher, error) {
func newIngressWatcherForTesting(cfakeClientset *fake.Clientset, ecfg *configenv.ConfigEnv) (*IngressWatcher, error) {
	cl, err := client.New(&rest.Config{}, client.Options{
		Cache: nil,
	})
	if err != nil {
		logger.Fatalf("unable to create client %v", err)
		return nil, err
	}
	return &IngressWatcher{
		Client:     &RealKubernetesClient{cfakeClientset},
		ClientObj:  cl,
		auditMutex: utils.NewNamedMutex(),
		Config:     ecfg,
	}, nil
}

// createIngressRules return a list of Ingress rules for the provided paths.
func createIngressRules(paths []string) []networkingv1.IngressRule {
	rulesIn := []networkingv1.IngressRule{}

	for _, path := range paths {
		rule := networkingv1.IngressRule{
			IngressRuleValue: networkingv1.IngressRuleValue{
				HTTP: &networkingv1.HTTPIngressRuleValue{
					Paths: []networkingv1.HTTPIngressPath{
						{
							Path: path,
						},
					},
				},
			},
		}
		rulesIn = append(rulesIn, rule)
	}

	return rulesIn
}

// generateIngressRules creates a list of Ingress rules for the provided paths.
func generateIngressRules(paths []string) []networkingv1.IngressRule {
	rulesIn := []networkingv1.IngressRule{}

	for _, path := range paths {
		rule := networkingv1.IngressRule{
			IngressRuleValue: networkingv1.IngressRuleValue{
				HTTP: &networkingv1.HTTPIngressRuleValue{
					Paths: []networkingv1.HTTPIngressPath{
						{
							Path: path,
						},
					},
				},
			},
		}
		rulesIn = append(rulesIn, rule)
	}

	return rulesIn
}

// generateIngress creates an Ingress object with the given name, namespace, labels, paths, and annotations.
func generateIngress(name, namespace string, labels map[string]string, paths []string, annotations map[string]string) *networkingv1.Ingress {
	ingressRules := generateIngressRules(paths)

	return &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: networkingv1.IngressSpec{
			Rules: ingressRules,
		},
	}
}

// generateTestCert creates a self-signed X.509 certificate with the provided expiration time.
func generateTestCert(expiration time.Time) ([]byte, error) {
	// Generate an ECDSA private key.
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %v", err)
	}

	// Define the template for the certificate.
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Org"},
			Country:      []string{"US"},
			Province:     []string{"California"},
			Locality:     []string{"San Francisco"},
		},
		NotBefore:             time.Now(),
		NotAfter:              expiration,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
		DNSNames:              []string{"localhost"},
	}

	// Create the self-signed certificate.
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %v", err)
	}

	return certDER, nil
}

// setupIngressWatcher initializes a mock IngressWatcher for testing purposes.
func setupIngressWatcher(client client.Client) (*IngressWatcher, error) {
	fakeClientset := fake.NewSimpleClientset()

	// Load environment variables configuration.
	ecfg := &configenv.ConfigEnv{
		RunMode:                     "dev",
		CertificateRenewalThreshold: 60,
		AnnotationRemovalDelay:      10,
		AdminUserPermission:         false,
		LogOutput:                   "console",
	}

	iw, err := newIngressWatcherForTesting(fakeClientset, ecfg)
	if err != nil {
		panic(fmt.Sprintf("Failed to create IngressWatcher: %v", err))
	}

	iw.ClientObj = client
	iw.Config = ecfg
	iw.auditMutex = utils.NewNamedMutex()

	return iw, nil
}

// setupIngressWatcher initializes a mock IngressWatcher for testing purposes.
func setupIngressWatcherMock(clientObj client.Client, client *FakeKubernetesClient) (*IngressWatcher, error) {
	fakeClientset := fake.NewSimpleClientset()

	// Load environment variables configuration.
	ecfg := &configenv.ConfigEnv{
		RunMode:                     "dev",
		CertificateRenewalThreshold: 60,
		AnnotationRemovalDelay:      10,
		AdminUserPermission:         false,
		LogOutput:                   "console",
	}

	// Create a new IngressWatcher.
	iw, err := newIngressWatcherForTesting(fakeClientset, ecfg)
	if err != nil {
		panic(fmt.Errorf("Failed to create IngressWatcher: %v", err))
	}
	// Set the client object.
	iw.ClientObj = clientObj
	// Set the audit mutex.
	iw.auditMutex = utils.NewNamedMutex()
	// Set the fake Kubernetes client.
	iw.Client = client

	return iw, nil
}

func TestNewIngressWatcher(t *testing.T) {
	fakeClientset := fake.NewSimpleClientset()

	// Load environment variables configuration.
	ecfg := &configenv.ConfigEnv{
		RunMode:                     "dev",
		CertificateRenewalThreshold: 60,
		AnnotationRemovalDelay:      10,
		AdminUserPermission:         false,
		LogOutput:                   "console",
	}

	t.Run("successfully initialize an IngressWatcher", func(t *testing.T) {
		// Mock the stopCh
		stopCh := make(chan struct{})
		defer close(stopCh)

		// TODO: Mock config.GetConfig() to return a dummy configuration

		iw, err := newIngressWatcherForTesting(fakeClientset, ecfg)
		assert.NoError(t, err)
		assert.NotNil(t, iw)

		// TODO: Check other attributes of iw to ensure they are correctly set up
	})

	// TODO: Add more test cases for negative scenarios like failing to get config, failing to set up the scheme, etc.
}

func TestAuditIngressResources(t *testing.T) {
	// Create a fake client with some Ingress resources.
	ctx := context.TODO()

	// 1. Setup fake client and resources
	fakeClient := fakec.NewClientBuilder().WithScheme(scheme.Scheme).Build()

	ingressWithLabel := generateIngress("ingress-with-label", "default", map[string]string{"nimble.opti.adapter/enabled": "true"}, nil, nil)
	ingressWithoutLabel := generateIngress("ingress-without-label", "default", nil, nil, nil)

	// Create the Ingress resources.
	err := fakeClient.Create(ctx, ingressWithLabel)
	if err != nil {
		t.Fatalf("Failed to create ingress with label: %v", err)
	}

	// Create the Ingress resources.
	err = fakeClient.Create(ctx, ingressWithoutLabel)
	if err != nil {
		t.Fatalf("Failed to create ingress without label: %v", err)
	}

	// Create the IngressWatcher.
	iw, err := setupIngressWatcher(fakeClient)
	if err != nil {
		t.Fatalf("Failed to setup IngressWatcher: %v", err)
	}

	// 2. Call the audit function
	iw.auditMutex.Unlock("default")
	err = iw.AuditIngressResources(ctx)
	if err != nil {
		t.Fatalf("Failed to audit ingress resources: %v", err)
	}
}

func TestStartCertificateRenewalAudit(t *testing.T) {
	ctx := context.TODO()

	tests := []struct {
		name         string
		initialPaths []string
		isRenewed    bool
	}{
		{
			name:         "Successful certificate renewal",
			initialPaths: []string{"/app", "/.well-known/acme-challenge"},
			isRenewed:    true,
		},
		{
			name:         "unsuccessful certificate renewal",
			initialPaths: []string{"/app", "/.well-known/acme-challenge"},
			isRenewed:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fakec.NewClientBuilder().WithScheme(scheme.Scheme).Build()

			// Create the initial Ingress object.
			ing := generateIngress("test-ingress", "default", nil, tt.initialPaths, nil)
			if err := fakeClient.Create(ctx, ing); err != nil {
				t.Fatalf("Failed to create initial ingress: %v", err)
			}

			// Create the IngressWatcher object.
			iw, err := setupIngressWatcher(fakeClient)
			if err != nil {
				t.Fatal(err)
			}
			iw.Config.AnnotationRemovalDelay = 5

			// Test
			gotRenewalCh := make(chan bool)
			errorCh := make(chan error)
			go func() {
				renewal, err := iw.startCertificateRenewalAudit(ctx, ing)
				if err != nil {
					errorCh <- err
					return
				}
				gotRenewalCh <- renewal
			}()

			if tt.isRenewed {
				// debug
				t.Log("Updating ingress to trigger renewal")

				time.Sleep(2 * time.Second)
				// Update the ingress with the final paths.
				ing.Spec.Rules = createIngressRules([]string{"/app"})
				if err := fakeClient.Update(context.TODO(), ing); err != nil {
					t.Fatalf("Failed to update ingress: %v", err)
				}
			}

			// Wait for a response or timeout after a few seconds.
			select {
			case gotRenewal := <-gotRenewalCh:
				assert.Equal(t, tt.isRenewed, gotRenewal)
			case err := <-errorCh:
				t.Fatalf("Received error: %v", err)
			case <-time.After(60 * time.Second): // Adjust as needed
				t.Fatal("Timeout while waiting for processIngressForRenewal response")
			}

			// Delete the Ingress object.
			if err := fakeClient.Delete(context.Background(), ing); err != nil {
				t.Fatalf("Failed to delete Ingress: %v", err)
			}
		})
	}
}

func TestChangeIngressSecretName(t *testing.T) {
	ctx := context.TODO()

	tests := []struct {
		name           string
		initialSecret  string
		changeToSecret string
		expectSecret   string
		shouldError    bool
	}{
		{
			name:           "Ingress has the secret and it's updated correctly",
			initialSecret:  "my-secret",
			changeToSecret: "my-secret",
			expectSecret:   "my-secret-v1", // Assuming that "ChangeSecretName" appends "-v1" for the first version.
			shouldError:    false,
		},
		// ... Add other test cases as needed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			ing := generateIngress("test-ingress", "default", nil, []string{"/"}, nil)
			ing.Spec.TLS = []networkingv1.IngressTLS{
				{
					SecretName: tt.initialSecret,
				},
			}
			fakeClient := fakec.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(ing).Build()

			iw, err := setupIngressWatcher(fakeClient)
			if err != nil {
				t.Fatal(err)
			}

			err = iw.changeIngressSecretName(ctx, ing, tt.changeToSecret)

			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Fetch the updated ingress and check the secret name
				updatedIngress := &networkingv1.Ingress{}
				err = fakeClient.Get(ctx, client.ObjectKey{Name: "test-ingress", Namespace: "default"}, updatedIngress)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectSecret, updatedIngress.Spec.TLS[0].SecretName)
			}
		})
	}
}

func TestDeleteIngressSecret(t *testing.T) {
	ctx := context.TODO()

	tests := []struct {
		name              string
		initialSecretName string
		deleteSecretName  string
		secretNamespace   string
		shouldError       bool
	}{
		{
			name:              "Secret is present and gets deleted successfully",
			initialSecretName: "existing-secret",
			deleteSecretName:  "existing-secret",
			secretNamespace:   "default",
			shouldError:       false,
		},
		{
			name:              "Secret isn't present and deletion returns an error",
			initialSecretName: "",
			deleteSecretName:  "non-existent-secret",
			secretNamespace:   "default",
			shouldError:       true,
		},
		// ... Add other test cases as needed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			var initialSecrets []client.Object
			if tt.initialSecretName != "" {
				initialSecrets = append(initialSecrets, &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      tt.initialSecretName,
						Namespace: tt.secretNamespace,
					},
				})
			}

			fakeClient := fakec.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(initialSecrets...).Build()
			iw := &IngressWatcher{ClientObj: fakeClient}

			err := iw.deleteIngressSecret(ctx, tt.deleteSecretName, tt.secretNamespace)

			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Check if the secret was actually deleted
				secret := &corev1.Secret{}
				err = fakeClient.Get(ctx, client.ObjectKey{Name: tt.deleteSecretName, Namespace: tt.secretNamespace}, secret)
				assert.True(t, client.IgnoreNotFound(err) == nil && err != nil, "Secret should be deleted but is still present")
			}
		})
	}
}
