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

package controller

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	v1alpha1 "github.com/d3vlo0p/TimeTerra/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ScheduleReconciler reconciles a Schedule object
type ScheduleReconciler struct {
	BaseReconciler
}

//+kubebuilder:rbac:groups=timeterra.d3vlo0p.dev,resources=schedules,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=timeterra.d3vlo0p.dev,resources=schedules/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=timeterra.d3vlo0p.dev,resources=schedules/finalizers,verbs=update

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *ScheduleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("reconciling schedule")

	instance := &v1alpha1.Schedule{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Schedule resource not found. object must has been deleted")
			return ctrl.Result{}, nil
		}
		logger.Info("Failed to get Schedule resource. Re-running reconcile")
		return ctrl.Result{}, err
	}

	if instance.Status.Conditions == nil {
		instance.Status.Conditions = make([]metav1.Condition, 0)
	}

	// Validate periods
	hasInvalidPeriod := false
	if len(instance.Spec.ActivePeriods) > 0 {
		for _, period := range instance.Spec.ActivePeriods {
			if period.Start.After(period.End.Time) {
				logger.Info("Start date is after end date in active period", "start", period.Start, "end", period.End)
				r.Recorder.Eventf(instance, corev1.EventTypeWarning, "InvalidPeriod", "Start date %s is after end date %s in ActivePeriods", period.Start.String(), period.End.String())
				hasInvalidPeriod = true
			}
		}
	}
	if len(instance.Spec.InactivePeriods) > 0 {
		for _, period := range instance.Spec.InactivePeriods {
			if period.Start.After(period.End.Time) {
				logger.Info("Start date is after end date in inactive period", "start", period.Start, "end", period.End)
				r.Recorder.Eventf(instance, corev1.EventTypeWarning, "InvalidPeriod", "Start date %s is after end date %s in InactivePeriods", period.Start.String(), period.End.String())
				hasInvalidPeriod = true
			}
		}
	}
	if hasInvalidPeriod {
		addToConditions(&instance.Status.Conditions, metav1.Condition{
			LastTransitionTime: metav1.Now(),
			Type:               "Ready",
			Status:             metav1.ConditionFalse,
			Reason:             "InvalidPeriod",
			Message:            "Start date must be before end date in periods",
		})
		return ctrl.Result{}, r.Status().Update(ctx, instance)
	}

	err = r.reconcile(ctx, instance)
	if err != nil {
		r.Recorder.Eventf(instance, corev1.EventTypeWarning, "ReconcileError", "Reconcile error: %s", err.Error())
		return ctrl.Result{}, err
	}

	err = r.Status().Update(ctx, instance)
	if err != nil {
		r.Recorder.Eventf(instance, corev1.EventTypeWarning, "ReconcileError", "Reconcile error: %s", err.Error())
		logger.Info("Failed to update Schedule resource status. Re-running reconcile")
		return ctrl.Result{}, err
	}

	r.Recorder.Eventf(instance, corev1.EventTypeNormal, "ReconcileSuccess", "Reconcile succeeded")
	return ctrl.Result{}, nil
}

