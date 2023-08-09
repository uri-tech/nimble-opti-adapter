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
	v1 "github.com/uri-tech/nimble-opti-adapter/api/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakec "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// Section: help functions

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

// generateIngress creates an Ingress object with the given name, namespace, and labels.
func generateIngress(name, namespace string, labels map[string]string, paths []string) *networkingv1.Ingress {
	var ingressRules []networkingv1.IngressRule

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
		ingressRules = append(ingressRules, rule)
	}

	return &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
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

	ingressWithLabel := generateIngress("ingress-with-label", "default", map[string]string{"nimble.opti.adapter/enabled": "true"}, nil)
	ingressWithoutLabel := generateIngress("ingress-without-label", "default", nil, nil)

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

	// Test: Add an ingress without the nimble.opti.adapter/enabled label.
	ing := generateIngress("test-ingress", "default", nil, nil)
	iw.handleIngressAdd(ing)
	val, ok := ing.Labels["nimble.opti.adapter/enabled"]
	assert.False(t, ok || val == "true", "Did not expect label to be present or set to true")

	// Test: Add an ingress with the nimble.opti.adapter/enabled label set to true.
	labels := map[string]string{"nimble.opti.adapter/enabled": "true"}
	ingWithLabel := generateIngress("test-ingress-with-label", "default", labels, nil)
	iw.handleIngressAdd(ingWithLabel)
	assert.Equal(t, "true", ingWithLabel.Labels["nimble.opti.adapter/enabled"], "Expected label to be present and set to true")
	// check if the nimbleopti object was created
	nimbleOpti := &v1.NimbleOpti{}
	err = iw.ClientObj.Get(context.TODO(), client.ObjectKey{Name: "default", Namespace: "default"}, nimbleOpti)
	assert.NoError(t, err)
	assert.NotNil(t, nimbleOpti)
}

func TestHandleIngressUpdate(t *testing.T) {
	fakeClient := fakec.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	iw, err := setupIngressWatcher(fakeClient)
	if err != nil {
		t.Fatalf("Failed to setup IngressWatcher: %v", err)
	}

	// Test: Update an ingress without any changes.
	oldIng := generateIngress("old-ingress", "default", nil, nil)
	newIng := generateIngress("old-ingress", "default", nil, nil)
	iw.handleIngressUpdate(oldIng, newIng)

	// Assert: Check the expected behavior here.
	// Assuming no changes were made, the label should still not exist.
	val, ok := newIng.Labels["nimble.opti.adapter/enabled"]
	assert.False(t, ok || val == "true", "Did not expect label to be present")

	// Test: Update an ingress with changes.
	labels := map[string]string{"nimble.opti.adapter/enabled": "true"}
	paths := []string{"/app"}
	oldIngDifferent := generateIngress("changed-ingress", "default", nil, nil)
	newIngDifferent := generateIngress("changed-ingress", "default", labels, paths)
	iw.handleIngressUpdate(oldIngDifferent, newIngDifferent)

	// Assert: Check the label has been processed.
	assert.Equal(t, "true", newIngDifferent.Labels["nimble.opti.adapter/enabled"], "Expected label to be present")
	// check if the nimbleopti object is exist or was created
	nimbleOpti := &v1.NimbleOpti{}
	err = iw.ClientObj.Get(context.TODO(), client.ObjectKey{Name: "default", Namespace: "default"}, nimbleOpti)
	assert.NoError(t, err)
	assert.NotNil(t, nimbleOpti)
}

// Section: 3 -

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
func TestIsAdapterEnabled(t *testing.T) {
	// Test if the function returns true when the label is present and set to "true".
	ingWithLabel := generateIngress("test-ingress-with-label", "default", map[string]string{"nimble.opti.adapter/enabled": "true"}, nil)
	assert.True(t, isAdapterEnabled(context.TODO(), ingWithLabel))

	// Test if the function returns false when the label is not present.
	ingWithoutLabel := generateIngress("test-ingress", "default", nil, nil)
	assert.False(t, isAdapterEnabled(context.TODO(), ingWithoutLabel))

	// Test if the function returns false when the label is present but not set to "true".
	ingWithFalseLabel := generateIngress("test-ingress-with-false-label", "default", map[string]string{"nimble.opti.adapter/enabled": "false"}, nil)
	assert.False(t, isAdapterEnabled(context.TODO(), ingWithFalseLabel))
}

