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

type AwsRdsAuroraClusterAction struct {
	Command AwsRdsAuroraClusterCommand `json:"command"`
}

type AwsRdsAuroraClusterIdentifier struct {
	Identifier string `json:"identifier"`
	Region     string `json:"region"`
}

// AwsRdsAuroraClusterSpec defines the desired state of AwsRdsAuroraCluster
type AwsRdsAuroraClusterSpec struct {
	Enabled              *bool                                `json:"enabled,omitempty"`
	Schedule             string                               `json:"schedule"`
	ServiceEndpoint      *string                              `json:"serviceEndpoint,omitempty"`
	DBClusterIdentifiers []AwsRdsAuroraClusterIdentifier      `json:"dbClusterIdentifiers"`
	Actions              map[string]AwsRdsAuroraClusterAction `json:"actions"`
}

// AwsRdsAuroraClusterStatus defines the observed state of AwsRdsAuroraCluster
type AwsRdsAuroraClusterStatus struct {
	Conditions []metav1.Condition `json:"conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// AwsRdsAuroraCluster is the Schema for the awsrdsauroraclusters API
type AwsRdsAuroraCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AwsRdsAuroraClusterSpec   `json:"spec,omitempty"`
	Status AwsRdsAuroraClusterStatus `json:"status,omitempty"`
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
