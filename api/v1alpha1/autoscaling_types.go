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

// AutoscalingAction defines the desired number of replicas for each action
type AutoscalingAction struct {
	MinReplicas int `json:"minReplicas"`
	MaxReplicas int `json:"maxReplicas"`
}

// AutoscalingSpec defines the desired state of Autoscaling
type AutoscalingSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Enabled       *bool                        `json:"enabled,omitempty"`
	Namespaces    []string                     `json:"namespaces"`
	LabelSelector metav1.LabelSelector         `json:"labelSelector"`
	Schedule      string                       `json:"schedule"`
	Actions       map[string]AutoscalingAction `json:"actions"`
}

// AutoscalingStatus defines the observed state of Autoscaling
type AutoscalingStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Conditions []metav1.Condition `json:"conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// Autoscaling is the Schema for the autoscalings API
type Autoscaling struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AutoscalingSpec   `json:"spec,omitempty"`
	Status AutoscalingStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AutoscalingList contains a list of Autoscaling
type AutoscalingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Autoscaling `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Autoscaling{}, &AutoscalingList{})
}
