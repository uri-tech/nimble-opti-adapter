package ingresswatcher

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/scheme"
	fakec "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

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
