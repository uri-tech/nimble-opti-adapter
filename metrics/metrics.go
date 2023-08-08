// metrics/metrics.go

// Package metrics provides Prometheus metrics utility functions for the Nimble Opti Adapter.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
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
	// Register the metrics.
	prometheus.MustRegister(CertificateRenewalsTotal)
	prometheus.MustRegister(AnnotationUpdatesDuration)
}

// IncrementCertificateRenewals increments the certificate renewals counter.
func IncrementCertificateRenewals() {
	CertificateRenewalsTotal.Inc()
}

// RecordAnnotationUpdateDuration records the duration for annotation updates.
func RecordAnnotationUpdateDuration(duration float64) {
	AnnotationUpdatesDuration.Observe(duration)
}
