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
	Scheme *runtime.Scheme
	Cron   *sc.ScheduleCron
}

//+kubebuilder:rbac:groups=core.timeterra.d3vlo0p.dev,resources=schedules,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.timeterra.d3vlo0p.dev,resources=schedules/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.timeterra.d3vlo0p.dev,resources=schedules/finalizers,verbs=update

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *ScheduleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info(fmt.Sprintf("reconciling object %#q", req.NamespacedName))
	instance := &corev1alpha1.Schedule{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Schedule resource not found. object must be deleted.")
			return ctrl.Result{}, nil
		}
		logger.Info("Failed to get Schedule resource. Re-running reconcile.")
		return ctrl.Result{}, err
	}

	if instance.Status.Conditions == nil {
		instance.Status.Conditions = make([]metav1.Condition, 0)
	}

	condition, err := r.reconcile(ctx, instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(instance.Status.Conditions) > 0 {
		// add condition if current status is false or if is different than last one
		if condition.Status != metav1.ConditionTrue || condition.Status != instance.Status.Conditions[len(instance.Status.Conditions)-1].Status {
			instance.Status.Conditions = append(instance.Status.Conditions, condition)
		}
		// keep only last ten conditions
		if len(instance.Status.Conditions) > 10 {
			instance.Status.Conditions = instance.Status.Conditions[len(instance.Status.Conditions)-10:]
		}
	} else {
		instance.Status.Conditions = append(instance.Status.Conditions, condition)
	}
	err = r.Status().Update(ctx, instance)
	if err != nil {
		logger.Info("Failed to update Schedule resource status. Re-running reconcile.")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ScheduleReconciler) reconcile(ctx context.Context, instance *corev1alpha1.Schedule) (metav1.Condition, error) {
	logger := log.FromContext(ctx)
	scheduleName := instance.Name
	// checking if the cron expression of the actions is correct
	for action, c := range instance.Spec.Actions {
		if !r.Cron.IsValidCron(c.Cron) {
			logger.Info(fmt.Sprintf("cron expression of action %s is invalid", action))
			return metav1.Condition{
				LastTransitionTime: metav1.Now(),
				Status:             metav1.ConditionFalse,
				Type:               "Ready",
				Reason:             "Action is invalid",
				Message:            fmt.Sprintf("cron expression of action %s is invalid", action),
			}, nil
		}
	}

	activeActions := r.Cron.GetActions(scheduleName)
	for action, resources := range activeActions {
		logger.Info(fmt.Sprintf("action %s is active", action))
		for resource, id := range resources {
			logger.Info(fmt.Sprintf("action %s is used by %s", action, resource))
			// check if some activities has been removed from the cron but there are still active
			if _, ok := instance.Spec.Actions[action]; !ok {
				logger.Info(fmt.Sprintf("action %s has been removed from the schedule", action))
				return metav1.Condition{
					LastTransitionTime: metav1.Now(),
					Status:             metav1.ConditionFalse,
					Type:               "Ready",
					Reason:             "Action is used",
					Message:            fmt.Sprintf("action %s is used by %s, but was removed from the schedule", action, resource),
				}, nil
			}
			// proceed to update spec on active cron
			updated := r.Cron.UpdateCronSpec(id, instance.Spec.Actions[action].Cron)
			if !updated {
				logger.Info(fmt.Sprintf("failed to update cron %d spec of action %s", id, action))
			} else {
				logger.Info(fmt.Sprintf("cron %d spec of action %s updated", id, action))
			}
		}
	}

	return metav1.Condition{
		LastTransitionTime: metav1.Now(),
		Status:             metav1.ConditionTrue,
		Type:               "Ready",
		Reason:             "Ready",
	}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ScheduleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.Schedule{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
