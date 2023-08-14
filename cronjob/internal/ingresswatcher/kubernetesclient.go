package ingresswatcher

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

// KubernetesClient defines methods we're using from the Kubernetes client.
type KubernetesClient interface {
	Watch(ctx context.Context, namespace, ingressName string) (watch.Interface, error)
}

// RealKubernetesClient is a structure that holds the Client for Kubernetes.
type RealKubernetesClient struct {
	kubernetes.Interface
}

// Watch implements the KubernetesClient interface.
func (r *RealKubernetesClient) Watch(ctx context.Context, namespace, ingressName string) (watch.Interface, error) {
	logger.Debugf("starting RealKubernetesClient.Watch, namespace: %v, ingressName: %v", namespace, ingressName)

	opts := metav1.SingleObject(metav1.ObjectMeta{Name: ingressName})
	return r.NetworkingV1().Ingresses(namespace).Watch(ctx, opts)
}
