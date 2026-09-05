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
	"errors"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"

	timeterrav1alpha1 "github.com/d3vlo0p/TimeTerra/api/v1alpha1"
	"github.com/d3vlo0p/TimeTerra/internal/action"
	"github.com/d3vlo0p/TimeTerra/notification"
	corev1 "k8s.io/api/core/v1"
)

var ErrActionTimeout = errors.New("action execution timed out")

// ActionExecutionReconciler reconciles a ActionExecution object
type ActionExecutionReconciler struct {
	BaseReconciler
	OperatorNamespace string
	ActionTimeout     time.Duration
}

//+kubebuilder:rbac:groups=timeterra.d3vlo0p.dev,resources=actionexecutions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=timeterra.d3vlo0p.dev,resources=actionexecutions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=timeterra.d3vlo0p.dev,resources=actionexecutions/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ActionExecution object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *ActionExecutionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	op := &timeterrav1alpha1.ActionExecution{}
	err := r.Get(ctx, req.NamespacedName, op)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Ignore resources that are already being deleted
	if !op.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	// Check if action execution has timed out
	timeout := r.ActionTimeout
	if timeout <= 0 {
		timeout = 30 * time.Minute
	}
	if !op.CreationTimestamp.IsZero() && time.Since(op.CreationTimestamp.Time) > timeout {
		timeoutErr := fmt.Errorf("%w after %v in phase %s", ErrActionTimeout, timeout, op.Status.Phase)
		logger.Info("ActionExecution timed out, finalizing and deleting", "timeout", timeout, "phase", op.Status.Phase)
		op.Status.Phase = "Timeout"
		r.finalizeAction(ctx, logger, op, timeoutErr)
		if delErr := client.IgnoreNotFound(r.Delete(ctx, op)); delErr != nil {
			logger.Error(delErr, "Failed to delete timed out ActionExecution")
			return ctrl.Result{}, delErr
		}
		return ctrl.Result{}, nil
	}

	var result ctrl.Result
	var handlerErr error
	var done bool

	switch op.Spec.ActionType {
	case "AuroraVerticalScale":
		handler := &action.AuroraScaleHandler{
			Client:            r.Client,
			OperatorNamespace: r.OperatorNamespace,
			SecretProvider:    getAwsCredentialProviderFromSecret,
		}
		result, done, handlerErr = handler.Execute(ctx, logger, op)
	case "DocumentDbVerticalScale":
		handler := &action.DocumentDbScaleHandler{
			Client:            r.Client,
			OperatorNamespace: r.OperatorNamespace,
			SecretProvider:    getAwsCredentialProviderFromSecret,
		}
		result, done, handlerErr = handler.Execute(ctx, logger, op)
	default:
		logger.Info("Unknown action type", "type", op.Spec.ActionType)
		return ctrl.Result{}, nil // Ignore
	}

	if done {
		r.finalizeAction(ctx, logger, op, handlerErr)
		if err := client.IgnoreNotFound(r.Delete(ctx, op)); err != nil {
			logger.Error(err, "Failed to delete ActionExecution")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if err := r.Status().Update(ctx, op); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to update ActionExecution status")
		return ctrl.Result{}, err
	}

	return result, handlerErr
}

func (r *ActionExecutionReconciler) finalizeAction(ctx context.Context, logger logr.Logger, op *timeterrav1alpha1.ActionExecution, handlerErr error) {
	statusStr := "Success"
	if handlerErr != nil {
		if errors.Is(handlerErr, ErrActionTimeout) {
			statusStr = "Timeout"
		} else {
			statusStr = "Error"
		}
	}

	metadata := map[string]interface{}{
		"phase": op.Status.Phase,
	}
	if handlerErr != nil {
		metadata["error"] = handlerErr.Error()
	}

	// 1. Notify
	if r.NotificationService != nil {
		r.NotificationService.Send(notification.NotificationBody{
			Schedule: op.Spec.ActionName, // For ActionExecution, ActionName usually holds the schedule/trigger name context
			Action:   op.Spec.ActionType,
			Resource: op.Spec.TargetResource.Name,
			Status:   statusStr,
			Metadata: metadata,
		})
	}

	// 2. Bubble up to target resource
	var target client.Object
	var conditions *[]metav1.Condition

	switch op.Spec.TargetResource.Kind {
	case "AwsRdsAuroraCluster":
		auroraTarget := &timeterrav1alpha1.AwsRdsAuroraCluster{}
		err := r.Get(ctx, types.NamespacedName{Name: op.Spec.TargetResource.Name, Namespace: op.Spec.TargetResource.Namespace}, auroraTarget)
		if err == nil {
			target = auroraTarget
			conditions = &auroraTarget.Status.Conditions
		} else {
			logger.Error(err, "Failed to get target resource for finalization")
		}
	case "AwsDocumentDBCluster":
		docdbTarget := &timeterrav1alpha1.AwsDocumentDBCluster{}
		err := r.Get(ctx, types.NamespacedName{Name: op.Spec.TargetResource.Name, Namespace: op.Spec.TargetResource.Namespace}, docdbTarget)
		if err == nil {
			target = docdbTarget
			conditions = &docdbTarget.Status.Conditions
		} else {
			logger.Error(err, "Failed to get target resource for finalization")
		}
	}

	if target != nil && conditions != nil {
		condType := "Action" + op.Spec.ActionType
		reason := "Active"
		status := metav1.ConditionTrue
		msg := "Action execution completed successfully"
		eventType := corev1.EventTypeNormal

		if handlerErr != nil {
			status = metav1.ConditionFalse
			msg = handlerErr.Error()
			eventType = corev1.EventTypeWarning
			if errors.Is(handlerErr, ErrActionTimeout) {
				reason = "Timeout"
			} else {
				reason = "Failed"
			}
		}

		meta.SetStatusCondition(conditions, metav1.Condition{
			Type:               condType,
			Status:             status,
			Reason:             reason,
			Message:            msg,
			LastTransitionTime: metav1.Now(),
		})
		if updateErr := r.Status().Update(ctx, target); updateErr != nil {
			logger.Error(updateErr, "Failed to bubble up conditions to parent resource")
		} else if r.Recorder != nil {
			// Emit K8s event on the parent
			r.Recorder.Event(target, eventType, reason, msg)
		}
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ActionExecutionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&timeterrav1alpha1.ActionExecution{}).
		Named("actionexecution").
		Complete(r)
}
