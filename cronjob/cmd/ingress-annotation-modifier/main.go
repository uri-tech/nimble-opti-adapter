package main

import (
	"context"
	"errors"
	"time"

	"github.com/uri-tech/nimble-opti-adapter/cronjob/configenv"
	"github.com/uri-tech/nimble-opti-adapter/cronjob/internal/ingresswatcher"
	"github.com/uri-tech/nimble-opti-adapter/cronjob/loggerpkg"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Initialize logger for the main application context
var logger = loggerpkg.GetNamedLogger("main")

// main is the entry point of the application.
func main() {
	logger.Debug("main")

	// Load environment variables configuration
	ecfg, err := configenv.LoadConfig()
	if err != nil {
		logger.Fatalf("Failed to load config: %v", err)
	}

	// log info of the setup
	logger.Infof("RUN_MODE: %s, ADMIN_USER_PERMISSION: %v", ecfg.RunMode, ecfg.AdminUserPermission)

	// Run the core functionality of the application
	if err := run(ecfg); err != nil {
		logger.Error(err)
	}
}

// run orchestrates the main flow of the application, setting up the Kubernetes client and initiating the IngressWatcher.
func run(ecfg *configenv.ConfigEnv) error {
	logger.Debug("run")

	// Set up Kubernetes client
	clientset, err := setupKubernetesClient()
	if err != nil {
		return err
	}
	// Run the IngressWatcher to monitor and modify ingress resources
	return runIngressWatcher(clientset, ecfg)
}

// setupKubernetesClient initializes and returns a Kubernetes clientset using the provided kubeconfig.
func setupKubernetesClient() (*kubernetes.Clientset, error) {
	logger.Debug("setupKubernetesClient")

	// Use in-cluster configuration
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.New("error getting in-cluster config: " + err.Error())
	}

	// Create and return a new Kubernetes clientset based on the configuration
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.New("error creating Kubernetes client: " + err.Error())
	}

	return clientset, nil
}

// runIngressWatcher creates a new IngressWatcher instance and initiates the audit of Ingress resources.
func runIngressWatcher(clientset *kubernetes.Clientset, ecfg *configenv.ConfigEnv) error {
	logger.Debug("runIngressWatcher")

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
