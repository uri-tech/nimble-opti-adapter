// internal/controller/ingress_watcher_test.go
package controller

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	v1 "github.com/uri-tech/nimble-opti-adapter/api/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakec "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// FakeKubernetesClient is a structure that holds the fake Client for Kubernetes.
type FakeKubernetesClient struct {
	mock.Mock
}

// Ensure FakeKubernetesClient implements KubernetesClient.
var _ KubernetesClient = &FakeKubernetesClient{}

// represents a stream of events that the watcher observes
type FakeWatcher struct {
	resultCh chan watch.Event
}

// closes the resultCh channel
func (f *FakeWatcher) Stop() {
	close(f.resultCh)
}

// eturns the resultCh channel for reading.
// ensuring that outside users of FakeWatcher can only read events from the channel and cannot accidentally send events into it.
func (f *FakeWatcher) ResultChan() <-chan watch.Event {
	return f.resultCh
}

func (m *FakeKubernetesClient) Watch(ctx context.Context, namespace, ingressName string) (watch.Interface, error) {
	args := m.Called(ctx, namespace, ingressName)
	return args.Get(0).(watch.Interface), args.Error(1)
}

// setupIngressWatcher initializes a mock IngressWatcher for testing purposes.
func setupIngressWatcherMock(clientObj client.Client, client *FakeKubernetesClient) (*IngressWatcher, error) {
	fakeClientset := fake.NewSimpleClientset()
	stopCh := make(chan struct{})

	// Add NimbleOpti to the scheme.
	err := v1.AddToScheme(scheme.Scheme)
	if err != nil {
		panic(fmt.Sprintf("Failed to add NimbleOpti to scheme: %v", err))
	}

	// Create a new IngressWatcher.
	iw, err := NewIngressWatcher(fakeClientset, stopCh)
	if err != nil {
		klog.ErrorS(err, "Failed to create IngressWatcher")
		return nil, err
	}
	// Set the client object.
	iw.ClientObj = clientObj
	// Set the audit mutex.
	iw.auditMutex = NewNamedMutex()
	// Set the fake Kubernetes client.
	iw.Client = client

	return iw, nil
}

// setupIngressWatcher initializes a mock IngressWatcher for testing purposes.
func setupIngressWatcher(client client.Client) (*IngressWatcher, error) {
	fakeClientset := fake.NewSimpleClientset()
	stopCh := make(chan struct{})

	// Add NimbleOpti to the scheme.
	err := v1.AddToScheme(scheme.Scheme)
	if err != nil {
		panic(fmt.Sprintf("Failed to add NimbleOpti to scheme: %v", err))
	}

	iw, err := NewIngressWatcher(fakeClientset, stopCh)
	if err != nil {
		klog.ErrorS(err, "Failed to create IngressWatcher")
		return nil, err
	}
	iw.ClientObj = client
	iw.auditMutex = NewNamedMutex()

	return iw, nil
}

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

