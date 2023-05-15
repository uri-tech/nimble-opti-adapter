// internal/controller/nimbleoptiadapter_controller_test.go

package controller

import (
	"context"

	nimbleoptiadapterv1 "github.com/uri-tech/nimble-opti-adapter/api/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("NimbleOptiAdapterReconciler", func() {
	var (
		namespace             = "default"
		nimbleOptiAdapterName = "default"
		ingressName           = "test-ingress"
	)

	Context("Reconcile", func() {
		It("should not return error if Ingress does not exist", func() {
			reconciler := &NimbleOptiAdapterReconciler{
				Client: k8sClient,
			}

			_, err := reconciler.Reconcile(context.Background(), reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      ingressName,
					Namespace: namespace,
				},
			})

			Expect(err).ToNot(HaveOccurred())
		})

		// Add other test cases here
	})

	Context("getOrCreateNimbleOptiAdapter", func() {
		It("should create a new NimbleOptiAdapter if none exists", func() {
			ctx := context.Background()
			reconciler := &NimbleOptiAdapterReconciler{
				Client: k8sClient,
			}

			_, err := reconciler.getOrCreateNimbleOptiAdapter(ctx, namespace)
			Expect(err).ToNot(HaveOccurred())

			nimbleOptiAdapter := &nimbleoptiadapterv1.NimbleOptiAdapter{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: nimbleOptiAdapterName, Namespace: namespace}, nimbleOptiAdapter)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return existing NimbleOptiAdapter if it exists", func() {
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
			Expect(k8sClient.Create(ctx, existingNimbleOptiAdapter)).To(Succeed())

			nimbleOptiAdapter, err := reconciler.getOrCreateNimbleOptiAdapter(ctx, namespace)
			Expect(err).ToNot(HaveOccurred())

			// Check if the returned NimbleOptiAdapter is the same as the one we created
			Expect(nimbleOptiAdapter.Name).To(Equal(existingNimbleOptiAdapter.Name))
			Expect(nimbleOptiAdapter.Namespace).To(Equal(existingNimbleOptiAdapter.Namespace))
		})

		Context("checkForAcmeChallengePath", func() {
			It("should return true if .well-known/acme-challenge path exists", func() {
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

				Expect(reconciler.checkForAcmeChallengePath(ingress)).To(BeTrue())
			})

			It("should return false if .well-known/acme-challenge path does not exist", func() {
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

				Expect(reconciler.checkForAcmeChallengePath(ingress)).To(BeFalse())
			})
		})

		// TODO: Add test cases for renewCertificate method
		// This might require more sophisticated setup, potentially using a fake clock to simulate the passage of time
	})
})