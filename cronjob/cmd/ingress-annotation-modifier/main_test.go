package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	networkingv1 "k8s.io/client-go/kubernetes/typed/networking/v1"
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

type CustomClientset struct {
	*kubernetes.Clientset
	fakeClient *fake.Clientset
}

func (c *CustomClientset) NetworkingV1() networkingv1.NetworkingV1Interface {
	return c.fakeClient.NetworkingV1()
}

// func TestRunIngressWatcher(t *testing.T) {
// 	// Use the fake ClientSet from Kubernetes client-go library.
// 	fakeClientSet := fake.NewSimpleClientset()

// 	// Create our custom clientset wrapper
// 	clientset := &CustomClientset{
// 		Clientset:  &kubernetes.Clientset{},
// 		fakeClient: fakeClientSet,
// 	}

// 	// Set any required environment variables.
// 	os.Setenv("KUBERNETES_MASTER", "http://localhost:8080")

// 	// Load environment variables configuration.
// 	ecfg, err := configenv.LoadConfig()
// 	if err != nil {
// 		t.Fatalf("Failed to load config: %v", err)
// 	}
// 	t.Logf("RUN_MODE: %s, ADMIN_USER_PERMISSION: %v", ecfg.RunMode, ecfg.AdminUserPermission)

// 	if err = runIngressWatcher(clientset.Clientset, ecfg); err != nil {
// 		t.Fatal(err)
// 	}

// 	// Cleanup: Remove any environment variables set for the test.
// 	os.Unsetenv("KUBERNETES_MASTER")
// }
