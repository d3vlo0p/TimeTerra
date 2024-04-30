package controller

import (
	"context"
	"fmt"
	"strings"

	corev1alpha1 "github.com/d3vlo0p/TimeTerra/api/v1alpha1"
	sc "github.com/d3vlo0p/TimeTerra/internal/cron"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func ResourceName(kind string, name string) string {
	return fmt.Sprintf("%s:%s", kind, name)
}

func ConditionTypeForAction(name string) string {
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

func disableResource(cron *sc.ScheduleCron, conditions *[]metav1.Condition, resourceName string) {
	cron.RemoveResource(resourceName)
	// remove condition if Type starts with "Action"
	if len(*conditions) > 0 {
		newConditions := make([]metav1.Condition, 0)
		for _, c := range *conditions {
			if !strings.HasPrefix(c.Type, "Action") {
				newConditions = append(newConditions, c)
			}
		}
		*conditions = newConditions
	}
	addToConditions(conditions, metav1.Condition{
		LastTransitionTime: metav1.Now(),
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		Reason:             "Disabled",
		Message:            "This Resource target is disabled",
	})
}

func reconcileResource[ActionType any](
	ctx context.Context,
	req ctrl.Request,
	cli client.Client,
	cron *sc.ScheduleCron,
	instanceActions map[string]ActionType,
	scheduleName string,
	resourceName string,
	job func(ctx context.Context, key types.NamespacedName, actionName string),
	conditions *[]metav1.Condition,
) error {
	logger := log.FromContext(ctx)

	schedule := &corev1alpha1.Schedule{}
	err := cli.Get(ctx, client.ObjectKey{Name: scheduleName}, schedule)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Schedule resource not found.")
			addToConditions(conditions, metav1.Condition{
				LastTransitionTime: metav1.Now(),
				Type:               "Ready",
				Status:             metav1.ConditionFalse,
				Reason:             "Schedule not found",
				Message:            fmt.Sprintf("Schedule %s not found", scheduleName),
			})
			return nil
		}
		logger.Info("Failed to get Schedule resource. Re-running reconcile.")
		return err
	}

	scheduledActions := make([]string, 0)
	for action, id := range cron.GetActionsOfResource(scheduleName, resourceName) {
		entry := cron.Get(id)
		if entry.Valid() {
			// delete scheduled action if action is not defined in instance spec
			if _, found := instanceActions[action]; !found {
				logger.Info(fmt.Sprintf("Action %q is no more defined in spec, removing it from cron", action))
				cron.Remove(scheduleName, action, resourceName)
				removeFromConditions(conditions, ConditionTypeForAction(action))
			} else {
				scheduledActions = append(scheduledActions, action)
			}
		}
	}

	for actionName := range instanceActions {
		actionName := actionName
		if !contains(scheduledActions, actionName) {
			scheduleAction, found := schedule.Spec.Actions[actionName]
			if !found {
				logger.Info(fmt.Sprintf("action %q is not defined in schedule %s", actionName, scheduleName))
				addToConditions(conditions, metav1.Condition{
					LastTransitionTime: metav1.Now(),
					Type:               "Ready",
					Status:             metav1.ConditionFalse,
					Reason:             "Acttion not found",
					Message:            fmt.Sprintf("action %s not found in %s", actionName, scheduleName),
				})
				return nil
			}
			logger.Info(fmt.Sprintf("action %q is not scheduled, scheduling it with %q", actionName, scheduleAction.Cron))
			_, err := cron.Add(scheduleName, actionName, resourceName, scheduleAction.Cron, func() {
				logger.Info(fmt.Sprintf("action %q is running", actionName))
				job(ctx, req.NamespacedName, actionName)
			})
			if err != nil {
				logger.Info(fmt.Sprintf("failed to add cron job: %q", err))
				return err
			}
			logger.Info(fmt.Sprintf("action %q is scheduled", actionName))
			addToConditions(conditions, metav1.Condition{
				LastTransitionTime: metav1.Now(),
				Type:               ConditionTypeForAction(actionName),
				Status:             metav1.ConditionTrue,
				Reason:             "Scheduled",
				Message:            fmt.Sprintf("action %s is scheduled", actionName),
			})
		}
	}
	addToConditions(conditions, metav1.Condition{
		LastTransitionTime: metav1.Now(),
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             "Ready",
	})
	return nil
}

func addToConditions(conditions *[]metav1.Condition, condition metav1.Condition) {
	meta.SetStatusCondition(conditions, condition)
}

func removeFromConditions(conditions *[]metav1.Condition, conditionType string) {
	meta.RemoveStatusCondition(conditions, conditionType)
}
