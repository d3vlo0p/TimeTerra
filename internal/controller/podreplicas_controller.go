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
	cron "github.com/robfig/cron/v3"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	corev1alpha1 "github.com/d3vlo0p/TimeTerra/api/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// PodReplicasReconciler reconciles a PodReplicas object
type PodReplicasReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Cron   *sc.ScheduleCron
}

//+kubebuilder:rbac:groups=core.timeterra.d3vlo0p.dev,resources=podreplicas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.timeterra.d3vlo0p.dev,resources=podreplicas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.timeterra.d3vlo0p.dev,resources=podreplicas/finalizers,verbs=update

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *PodReplicasReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info(fmt.Sprintf("reconciling object %#q", req.NamespacedName))

	resourceName := ResourceName("PodReplicas", req.Name)
	instance := &corev1alpha1.PodReplicas{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("PodReplicas resource not found. object must be deleted.")
			r.Cron.RemoveResource(resourceName)
			return ctrl.Result{}, nil
		}
		logger.Info("Failed to get PodReplicas resource. Re-running reconcile.")
		return ctrl.Result{}, err
	}

	if instance.Status.Conditions == nil {
		instance.Status.Conditions = make([]metav1.Condition, 0)
	}

	schedule := &corev1alpha1.Schedule{}
	err = r.Get(ctx, client.ObjectKey{Name: instance.Spec.Schedule}, schedule)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Schedule resource not found. Re-running reconcile.")
			return ctrl.Result{}, err
		}
		logger.Info("Failed to get Schedule resource. Re-running reconcile.")
		return ctrl.Result{}, err
	}

	condition, err := r.reconcile(ctx, instance, schedule, resourceName)
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
		logger.Info(fmt.Sprintf("failed to update status: %q", err))
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *PodReplicasReconciler) reconcile(ctx context.Context, instance *corev1alpha1.PodReplicas, schedule *corev1alpha1.Schedule, resourceName string) (metav1.Condition, error) {
	logger := log.FromContext(ctx)
	scheduleName := schedule.Name
	scheduledActions := []string{}
	for action, id := range r.Cron.GetActionsOfResource(scheduleName, resourceName) {
		entry := r.Cron.Get(id)
		if entry.Valid() {
			// delete scheduled action if action is not defined in spec
			if _, found := instance.Spec.Actions[action]; !found {
				logger.Info(fmt.Sprintf("job action %q is no more defined in spec, removing it from status", action))
				r.Cron.Remove(scheduleName, action, resourceName)
			} else {
				scheduledActions = append(scheduledActions, action)
			}
		}
	}

	for actionName, action := range instance.Spec.Actions {
		actionName := actionName
		action := action
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
		if !contains(scheduledActions, actionName) {
			logger.Info(fmt.Sprintf("action %q is not scheduled, scheduling it with %q", actionName, scheduleAction.Cron))
			_, err := r.Cron.Add(scheduleName, actionName, resourceName, scheduleAction.Cron, func() {
				logger.Info(fmt.Sprintf("action %q is running", actionName))
				r.setReplicas(ctx, instance.Spec, action)
			})
			if err != nil {
				logger.Info(fmt.Sprintf("failed to add cron job: %q", err))
				return metav1.Condition{}, err
			}
			logger.Info(fmt.Sprintf("action %q is scheduled", actionName))
		} else {
			// check if cron is changed
			logger.Info(fmt.Sprintf("action %q is scheduled", actionName))
			currentSchedule, err := cron.ParseStandard(scheduleAction.Cron)
			if err != nil {
				logger.Info(fmt.Sprintf("failed to parse cron schedule: %q", err))
				return metav1.Condition{}, err
			}
			currentCronID := r.Cron.GetActionsOfResource(scheduleName, resourceName)[actionName]
			entry := r.Cron.Get(currentCronID)
			if entry.Schedule != currentSchedule {
				logger.Info(fmt.Sprintf("action %q schedule is changed, updating it", actionName))
				r.Cron.Remove(scheduleName, actionName, resourceName)
				_, err := r.Cron.Add(scheduleName, actionName, resourceName, scheduleAction.Cron, func() {
					logger.Info(fmt.Sprintf("action %q is running", actionName))
					r.setReplicas(ctx, instance.Spec, action)
				})
				if err != nil {
					logger.Info(fmt.Sprintf("failed to schedule: %q", err))
					return metav1.Condition{}, err
				}
				logger.Info(fmt.Sprintf("action %q is scheduled", actionName))
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

func contains(actions []string, action string) bool {
	for _, a := range actions {
		if a == action {
			return true
		}
	}
	return false
}

func (r *PodReplicasReconciler) setReplicas(ctx context.Context, spec corev1alpha1.PodReplicasSpec, action corev1alpha1.PodReplicasAction) {
	logger := log.FromContext(ctx)
	selector := labels.SelectorFromSet(spec.LabelSelector.MatchLabels)
	switch kind := spec.ResourceType; kind {
	case corev1alpha1.Deployment:
		// retrive Deployments with specified labels
		for _, namespace := range spec.Namespaces {
			deploymentList := &appsv1.DeploymentList{}
			err := r.List(ctx, deploymentList, &client.ListOptions{
				LabelSelector: selector,
				Namespace:     namespace,
			})
			if err != nil {
				logger.Info(fmt.Sprintf("failed to list deployments: %q", err))
				return
			}

			replicasInt32 := int32(action.Replicas)
			for _, deployment := range deploymentList.Items {
				deployment.Spec.Replicas = &replicasInt32
				err := r.Update(ctx, &deployment)
				if err != nil {
					logger.Info(fmt.Sprintf("failed to update deployment: %q", err))
					return
				}
				logger.Info(fmt.Sprintf("updated deployment %q/%q to %d replicas", deployment.Namespace, deployment.Name, action.Replicas))
			}
		}
	case corev1alpha1.StatefulSet:
		// retrive StatefulSets with specified labels
		for _, namespace := range spec.Namespaces {
			statefulSetList := &appsv1.StatefulSetList{}
			err := r.List(ctx, statefulSetList, &client.ListOptions{
				LabelSelector: selector,
				Namespace:     namespace,
			})
			if err != nil {
				logger.Info(fmt.Sprintf("failed to list statefulsets: %q", err))
				return
			}

			replicasInt32 := int32(action.Replicas)
			for _, statefulSet := range statefulSetList.Items {
				statefulSet.Spec.Replicas = &replicasInt32
				err := r.Update(ctx, &statefulSet)
				if err != nil {
					logger.Info(fmt.Sprintf("failed to update statefulset: %q", err))
					return
				}
				logger.Info(fmt.Sprintf("updated statefulset %q/%q to %d replicas", statefulSet.Namespace, statefulSet.Name, action.Replicas))
			}
		}
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodReplicasReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.PodReplicas{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
