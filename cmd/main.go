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
	"context"
	"flag"
	"net/http"
	"os"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	adapterv1 "github.com/uri-tech/nimble-opti-adapter/api/v1"
	"github.com/uri-tech/nimble-opti-adapter/internal/controller"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// Define global variables.
var (
	// Addresses for metrics and health probes.
	metricsAddr, probeAddr string
	// Flag to enable leader election.
	enableLeaderElection bool
	// Configuration options for the zap logger.
	opts = zap.Options{
		Development: true,
	}
	// Logger for setup processes.
	setupLog = ctrl.Log.WithName("setup")
)

// Initialize command line flags.
func init() {
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false, "Enable leader election for controller manager.")
	opts.BindFlags(flag.CommandLine)
}

// createCertManagerCertificate creates a Certificate resource for cert-manager.
// This will lead cert-manager to generate a TLS certificate for the webhook server.
func createCertManagerCertificate(client client.Client) error {
	cert := &cmv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "webhook-certificate",
			Namespace: "nimble-opti-adapter-system",
		},
		Spec: cmv1.CertificateSpec{
			SecretName: "webhook-certificate-secret",
			// IssuerRef: corev1.ObjectReference{
			// 	Name:  "selfsigned-issuer-nimble-opti",
			// 	Kind:  "ClusterIssuer",
			// 	Group: "cert-manager.io",
			// 	Namespace: "nimble-opti-adapter-system",
			// },
			CommonName: "webhook.noa.svc",
			DNSNames: []string{
				"webhook.noa.svc.cluster.local",
			},
			// PrivateKey: cmv1.CertificatePrivateKey{
			// 	Algorithm: cmv1.PrivateKeyAlgorithm("RSA"),
			// 	Size:      2048,
			// },
		},
	}

	// Create or update the Certificate resource.
	if err := client.Create(context.TODO(), cert); err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
		if err := client.Update(context.TODO(), cert); err != nil {
			return err
		}
	}
	return nil
}

// Entry point of the program.
func main() {
	// Parse command line flags.
	flag.Parse()

	// Set up the logger.
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// Initialize the manager with configurations.
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "8f24f142.uri-tech.github.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// // Create the cert-manager Certificate for the webhook.
	// if err := createCertManagerCertificate(mgr.GetClient()); err != nil {
	// 	setupLog.Error(err, "unable to create cert-manager certificate")
	// 	os.Exit(1)
	// }

	// Set up Prometheus metrics server.
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(metricsAddr, nil)

	// Initialize the ingress watcher.
	kubernetesClient := kubernetes.NewForConfigOrDie(mgr.GetConfig())
	ingressWatcher, err := controller.NewIngressWatcher(kubernetesClient, make(chan struct{}))
	if err != nil {
		setupLog.Error(err, "unable to create ingress watcher")
		os.Exit(1)
	}
	ingressWatcher.StartAudit(make(chan struct{}))

	// Set up the custom reconciler.
	if err = (&controller.NimbleOptiReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		KubernetesClient: kubernetesClient,
		IngressWatcher:   ingressWatcher,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NimbleOpti")
		os.Exit(1)
	}

	// Set up the webhook server.
	if err = (&adapterv1.NimbleOpti{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "NimbleOpti")
		os.Exit(1)
	}

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
