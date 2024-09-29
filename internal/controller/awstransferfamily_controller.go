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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/transfer"
	v1alpha1 "github.com/d3vlo0p/TimeTerra/api/v1alpha1"
	"github.com/d3vlo0p/TimeTerra/internal/cron"
	"github.com/d3vlo0p/TimeTerra/notification"
	"github.com/go-logr/logr"
)

// AwsTransferFamilyReconciler reconciles a AwsTransferFamily object
type AwsTransferFamilyReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	Cron                *cron.ScheduleService
	NotificationService *notification.NotificationService
	Recorder            record.EventRecorder
	OperatorNamespace   string
}

//+kubebuilder:rbac:groups=timeterra.d3vlo0p.dev,resources=awstransferfamilies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=timeterra.d3vlo0p.dev,resources=awstransferfamilies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=timeterra.d3vlo0p.dev,resources=awstransferfamilies/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the server closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AwsTransferFamily object against the actual server state, and then
// perform operations to make the server state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *AwsTransferFamilyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("reconciling resource")

	resourceName := resourceName("v1alpha1.AwsTransferFamily", req.Name)
	instance := &v1alpha1.AwsTransferFamily{}
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

	if instance.Spec.Credentials != nil {
		secret := &corev1.Secret{}
		key := types.NamespacedName{Name: instance.Spec.Credentials.SecretName, Namespace: defaultNamespace(r.OperatorNamespace, instance.Spec.Credentials.Namespace)}
		log.Log.Info("check secret", "key", key)
		err = r.Get(ctx, key, secret)
		if err != nil {
			logger.Error(err, "Failed to get Secret.", "key", key)
			r.Recorder.Eventf(instance, corev1.EventTypeWarning, "SecretError", "Secret %q error: %s", key.String(), err.Error())
			return ctrl.Result{}, err
		}
	}

	return reconcileResource(
		ctx,
		r,
		instance,
		resourceName,
		instance.Spec.Actions,
		r.startStopServer,
	)
}

func (r *AwsTransferFamilyReconciler) GetScheduleService() *cron.ScheduleService {
	return r.Cron
}

func (r *AwsTransferFamilyReconciler) GetNotificationService() *notification.NotificationService {
	return r.NotificationService
}

func (r *AwsTransferFamilyReconciler) GetRecorder() record.EventRecorder {
	return r.Recorder
}

func (r *AwsTransferFamilyReconciler) SetConditions(obj client.Object, conditions []metav1.Condition) {
	obj.(*v1alpha1.AwsTransferFamily).Status.Conditions = conditions
}

func (r *AwsTransferFamilyReconciler) startStopServer(ctx context.Context, logger logr.Logger, key types.NamespacedName, actionName string) (JobResult, JobMetadata) {
	metadata := JobMetadata{}
	start := time.Now()
	obj := &v1alpha1.AwsTransferFamily{}
	err := r.Get(ctx, key, obj)
	if err != nil {
		logger.Error(err, "Failed to get AwsTransferFamily resource. Re-running reconcile")
		metadata["error"] = err.Error()
		return JobResultError, metadata
	}
	action, ok := obj.Spec.Actions[actionName]
	if !ok {
		logger.Info("Action not found in AWSTransferFamily resource")
		metadata["error"] = fmt.Sprintf("Action %q not found in AWSTransferFamily resource", actionName)
		return JobResultError, metadata
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		logger.Error(err, "unable to load SDK config")
		metadata["error"] = err.Error()
		return JobResultError, metadata
	}

	if obj.Spec.Credentials != nil {
		secret := &corev1.Secret{}
		err = r.Get(ctx, types.NamespacedName{Name: obj.Spec.Credentials.SecretName, Namespace: defaultNamespace(r.OperatorNamespace, obj.Spec.Credentials.Namespace)}, secret)
		if err != nil {
			logger.Error(err, "Failed to get Secret resource")
			metadata["error"] = err.Error()
			return JobResultError, metadata
		}

		cfg.Credentials, err = getAwsCredentialProviderFromSecret(secret, obj.Spec.Credentials.KeysRef)
		if err != nil {
			logger.Error(err, "Failed to configure credentials")
			metadata["error"] = err.Error()
			return JobResultError, metadata
		}
	}

	actionType := conditionTypeForAction(actionName)
	errorsList := make([]string, 0)
	transferClient := transfer.NewFromConfig(cfg)
	for _, server := range obj.Spec.ServerIds {
		opts := func(o *transfer.Options) {
			o.Region = server.Region
			if obj.Spec.ServiceEndpoint != nil {
				o.BaseEndpoint = obj.Spec.ServiceEndpoint
			}
		}

		switch action.Command {

		case v1alpha1.AwsTransferFamilyCommandStart:
			_, err := transferClient.StartServer(ctx, &transfer.StartServerInput{
				ServerId: &server.Id,
			}, opts)
			if err != nil {
				msg := fmt.Sprintf("unable to start server %q", server.Id)
				logger.Error(err, msg)
				errorsList = append(errorsList, msg)
				r.Recorder.Eventf(obj, corev1.EventTypeWarning, "StartServerFailed", msg)
				continue
			}

			r.Recorder.Eventf(obj, corev1.EventTypeNormal, "StartServerSucceeded", "Server %q is starting", server.Id)
			logger.Info("Server is starting", "id", server.Id)

		case v1alpha1.AwsTransferFamilyCommandStop:
			_, err := transferClient.StopServer(ctx, &transfer.StopServerInput{
				ServerId: &server.Id,
			}, opts)
			if err != nil {
				msg := fmt.Sprintf("unable to stop server %q", server.Id)
				logger.Error(err, msg)
				errorsList = append(errorsList, msg)
				r.Recorder.Eventf(obj, corev1.EventTypeWarning, "StopServerFailed", msg)
				continue
			}

			r.Recorder.Eventf(obj, corev1.EventTypeNormal, "StopServerSucceeded", "Server %q is stopping", server.Id)
			logger.Info("Server is stopping", "id", server.Id)
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
func (r *AwsTransferFamilyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.AwsTransferFamily{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
