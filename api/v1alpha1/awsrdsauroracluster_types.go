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

type AwsRdsAuroraClusterCommand string

const (
	AwsRdsAuroraClusterCommandStop  AwsRdsAuroraClusterCommand = "stop"
	AwsRdsAuroraClusterCommandStart AwsRdsAuroraClusterCommand = "start"
)

func (r AwsRdsAuroraClusterCommand) String() string {
	return string(r)
}

func (r AwsRdsAuroraClusterAction) IsActive() bool {
	return r.Enabled == nil || *r.Enabled
}

type AwsRdsAuroraClusterAction struct {
	Enabled *bool `json:"enabled,omitempty"`
	// +kubebuilder:validation:Enum:=stop;start
	Command AwsRdsAuroraClusterCommand `json:"command"`
}

type AwsRdsAuroraClusterIdentifier struct {
	Identifier string `json:"identifier"`
	Region     string `json:"region"`
}

func (r AwsRdsAuroraClusterSpec) IsActive() bool {
	return r.Enabled == nil || *r.Enabled
}

// AwsRdsAuroraClusterSpec defines the desired state of AwsRdsAuroraCluster
type AwsRdsAuroraClusterSpec struct {
	Enabled              *bool                                `json:"enabled,omitempty"`
	Schedule             string                               `json:"schedule"`
	ServiceEndpoint      *string                              `json:"serviceEndpoint,omitempty"`
	DBClusterIdentifiers []AwsRdsAuroraClusterIdentifier      `json:"dbClusterIdentifiers"`
	Actions              map[string]AwsRdsAuroraClusterAction `json:"actions"`
	Credentials          *AwsCredentialsSpec                  `json:"credentials,omitempty"`
}

func (r AwsRdsAuroraCluster) IsActive() bool {
	return r.Spec.Enabled == nil || *r.Spec.Enabled
}

func (r AwsRdsAuroraCluster) GetSchedule() string {
	return r.Spec.Schedule
}

func (r AwsRdsAuroraCluster) GetStatus() Status {
	return r.Status
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster
//+kubebuilder:printcolumn:name="Schedule",type="string",JSONPath=`.spec.schedule`
//+kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=`.status.conditions[?(@.type=="Ready")].status`

// AwsRdsAuroraCluster is the Schema for the awsrdsauroraclusters API
type AwsRdsAuroraCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec       AwsRdsAuroraClusterSpec `json:"spec,omitempty"`
	StatusType `json:",inline"`
}

//+kubebuilder:object:root=true

// AwsRdsAuroraClusterList contains a list of AwsRdsAuroraCluster
type AwsRdsAuroraClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AwsRdsAuroraCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AwsRdsAuroraCluster{}, &AwsRdsAuroraClusterList{})
}
