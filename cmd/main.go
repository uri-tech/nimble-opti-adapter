/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// starting one controller (NimbleOptiReconciler), with also a watcher (ingressWatcher) that assists this controller.

// cmd/main.go

package main

import (
	"flag"
	"os"

	adapterv1 "github.com/uri-tech/nimble-opti-adapter/api/v1"
	"github.com/uri-tech/nimble-opti-adapter/internal/controller"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// Define global variables.
var (
	// Define scheme for the runtime object.
	scheme = runtime.NewScheme()
	// Addresses for metrics and health probes.
	metricsAddr, probeAddr string
	// Flag to enable leader election.
	enableLeaderElection bool
	// Configuration options for the zap logger.
	opts = zap.Options{
		Development: false,
		// Only show warnings and above
		Level: zapcore.Level(-2), // Only show warnings and above
	}

	// Logger for setup processes.
	setupLog = ctrl.Log.WithName("setup")
)

// Initialize command line flags.
func init() {
	// Add schemes for client-go and adapterv1.
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(adapterv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme

	// Parse CLI flags.
	parseFlags()
}

// parseFlags sets up and parses command-line flags.
func parseFlags() {
	// Define command-line flags.
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

}

// Entry point of the program.
func main() {
	// Set up the logger.
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// Initialize the manager with configurations.
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		// MetricsBindAddress:     "0",
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "8f24f142.uri-tech.github.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Initialize the Kubernetes client.
	stopCh := make(chan struct{})
	defer close(stopCh)

	kubernetesClient := kubernetes.NewForConfigOrDie(mgr.GetConfig())
	ingressWatcher, err := controller.NewIngressWatcher(kubernetesClient, stopCh)
	if err != nil {
		setupLog.Error(err, "unable to create ingress watcher")
		os.Exit(1)
	}

	// Start the daily audit for the ingress watcher.
	ingressWatcher.StartAudit(stopCh)

	// Setup the reconciler with the manager.
	if err = (&controller.NimbleOptiReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		KubernetesClient: kubernetesClient,
		IngressWatcher:   ingressWatcher,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NimbleOpti")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	// Set up the webhook server.
	if err = (&adapterv1.NimbleOpti{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "NimbleOpti")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	// Add health checks.
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}

	// Add readiness checks.
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	// Start the manager and listen for termination signals.
	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
