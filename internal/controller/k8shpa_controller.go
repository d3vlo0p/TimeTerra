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

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	corev1alpha1 "github.com/d3vlo0p/TimeTerra/api/v1alpha1"
	"github.com/d3vlo0p/TimeTerra/internal/cron"
)

// K8sHpaReconciler reconciles a K8sHpa object
type K8sHpaReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Cron   *cron.ScheduleCron
}

//+kubebuilder:rbac:groups=core.timeterra.d3vlo0p.dev,resources=k8shpas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.timeterra.d3vlo0p.dev,resources=k8shpas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.timeterra.d3vlo0p.dev,resources=k8shpas/finalizers,verbs=update
//+kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the K8sHpa object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *K8sHpaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	logger := log.FromContext(ctx)

	logger.Info(fmt.Sprintf("reconciling object %#q", req.NamespacedName))

	resourceName := ResourceName("K8sHpa", req.Name)
	instance := &corev1alpha1.K8sHpa{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("AutoScaling resource not found. object must be deleted.")
			r.Cron.RemoveResource(resourceName)
			return ctrl.Result{}, nil
		}
		logger.Info("Failed to get AutoScaling resource. Re-running reconcile.")
		return ctrl.Result{}, err
	}

	if instance.Status.Conditions == nil {
		instance.Status.Conditions = make([]metav1.Condition, 0)
	}

	if instance.Spec.Enabled != nil && !*instance.Spec.Enabled {
		r.Cron.RemoveResource(resourceName)
		instance.Status.Conditions = addCondition(instance.Status.Conditions, metav1.Condition{
			LastTransitionTime: metav1.Now(),
			Type:               "Enabled",
			Status:             metav1.ConditionFalse,
			Reason:             "Disabled",
			Message:            "This Resource target is disabled",
		})
	} else {
		condition, err := reconcileResource(ctx, r.Client, r.Cron, instance.Spec.Actions, instance.Spec, instance.Spec.Schedule, resourceName, r.setAutoscaling)
		if err != nil {
			return ctrl.Result{}, err
		}
		instance.Status.Conditions = addCondition(instance.Status.Conditions, condition)
	}

	err = r.Status().Update(ctx, instance)
	if err != nil {
		logger.Info("Failed to update AutoScaling resource. Re-running reconcile.")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *K8sHpaReconciler) setAutoscaling(ctx context.Context, spec corev1alpha1.K8sHpaSpec, action corev1alpha1.K8sHpaAction) {
	logger := log.FromContext(ctx)
	selector := labels.SelectorFromSet(spec.LabelSelector.MatchLabels)

	for _, namespace := range spec.Namespaces {
		list := &autoscalingv2.HorizontalPodAutoscalerList{}
		err := r.List(ctx, list, &client.ListOptions{Namespace: namespace, LabelSelector: selector})
		if err != nil {
			logger.Error(err, "Failed to list HorizontalPodAutoscaler")
			return
		}
		logger.Info(fmt.Sprintf("updating HorizontalPodAutoscaler in namespace %q", namespace))
		minReplicas := int32(action.MinReplicas)
		maxReplicas := int32(action.MaxReplicas)
		for _, hpa := range list.Items {
			hpa.Spec.MinReplicas = &minReplicas
			hpa.Spec.MaxReplicas = maxReplicas
			err = r.Update(ctx, &hpa)
			if err != nil {
				logger.Error(err, "Failed to update HorizontalPodAutoscaler")
			}
		}
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *K8sHpaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.K8sHpa{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