// generateIngress creates an Ingress object with the given name, namespace, and labels.
func generateIngress(name, namespace string, labels map[string]string, paths []string, annotations map[string]string) *networkingv1.Ingress {
	ingressRules := createIngressRules(paths)

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

// generateTestCert creates a self-signed X.509 certificate for testing purposes.
// The certificate is generated with the provided expiration time.
//
// Parameters:
//   - expiration: The expiration time of the certificate.
//
// Returns:
//   - []byte: The DER-encoded (binary format) certificate.
//   - error: An error object if there's an issue during certificate generation.
func generateTestCert(expiration time.Time) ([]byte, error) {
	// Generate a new ECDSA private key.
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

// Section: 2

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
	err = iw.auditIngressResources(ctx)
	if err != nil {
		t.Fatalf("Failed to audit ingress resources: %v", err)
	}
	// assert.Nil(t, err)
}

func TestHandleIngressAdd(t *testing.T) {
	fakeClient := fakec.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	iw, err := setupIngressWatcher(fakeClient)
	if err != nil {
		t.Fatalf("Failed to setup IngressWatcher: %v", err)
	}

	// Test: Add an ingress with label, annotation and ".well-known/acme-challenge" in it path.
	labels := map[string]string{
		"nimble.opti.adapter/enabled": "true",
	}
	httpsAnnotation := map[string]string{
		"nginx.ingress.kubernetes.io/backend-protocol": "HTTPS",
	}
	paths := []string{
		"/app",
		"/.well-known/acme-challenge",
	}
	ingWithLabelAndAnnotation := generateIngress(
		"default",
		"default",
		labels,
		paths,
		httpsAnnotation,
	)
	// Create the Ingress object using the fake client.
	if err := fakeClient.Create(context.TODO(), ingWithLabelAndAnnotation); err != nil {
		t.Fatalf("Failed to create Ingress: %v", err)
	}

	// Call the handleIngressAdd function.
	iw.handleIngressAdd(ingWithLabelAndAnnotation)

	// check if the nimbleopti object was created
	nimbleOpti := &v1.NimbleOpti{}
	err = iw.ClientObj.Get(context.TODO(), client.ObjectKey{Name: "default", Namespace: "default"}, nimbleOpti)
	assert.NoError(t, err)
	assert.NotNil(t, nimbleOpti)
}

// Section: 3

func TestGetOrCreateNimbleOpti(t *testing.T) {
	fakeClient := fakec.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	iw, err := setupIngressWatcher(fakeClient)
	if err != nil {
		t.Fatalf("Failed to setup IngressWatcher: %v", err)
	}

	// Scenario: NimbleOpti doesn't exist.
	// Try to get or create a NimbleOpti in the "default" namespace.
	nimbleOpti, err := iw.getOrCreateNimbleOpti(context.TODO(), "default")
	assert.NoError(t, err)
	assert.NotNil(t, nimbleOpti)

	// Use the fakeClient to retrieve the NimbleOpti to confirm it was created.
	retrievedNimbleOpti := &v1.NimbleOpti{}
	err = iw.ClientObj.Get(context.TODO(), client.ObjectKey{Name: "default", Namespace: "default"}, retrievedNimbleOpti)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedNimbleOpti)

	// Scenario: NimbleOpti exists.
	// Try to get or create again a NimbleOpti in the "default" namespace.
	secondNimbleOpti, err := iw.getOrCreateNimbleOpti(context.TODO(), "default")
	assert.NoError(t, err)
	assert.NotNil(t, secondNimbleOpti)

	// The returned NimbleOpti should have the same UID as the first one, which means it wasn't recreated.
	assert.Equal(t, nimbleOpti.GetUID(), secondNimbleOpti.GetUID())
}

func TestIsAdapterEnabledLabel(t *testing.T) {
	// Test if the function returns true when the label is present and set to "true".
	ingWithLabel := generateIngress("test-ingress-with-label", "default", map[string]string{"nimble.opti.adapter/enabled": "true"}, nil, nil)
	assert.True(t, isAdapterEnabledLabel(context.TODO(), ingWithLabel))

	// Test if the function returns false when the label is not present.
	ingWithoutLabel := generateIngress("test-ingress", "default", nil, nil, nil)
	assert.False(t, isAdapterEnabledLabel(context.TODO(), ingWithoutLabel))

	// Test if the function returns false when the label is present but not set to "true".
	ingWithFalseLabel := generateIngress("test-ingress-with-false-label", "default", map[string]string{"nimble.opti.adapter/enabled": "false"}, nil, nil)
	assert.False(t, isAdapterEnabledLabel(context.TODO(), ingWithFalseLabel))
}

