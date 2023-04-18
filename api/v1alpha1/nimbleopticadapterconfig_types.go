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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NimbleOpticAdapterConfigSpec defines the desired state of NimbleOpticAdapterConfig
type NimbleOpticAdapterConfigSpec struct {
	// TargetNamespace is the namespace where the operator should manage certificates
	TargetNamespace string `json:"targetNamespace"`
	// CertificateRenewalThreshold is the waiting time (in days) before the certificate expires to trigger renewal
	CertificateRenewalThreshold int `json:"certificateRenewalThreshold"`
	// AnnotationRemovalDelay is the delay (in seconds) after removing the "nginx.ingress.kubernetes.io/backend-protocol: HTTPS" annotation before re-adding it
	AnnotationRemovalDelay int `json:"annotationRemovalDelay"`
	// RenewalCheckInterval is the interval (in minutes) for checking certificate renewals
	RenewalCheckInterval int `json:"renewalCheckInterval"`
}

// NimbleOpticAdapterConfigStatus defines the observed state of NimbleOpticAdapterConfig
type NimbleOpticAdapterConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// NimbleOpticAdapterConfig is the Schema for the nimbleopticadapterconfigs API
type NimbleOpticAdapterConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NimbleOpticAdapterConfigSpec   `json:"spec,omitempty"`
	Status NimbleOpticAdapterConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NimbleOpticAdapterConfigList contains a list of NimbleOpticAdapterConfig
type NimbleOpticAdapterConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NimbleOpticAdapterConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NimbleOpticAdapterConfig{}, &NimbleOpticAdapterConfigList{})
}
