package main

import (
	"context"
	"errors"
	"flag"
	"os"
	"time"

	"github.com/uri-tech/nimble-opti-adapter/cronjob/configenv"
	"github.com/uri-tech/nimble-opti-adapter/cronjob/internal/ingresswatcher"
	"github.com/uri-tech/nimble-opti-adapter/cronjob/loggerpkg"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// main is the entry point of the application.
func main() {
	// Initialize logger for the main application context.
	logger := loggerpkg.GetNamedLogger("main")

	// Load environment variables configuration.
	ecfg, err := configenv.LoadConfig()
	if err != nil {
		logger.Fatalf("Failed to load config: %v", err)
	}

	// Parse command-line arguments to fetch kubeconfig path.
	kubeconfig := parseFlags()

	// Run the core functionality of the application.
	if err := run(kubeconfig, ecfg); err != nil {
		logger.Error(err)
	}
}

// parseFlags reads and parses command-line flags, returning the kubeconfig path.
func parseFlags() string {
	// Set default path for kubeconfig.
	defaultKubeconfig := os.Getenv("HOME") + "/.kube/config"
	var kubeconfig string

	// Define the command-line flag for kubeconfig.
	flag.StringVar(&kubeconfig, "kubeconfig", defaultKubeconfig, "Path to a kubeconfig. Only required if out-of-cluster.")

	// Parse all provided command-line flags.
	flag.Parse()

	return kubeconfig
}

// run orchestrates the main flow of the application, setting up the Kubernetes client and initiating the IngressWatcher.
func run(kubeconfig string, ecfg *configenv.ConfigEnv) error {
	// Set up Kubernetes client using provided kubeconfig.
	clientset, err := setupKubernetesClient(kubeconfig, clientcmd.BuildConfigFromFlags)

	if err != nil {
		return err
	}

	// Run the IngressWatcher to monitor and modify ingress resources.
	return runIngressWatcher(clientset, ecfg)
}

// setupKubernetesClient initializes and returns a Kubernetes clientset using the provided kubeconfig.
func setupKubernetesClient(kubeconfig string, buildConfigFunc func(string, string) (*rest.Config, error)) (*kubernetes.Clientset, error) {
	// Build the configuration for Kubernetes using the kubeconfig.
	config, err := buildConfigFunc("", kubeconfig)
	if err != nil {
		return nil, errors.New("error building kubeconfig: " + err.Error())
	}

	// Create and return a new Kubernetes clientset based on the configuration.
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.New("error creating Kubernetes client: " + err.Error())
	}

	return clientset, nil
}

// runIngressWatcher creates a new IngressWatcher instance and initiates the audit of Ingress resources.
func runIngressWatcher(clientset *kubernetes.Clientset, ecfg *configenv.ConfigEnv) error {
	// Create a new IngressWatcher instance.
	iw, err := ingresswatcher.NewIngressWatcher(clientset, ecfg)
	if err != nil {
		return errors.New("error creating IngressWatcher: " + err.Error())
	}

	// Define a context with a 10-minute timeout for the auditing process.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Audit all Ingress resources in the cluster.
	if err := iw.AuditIngressResources(ctx); err != nil {
		return errors.New("error auditing Ingress resources: " + err.Error())
	}

	return nil
}