func TestHasIngressChanged(t *testing.T) {
	oldIng := generateIngress("old-ingress", "default", nil, nil, nil)
	newIng := generateIngress("new-ingress", "default", map[string]string{"nimble.opti.adapter/enabled": "true"}, nil, nil)

	// Test for changes in spec.
	newIng.Spec.Rules = append(newIng.Spec.Rules, networkingv1.IngressRule{Host: "new-host"})
	assert.True(t, hasIngressChanged(context.TODO(), oldIng, newIng))

	// Test for changes in the important labels.
	if oldIng.Labels == nil {
		oldIng.Labels = make(map[string]string)
	}
	oldIng.Labels["nimble.opti.adapter/enabled"] = "false"
	assert.True(t, hasIngressChanged(context.TODO(), oldIng, newIng))

	// Test for changes in annotations - do not need for now.
	if oldIng.Annotations == nil {
		oldIng.Annotations = make(map[string]string)
	}
	newIng.Annotations = map[string]string{"new-annotation": "value"}
	assert.True(t, hasIngressChanged(context.TODO(), oldIng, newIng))

	// Test for no changes.
	assert.False(t, hasIngressChanged(context.TODO(), oldIng, oldIng))
}

const httpsAnnotation = "nginx.ingress.kubernetes.io/backend-protocol"

func TestRemoveHTTPSAnnotation(t *testing.T) {
	fakeClient := fakec.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	iw, err := setupIngressWatcher(fakeClient)
	if err != nil {
		t.Fatalf("Failed to setup IngressWatcher: %v", err)
	}

	ctx := context.TODO()

	ing := generateIngress("test-ingress", "default", nil, nil, nil)
	ing.Annotations = map[string]string{httpsAnnotation: "HTTPS"}

	// First, create the Ingress object using the fake client.
	if err := fakeClient.Create(ctx, ing); err != nil {
		t.Fatalf("Failed to create Ingress: %v", err)
	}

	// Then, remove the HTTPS annotation.
	if err := iw.removeHTTPSAnnotation(ctx, ing); err != nil {
		t.Fatalf("Failed to remove HTTPS annotation: %v", err)
	}
	_, exists := ing.Annotations[httpsAnnotation]
	assert.False(t, exists, "Expected HTTPS annotation to be removed")
}

func TestAddHTTPSAnnotation(t *testing.T) {
	fakeClient := fakec.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	iw, err := setupIngressWatcher(fakeClient)
	if err != nil {
		t.Fatalf("Failed to setup IngressWatcher: %v", err)
	}

	ctx := context.TODO()

	ing := generateIngress("test-ingress", "default", nil, nil, nil)

	// Create the Ingress object using the fake client.
	if err := fakeClient.Create(ctx, ing); err != nil {
		t.Fatalf("Failed to create Ingress: %v", err)
	}

	// Then, add the HTTPS annotation.
	if err := iw.addHTTPSAnnotation(ctx, ing); err != nil {
		t.Fatalf("Failed to add HTTPS annotation: %v", err)
	}
	// _, exists := ing.Annotations[httpsAnnotation]
	// assert.True(t, exists, "Expected HTTPS annotation to be added")
	val, exists := ing.Annotations[httpsAnnotation]
	assert.True(t, exists && val == "HTTPS", "Expected HTTPS annotation to be added")
}

func TestProcessIngressForRenewal(t *testing.T) {
	ctx := context.TODO()

	tests := []struct {
		name          string
		ingressLabels map[string]string
		ingressPaths  []string
		wantRenewal   bool
	}{
		{
			name: "Ingress with ACME challenge path should trigger renewal",
			ingressLabels: map[string]string{
				"nimble.opti.adapter/enabled":                  "true",
				"nginx.ingress.kubernetes.io/backend-protocol": "HTTPS",
			},
			ingressPaths: []string{
				"/app",
				"/.well-known/acme-challenge",
			},
			wantRenewal: true,
		},
		{
			name: "Ingress without ACME challenge path should not trigger renewal",
			ingressLabels: map[string]string{
				"nimble.opti.adapter/enabled":                  "true",
				"nginx.ingress.kubernetes.io/backend-protocol": "HTTPS",
			},
			ingressPaths: []string{
				"/app",
			},
			wantRenewal: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			fakeClient := fakec.NewClientBuilder().WithScheme(scheme.Scheme).Build()
			iw, err := setupIngressWatcher(fakeClient)
			assert.Nil(t, err)

			// Create the initial ingress.
			ing := generateIngress("test-ingress", "default", tt.ingressLabels, tt.ingressPaths, nil)
			assert.Nil(t, fakeClient.Create(ctx, ing))

			// Test
			gotRenewalCh := make(chan bool)
			errorCh := make(chan error)
			go func() {
				renewal, err := iw.processIngressForRenewal(ctx, ing)
				if err != nil {
					errorCh <- err
					return
				}
				gotRenewalCh <- renewal
			}()

			if tt.wantRenewal {
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
				assert.Equal(t, tt.wantRenewal, gotRenewal)
			case err := <-errorCh:
				t.Fatalf("Received error: %v", err)
			case <-time.After(20 * time.Second): // Adjust as needed
				t.Fatal("Timeout while waiting for processIngressForRenewal response")
			}

			// Delete the Ingress object.
			if err := fakeClient.Delete(context.Background(), ing); err != nil {
				t.Fatalf("Failed to delete Ingress: %v", err)
			}
		})
	}
}

