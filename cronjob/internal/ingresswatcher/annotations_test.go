package ingresswatcher

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/scheme"
	fakec "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	httpsAnnotation = "nginx.ingress.kubernetes.io/backend-protocol"
)

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

	// // Fetch the latest version of the Ingress object
	// updatedIng := &networkingv1.Ingress{}
	// if err := fakeClient.Get(ctx, client.ObjectKey{Name: ing.Name, Namespace: ing.Namespace}, updatedIng); err != nil {
	// 	t.Fatalf("Failed to fetch updated Ingress: %v", err)
	// }

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

	val, exists := ing.Annotations[httpsAnnotation]
	assert.True(t, exists && val == "HTTPS", "Expected HTTPS annotation to be added")
}
