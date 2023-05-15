// tests/controller/controller_test.go

package controller_test

// func createReconciler() *Reconciler {
// 	fakeClient := fake.NewClientBuilder().Build()
// 	return &Reconciler{
// 		Client: fakeClient,
// 		Scheme: scheme.Scheme,
// 	}
// }

// func TestReconcile(t *testing.T) {
// 	ctx := context.Background()
// 	r := createReconciler()

// 	// TODO: Modify this example ingress based on your needs
// 	ingress := &networkingv1.Ingress{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      "test-ingress",
// 			Namespace: "default",
// 			Labels: map[string]string{
// 				"nimble.opti.adapter/enabled": "true",
// 			},
// 		},
// 	}

// 	// Insert the test ingress into the fake client's cache
// 	err := r.Client.Create(ctx, ingress)
// 	assert.NoError(t, err)

// 	// Call the Reconcile method and assert there are no errors
// 	_, err = r.Reconcile(ctx, ctrl.Request{
// 		NamespacedName: metav1.NamespacedName{
// 			Name:      ingress.Name,
// 			Namespace: ingress.Namespace,
// 		},
// 	})
// 	assert.NoError(t, err)
// }

// func TestGetOrCreateNimbleOptiAdapter(t *testing.T) {
// 	ctx := context.Background()
// 	r := createReconciler()

// 	// Call the method and assert there are no errors
// 	_, err := r.GetOrCreateNimbleOptiAdapter(ctx, "default")
// 	assert.NoError(t, err)
// }

// func TestCheckForAcmeChallengePath(t *testing.T) {
// 	r := createReconciler()

// 	// Test Ingress without .well-known/acme-challenge path
// 	ingressNoAcme := &networkingv1.Ingress{}
// 	assert.False(t, r.CheckForAcmeChallengePath(ingressNoAcme))

// 	// Test Ingress with .well-known/acme-challenge path
// 	ingressWithAcme := &networkingv1.Ingress{
// 		Spec: networkingv1.IngressSpec{
// 			Rules: []networkingv1.IngressRule{
// 				{
// 					HTTP: &networkingv1.HTTPIngressRuleValue{
// 						Paths: []networkingv1.HTTPIngressPath{
// 							{
// 								Path: "/.well-known/acme-challenge",
// 							},
// 						},
// 					},
// 				},
// 			},
// 		},
// 	}
// 	assert.True(t, r.CheckForAcmeChallengePath(ingressWithAcme))
// }

// func TestRenewCertificate(t *testing.T) {
// 	ctx := context.Background()
// 	r := createReconciler()

// 	// TODO: Modify this example ingress and nimbleOptiAdapter based on your needs
// 	ingress := &networkingv1.Ingress{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      "test-ingress",
// 			Namespace: "default",
// 			Annotations: map[string]string{
// 				"nginx.ingress.kubernetes.io/backend-protocol": "HTTPS",
// 			},
// 		},
// 	}

// 	nimbleOptiAdapter := &nimbleoptiadapterv1.NimbleOptiAdapter{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      "default",
// 			Namespace: "default",
// 		},
// 		Spec: nimbleoptiadapterv1.NimbleOptiAdapterSpec{
// 			AnnotationRemovalDelay: 300, // 5 minutes in seconds
// 		},
// 	}

// 	err := r.Client.Create(ctx, ingress)
// 	assert.NoError(t, err)

// 	err = r.Client.Create(ctx, nimbleOptiAdapter)
// 	assert.NoError(t, err)

// 	// Call the method and assert there are no errors
// 	err = r.RenewCertificate(ctx, ingress, nimbleOptiAdapter)
// 	assert.NoError(t, err)
// }
