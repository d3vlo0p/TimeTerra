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

	appsv1 "k8s.io/api/apps/v1"
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

// K8sPodReplicasReconciler reconciles a K8sPodReplicas object
type K8sPodReplicasReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Cron   *cron.ScheduleCron
}

//+kubebuilder:rbac:groups=core.timeterra.d3vlo0p.dev,resources=k8spodreplicas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.timeterra.d3vlo0p.dev,resources=k8spodreplicas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.timeterra.d3vlo0p.dev,resources=k8spodreplicas/finalizers,verbs=update

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
		condition, err := reconcileResource(ctx, r.Client, r.Cron, instance.Spec.Actions, instance.Spec, instance.Spec.Schedule, resourceName, r.setReplicas)
		if err != nil {
			return ctrl.Result{}, err
		}
		instance.Status.Conditions = addCondition(instance.Status.Conditions, condition)
	}

	err = r.Status().Update(ctx, instance)
	if err != nil {
		logger.Info(fmt.Sprintf("failed to update status: %q", err))
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *K8sPodReplicasReconciler) setReplicas(ctx context.Context, spec corev1alpha1.K8sPodReplicasSpec, action corev1alpha1.K8sPodReplicasAction) {
	logger := log.FromContext(ctx)
	selector := labels.SelectorFromSet(spec.LabelSelector.MatchLabels)
	switch kind := spec.ResourceType; kind {
	case corev1alpha1.K8sDeployment:
		// retrive Deployments with specified labels
		for _, namespace := range spec.Namespaces {
			deploymentList := &appsv1.DeploymentList{}
			err := r.List(ctx, deploymentList, &client.ListOptions{
				LabelSelector: selector,
				Namespace:     namespace,
			})
			if err != nil {
				logger.Error(err, "failed to list deployments")
				return
			}

			replicasInt32 := int32(action.Replicas)
			for _, deployment := range deploymentList.Items {
				deployment.Spec.Replicas = &replicasInt32
				err := r.Update(ctx, &deployment)
				if err != nil {
					logger.Error(err, "failed to update deployment")
					return
				}
				logger.Info(fmt.Sprintf("updated deployment %q/%q to %d replicas", deployment.Namespace, deployment.Name, action.Replicas))
			}
		}
	case corev1alpha1.K8sStatefulSet:
		// retrive StatefulSets with specified labels
		for _, namespace := range spec.Namespaces {
			statefulSetList := &appsv1.StatefulSetList{}
			err := r.List(ctx, statefulSetList, &client.ListOptions{
				LabelSelector: selector,
				Namespace:     namespace,
			})
			if err != nil {
				logger.Error(err, "failed to list statefulsets")
				return
			}

			replicasInt32 := int32(action.Replicas)
			for _, statefulSet := range statefulSetList.Items {
				statefulSet.Spec.Replicas = &replicasInt32
				err := r.Update(ctx, &statefulSet)
				if err != nil {
					logger.Error(err, "failed to update statefulset")
					return
				}
				logger.Info(fmt.Sprintf("updated statefulset %q/%q to %d replicas", statefulSet.Namespace, statefulSet.Name, action.Replicas))
			}
		}
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *K8sPodReplicasReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.K8sPodReplicas{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}