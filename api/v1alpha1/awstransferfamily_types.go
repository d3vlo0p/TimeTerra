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

type AwsTransferFamilyCommand string

const (
	AwsTransferFamilyCommandStop  AwsTransferFamilyCommand = "stop"
	AwsTransferFamilyCommandStart AwsTransferFamilyCommand = "start"
)

func (r AwsTransferFamilyCommand) String() string {
	return string(r)
}

func (r AwsTransferFamilyAction) IsActive() bool {
	return r.Enabled == nil || *r.Enabled
}

type AwsTransferFamilyAction struct {
	Enabled *bool `json:"enabled,omitempty"`
	// +kubebuilder:validation:Enum:=stop;start
	Command AwsTransferFamilyCommand `json:"command"`
}

type AwsTransferFamilyIdentifier struct {
	Id     string `json:"id"`
	Region string `json:"region"`
}

func (r AwsTransferFamilySpec) IsActive() bool {
	return r.Enabled == nil || *r.Enabled
}

// AwsTransferFamilySpec defines the desired state of AwsTransferFamily
type AwsTransferFamilySpec struct {
	Enabled         *bool                              `json:"enabled,omitempty"`
	Schedule        string                             `json:"schedule"`
	ServiceEndpoint *string                            `json:"serviceEndpoint,omitempty"`
	ServerIds       []AwsTransferFamilyIdentifier      `json:"serverIds"`
	Actions         map[string]AwsTransferFamilyAction `json:"actions"`
	Credentials     *AwsCredentialsSpec                `json:"credentials,omitempty"`
}

// AwsTransferFamilyStatus defines the observed state of AwsTransferFamily
type AwsTransferFamilyStatus struct {
	Conditions []metav1.Condition `json:"conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster
//+kubebuilder:printcolumn:name="Schedule",type="string",JSONPath=`.spec.schedule`
//+kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=`.status.conditions[?(@.type=="Ready")].status`

// AwsTransferFamily is the Schema for the awstransferfamilies API
type AwsTransferFamily struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AwsTransferFamilySpec   `json:"spec,omitempty"`
	Status AwsTransferFamilyStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AwsTransferFamilyList contains a list of AwsTransferFamily
type AwsTransferFamilyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AwsTransferFamily `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AwsTransferFamily{}, &AwsTransferFamilyList{})
}
