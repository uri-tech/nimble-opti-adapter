package ingresswatcher

import (
	"context"

	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/watch"
	// "k8s.io/client-go/kubernetes"
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

// func TestKubernetesClient_Watch(t *testing.T) {
// 	mockClient := new(FakeKubernetesClient)
// 	fakeWatch := &FakeWatcher{
// 		resultCh: make(chan watch.Event),
// 	}

// 	expectedNamespace := "default"
// 	expectedIngressName := "test-ingress"

// 	mockClient.On("Watch", mock.Anything, expectedNamespace, expectedIngressName).Return(fakeWatch, nil)

// 	watcher, err := mockClient.Watch(context.Background(), expectedNamespace, expectedIngressName)
// 	t.Logf("watcher: %v", watcher)

// 	if err != nil {
// 		t.Fatalf("expected no error, but got: %v", err)
// 	}

// 	if watcher != fakeWatch {
// 		t.Fatalf("expected returned watcher to be the fake watcher, but it wasn't")
// 	}

// 	// Ensure that the mock's expected methods were called
// 	mockClient.AssertExpectations(t)
// }
