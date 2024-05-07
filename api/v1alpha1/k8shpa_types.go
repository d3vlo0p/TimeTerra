/*
Copyright 2024.

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

func (r K8sHpaAction) IsActive() bool {
	return r.Enabled == nil || *r.Enabled
}

type K8sHpaAction struct {
	Enabled     *bool `json:"enabled,omitempty"`
	MinReplicas int   `json:"minReplicas"`
	MaxReplicas int   `json:"maxReplicas"`
}

func (r K8sHpaSpec) IsActive() bool {
	return r.Enabled == nil || *r.Enabled
}

// K8sHpaSpec defines the desired state of K8sHpa
type K8sHpaSpec struct {
	Enabled       *bool                   `json:"enabled,omitempty"`
	Namespaces    []string                `json:"namespaces,omitempty"`
	LabelSelector metav1.LabelSelector    `json:"labelSelector"`
	Schedule      string                  `json:"schedule"`
	Actions       map[string]K8sHpaAction `json:"actions"`
}

// K8sHpaStatus defines the observed state of K8sHpa
type K8sHpaStatus struct {
	Conditions []metav1.Condition `json:"conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// K8sHpa is the Schema for the k8shpas API
type K8sHpa struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   K8sHpaSpec   `json:"spec,omitempty"`
	Status K8sHpaStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// K8sHpaList contains a list of K8sHpa
type K8sHpaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []K8sHpa `json:"items"`
}

func init() {
	SchemeBuilder.Register(&K8sHpa{}, &K8sHpaList{})
}
