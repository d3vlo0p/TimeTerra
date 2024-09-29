package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Status struct {
	Conditions []metav1.Condition `json:"conditions"`
}

type StatusType struct {
	Status Status `json:"status,omitempty"`
}