// TestWaitForChallengeAbsence tests the waitForChallengeAbsence function.
//  1. We use two test cases: one where the ACME challenge path is initially present and then removed,
//     and another where it's always present.
//  2. We first create an Ingress with the initial paths.
//  3. Then we run the waitForChallengeAbsence function in a goroutine.
//  4. After a short delay, we update the Ingress with the final paths.
//  5. Finally, we check whether the ticker is still running or not, based on our expectations for each test case.
func TestWaitForChallengeAbsence(t *testing.T) {
	ctx := context.TODO()

	tests := []struct {
		name              string
		initialPaths      []string
		finalPaths        []string
		expectPathAbsence bool // true means we expect the path to be absent at the end of the test
	}{
		{
			name:              "Path removed within the timeout duration",
			initialPaths:      []string{"/app", "/.well-known/acme-challenge"},
			finalPaths:        []string{"/app"},
			expectPathAbsence: true,
		},
		{
			name:              "Path persists beyond the timeout duration",
			initialPaths:      []string{"/app", "/.well-known/acme-challenge"},
			finalPaths:        []string{"/app", "/.well-known/acme-challenge"},
			expectPathAbsence: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			fakeClient := fakec.NewClientBuilder().WithScheme(scheme.Scheme).Build()

			// Create the initial ingress.
			ing := generateIngress("test-ingress", "default", nil, tt.initialPaths, nil)
			if err := fakeClient.Create(ctx, ing); err != nil {
				t.Fatalf("Failed to create initial ingress: %v", err)
			}

			// Create the IngressWatcher.
			iw, err := setupIngressWatcher(fakeClient)
			if err != nil {
				t.Fatal(err)
			}

			// Test
			timeout := 5 * time.Second
			resultCh := make(chan time.Duration)
			errorCh := make(chan error)
			go func() {
				res, err := iw.waitForChallengeAbsence(ctx, timeout, "default", "test-ingress")
				if err != nil {
					errorCh <- err
					return
				}
				resultCh <- res
			}()

			time.Sleep(2 * time.Second)

			// Update the ingress with the final paths.
			ing.Spec.Rules = createIngressRules(tt.finalPaths)
			if err := fakeClient.Update(ctx, ing); err != nil {
				t.Fatalf("Failed to update ingress: %v", err)
			}

			// Assertions
			select {
			case err := <-errorCh:
				t.Fatalf("Error from waitForChallengeAbsence: %v", err)
			case res := <-resultCh:
				if tt.expectPathAbsence {
					assert.GreaterOrEqual(t, timeout, res)
				} else {
					assert.Less(t, timeout, res)
				}
				// assert.(t, tt.expectPathAbsence, res)
			case <-time.After(timeout + 1*time.Second):
				t.Fatal("Test timeout exceeded")
			}
		})
	}
}

