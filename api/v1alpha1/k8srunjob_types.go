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
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

func (r K8sRunJobAction) IsActive() bool {
	return r.Enabled == nil || *r.Enabled
}

type K8sRunJobAction struct {
	Enabled *bool           `json:"enabled,omitempty"`
	Job     batchv1.JobSpec `json:"job"`
}

func (r K8sRunJobSpec) IsActive() bool {
	return r.Enabled == nil || *r.Enabled
}

// K8sRunJobSpec defines the desired state of K8sRunJob
type K8sRunJobSpec struct {
	Enabled    *bool                      `json:"enabled,omitempty"`
	Namespaces []string                   `json:"namespaces,omitempty"`
	Schedule   string                     `json:"schedule"`
	Actions    map[string]K8sRunJobAction `json:"actions"`
}

// K8sRunJobStatus defines the observed state of K8sRunJob
type K8sRunJobStatus struct {
	Conditions []metav1.Condition `json:"conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster
//+kubebuilder:printcolumn:name="Schedule",type="string",JSONPath=`.spec.schedule`
//+kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=`.status.conditions[?(@.type=="Ready")].status`

// K8sRunJob is the Schema for the k8srunjobs API
type K8sRunJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   K8sRunJobSpec   `json:"spec,omitempty"`
	Status K8sRunJobStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// K8sRunJobList contains a list of K8sRunJob
type K8sRunJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []K8sRunJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&K8sRunJob{}, &K8sRunJobList{})
}
