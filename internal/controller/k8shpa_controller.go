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
	"time"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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
			logger.Info("K8sHpa resource not found. object must be deleted.")
			r.Cron.RemoveResource(resourceName)
			return ctrl.Result{}, nil
		}
		logger.Info("Failed to get K8sHpa resource. Re-running reconcile.")
		return ctrl.Result{}, err
	}

	if instance.Status.Conditions == nil {
		instance.Status.Conditions = make([]metav1.Condition, 0)
	}

	if instance.Spec.Enabled != nil && !*instance.Spec.Enabled {
		disableResource(r.Cron, &instance.Status.Conditions, resourceName)
	} else {
		err := reconcileResource(ctx, req, r.Client, r.Cron, instance.Spec.Actions, instance.Spec.Schedule, resourceName, r.setAutoscaling, &instance.Status.Conditions)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	err = r.Status().Update(ctx, instance)
	if err != nil {
		logger.Info("Failed to update K8sHpa resource. Re-running reconcile.")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *K8sHpaReconciler) setAutoscaling(ctx context.Context, key types.NamespacedName, actionName string) {
	logger := log.FromContext(ctx)
	start := time.Now()
	obj := &corev1alpha1.K8sHpa{}
	err := r.Get(ctx, key, obj)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("K8sHpa resource not found. object must be deleted.")
			return
		}
		logger.Info("Failed to get K8sHpa resource.")
		return
	}
	action, ok := obj.Spec.Actions[actionName]
	if !ok {
		logger.Info("Action not found")
		return
	}

	selector := labels.SelectorFromSet(obj.Spec.LabelSelector.MatchLabels)
	if len(obj.Spec.Namespaces) == 0 {
		obj.Spec.Namespaces = []string{metav1.NamespaceAll}
	}
	actionType := ConditionTypeForAction(actionName)
	errorsList := make([]string, 0)
	for _, namespace := range obj.Spec.Namespaces {
		list := &autoscalingv2.HorizontalPodAutoscalerList{}
		err := r.List(ctx, list, &client.ListOptions{Namespace: namespace, LabelSelector: selector})
		if err != nil {
			msg := fmt.Sprintf("Failed to list HorizontalPodAutoscaler in namespace %q", namespace)
			logger.Error(err, msg)
			errorsList = append(errorsList, msg)
		} else {
			minReplicas := int32(action.MinReplicas)
			maxReplicas := int32(action.MaxReplicas)
			for _, hpa := range list.Items {
				hpa.Spec.MinReplicas = &minReplicas
				hpa.Spec.MaxReplicas = maxReplicas
				err = r.Update(ctx, &hpa)
				if err != nil {
					msg := fmt.Sprintf("Failed to update HorizontalPodAutoscaler %q/%q", hpa.Namespace, hpa.Name)
					logger.Error(err, msg)
					errorsList = append(errorsList, msg)
				}
				logger.Info(fmt.Sprintf("HorizontalPodAutoscaler %q/%q updated min:%d max:%d", hpa.Namespace, hpa.Name, minReplicas, maxReplicas))
			}
		}
	}
	if len(errorsList) > 0 {
		addToConditions(&obj.Status.Conditions, metav1.Condition{
			LastTransitionTime: metav1.Now(),
			Type:               actionType,
			Status:             metav1.ConditionFalse,
			Reason:             "Failed",
			Message:            strings.Join(errorsList, ";"),
		})
	} else {
		addToConditions(&obj.Status.Conditions, metav1.Condition{
			LastTransitionTime: metav1.Now(),
			Type:               actionType,
			Status:             metav1.ConditionTrue,
			Reason:             "Success",
			Message:            fmt.Sprintf("Action %q, last execution start:%q end:%q", actionName, start.Format(time.RFC3339), time.Now().Format(time.RFC3339)),
		})
	}
	r.Status().Update(ctx, obj)
}

// SetupWithManager sets up the controller with the Manager.
func (r *K8sHpaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.K8sHpa{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
