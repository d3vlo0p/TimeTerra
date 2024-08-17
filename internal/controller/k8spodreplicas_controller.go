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

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	corev1alpha1 "github.com/d3vlo0p/TimeTerra/api/v1alpha1"
	"github.com/d3vlo0p/TimeTerra/internal/cron"
	"github.com/d3vlo0p/TimeTerra/notification"
)

// K8sPodReplicasReconciler reconciles a K8sPodReplicas object
type K8sPodReplicasReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	Cron                *cron.ScheduleCron
	NotificationService *notification.NotificationService
	Recorder            record.EventRecorder
}

//+kubebuilder:rbac:groups=core.timeterra.d3vlo0p.dev,resources=k8spodreplicas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.timeterra.d3vlo0p.dev,resources=k8spodreplicas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.timeterra.d3vlo0p.dev,resources=k8spodreplicas/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=apps,resources=deployments/scale,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=apps,resources=statefulsets/scale,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the K8sPodReplicas object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *K8sPodReplicasReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info(fmt.Sprintf("reconciling object %#q", req.NamespacedName))

	resourceName := ResourceName("K8sPodReplicas", req.Name)
	instance := &corev1alpha1.K8sPodReplicas{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("PodReplicas resource not found. object must has been deleted.")
			r.Cron.RemoveResource(resourceName)
			return ctrl.Result{}, nil
		}
		logger.Info("Failed to get PodReplicas resource. Re-running reconcile.")
		return ctrl.Result{}, err
	}

	if instance.Status.Conditions == nil {
		instance.Status.Conditions = make([]metav1.Condition, 0)
	}
	if instance.Spec.Enabled != nil && !*instance.Spec.Enabled {
		disableResource(r.Cron, &instance.Status.Conditions, resourceName)
	} else {
		err := reconcileResource(ctx, req, r.Client, r.Cron, r.NotificationService, instance.Spec.Actions, instance.Spec.Schedule, resourceName, r.setReplicas, &instance.Status.Conditions)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	err = r.Status().Update(ctx, instance)
	if err != nil {
		logger.Info(fmt.Sprintf("failed to update status: %q", err))
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *K8sPodReplicasReconciler) setReplicas(ctx context.Context, key types.NamespacedName, actionName string) JobResult {
	logger := log.FromContext(ctx)
	start := time.Now()
	obj := &corev1alpha1.K8sPodReplicas{}
	err := r.Get(ctx, key, obj)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Error(err, "PodReplicas resource not found. object must has been deleted.")
			return JobResultError
		}
		logger.Error(err, "Failed to get PodReplicas resource.")
		return JobResultError
	}
	action, ok := obj.Spec.Actions[actionName]
	if !ok {
		logger.Info("Action not found")
		return JobResultError
	}
	selector := labels.SelectorFromSet(obj.Spec.LabelSelector.MatchLabels)
	if len(obj.Spec.Namespaces) == 0 {
		obj.Spec.Namespaces = []string{metav1.NamespaceAll}
	}
	actionType := ConditionTypeForAction(actionName)
	errorsList := make([]string, 0)
	for _, namespace := range obj.Spec.Namespaces {
		switch kind := obj.Spec.ResourceType; kind {
		case corev1alpha1.K8sDeployment:
			// retrive Deployments with specified labels
			deploymentList := &appsv1.DeploymentList{}
			err := r.List(ctx, deploymentList, &client.ListOptions{
				LabelSelector: selector,
				Namespace:     namespace,
			})
			if err != nil {
				msg := fmt.Sprintf("Action %q failed to list deployments for namespace %q", actionName, namespace)
				logger.Error(err, msg)
				errorsList = append(errorsList, msg)
				r.Recorder.Eventf(obj, "Warning", "Failed", msg)
				continue
			}

			replicasInt32 := int32(action.Replicas)
			for _, deployment := range deploymentList.Items {
				deployment.Spec.Replicas = &replicasInt32
				err := r.Update(ctx, &deployment)
				if err != nil {
					msg := fmt.Sprintf("failed to update deployment %q/%q", deployment.Namespace, deployment.Name)
					logger.Error(err, msg)
					errorsList = append(errorsList, msg)
					r.Recorder.Eventf(obj, "Warning", "Failed", msg)
					continue
				}

				r.Recorder.Eventf(obj, "Normal", "Success", "updated deployment %q/%q to %d replicas", deployment.Namespace, deployment.Name, action.Replicas)
				logger.Info(fmt.Sprintf("updated deployment %q/%q to %d replicas", deployment.Namespace, deployment.Name, action.Replicas))
			}

		case corev1alpha1.K8sStatefulSet:
			// retrive StatefulSets with specified labels
			statefulSetList := &appsv1.StatefulSetList{}
			err := r.List(ctx, statefulSetList, &client.ListOptions{
				LabelSelector: selector,
				Namespace:     namespace,
			})
			if err != nil {
				msg := fmt.Sprintf("Action %q failed to list statefulsets for namespace %q", actionName, namespace)
				logger.Error(err, msg)
				errorsList = append(errorsList, msg)
				r.Recorder.Eventf(obj, "Warning", "Failed", msg)
				continue
			}

			replicasInt32 := int32(action.Replicas)
			for _, statefulSet := range statefulSetList.Items {
				statefulSet.Spec.Replicas = &replicasInt32
				err := r.Update(ctx, &statefulSet)
				if err != nil {
					msg := fmt.Sprintf("failed to update statefulset %q/%q", statefulSet.Namespace, statefulSet.Name)
					logger.Error(err, msg)
					errorsList = append(errorsList, msg)
					r.Recorder.Eventf(obj, "Warning", "Failed", msg)
					continue
				}

				r.Recorder.Eventf(obj, "Normal", "Success", "updated statefulset %q/%q to %d replicas", statefulSet.Namespace, statefulSet.Name, action.Replicas)
				logger.Info(fmt.Sprintf("updated statefulset %q/%q to %d replicas", statefulSet.Namespace, statefulSet.Name, action.Replicas))
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
		r.Status().Update(ctx, obj)
		return JobResultFailure
	}

	addToConditions(&obj.Status.Conditions, metav1.Condition{
		LastTransitionTime: metav1.Now(),
		Type:               actionType,
		Status:             metav1.ConditionTrue,
		Reason:             "Success",
		Message:            fmt.Sprintf("Action %q, last execution start:%q end:%q", actionName, start.Format(time.RFC3339), time.Now().Format(time.RFC3339)),
	})
	r.Status().Update(ctx, obj)
	return JobResultSuccess
}

// SetupWithManager sets up the controller with the Manager.
func (r *K8sPodReplicasReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.K8sPodReplicas{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
