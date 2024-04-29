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

type K8sPodReplicasResourceType string

const (
	K8sDeployment  K8sPodReplicasResourceType = "Deployment"
	K8sStatefulSet K8sPodReplicasResourceType = "StatefulSet"
)

func (r K8sPodReplicasResourceType) String() string {
	return string(r)
}

type K8sPodReplicasAction struct {
	Replicas int `json:"replicas"`
}

// K8sPodReplicasSpec defines the desired state of K8sPodReplicas
type K8sPodReplicasSpec struct {
	Enabled    *bool    `json:"enabled,omitempty"`
	Namespaces []string `json:"namespaces,omitempty"`
	// +kubebuilder:validation:Enum:=Deployment;StatefulSet
	ResourceType  K8sPodReplicasResourceType      `json:"resourceType"`
	LabelSelector metav1.LabelSelector            `json:"labelSelector"`
	Schedule      string                          `json:"schedule"`
	Actions       map[string]K8sPodReplicasAction `json:"actions"`
}

// K8sPodReplicasStatus defines the observed state of K8sPodReplicas
type K8sPodReplicasStatus struct {
	Conditions []metav1.Condition `json:"conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// K8sPodReplicas is the Schema for the k8spodreplicas API
type K8sPodReplicas struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   K8sPodReplicasSpec   `json:"spec,omitempty"`
	Status K8sPodReplicasStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// K8sPodReplicasList contains a list of K8sPodReplicas
type K8sPodReplicasList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []K8sPodReplicas `json:"items"`
}

func init() {
	SchemeBuilder.Register(&K8sPodReplicas{}, &K8sPodReplicasList{})
}