func TestStartCertificateRenewal(t *testing.T) {
	ctx := context.TODO()

	tests := []struct {
		name         string
		initialPaths []string
		isRenewed    bool
	}{
		{
			name:         "Successful certificate renewal",
			initialPaths: []string{"/app"},
			isRenewed:    true,
		},
		{
			name:         "Failure at removing HTTPS annotation",
			initialPaths: []string{"/app", "/.well-known/acme-challenge"},
			isRenewed:    false,
		},
		// Add more test scenarios as needed.
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

			// Create the NimbleOpti object.
			nimbleOpti := &v1.NimbleOpti{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "default",
					Namespace: "default",
				},
				Spec: v1.NimbleOptiSpec{
					TargetNamespace:             "default",
					CertificateRenewalThreshold: 3,
					AnnotationRemovalDelay:      5,
				},
			}
			if err := fakeClient.Create(ctx, nimbleOpti); err != nil {
				t.Fatalf("Failed to create NimbleOpti: %v", err)
			}

			isRenew, err := iw.startCertificateRenewal(ctx, ing, nimbleOpti)
			if err != nil {
				t.Fatalf("startCertificateRenewal failed: %v", err)
			}
			assert.Equal(t, isRenew, tt.isRenewed)

		})
	}
}

func TestRenewValidCertificateIfNecessary(t *testing.T) {
	ctx := context.TODO()

	tests := []struct {
		name           string
		tlsSecrets     map[string][]byte // map of secret names to their 'tls.crt' content
		tlsSecretsTime time.Duration
		wantRenewal    bool // Whether we expect a certificate renewal to be initiated
		initialPaths   []string
		middlePaths    []string
		finalPaths     []string
	}{
		{
			name:           "Certificate is valid but about to expire according to threshold",
			tlsSecrets:     map[string][]byte{},
			tlsSecretsTime: 1,
			wantRenewal:    true,
			initialPaths:   []string{"/app"},
			middlePaths:    []string{"/app", "/.well-known/acme-challenge"},
			finalPaths:     []string{"/app"},
		},
		{
			name:           "Certificate is not about to expire",
			tlsSecrets:     map[string][]byte{},
			tlsSecretsTime: 100,
			wantRenewal:    false,
			initialPaths:   []string{"/app"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate the test certificate and handle errors
			certDER, err := generateTestCert(time.Now().Add(tt.tlsSecretsTime * time.Hour))
			if err != nil {
				t.Fatalf("Failed to generate test certificate: %v", err)
			}
			tt.tlsSecrets["test-secret"] = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

			// Create the mock client for ingress watching
			fakeK8sClient := &FakeKubernetesClient{}
			// watchChan := make(chan watch.Event)
			// fakeK8sClient.On("Watch", mock.Anything, mock.Anything, mock.Anything).Return(watchChan, nil)
			watchChan := make(chan watch.Event)
			fakeWatcher := &FakeWatcher{
				resultCh: watchChan,
			}
			fakeK8sClient.On("Watch", mock.Anything, mock.Anything, mock.Anything).Return(fakeWatcher, nil)

			// Create the mock client for k8s resources
			mockClient := fakec.NewClientBuilder().WithScheme(scheme.Scheme).Build()

			// Use fakeK8sClient for IngressWatcher
			iw, err := setupIngressWatcherMock(mockClient, fakeK8sClient)
			assert.NoError(t, err)

			// Create the Ingress with the TLS spec.
			ing := generateIngress("test-ingress", "default", nil, tt.initialPaths, nil)
			ing.Spec.TLS = []networkingv1.IngressTLS{
				{
					SecretName: "test-secret",
				},
			}
			// Create the Ingress.
			if err := mockClient.Create(ctx, ing); err != nil {
				t.Fatalf("Failed to create initial ingress: %v", err)
			}

			// Create the associated Secret objects.
			for secretName, certContent := range tt.tlsSecrets {
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName,
						Namespace: "default",
					},
					Data: map[string][]byte{
						"tls.crt": certContent,
					},
				}
				if err := mockClient.Create(ctx, secret); err != nil {
					t.Fatalf("Failed to create secret %s: %v", secretName, err)
				}
			}

			// Create the NimbleOpti object.
			nimbleOpti := &v1.NimbleOpti{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "default",
					Namespace: "default",
				},
				Spec: v1.NimbleOptiSpec{
					TargetNamespace:             "default",
					CertificateRenewalThreshold: 3, // Renew if certificate expires within 3 days
					AnnotationRemovalDelay:      5,
				},
			}
			if err := mockClient.Create(ctx, nimbleOpti); err != nil {
				t.Fatalf("Failed to create NimbleOpti: %v", err)
			}

			// Start the certificate renewal process.
			errorCh := make(chan error)
			go func() {
				err := iw.renewValidCertificateIfNecessary(ctx, ing) // Use iwMock here
				if err != nil {
					errorCh <- err
					return
				}
				errorCh <- nil
			}()

			if tt.wantRenewal {
				// debug
				t.Log("updating middlePaths")

				time.Sleep(2 * time.Second) // Give it some delay
				// Update the ingress with the final paths.
				ing.Spec.Rules = createIngressRules(tt.middlePaths)
				if err := mockClient.Update(context.TODO(), ing); err != nil {
					t.Fatalf("Failed to update ingress: %v", err)
				}

				// will block the current goroutine until the data is read from the other side of the channel
				// watchChan <- watch.Event{Type: watch.Modified, Object: ing}

				go func() {
					watchChan <- watch.Event{Type: watch.Modified, Object: ing}
				}()

				// debug
				t.Log("updating finalPaths")

				time.Sleep(2 * time.Second) // Give it some delay
				// Update the ingress with the final paths.
				ing.Spec.Rules = createIngressRules(tt.finalPaths)
				if err := mockClient.Update(context.TODO(), ing); err != nil {
					t.Fatalf("Failed to update ingress: %v", err)
				}
			}

			// Wait for a response or timeout after a few seconds.
			select {
			case err := <-errorCh:
				assert.Nil(t, err)
			case <-time.After(20 * time.Second): // Adjust as needed
				t.Fatal("Timeout while waiting for processIngressForRenewal response")
			}

			// Delete the Ingress object.
			if err := mockClient.Delete(context.Background(), ing); err != nil {
				t.Fatalf("Failed to delete Ingress: %v", err)
			}

		})
	}
}

