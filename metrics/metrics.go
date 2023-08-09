// metrics/metrics.go

// Package metrics provides Prometheus metrics utility functions for the Nimble Opti Adapter.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/klog/v2"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// CertificateRenewalsTotal counts the total number of certificate renewals.
	CertificateRenewalsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nimble_opti_adapter_certificate_renewals_total",
			Help: "Total number of certificate renewals",
		},
	)

	// AnnotationUpdatesDuration measures the duration (in seconds) of annotation updates during each renewal.
	AnnotationUpdatesDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "nimble_opti_adapter_annotation_updates_duration_seconds",
			Help:    "Duration (in seconds) of annotation updates during each renewal",
			Buckets: prometheus.DefBuckets,
		},
	)
)

func init() {
	klog.Info("Metrics init")
	// Register the metrics.
	// prometheus.MustRegister(CertificateRenewalsTotal)
	// prometheus.MustRegister(AnnotationUpdatesDuration)

	// Register the metrics with controller-runtime's default registry.

	if err := ctrlmetrics.Registry.Register(CertificateRenewalsTotal); err != nil {
		klog.Errorf("Error registering CertificateRenewalsTotal metric: %v", err)
	}

	if err := ctrlmetrics.Registry.Register(AnnotationUpdatesDuration); err != nil {
		klog.Errorf("Error registering AnnotationUpdatesDuration metric: %v", err)
	}
}

// IncrementCertificateRenewals increments the certificate renewals counter.
func IncrementCertificateRenewals() {
	CertificateRenewalsTotal.Inc()
}

// RecordAnnotationUpdateDuration records the duration for annotation updates.
func RecordAnnotationUpdateDuration(duration float64) {
	AnnotationUpdatesDuration.Observe(duration)
}
