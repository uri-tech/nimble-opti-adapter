package controller

import (
	"context"
	"testing"

	nimbleoptiadapterv1 "github.com/uri-tech/nimble-opti-adapter/api/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/stretchr/testify/assert"
)

// TestNimbleOptiAdapterReconciler contains all test cases for NimbleOptiAdapterReconciler
func TestNimbleOptiAdapterReconciler(t *testing.T) {
	var (
		namespace             = "default"
		nimbleOptiAdapterName = "default"
		ingressName           = "test-ingress"
	)

	// Test case: Reconcile should not return error if Ingress does not exist
	t.Run("Reconcile", func(t *testing.T) {
		reconciler := &NimbleOptiAdapterReconciler{
			Client: k8sClient,
		}

		_, err := reconciler.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      ingressName,
				Namespace: namespace,
			},
		})

		assert.NoError(t, err, "Reconcile should not return an error")
	})

	// Test case: getOrCreateNimbleOptiAdapter should create a new NimbleOptiAdapter if none exists
	t.Run("getOrCreateNimbleOptiAdapter - new", func(t *testing.T) {
		ctx := context.Background()
		reconciler := &NimbleOptiAdapterReconciler{
			Client: k8sClient,
		}

		_, err := reconciler.getOrCreateNimbleOptiAdapter(ctx, namespace)
		assert.NoError(t, err, "getOrCreateNimbleOptiAdapter should not return an error when creating a new NimbleOptiAdapter")

		nimbleOptiAdapter := &nimbleoptiadapterv1.NimbleOptiAdapter{}
		err = k8sClient.Get(ctx, types.NamespacedName{Name: nimbleOptiAdapterName, Namespace: namespace}, nimbleOptiAdapter)
		assert.NoError(t, err, "NimbleOptiAdapter should exist after getOrCreateNimbleOptiAdapter is called")
	})

	// Test case: getOrCreateNimbleOptiAdapter should return existing NimbleOptiAdapter if it exists
	t.Run("getOrCreateNimbleOptiAdapter - existing", func(t *testing.T) {
		ctx := context.Background()
		reconciler := &NimbleOptiAdapterReconciler{
			Client: k8sClient,
		}

		existingNimbleOptiAdapter := &nimbleoptiadapterv1.NimbleOptiAdapter{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nimbleOptiAdapterName,
				Namespace: namespace,
			},
			Spec: nimbleoptiadapterv1.NimbleOptiAdapterSpec{
				CertificateRenewalThreshold: 30,
				AnnotationRemovalDelay:      5 * 60, // 5 minutes in seconds
			},
		}
		assert.NoError(t, k8sClient.Create(ctx, existingNimbleOptiAdapter), "Creating existing Nimble 		OptiAdapter should not return an error")

		nimbleOptiAdapter, err := reconciler.getOrCreateNimbleOptiAdapter(ctx, namespace)
		assert.NoError(t, err, "getOrCreateNimbleOptiAdapter should not return an error when NimbleOptiAdapter exists")

		// Check if the returned NimbleOptiAdapter is the same as the one we created
		assert.Equal(t, existingNimbleOptiAdapter.Name, nimbleOptiAdapter.Name, "NimbleOptiAdapter names should match")
		assert.Equal(t, existingNimbleOptiAdapter.Namespace, nimbleOptiAdapter.Namespace, "NimbleOptiAdapter namespaces should match")
	})

	// Test case: checkForAcmeChallengePath should return true if .well-known/acme-challenge path exists
	t.Run("checkForAcmeChallengePath - exists", func(t *testing.T) {
		reconciler := &NimbleOptiAdapterReconciler{
			Client: k8sClient,
		}

		ingress := &networkingv1.Ingress{
			Spec: networkingv1.IngressSpec{
				Rules: []networkingv1.IngressRule{
					{
						IngressRuleValue: networkingv1.IngressRuleValue{
							HTTP: &networkingv1.HTTPIngressRuleValue{
								Paths: []networkingv1.HTTPIngressPath{
									{
										Path: "/.well-known/acme-challenge",
									},
								},
							},
						},
					},
				},
			},
		}

		assert.True(t, reconciler.checkForAcmeChallengePath(ingress), "checkForAcmeChallengePath should return true when path exists")
	})

	// Test case: checkForAcmeChallengePath should return false if .well-known/acme-challenge path does not exist
	t.Run("checkForAcmeChallengePath - does not exist", func(t *testing.T) {
		reconciler := &NimbleOptiAdapterReconciler{
			Client: k8sClient,
		}

		ingress := &networkingv1.Ingress{
			Spec: networkingv1.IngressSpec{
				Rules: []networkingv1.IngressRule{
					{
						IngressRuleValue: networkingv1.IngressRuleValue{
							HTTP: &networkingv1.HTTPIngressRuleValue{
								Paths: []networkingv1.HTTPIngressPath{
									{
										Path: "/test",
									},
								},
							},
						},
					},
				},
			},
		}

		assert.False(t, reconciler.checkForAcmeChallengePath(ingress), "checkForAcmeChallengePath should return false when path does not exist")
	})

	// TODO: Add test cases for renewCertificate method
	// This might require more sophisticated setup, potentially using a fake clock to simulate the passage of time
}
