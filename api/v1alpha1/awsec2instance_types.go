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

type AwsEc2InstanceCommand string

const (
	AwsEc2InstanceCommandStop      AwsEc2InstanceCommand = "stop"
	AwsEc2InstanceCommandStart     AwsEc2InstanceCommand = "start"
	AwsEc2InstanceCommandHibernate AwsEc2InstanceCommand = "hibernate"
)

func (r AwsEc2InstanceCommand) String() string {
	return string(r)
}

func (r AwsEc2InstanceAction) IsActive() bool {
	return r.Enabled == nil || *r.Enabled
}

type AwsEc2InstanceAction struct {
	Enabled *bool `json:"enabled,omitempty"`
	// +kubebuilder:validation:Enum:=stop;start;hibernate
	Command AwsEc2InstanceCommand `json:"command"`
	Force   *bool                 `json:"force,omitempty"`
}

type AwsEc2InstanceIdentifier struct {
	Id     string `json:"id"`
	Region string `json:"region"`
}

func (r AwsEc2InstanceSpec) IsActive() bool {
	return r.Enabled == nil || *r.Enabled
}

// AwsEc2InstanceSpec defines the desired state of AwsEc2Instance
type AwsEc2InstanceSpec struct {
	Enabled         *bool                           `json:"enabled,omitempty"`
	Schedule        string                          `json:"schedule"`
	ServiceEndpoint *string                         `json:"serviceEndpoint,omitempty"`
	Instances       []AwsEc2InstanceIdentifier      `json:"instances"`
	Actions         map[string]AwsEc2InstanceAction `json:"actions"`
}

// AwsEc2InstanceStatus defines the observed state of AwsEc2Instance
type AwsEc2InstanceStatus struct {
	Conditions []metav1.Condition `json:"conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster
//+kubebuilder:printcolumn:name="Schedule",type="string",JSONPath=`.spec.schedule`
//+kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=`.status.conditions[?(@.type=="Ready")].status`

// AwsEc2Instance is the Schema for the awsec2instances API
type AwsEc2Instance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AwsEc2InstanceSpec   `json:"spec,omitempty"`
	Status AwsEc2InstanceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AwsEc2InstanceList contains a list of AwsEc2Instance
type AwsEc2InstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AwsEc2Instance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AwsEc2Instance{}, &AwsEc2InstanceList{})
}
