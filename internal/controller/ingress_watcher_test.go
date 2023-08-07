// internal/controller/ingress_watcher_test.go
package controller

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "github.com/uri-tech/nimble-opti-adapter/api/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakec "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// Section: 1 -

// setupIngressWatcher initializes a mock IngressWatcher for testing purposes.
func setupIngressWatcher(client client.Client) *IngressWatcher {
	fakeClientset := fake.NewSimpleClientset()
	stopCh := make(chan struct{})

	// Add NimbleOpti to the scheme.
	err := v1.AddToScheme(scheme.Scheme)
	if err != nil {
		panic(fmt.Sprintf("Failed to add NimbleOpti to scheme: %v", err))
	}

	iw := NewIngressWatcher(fakeClientset, stopCh)
	iw.ClientObj = client
	return iw
}

// generateIngress creates an Ingress object with the given name, namespace, and labels.
func generateIngress(name, namespace string, labels map[string]string) *networkingv1.Ingress {
	return &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
	}
}

// Section: 2 -

func TestHandleIngressAdd(t *testing.T) {
	fakeClient := fakec.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	iw := setupIngressWatcher(fakeClient)

	// Test: Add an ingress without the nimble.opti.adapter/enabled label.
	ing := generateIngress("test-ingress", "default", nil)
	iw.handleIngressAdd(ing)
	val, ok := ing.Labels["nimble.opti.adapter/enabled"]
	assert.False(t, ok || val == "true", "Did not expect label to be present or set to true")

	// Test: Add an ingress with the nimble.opti.adapter/enabled label set to true.
	labels := map[string]string{"nimble.opti.adapter/enabled": "true"}
	ingWithLabel := generateIngress("test-ingress-with-label", "default", labels)
	iw.handleIngressAdd(ingWithLabel)
	assert.Equal(t, "true", ingWithLabel.Labels["nimble.opti.adapter/enabled"], "Expected label to be present and set to true")
	// check if the nimbleopti object was created
	nimbleOpti := &v1.NimbleOpti{}
	err := iw.ClientObj.Get(context.TODO(), client.ObjectKey{Name: "default", Namespace: "default"}, nimbleOpti)
	assert.NoError(t, err)
	assert.NotNil(t, nimbleOpti)
}

func TestHandleIngressUpdate(t *testing.T) {
	fakeClient := fakec.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	iw := setupIngressWatcher(fakeClient)

	// Test: Update an ingress without any changes.
	oldIng := generateIngress("old-ingress", "default", nil)
	newIng := generateIngress("old-ingress", "default", nil)
	iw.handleIngressUpdate(oldIng, newIng)

	// Assert: Check the expected behavior here.
	// Assuming no changes were made, the label should still not exist.
	val, ok := newIng.Labels["nimble.opti.adapter/enabled"]
	assert.False(t, ok || val == "true", "Did not expect label to be present")

	// Test: Update an ingress with changes.
	labels := map[string]string{"nimble.opti.adapter/enabled": "true"}
	oldIngDifferent := generateIngress("changed-ingress", "default", nil)
	newIngDifferent := generateIngress("changed-ingress", "default", labels)
	iw.handleIngressUpdate(oldIngDifferent, newIngDifferent)

	// Assert: Check the label has been processed.
	assert.Equal(t, "true", newIngDifferent.Labels["nimble.opti.adapter/enabled"], "Expected label to be present")
	// check if the nimbleopti object is exist or was created
	nimbleOpti := &v1.NimbleOpti{}
	err := iw.ClientObj.Get(context.TODO(), client.ObjectKey{Name: "default", Namespace: "default"}, nimbleOpti)
	assert.NoError(t, err)
	assert.NotNil(t, nimbleOpti)
}

//
//
//

func TestGetOrCreateNimbleOpti(t *testing.T) {
	fakeClient := fakec.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	iw := setupIngressWatcher(fakeClient)

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
	ingWithLabel := generateIngress("test-ingress-with-label", "default", map[string]string{"nimble.opti.adapter/enabled": "true"})
	assert.True(t, isAdapterEnabled(context.TODO(), ingWithLabel))

	// Test if the function returns false when the label is not present.
	ingWithoutLabel := generateIngress("test-ingress", "default", nil)
	assert.False(t, isAdapterEnabled(context.TODO(), ingWithoutLabel))

	// Test if the function returns false when the label is present but not set to "true".
	ingWithFalseLabel := generateIngress("test-ingress-with-false-label", "default", map[string]string{"nimble.opti.adapter/enabled": "false"})
	assert.False(t, isAdapterEnabled(context.TODO(), ingWithFalseLabel))
}

func TestHasIngressChanged(t *testing.T) {
	oldIng := generateIngress("old-ingress", "default", nil)
	newIng := generateIngress("new-ingress", "default", map[string]string{"nimble.opti.adapter/enabled": "true"})

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

// func TestProcessIngressForAdapter(t *testing.T) {
// 	fakeClient := fakec.NewClientBuilder().WithScheme(scheme.Scheme).Build()
// 	iw := setupIngressWatcher(fakeClient)

// 	// TODO: Mock getOrCreateNimbleOpti to return a dummy NimbleOpti without errors.

// 	// Test with Ingress containing a rule with .well-known/acme-challenge path.
// 	ing := generateIngress("test-ingress", "default", nil)
// 	ing.Spec.Rules = []networkingv1.IngressRule{
// 		{
// 			HTTP: &networkingv1.HTTPIngressRuleValue{
// 				Paths: []networkingv1.HTTPIngressPath{
// 					{Path: "/.well-known/acme-challenge/test"},
// 				},
// 			},
// 		},
// 	}
// 	iw.processIngressForAdapter(context.TODO(), ing)
// 	// TODO: Assert that startCertificateRenewal is called.

// 	// Test with Ingress without .well-known/acme-challenge path.
// 	ingWithoutPath := generateIngress("test-ingress-without-path", "default", nil)
// 	iw.processIngressForAdapter(context.TODO(), ingWithoutPath)
// 	// TODO: Assert that startCertificateRenewal is not called.
// }
