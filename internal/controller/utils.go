package controller

import (
	"context"
	"fmt"

	corev1alpha1 "github.com/d3vlo0p/TimeTerra/api/v1alpha1"
	sc "github.com/d3vlo0p/TimeTerra/internal/cron"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func ResourceName(kind string, name string) string {
	return fmt.Sprintf("%s:%s", kind, name)
}

func contains(actions []string, action string) bool {
	for _, a := range actions {
		if a == action {
			return true
		}
	}
	return false
}

func rec[ActionType any, SpecType any](
	ctx context.Context,
	cli client.Client,
	cron *sc.ScheduleCron,
	instanceActions map[string]ActionType,
	instanceSpec SpecType,
	scheduleName string,
	resourceName string,
	job func(ctx context.Context, instanceSpec SpecType, action ActionType),
) (metav1.Condition, error) {
	logger := log.FromContext(ctx)

	schedule := &corev1alpha1.Schedule{}
	err := cli.Get(ctx, client.ObjectKey{Name: scheduleName}, schedule)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Schedule resource not found.")
			return metav1.Condition{
				LastTransitionTime: metav1.Now(),
				Status:             metav1.ConditionFalse,
				Type:               "Ready",
				Reason:             "Schedule not found",
				Message:            fmt.Sprintf("Schedule %s not found", scheduleName),
			}, nil
		}
		logger.Info("Failed to get Schedule resource. Re-running reconcile.")
		return metav1.Condition{}, err
	}

	scheduledActions := make([]string, 0)
	for action, id := range cron.GetActionsOfResource(scheduleName, resourceName) {
		entry := cron.Get(id)
		if entry.Valid() {
			// delete scheduled action if action is not defined in instance spec
			if _, found := instanceActions[action]; !found {
				logger.Info(fmt.Sprintf("Action %q is no more defined in spec, removing it from cron", action))
				cron.Remove(scheduleName, action, resourceName)
			} else {
				scheduledActions = append(scheduledActions, action)
			}
		}
	}

	for actionName, action := range instanceActions {
		actionName := actionName
		action := action
		if !contains(scheduledActions, actionName) {
			scheduleAction, found := schedule.Spec.Actions[actionName]
			if !found {
				logger.Info(fmt.Sprintf("action %q is not defined in schedule", actionName))
				return metav1.Condition{
					LastTransitionTime: metav1.Now(),
					Status:             metav1.ConditionFalse,
					Type:               "Ready",
					Reason:             "Acttion not found",
					Message:            fmt.Sprintf("action %s not found", actionName),
				}, nil
			}
			logger.Info(fmt.Sprintf("action %q is not scheduled, scheduling it with %q", actionName, scheduleAction.Cron))
			_, err := cron.Add(scheduleName, actionName, resourceName, scheduleAction.Cron, func() {
				logger.Info(fmt.Sprintf("action %q is running", actionName))
				job(ctx, instanceSpec, action)
			})
			if err != nil {
				logger.Info(fmt.Sprintf("failed to add cron job: %q", err))
				return metav1.Condition{}, err
			}
			logger.Info(fmt.Sprintf("action %q is scheduled", actionName))
		}
	}
	return metav1.Condition{
		LastTransitionTime: metav1.Now(),
		Status:             metav1.ConditionTrue,
		Type:               "Ready",
		Reason:             "Ready",
	}, nil
}