func TestWaitForAcmeChallenge(t *testing.T) {
	ctx := context.TODO()

	// Configuration & Mock setup
	// Create the mock client for ingress watching
	fakeK8sClient := &FakeKubernetesClient{}
	watchChan := make(chan watch.Event)
	fakeWatcher := &FakeWatcher{
		resultCh: watchChan,
	}
	fakeK8sClient.On("Watch", mock.Anything, mock.Anything, mock.Anything).Return(fakeWatcher, nil)

	// Create the mock client for k8s resources
	mockClient := fakec.NewClientBuilder().WithScheme(scheme.Scheme).Build()

	// Use fakeK8sClient for IngressWatcher
	iw, err := setupIngressWatcherMock(mockClient, fakeK8sClient)
	if err != nil {
		t.Fatalf("Failed to setup IngressWatcher: %v", err)
	}

	// Define the namespace and ingress name.
	namespace := "default"
	ingressName := "test-ingress"

	// Create an ingress without the acme challenge path.
	ing := generateIngress(ingressName, namespace, nil, []string{"/testpath"}, nil)
	if err := mockClient.Create(ctx, ing); err != nil {
		t.Fatalf("Failed to create initial ingress: %v", err)
	}

	// Run waitForAcmeChallenge in a goroutine.
	goErrCh := make(chan error)
	go func() {
		goErrCh <- iw.waitForAcmeChallenge(ctx, namespace, ingressName)
	}()

	// Simulate real-world delay before an update
	time.Sleep(2 * time.Second)

	// Update the ingress to include the acme challenge path.
	ing.Spec.Rules[0].IngressRuleValue.HTTP.Paths = append(ing.Spec.Rules[0].IngressRuleValue.HTTP.Paths, networkingv1.HTTPIngressPath{Path: "/.well-known/acme-challenge"})
	if err := mockClient.Update(ctx, ing); err != nil {
		t.Fatalf("Failed to update ingress: %v", err)
	}

	// Trigger the watch event
	watchChan <- watch.Event{Type: watch.Modified, Object: ing}

	// Wait for the goroutine to finish and check the result.
	select {
	case err := <-goErrCh:
		assert.Nil(t, err)
	case <-time.After(20 * time.Second): // Adjust as needed
		t.Fatal("Timeout while waiting for waitForAcmeChallenge response")
	}
}