func TestHasIngressChanged(t *testing.T) {
	oldIng := generateIngress("old-ingress", "default", nil, nil)
	newIng := generateIngress("new-ingress", "default", map[string]string{"nimble.opti.adapter/enabled": "true"}, nil)

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

	ing := generateIngress("test-ingress", "default", nil, nil)
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

	ing := generateIngress("test-ingress", "default", nil, nil)

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
	fakeClient := fakec.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	iw, err := setupIngressWatcher(fakeClient)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		ingressLabels map[string]string
		ingressPaths  []string
		wantRenewal   bool
	}{
		{
			name: "Ingress with ACME challenge path should trigger renewal",
			ingressLabels: map[string]string{
				"nimble.opti.adapter/enabled": "true",
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
				"nimble.opti.adapter/enabled": "true",
			},
			ingressPaths: []string{
				"/app",
			},
			wantRenewal: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate the Ingress object with the specified labels and paths.
			ing := generateIngress("test-ingress", "default", tt.ingressLabels, tt.ingressPaths)

			// Create the Ingress object using the fake client.
			if err := fakeClient.Create(context.TODO(), ing); err != nil {
				t.Fatalf("Failed to create Ingress: %v", err)
			}

			gotRenewal, err := iw.processIngressForRenewal(context.TODO(), ing)
			if err != nil {
				t.Fatalf("processIngressForRenewal() returned an error: %v", err)
			}

			if gotRenewal != tt.wantRenewal {
				t.Fatalf("processIngressForRenewal() = %v; want %v", gotRenewal, tt.wantRenewal)
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
			ing := generateIngress("test-ingress", "default", nil, tt.initialPaths)
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
			resultCh := make(chan bool)
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

			ing.Spec.Rules = nil
			for _, path := range tt.finalPaths {
				ing.Spec.Rules = append(ing.Spec.Rules, networkingv1.IngressRule{
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path: path,
								},
							},
						},
					},
				})
			}
			if err := fakeClient.Update(ctx, ing); err != nil {
				t.Fatalf("Failed to update ingress: %v", err)
			}

			// Assertions
			select {
			case err := <-errorCh:
				t.Fatalf("Error from waitForChallengeAbsence: %v", err)
			case res := <-resultCh:
				assert.Equal(t, tt.expectPathAbsence, res)
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
	}{
		{
			name:         "Successful certificate renewal",
			initialPaths: []string{"/app"},
		},
		{
			name:         "Failure at removing HTTPS annotation",
			initialPaths: []string{"/app", "/.well-known/acme-challenge"},
		},
		// Add more test scenarios as needed.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fakec.NewClientBuilder().WithScheme(scheme.Scheme).Build()

			// Create the initial Ingress object.
			ing := generateIngress("test-ingress", "default", nil, tt.initialPaths)
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

			if err = iw.startCertificateRenewal(ctx, ing, nimbleOpti); err != nil {
				t.Fatalf("startCertificateRenewal failed: %v", err)
			}
		})
	}
}

func TestRenewCertificateIfNecessary(t *testing.T) {
	ctx := context.TODO()

	tests := []struct {
		name           string
		tlsSecrets     map[string][]byte // map of secret names to their 'tls.crt' content
		tlsSecretsTime time.Duration
		wantRenewal    bool // Whether we expect a certificate renewal to be initiated
	}{
		{
			name:           "Certificate is about to expire",
			tlsSecrets:     map[string][]byte{},
			tlsSecretsTime: 1,
			wantRenewal:    true,
		},
		{
			name:           "Certificate is not about to expire",
			tlsSecrets:     map[string][]byte{},
			tlsSecretsTime: 100,
			wantRenewal:    false,
		},
		{
			name:           "tls.crt missing in secret",
			tlsSecrets:     map[string][]byte{}, // empty content indicates no 'tls.crt' in the secret
			tlsSecretsTime: 0,
			wantRenewal:    false,
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

			fakeClient := fakec.NewClientBuilder().WithScheme(scheme.Scheme).Build()

			// Create the Ingress with the TLS spec.
			ing := generateIngress("test-ingress", "default", nil, nil)
			ing.Spec.TLS = []networkingv1.IngressTLS{
				{
					SecretName: "test-secret",
				},
			}
			if err := fakeClient.Create(ctx, ing); err != nil {
				t.Fatalf("Failed to create ingress: %v", err)
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
				if err := fakeClient.Create(ctx, secret); err != nil {
					t.Fatalf("Failed to create secret %s: %v", secretName, err)
				}
			}

			// Create the IngressWatcher.
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
					CertificateRenewalThreshold: 3, // Renew if certificate expires within 3 days
					AnnotationRemovalDelay:      5,
				},
			}
			if err := fakeClient.Create(ctx, nimbleOpti); err != nil {
				t.Fatalf("Failed to create NimbleOpti: %v", err)
			}

			// Call the method.
			if err := iw.renewCertificateIfNecessary(ctx, ing); err != nil {
				t.Errorf("Error in renewCertificateIfNecessary: %v", err)
			}
		})
	}
}
