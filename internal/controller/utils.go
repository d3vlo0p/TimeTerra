package controller

import (
	"fmt"
)

func resourceName(kind string, name string) string {
	return fmt.Sprintf("%s:%s", kind, name)
}

func conditionTypeForAction(name string) string {
	return fmt.Sprintf("Action-%s", name)
}

func contains(actions []string, action string) bool {
	for _, a := range actions {
		if a == action {
			return true
		}
	}
	return false
}

func defaultNamespace(operatorNamespace string, namespace string) string {
	if namespace == "" {
		return operatorNamespace
	}
	return namespace
}
