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

	sc "github.com/d3vlo0p/TimeTerra/internal/cron"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	corev1alpha1 "github.com/d3vlo0p/TimeTerra/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ScheduleReconciler reconciles a Schedule object
type ScheduleReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Cron     *sc.ScheduleCron
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=core.timeterra.d3vlo0p.dev,resources=schedules,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.timeterra.d3vlo0p.dev,resources=schedules/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.timeterra.d3vlo0p.dev,resources=schedules/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *ScheduleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info(fmt.Sprintf("reconciling object %#q", req.NamespacedName))
	instance := &corev1alpha1.Schedule{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Schedule resource not found. object must has been deleted.")
			return ctrl.Result{}, nil
		}
		logger.Info("Failed to get Schedule resource. Re-running reconcile.")
		return ctrl.Result{}, err
	}

	if instance.Status.Conditions == nil {
		instance.Status.Conditions = make([]metav1.Condition, 0)
	}

	// Check periods
	update := false
	if len(instance.Spec.ActivePeriods) > 0 {
		for i := 0; i < len(instance.Spec.ActivePeriods); i++ {
			period := instance.Spec.ActivePeriods[0]
			if period.Start.After(period.End.Time) {
				logger.Info("Start date is after end date, switching them")
				instance.Spec.ActivePeriods[i].End = period.Start
				instance.Spec.ActivePeriods[i].Start = period.End
				update = true
			}
		}
	}
	if len(instance.Spec.InactivePeriods) > 0 {
		for i := 0; i < len(instance.Spec.InactivePeriods); i++ {
			period := instance.Spec.InactivePeriods[0]
			if period.Start.After(period.End.Time) {
				logger.Info("Start date is after end date, switching them")
				instance.Spec.InactivePeriods[i].End = period.Start
				instance.Spec.InactivePeriods[i].Start = period.End
				update = true
			}
		}
	}
	if update {
		err = r.Update(ctx, instance)
		if err != nil {
			logger.Info("Failed to update Schedule resource. Re-running reconcile.")
			return ctrl.Result{}, err
		}
		// return and trigger another reconcile for the update
		return ctrl.Result{}, nil
	}

	err = r.reconcile(ctx, instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.Status().Update(ctx, instance)
	if err != nil {
		logger.Info("Failed to update Schedule resource status. Re-running reconcile.")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ScheduleReconciler) reconcile(ctx context.Context, instance *corev1alpha1.Schedule) error {
	logger := log.FromContext(ctx)
	scheduleName := instance.Name
	// checking if the cron expression of the actions is correct
	ret := false
	specActions := make([]string, 0)
	for action, c := range instance.Spec.Actions {
		specActions = append(specActions, action)
		actionType := ConditionTypeForAction(action)
		if !r.Cron.IsValidCron(c.Cron) {
			logger.Info(fmt.Sprintf("cron expression of action %s is invalid", action))
			addToConditions(&instance.Status.Conditions, metav1.Condition{
				LastTransitionTime: metav1.Now(),
				Status:             metav1.ConditionFalse,
				Type:               actionType,
				Reason:             "Invalid",
				Message:            fmt.Sprintf("cron expression of action %s is invalid", action),
			})
			ret = true
		} else if !c.IsActive() {
			logger.Info(fmt.Sprintf("action %s is not active", action))
			addToConditions(&instance.Status.Conditions, metav1.Condition{
				LastTransitionTime: metav1.Now(),
				Status:             metav1.ConditionFalse,
				Type:               actionType,
				Reason:             "NotActive",
				Message:            fmt.Sprintf("action %s is not active", action),
			})
		} else {
			logger.Info(fmt.Sprintf("action %s is active", action))
			addToConditions(&instance.Status.Conditions, metav1.Condition{
				LastTransitionTime: metav1.Now(),
				Status:             metav1.ConditionTrue,
				Type:               actionType,
				Reason:             "Ready",
				Message:            fmt.Sprintf("action %s is ready", action),
			})
		}
	}

	removeMissingActionFromConditions(&instance.Status.Conditions, specActions)

	if ret {
		return nil
	}

	scheduledActions := r.Cron.GetActions(scheduleName)
	for action, resources := range scheduledActions {
		for resource := range resources {
			// check if some activities has been removed from the cron but there are still active
			if _, ok := instance.Spec.Actions[action]; !ok {
				logger.Info(fmt.Sprintf("action %s is used by %s, but was removed from the schedule", action, resource))
				addToConditions(&instance.Status.Conditions, metav1.Condition{
					LastTransitionTime: metav1.Now(),
					Status:             metav1.ConditionFalse,
					Type:               "Ready",
					Reason:             "MissingAction",
					Message:            fmt.Sprintf("action %s is used by %s, but was removed from the schedule", action, resource),
				})
				r.Recorder.Eventf(instance, "Warning", "MissingAction", "action %s is used by %s, but was removed from the schedule", action, resource)
				return nil
			}
			// proceed to refresh spec on active cron
			updated := r.Cron.UpdateCronSpec(scheduleName, action, resource, instance.Spec.Actions[action].Cron)
			if !updated {
				logger.Info(fmt.Sprintf("failed to update resource %s cron spec for action %s", resource, action))
				r.Recorder.Eventf(instance, "Warning", "FailedUpdate", "failed to update resource %s cron spec for action %s", resource, action)
			} else {
				logger.Info(fmt.Sprintf("resource %s cron spec for action %s has been updated", resource, action))
				r.Recorder.Eventf(instance, "Normal", "Updated", "resource %s cron spec for action %s has been updated", resource, action)
			}
		}
	}

	if instance.Spec.IsActive() {
		addToConditions(&instance.Status.Conditions, metav1.Condition{
			LastTransitionTime: metav1.Now(),
			Status:             metav1.ConditionTrue,
			Type:               "Ready",
			Reason:             "Ready",
		})
	} else {
		addToConditions(&instance.Status.Conditions, metav1.Condition{
			LastTransitionTime: metav1.Now(),
			Status:             metav1.ConditionFalse,
			Type:               "Ready",
			Reason:             "Disabled",
		})
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ScheduleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.Schedule{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
