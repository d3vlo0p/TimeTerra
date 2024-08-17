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

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	corev1alpha1 "github.com/d3vlo0p/TimeTerra/api/v1alpha1"
	"github.com/d3vlo0p/TimeTerra/notification"
)

// NotificationPolicyReconciler reconciles a NotificationPolicy object
type NotificationPolicyReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	NotificationService *notification.NotificationService
	Recorder            record.EventRecorder
}

//+kubebuilder:rbac:groups=core.timeterra.d3vlo0p.dev,resources=notificationpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.timeterra.d3vlo0p.dev,resources=notificationpolicies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.timeterra.d3vlo0p.dev,resources=notificationpolicies/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NotificationPolicy object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.3/pkg/reconcile
func (r *NotificationPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info(fmt.Sprintf("reconciling object %#q", req.NamespacedName))
	instance := &corev1alpha1.NotificationPolicy{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("NotificationPolicy resource not found. object must has been deleted.")
			r.NotificationService.RemoveRecipient(instance.Name)
			return ctrl.Result{}, nil
		}
		logger.Info("Failed to get NotificationPolicy resource. Re-running reconcile.")
		return ctrl.Result{}, err
	}

	if instance.Status.Conditions == nil {
		instance.Status.Conditions = make([]metav1.Condition, 0)
	}

	err = r.reconcile(ctx, instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.Status().Update(ctx, instance)
	if err != nil {
		logger.Info("Failed to update NotificationPolicy resource status. Re-running reconcile.")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *NotificationPolicyReconciler) reconcile(ctx context.Context, instance *corev1alpha1.NotificationPolicy) error {
	logger := log.FromContext(ctx)
	r.NotificationService.RemoveRecipient(instance.Name)
	switch instance.Spec.Type {
	case corev1alpha1.NotificationTypeApi:
		logger.Info("Handling API notification")
		if instance.Spec.Api == nil {
			return fmt.Errorf("api notification type requires api configuration")
		}
		api := notification.NewApiNotification(ctx, instance.Name, instance.Spec.Api.Url)
		for _, schedule := range instance.Spec.Schedules {
			r.NotificationService.AddRecipientToSchedule(schedule, instance.Name, api)
		}
	default:
		logger.Info("Unknown notification type")
		return fmt.Errorf("unknown notification type: %s", instance.Spec.Type)
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NotificationPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.NotificationPolicy{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
