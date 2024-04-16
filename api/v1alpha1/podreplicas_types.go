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

type ResourceType string

const (
	Deployment  ResourceType = "Deployment"
	StatefulSet ResourceType = "StatefulSet"
)

func (r ResourceType) String() string {
	return string(r)
}

// PodReplicasAction defines the desired number of replicas for each action
type PodReplicasAction struct {
	Replicas int `json:"replicas"`
}

// PodReplicasSpec defines the desired state of PodReplicas
type PodReplicasSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Enabled       *bool                        `json:"enabled,omitempty"`
	Namespaces    []string                     `json:"namespaces"`
	ResourceType  ResourceType                 `json:"resourceType"`
	LabelSelector metav1.LabelSelector         `json:"labelSelector"`
	Schedule      string                       `json:"schedule"`
	Actions       map[string]PodReplicasAction `json:"actions"`
}

// PodReplicasStatus defines the observed state of PodReplicas
type PodReplicasStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Conditions []metav1.Condition `json:"conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// PodReplicas is the Schema for the podreplicas API
type PodReplicas struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PodReplicasSpec   `json:"spec,omitempty"`
	Status PodReplicasStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PodReplicasList contains a list of PodReplicas
type PodReplicasList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PodReplicas `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PodReplicas{}, &PodReplicasList{})
}