func (r *ScheduleReconciler) reconcile(ctx context.Context, instance *v1alpha1.Schedule) error {
	logger := log.FromContext(ctx).WithValues("schedule", instance.Name)
	scheduleName := instance.Name
	// checking if the cron expression of the actions is correct
	ret := false
	specActions := make([]string, 0)
	for action, c := range instance.Spec.Actions {
		specActions = append(specActions, action)
		actionType := conditionTypeForAction(action)
		if !r.Cron.IsValidCron(c.Cron) {
			logger.Info(fmt.Sprintf("cron expression of action %q is invalid", action))
			addToConditions(&instance.Status.Conditions, metav1.Condition{
				LastTransitionTime: metav1.Now(),
				Type:               actionType,
				Status:             metav1.ConditionFalse,
				Reason:             "InvalidCronExpression",
				Message:            fmt.Sprintf("cron expression %q is invalid", c.Cron),
			})
			ret = true
		} else if !c.IsActive() {
			logger.Info(fmt.Sprintf("action %q is not active", action))
			addToConditions(&instance.Status.Conditions, metav1.Condition{
				LastTransitionTime: metav1.Now(),
				Type:               actionType,
				Status:             metav1.ConditionFalse,
				Reason:             "NotActive",
			})
		} else {
			logger.Info(fmt.Sprintf("action %q is active", action))
			addToConditions(&instance.Status.Conditions, metav1.Condition{
				LastTransitionTime: metav1.Now(),
				Type:               actionType,
				Status:             metav1.ConditionTrue,
				Reason:             "Active",
			})
		}
	}

	removeMissingActionFromConditions(&instance.Status.Conditions, specActions)

	if ret {
		addToConditions(&instance.Status.Conditions, metav1.Condition{
			LastTransitionTime: metav1.Now(),
			Type:               "Ready",
			Status:             metav1.ConditionFalse,
			Reason:             "InvalidCronExpression",
			Message:            "One or more action cron expressions are invalid",
		})
		return nil
	}

	scheduledActions := r.Cron.GetActions(scheduleName)
	for action, resources := range scheduledActions {
		for resource := range resources {
			// check if some activities has been removed from the cron but there are still active
			if _, ok := instance.Spec.Actions[action]; !ok {
				logger.Info(fmt.Sprintf("action %q is used by %q, but was removed from the schedule", action, resource))
				addToConditions(&instance.Status.Conditions, metav1.Condition{
					LastTransitionTime: metav1.Now(),
					Type:               "Ready",
					Status:             metav1.ConditionFalse,
					Reason:             "MissingAction",
					Message:            fmt.Sprintf("action %q is used by %q, but was removed from the schedule", action, resource),
				})
				r.Recorder.Eventf(instance, corev1.EventTypeWarning, "MissingAction", "action %q is used by %q, but was removed from the schedule", action, resource)
				return nil
			}
			// proceed to refresh spec on active cron
			updated := r.Cron.UpdateCronSpec(scheduleName, action, resource, instance.Spec.Actions[action].Cron)
			if !updated {
				logger.Info(fmt.Sprintf("failed to update resource %q cron spec for action %q", resource, action))
				r.Recorder.Eventf(instance, corev1.EventTypeWarning, "FailedUpdate", "failed to update resource %q cron spec for action %q", resource, action)
			} else {
				logger.Info(fmt.Sprintf("resource %q cron spec for action %q has been updated", resource, action))
				r.Recorder.Eventf(instance, corev1.EventTypeNormal, "Updated", "resource %q cron spec for action %q has been updated", resource, action)
			}
		}
	}

	if !instance.Spec.IsActive() {
		addToConditions(&instance.Status.Conditions, metav1.Condition{
			LastTransitionTime: metav1.Now(),
			Type:               "Ready",
			Status:             metav1.ConditionFalse,
			Reason:             "Disabled",
		})
	} else {
		addToConditions(&instance.Status.Conditions, metav1.Condition{
			LastTransitionTime: metav1.Now(),
			Type:               "Ready",
			Status:             metav1.ConditionTrue,
			Reason:             "Active",
		})
	}
	return nil
}

func removeMissingActionFromConditions(conditions *[]metav1.Condition, actions []string) {
	if conditions == nil || len(*conditions) == 0 {
		return
	}
	activeActionTypes := make(map[string]bool, len(actions))
	for _, action := range actions {
		activeActionTypes[conditionTypeForAction(action)] = true
	}
	filtered := make([]metav1.Condition, 0, len(*conditions))
	for _, c := range *conditions {
		// Only consider Action- prefixed conditions for pruning
		if strings.HasPrefix(c.Type, "Action-") {
			if activeActionTypes[c.Type] {
				filtered = append(filtered, c)
			}
		} else {
			// Keep non-action conditions like "Ready"
			filtered = append(filtered, c)
		}
	}
	*conditions = filtered
}

// SetupWithManager sets up the controller with the Manager.
func (r *ScheduleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Schedule{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Named("schedule").
		Complete(r)
}
