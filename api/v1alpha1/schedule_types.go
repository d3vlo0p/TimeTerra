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

func (r ScheduleAction) IsActive() bool {
	return r.Enabled == nil || *r.Enabled
}

// ScheduleAction defines the time to performe an action
type ScheduleAction struct {
	Enabled *bool  `json:"enabled,omitempty"`
	Cron    string `json:"cron"`
}

func (r ScheduleSpec) IsActive() bool {
	return r.Enabled == nil || *r.Enabled
}

type TimePeriod struct {
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Format=date-time
	Start metav1.Time `json:"start"`
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Format=date-time
	End metav1.Time `json:"end"`
}

// ScheduleSpec defines the desired state of Schedule
type ScheduleSpec struct {
	Enabled         *bool                     `json:"enabled,omitempty"`
	Actions         map[string]ScheduleAction `json:"actions,omitempty"`
	ActivePeriods   []TimePeriod              `json:"activePeriods,omitempty"`
	InactivePeriods []TimePeriod              `json:"inactivePeriods,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster
//+kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=`.status.conditions[?(@.type=="Ready")].status`

// Schedule is the Schema for the schedules API
type Schedule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec       ScheduleSpec `json:"spec,omitempty"`
	StatusType `json:",inline"`
}

//+kubebuilder:object:root=true

// ScheduleList contains a list of Schedule
type ScheduleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Schedule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Schedule{}, &ScheduleList{})
}
