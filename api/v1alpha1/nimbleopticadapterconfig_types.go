// ./api/v1alpha1/nimbleoptiadapterconfig_types.go

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NimbleOptiAdapterConfigSpec defines the desired state of NimbleOptiAdapterConfig
type NimbleOptiAdapterConfigSpec struct {
	// TargetNamespace is the namespace where the operator should manage certificates
	// +kubebuilder:validation:MinLength=1
	TargetNamespace string `json:"targetNamespace"`

	// CertificateRenewalThreshold is the waiting time (in days) before the certificate expires to trigger renewal
	// +kubebuilder:validation:Minimum=1
	CertificateRenewalThreshold int `json:"certificateRenewalThreshold"`

	// AnnotationRemovalDelay is the delay (in seconds) after removing the "nginx.ingress.kubernetes.io/backend-protocol: HTTPS" annotation before re-adding it
	// +kubebuilder:validation:Minimum=1
	AnnotationRemovalDelay int `json:"annotationRemovalDelay"`

	// RenewalCheckInterval is the interval (in minutes) for checking certificate renewals
	// +kubebuilder:validation:Minimum=1
	RenewalCheckInterval int `json:"renewalCheckInterval"`
}

// NimbleOptiAdapterConfigStatus defines the observed state of NimbleOptiAdapterConfig
type NimbleOptiAdapterConfigStatus struct {
	// Conditions are the conditions for this resource.
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// IngressPathsForRenewal is a list of ingress paths for which certificates need to be renewed.
	IngressPathsForRenewal []string `json:"ingressPathsForRenewal,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// NimbleOptiAdapterConfig is the Schema for the nimbleoptiadapterconfigs API
type NimbleOptiAdapterConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NimbleOptiAdapterConfigSpec   `json:"spec,omitempty"`
	Status NimbleOptiAdapterConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NimbleOptiAdapterConfigList contains a list of NimbleOptiAdapterConfig
type NimbleOptiAdapterConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NimbleOptiAdapterConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NimbleOptiAdapterConfig{}, &NimbleOptiAdapterConfigList{})
}