func TestNewIngressWatcher(t *testing.T) {
	fakeClientset := fake.NewSimpleClientset()

	t.Run("successfully initialize an IngressWatcher", func(t *testing.T) {
		// Mock the stopCh
		stopCh := make(chan struct{})
		defer close(stopCh)

		// TODO: Mock config.GetConfig() to return a dummy configuration

		iw, err := NewIngressWatcher(fakeClientset, stopCh)
		assert.NoError(t, err)
		assert.NotNil(t, iw)

		// TODO: Check other attributes of iw to ensure they are correctly set up
	})

	// TODO: Add more test cases for negative scenarios like failing to get config, failing to set up the scheme, etc.
}

func TestIsBackendHttpsAnnotations(t *testing.T) {
	ctx := context.TODO()

	t.Run("returns true when backend protocol is HTTPS", func(t *testing.T) {
		ing := &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"nginx.ingress.kubernetes.io/backend-protocol": "HTTPS",
				},
			},
		}

		result := isBackendHttpsAnnotations(ctx, ing)
		assert.True(t, result)
	})

	t.Run("returns false when backend protocol label is missing", func(t *testing.T) {
		ing := &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{},
			},
		}

		result := isBackendHttpsAnnotations(ctx, ing)
		assert.False(t, result)
	})

	t.Run("returns false when backend protocol is not HTTPS", func(t *testing.T) {
		ing := &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"nginx.ingress.kubernetes.io/backend-protocol": "HTTP",
				},
			},
		}

		result := isBackendHttpsAnnotations(ctx, ing)
		assert.False(t, result)
	})
}

func TestIsAcmeChallengePath(t *testing.T) {
	ctx := context.TODO()

	t.Run("returns true when string contains .well-known/acme-challenge", func(t *testing.T) {
		p := "/.well-known/acme-challenge/test"
		result := isAcmeChallengePath(ctx, p)
		assert.True(t, result)
	})

	t.Run("returns false when string does not contain .well-known/acme-challenge", func(t *testing.T) {
		p := "/testpath/test"
		result := isAcmeChallengePath(ctx, p)
		assert.False(t, result)
	})

	t.Run("returns true when .well-known/acme-challenge is embedded in string", func(t *testing.T) {
		p := "/test/.well-known/acme-challenge/testpath"
		result := isAcmeChallengePath(ctx, p)
		assert.True(t, result)
	})
}

func TestContainsAcmeChallenge(t *testing.T) {
	ctx := context.TODO()

	t.Run("returns true when Ingress contains .well-known/acme-challenge in a path", func(t *testing.T) {

		paths := []string{"/.well-known/acme-challenge/test", "/testpath/test"}
		rules := createIngressRules(paths)

		ing := &networkingv1.Ingress{
			Spec: networkingv1.IngressSpec{
				Rules: rules,
			},
		}
		result := containsAcmeChallenge(ctx, ing)
		assert.True(t, result)
	})

	t.Run("returns false when Ingress does not contain .well-known/acme-challenge in any path", func(t *testing.T) {
		paths := []string{"/testpath1/test", "/testpath2/test"}
		rules := createIngressRules(paths)

		ing := &networkingv1.Ingress{
			Spec: networkingv1.IngressSpec{
				Rules: rules,
			},
		}
		result := containsAcmeChallenge(ctx, ing)
		assert.False(t, result)
	})
}
