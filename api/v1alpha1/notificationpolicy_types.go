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

type NotificationPolicyApiSpec struct {
	Url string `json:"url"`
	// +kubebuilder:validation:Enum:=post;put;patch
	Method string `json:"method"`
}

func (r NotificationPolicySpec) IsActive() bool {
	return r.Enabled == nil || *r.Enabled
}

type NotificationType string

const (
	NotificationTypeApi NotificationType = "api"
)

func (r NotificationType) String() string {
	return string(r)
}

// NotificationPolicySpec defines the desired state of NotificationPolicy
type NotificationPolicySpec struct {
	Enabled   *bool    `json:"enabled,omitempty"`
	Schedules []string `json:"schedules"`
	// +kubebuilder:validation:Enum:=api
	Type NotificationType           `json:"type"`
	Api  *NotificationPolicyApiSpec `json:"api,omitempty"`
}

// NotificationPolicyStatus defines the observed state of NotificationPolicy
type NotificationPolicyStatus struct {
	Conditions []metav1.Condition `json:"conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// NotificationPolicy is the Schema for the notificationpolicies API
type NotificationPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NotificationPolicySpec   `json:"spec,omitempty"`
	Status NotificationPolicyStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NotificationPolicyList contains a list of NotificationPolicy
type NotificationPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NotificationPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NotificationPolicy{}, &NotificationPolicyList{})
}
