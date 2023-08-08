package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"k8s.io/klog/v2"
)

func TestIncrementCertificateRenewals(t *testing.T) {
	// Clear the metric for testing
	// CertificateRenewalsTotal.Reset()

	IncrementCertificateRenewals()
	assert.Equal(t, float64(1), testutil.ToFloat64(CertificateRenewalsTotal), "Expected CertificateRenewalsTotal to be incremented")

	IncrementCertificateRenewals()
	assert.Equal(t, float64(2), testutil.ToFloat64(CertificateRenewalsTotal), "Expected CertificateRenewalsTotal to be incremented again")
}

func TestRecordAnnotationUpdateDuration(t *testing.T) {
	// Collect the metric's data into a channel
	ch := make(chan prometheus.Metric, 1)
	AnnotationUpdatesDuration.Collect(ch)

	// Convert the metric into a dto.Metric to extract the histogram data
	metric := dto.Metric{}
	_ = (<-ch).Write(&metric)

	// Check the initial count of observations
	initialCount := metric.Histogram.GetSampleCount()

	RecordAnnotationUpdateDuration(0.5)

	// debug
	klog.Infof("Histogram.GetSampleCount()-1: %s", metric.Histogram.GetSampleCount())

	// Collect the metric's data again
	ch = make(chan prometheus.Metric, 1)
	AnnotationUpdatesDuration.Collect(ch)
	_ = (<-ch).Write(&metric)

	// Check the count of observations after the first recording
	assert.Equal(t, initialCount+1, metric.Histogram.GetSampleCount(), "Expected one observation in AnnotationUpdatesDuration")

	RecordAnnotationUpdateDuration(1.5)

	// debug
	klog.Infof("Histogram.GetSampleCount()-2: %s", metric.Histogram.GetSampleCount())

	// Collect the metric's data again
	ch = make(chan prometheus.Metric, 1)
	AnnotationUpdatesDuration.Collect(ch)
	_ = (<-ch).Write(&metric)

	// debug
	klog.Infof("Histogram.GetSampleCount()-3: %s", metric.Histogram.GetSampleCount())

	// Check the count of observations after the second recording
	assert.Equal(t, initialCount+2, metric.Histogram.GetSampleCount(), "Expected another observation in AnnotationUpdatesDuration")
}
