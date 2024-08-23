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
	"github.com/aws/aws-sdk-go-v2/service/rds"
	v1alpha1 "github.com/d3vlo0p/TimeTerra/api/v1alpha1"
	"github.com/d3vlo0p/TimeTerra/internal/cron"
	"github.com/d3vlo0p/TimeTerra/notification"
)

// AwsRdsAuroraClusterReconciler reconciles a AwsRdsAuroraCluster object
type AwsRdsAuroraClusterReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	Cron                *cron.ScheduleService
	NotificationService *notification.NotificationService
	Recorder            record.EventRecorder
	OperatorNamespace   string
}

//+kubebuilder:rbac:groups=timeterra.d3vlo0p.dev,resources=awsrdsauroraclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=timeterra.d3vlo0p.dev,resources=awsrdsauroraclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=timeterra.d3vlo0p.dev,resources=awsrdsauroraclusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AwsRdsAuroraCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *AwsRdsAuroraClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info(fmt.Sprintf("reconciling object %#q", req.NamespacedName))

	resourceName := ResourceName("AwsRdsAuroraCluster", req.Name)
	instance := &v1alpha1.AwsRdsAuroraCluster{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("AwsRdsAuroraCluster resource not found. object must has been deleted.")
			r.Cron.RemoveResource(resourceName)
			return ctrl.Result{}, nil
		}
		logger.Info("Failed to get AwsRdsAuroraCluster resource. Re-running reconcile.")
		return ctrl.Result{}, err
	}

	if instance.Status.Conditions == nil {
		instance.Status.Conditions = make([]metav1.Condition, 0)
	}

	if instance.Spec.Credentials != nil {
		secret := &corev1.Secret{}
		key := types.NamespacedName{Name: instance.Spec.Credentials.SecretName, Namespace: defaultNamespace(r.OperatorNamespace, instance.Spec.Credentials.Namespace)}
		log.Log.Info("check secret", "key", key)
		err = r.Get(ctx, key, secret)
		if err != nil {
			logger.Error(err, "Failed to get Secret.", "key", key)
			r.Recorder.Eventf(instance, corev1.EventTypeWarning, "SecretError", "Secret %s error: %s", key.String(), err.Error())
			return ctrl.Result{}, err
		}
	}

	if instance.Spec.Enabled != nil && !*instance.Spec.Enabled {
		disableResource(r.Cron, &instance.Status.Conditions, resourceName)
	} else {
		err := reconcileResource(ctx, req, r.Client, r.Cron, r.NotificationService, instance.Spec.Actions, instance.Spec.Schedule, resourceName, r.startStopCluster, &instance.Status.Conditions)
		if err != nil {
			r.Recorder.Eventf(instance, corev1.EventTypeWarning, "ReconcileError", "Reconcile error: %s", err.Error())
			return ctrl.Result{}, err
		}
	}

	err = r.Status().Update(ctx, instance)
	if err != nil {
		r.Recorder.Eventf(instance, corev1.EventTypeWarning, "ReconcileError", "Reconcile error: %s", err.Error())
		logger.Info("Failed to update AwsRdsAuroraCluster resource. Re-running reconcile.")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *AwsRdsAuroraClusterReconciler) startStopCluster(ctx context.Context, key types.NamespacedName, actionName string) (JobResult, JobMetadata) {
	metadata := JobMetadata{}
	logger := log.FromContext(ctx)
	start := time.Now()
	obj := &v1alpha1.AwsRdsAuroraCluster{}
	err := r.Get(ctx, key, obj)
	if err != nil {
		logger.Error(err, "Failed to get AwsRdsAuroraCluster resource. Re-running reconcile.")
		metadata["error"] = err.Error()
		return JobResultError, metadata
	}
	action, ok := obj.Spec.Actions[actionName]
	if !ok {
		logger.Info(fmt.Sprintf("Action %q not found in AWSRdsCluster resource.", actionName))
		metadata["error"] = fmt.Sprintf("Action %q not found in AWSRdsCluster resource.", actionName)
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
			logger.Error(err, "Failed to get Secret resource.")
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

	actionType := ConditionTypeForAction(actionName)
	errorsList := make([]string, 0)
	rdsClient := rds.NewFromConfig(cfg)
	for _, cluster := range obj.Spec.DBClusterIdentifiers {
		opts := func(o *rds.Options) {
			o.Region = cluster.Region
			if obj.Spec.ServiceEndpoint != nil {
				o.BaseEndpoint = obj.Spec.ServiceEndpoint
			}
		}

		switch action.Command {

		case v1alpha1.AwsRdsAuroraClusterCommandStart:
			_, err := rdsClient.StartDBCluster(ctx, &rds.StartDBClusterInput{
				DBClusterIdentifier: &cluster.Identifier,
			}, opts)
			if err != nil {
				msg := fmt.Sprintf("unable to start cluster %s", cluster.Identifier)
				logger.Error(err, msg)
				errorsList = append(errorsList, msg)
				r.Recorder.Eventf(obj, corev1.EventTypeWarning, "StartClusterFailed", msg)
				continue
			}

			r.Recorder.Eventf(obj, corev1.EventTypeNormal, "StartClusterSucceeded", "Cluster %s is starting", cluster.Identifier)
			logger.Info("Cluster is starting", "identifier", cluster.Identifier)

		case v1alpha1.AwsRdsAuroraClusterCommandStop:
			_, err := rdsClient.StopDBCluster(ctx, &rds.StopDBClusterInput{
				DBClusterIdentifier: &cluster.Identifier,
			}, opts)
			if err != nil {
				msg := fmt.Sprintf("unable to stop cluster %s", cluster.Identifier)
				logger.Error(err, msg)
				errorsList = append(errorsList, msg)
				r.Recorder.Eventf(obj, corev1.EventTypeWarning, "StopClusterFailed", msg)
				continue
			}

			r.Recorder.Eventf(obj, corev1.EventTypeNormal, "StopClusterSucceeded", "Cluster %s is stopping", cluster.Identifier)
			logger.Info("Cluster is stopping", "identifier", cluster.Identifier)
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
		Reason:             "Success",
		Message:            fmt.Sprintf("Action %q, last execution started:%q ended:%q", actionName, start.Format(time.RFC3339), time.Now().Format(time.RFC3339)),
	})
	r.Status().Update(ctx, obj)
	return JobResultSuccess, metadata
}

// SetupWithManager sets up the controller with the Manager.
func (r *AwsRdsAuroraClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.AwsRdsAuroraCluster{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
