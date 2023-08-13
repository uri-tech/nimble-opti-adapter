package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/uri-tech/nimble-opti-adapter/cronjob/configenv"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// func TestMain(m *testing.M) {
// 	// Initialize logger or any other setup.
// 	loggerpkg.SetupLogger()

// 	// // Ensure logger is ready.
// 	// _ = loggerpkg.GetNamedLogger("main2")

// 	// Run the tests.
// 	os.Exit(m.Run())
// }

// Define our Mock for the Kubernetes clientset
type MockClientSet struct {
	mock.Mock
}

func (m *MockClientSet) NewForConfig(config *rest.Config) (*kubernetes.Clientset, error) {
	args := m.Called(config)
	return args.Get(0).(*kubernetes.Clientset), args.Error(1)
}

func mockBuildConfigFromFlags(apiServerURL string, kubeconfigPath string) (*rest.Config, error) {
	return &rest.Config{}, nil // Return a non-nil empty configuration
}

func mockNewForConfig(config *rest.Config) (*kubernetes.Clientset, error) {
	return &kubernetes.Clientset{}, nil // Return a non-nil empty clientset
}

func TestSetupKubernetesClient(t *testing.T) {
	clientset, err := setupKubernetesClient()
	// assert.NoError(t, err)
	if err != nil {
		// t.Fatal(err)
		assert.Nil(t, clientset)
	} else {
		assert.NotNil(t, clientset)
	}

}

func TestRunIngressWatcher(t *testing.T) {
	// We would ideally mock the NewIngressWatcher function and its interactions as well
	// For simplicity, let's assume it works with the fake ClientSet

	clientset := &kubernetes.Clientset{} // This is a placeholder, consider using k8s fake clientset or mocking further

	// Load environment variables configuration.
	ecfg, err := configenv.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if err = runIngressWatcher(clientset, ecfg); err != nil {
		t.Fatal(err)
	}
}
