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
	corev1 "k8s.io/api/core/v1"
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

	v1alpha1 "github.com/d3vlo0p/TimeTerra/api/v1alpha1"
	"github.com/d3vlo0p/TimeTerra/internal/cron"
	"github.com/d3vlo0p/TimeTerra/notification"
	"github.com/go-logr/logr"
)

// K8sPodReplicasReconciler reconciles a K8sPodReplicas object
type K8sPodReplicasReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	Cron                *cron.ScheduleService
	NotificationService *notification.NotificationService
	Recorder            record.EventRecorder
}

//+kubebuilder:rbac:groups=timeterra.d3vlo0p.dev,resources=k8spodreplicas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=timeterra.d3vlo0p.dev,resources=k8spodreplicas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=timeterra.d3vlo0p.dev,resources=k8spodreplicas/finalizers,verbs=update
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

	logger.Info("reconciling resource")

	resourceName := resourceName("v1alpha1.K8sPodReplicas", req.Name)
	instance := &v1alpha1.K8sPodReplicas{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Resource not found. object must has been deleted")
			r.Cron.RemoveResource(resourceName)
			return ctrl.Result{}, nil
		}
		logger.Info("Failed to get resource. Re-running reconcile")
		return ctrl.Result{}, err
	}

	return reconcileResource(
		ctx,
		r,
		instance,
		resourceName,
		instance.Spec.Actions,
		r.setReplicas,
	)
}

func (r *K8sPodReplicasReconciler) GetScheduleService() *cron.ScheduleService {
	return r.Cron
}

func (r *K8sPodReplicasReconciler) GetNotificationService() *notification.NotificationService {
	return r.NotificationService
}

func (r *K8sPodReplicasReconciler) GetRecorder() record.EventRecorder {
	return r.Recorder
}

func (r *K8sPodReplicasReconciler) SetConditions(obj client.Object, conditions []metav1.Condition) {
	obj.(*v1alpha1.K8sPodReplicas).Status.Conditions = conditions
}

func (r *K8sPodReplicasReconciler) setReplicas(ctx context.Context, logger logr.Logger, key types.NamespacedName, actionName string) (JobResult, JobMetadata) {
	metadata := JobMetadata{}
	start := time.Now()
	obj := &v1alpha1.K8sPodReplicas{}
	err := r.Get(ctx, key, obj)
	if err != nil {
		logger.Error(err, "Failed to get PodReplicas resource")
		metadata["error"] = err.Error()
		return JobResultError, metadata
	}
	action, ok := obj.Spec.Actions[actionName]
	if !ok {
		logger.Info("Action not found in PodReplicas resource")
		metadata["error"] = fmt.Sprintf("Action %q not found in PodReplicas resource", actionName)
		return JobResultError, metadata
	}
	selector := labels.SelectorFromSet(obj.Spec.LabelSelector.MatchLabels)
	if len(obj.Spec.Namespaces) == 0 {
		obj.Spec.Namespaces = []string{metav1.NamespaceAll}
	}
	actionType := conditionTypeForAction(actionName)
	errorsList := make([]string, 0)
	for _, namespace := range obj.Spec.Namespaces {
		switch kind := obj.Spec.ResourceType; kind {
		case v1alpha1.K8sDeployment:
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
				r.Recorder.Eventf(obj, corev1.EventTypeWarning, "Failed", msg)
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
					r.Recorder.Eventf(obj, corev1.EventTypeWarning, "Failed", msg)
					continue
				}

				r.Recorder.Eventf(obj, corev1.EventTypeNormal, "Success", "updated deployment %q/%q to %d replicas", deployment.Namespace, deployment.Name, action.Replicas)
				logger.Info(fmt.Sprintf("updated deployment %q/%q to %d replicas", deployment.Namespace, deployment.Name, action.Replicas))
			}

		case v1alpha1.K8sStatefulSet:
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
				r.Recorder.Eventf(obj, corev1.EventTypeWarning, "Failed", msg)
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
					r.Recorder.Eventf(obj, corev1.EventTypeWarning, "Failed", msg)
					continue
				}

				r.Recorder.Eventf(obj, corev1.EventTypeNormal, "Success", "updated statefulset %q/%q to %d replicas", statefulSet.Namespace, statefulSet.Name, action.Replicas)
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
		metadata["error_list"] = errorsList
		return JobResultFailure, metadata
	}

	addToConditions(&obj.Status.Conditions, metav1.Condition{
		LastTransitionTime: metav1.Now(),
		Type:               actionType,
		Status:             metav1.ConditionTrue,
		Reason:             "Active",
		Message:            fmt.Sprintf("last execution started:%q ended:%q", start.Format(time.RFC3339), time.Now().Format(time.RFC3339)),
	})
	r.Status().Update(ctx, obj)
	return JobResultSuccess, metadata
}

// SetupWithManager sets up the controller with the Manager.
func (r *K8sPodReplicasReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.K8sPodReplicas{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
