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

// AutoscalingReconciler reconciles a Autoscaling object
type AutoscalingReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Cron   *sc.ScheduleCron
}

//+kubebuilder:rbac:groups=core.timeterra.d3vlo0p.dev,resources=autoscalings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.timeterra.d3vlo0p.dev,resources=autoscalings/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.timeterra.d3vlo0p.dev,resources=autoscalings/finalizers,verbs=update

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *AutoscalingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info(fmt.Sprintf("reconciling object %#q", req.NamespacedName))

	resourceName := ResourceName("Autoscaling", req.Name)
	instance := &corev1alpha1.Autoscaling{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("AutoScaling resource not found. object must be deleted.")
			return ctrl.Result{}, nil
		}
		logger.Info("Failed to get AutoScaling resource. Re-running reconcile.")
		return ctrl.Result{}, err
	}

	if instance.Status.Conditions == nil {
		instance.Status.Conditions = make([]metav1.Condition, 0)
	}

	condition, err := rec(ctx, r.Client, r.Cron, instance.Spec.Actions, instance.Spec, instance.Spec.Schedule, resourceName, r.setAutoscaling)
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
		logger.Info("Failed to update AutoScaling resource. Re-running reconcile.")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *AutoscalingReconciler) setAutoscaling(ctx context.Context, spec corev1alpha1.AutoscalingSpec, action corev1alpha1.AutoscalingAction) {
	// TODO: update HPA settings
}

// SetupWithManager sets up the controller with the Manager.
func (r *AutoscalingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.Autoscaling{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
