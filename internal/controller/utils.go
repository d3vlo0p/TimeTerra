package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

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

type Activable interface {
	IsActive() bool
}

func reconcileResource[ActionType Activable](
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
				Reason:             "Error",
				Message:            fmt.Sprintf("Schedule %s not found", scheduleName),
			})
			return nil
		}
		logger.Info("Failed to get Schedule resource. Re-running reconcile.")
		return err
	}

	removedActions := cron.RemoveActionsOfResourceFromNonCurrentSchedule(scheduleName, resourceName)
	if len(removedActions) > 0 {
		logger.Info("Schedule has been changed, removed cron jobs of previous schedule")
		for _, action := range removedActions {
			removeFromConditions(conditions, ConditionTypeForAction(action))
		}
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

	for actionName, action := range instanceActions {
		actionName := actionName
		if !action.IsActive() {
			logger.Info(fmt.Sprintf("Action %q is not active, removing it from cron", actionName))
			cron.Remove(scheduleName, actionName, resourceName)
			addToConditions(conditions, metav1.Condition{
				LastTransitionTime: metav1.Now(),
				Type:               ConditionTypeForAction(actionName),
				Status:             metav1.ConditionFalse,
				Reason:             "Disabled",
				Message:            "This Action target is disabled",
			})
			continue
		}
		if !contains(scheduledActions, actionName) {
			scheduleAction, found := schedule.Spec.Actions[actionName]
			if !found {
				logger.Info(fmt.Sprintf("action %q is not defined in schedule %s", actionName, scheduleName))
				addToConditions(conditions, metav1.Condition{
					LastTransitionTime: metav1.Now(),
					Type:               "Ready",
					Status:             metav1.ConditionFalse,
					Reason:             "MissingAction",
					Message:            fmt.Sprintf("action %s not found in %s", actionName, scheduleName),
				})
				continue
			}
			logger.Info(fmt.Sprintf("action %q is not scheduled, scheduling it with %q", actionName, scheduleAction.Cron))
			_, err := cron.Add(scheduleName, actionName, resourceName, scheduleAction.Cron, func() {
				// retrive schedule for cheking if it is active
				// we do the check here because the check is simpler, and it avoids us having to delete and create objects in cron schedule
				s := &corev1alpha1.Schedule{}
				err := cli.Get(ctx, client.ObjectKey{Name: scheduleName}, s)
				if err != nil {
					logger.Error(err, "Cron run, failed to get schedule", "name", scheduleName)
					return
				}

				if !s.Spec.IsActive() {
					logger.Info(fmt.Sprintf("schedule %q is not active, skipping execution", scheduleName))
					return
				}

				if a, ok := s.Spec.Actions[actionName]; !ok || !a.IsActive() {
					logger.Info(fmt.Sprintf("action %q is not active, skipping execution", actionName))
					return
				}

				//logic for managing time periods
				now := time.Now()
				if len(s.Spec.ActivePeriods) > 0 {
					active := false
					// chack if current date is inside an Active Period, if not then skip exec
					for _, p := range s.Spec.ActivePeriods {
						if (now.After(p.Start.Time) && now.Before(p.End.Time)) || now.Equal(p.Start.Time) || now.Equal(p.End.Time) {
							active = true
							break
						}
					}
					if !active {
						logger.Info(fmt.Sprintf("Schedule %q is outside an active period, skipping execution", scheduleName))
						return
					}
				}
				if len(s.Spec.InactivePeriods) > 0 {
					active := true
					// chack if current date is inside an Inactive Period, if it is then skip exec
					for _, p := range s.Spec.InactivePeriods {
						if (now.After(p.Start.Time) && now.Before(p.End.Time)) || now.Equal(p.Start.Time) || now.Equal(p.End.Time) {
							active = false
							break
						}
					}
					if !active {
						logger.Info(fmt.Sprintf("Schedule %q is inside an inactive period, skipping execution", scheduleName))
						return
					}
				}

				logger.Info(fmt.Sprintf("action %q is starting for resource %q", actionName, resourceName))
				job(ctx, req.NamespacedName, actionName)
			})
			if err != nil {
				logger.Error(err, "failed to add cron job")
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

func removeMissingActionFromConditions(conditions *[]metav1.Condition, actions []string) {
	// remove from orphanedConditions all current action, so will remain only conditions without actions
	orphanedConditions := *conditions
	for _, action := range actions {
		cond := meta.FindStatusCondition(orphanedConditions, ConditionTypeForAction(action))
		if cond != nil {
			// find cond inside orphanedConditions and remove it
			for _, c := range orphanedConditions {
				if c.Type == cond.Type {
					removeFromConditions(&orphanedConditions, c.Type)
					break
				}
			}
		}
	}
	// remove orphaned conditions
	for _, c := range orphanedConditions {
		removeFromConditions(conditions, c.Type)
	}
}
