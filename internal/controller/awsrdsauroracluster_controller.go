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
	rdsTypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	v1alpha1 "github.com/d3vlo0p/TimeTerra/api/v1alpha1"
	"github.com/d3vlo0p/TimeTerra/internal/cron"
	"github.com/d3vlo0p/TimeTerra/notification"
	"github.com/go-logr/logr"
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

	logger.Info("reconciling resource")

	resourceName := resourceName("v1alpha1.AwsRdsAuroraCluster", req.Name)
	instance := &v1alpha1.AwsRdsAuroraCluster{}
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
		r.startStopCluster,
	)
}

func (r *AwsRdsAuroraClusterReconciler) GetScheduleService() *cron.ScheduleService {
	return r.Cron
}

func (r *AwsRdsAuroraClusterReconciler) GetNotificationService() *notification.NotificationService {
	return r.NotificationService
}

func (r *AwsRdsAuroraClusterReconciler) GetRecorder() record.EventRecorder {
	return r.Recorder
}

func (r *AwsRdsAuroraClusterReconciler) SetConditions(obj client.Object, conditions []metav1.Condition) {
	obj.(*v1alpha1.AwsRdsAuroraCluster).Status.Conditions = conditions
}

func (r *AwsRdsAuroraClusterReconciler) startStopCluster(ctx context.Context, logger logr.Logger, key types.NamespacedName, actionName string) (JobResult, JobMetadata) {
	metadata := JobMetadata{}
	start := time.Now()
	obj := &v1alpha1.AwsRdsAuroraCluster{}
	err := r.Get(ctx, key, obj)
	if err != nil {
		logger.Error(err, "Failed to get AwsRdsAuroraCluster resource. Re-running reconcile")
		metadata["error"] = err.Error()
		return JobResultError, metadata
	}
	action, ok := obj.Spec.Actions[actionName]
	if !ok {
		logger.Info("Action not found in AWSRdsCluster resource")
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
				msg := fmt.Sprintf("unable to start cluster %q", cluster.Identifier)
				logger.Error(err, msg)
				errorsList = append(errorsList, msg)
				r.Recorder.Eventf(obj, corev1.EventTypeWarning, "StartClusterFailed", msg)
				continue
			}

			r.Recorder.Eventf(obj, corev1.EventTypeNormal, "StartClusterSucceeded", "Cluster %q is starting", cluster.Identifier)
			logger.Info("Cluster is starting", "identifier", cluster.Identifier)

		case v1alpha1.AwsRdsAuroraClusterCommandStop:
			_, err := rdsClient.StopDBCluster(ctx, &rds.StopDBClusterInput{
				DBClusterIdentifier: &cluster.Identifier,
			}, opts)
			if err != nil {
				msg := fmt.Sprintf("unable to stop cluster %q", cluster.Identifier)
				logger.Error(err, msg)
				errorsList = append(errorsList, msg)
				r.Recorder.Eventf(obj, corev1.EventTypeWarning, "StopClusterFailed", msg)
				continue
			}

			r.Recorder.Eventf(obj, corev1.EventTypeNormal, "StopClusterSucceeded", "Cluster %q is stopping", cluster.Identifier)
			logger.Info("Cluster is stopping", "identifier", cluster.Identifier)

		case v1alpha1.AwsRdsAuroraClusterCommandScale:
			c, err := getClusterInfo(ctx, rdsClient, cluster.Identifier)
			if err != nil {
				msg := fmt.Sprintf("unable to get cluster info %q", cluster.Identifier)
				logger.Error(err, msg)
				errorsList = append(errorsList, msg)
				r.Recorder.Eventf(obj, corev1.EventTypeWarning, "ScaleClusterFailed", msg)
				continue
			}

			if len(c.DBClusterMembers) == 1 {
				instanceId := *c.DBClusterMembers[0].DBInstanceIdentifier
				err = scaleDBInstance(ctx, rdsClient, instanceId, action.Scale.Primary.InstanceClass)
				if err != nil {
					msg := fmt.Sprintf("unable to scale cluster %q", cluster.Identifier)
					logger.Error(err, msg)
					errorsList = append(errorsList, msg)
					r.Recorder.Eventf(obj, corev1.EventTypeWarning, "ScaleClusterFailed", msg)
					continue
				}
			} else if len(c.DBClusterMembers) > 1 {
				var newPrimary string
				var otherInstances []string
				for _, member := range c.DBClusterMembers {
					if !*member.IsClusterWriter && newPrimary == "" {
						newPrimary = *member.DBInstanceIdentifier
					} else {
						otherInstances = append(otherInstances, *member.DBInstanceIdentifier)
					}
				}

				err = scaleDBInstance(ctx, rdsClient, newPrimary, action.Scale.Primary.InstanceClass)
				if err != nil {
					msg := fmt.Sprintf("unable to scale cluster %q", cluster.Identifier)
					logger.Error(err, msg)
					errorsList = append(errorsList, msg)
					r.Recorder.Eventf(obj, corev1.EventTypeWarning, "ScaleClusterFailed", msg)
					continue
				}

				err = waitForDBInstanceAvailable(ctx, rdsClient, newPrimary)
				if err != nil {
					msg := fmt.Sprintf("unable to scale cluster %q", cluster.Identifier)
					logger.Error(err, msg)
					errorsList = append(errorsList, msg)
					r.Recorder.Eventf(obj, corev1.EventTypeWarning, "ScaleClusterFailed", msg)
					continue
				}

				err = failoverDBCluster(ctx, rdsClient, cluster.Identifier, newPrimary)
				if err != nil {
					msg := fmt.Sprintf("unable to scale cluster %q", cluster.Identifier)
					logger.Error(err, msg)
					errorsList = append(errorsList, msg)
					r.Recorder.Eventf(obj, corev1.EventTypeWarning, "ScaleClusterFailed", msg)
					continue
				}

				if action.Scale.Replicas != nil {
					for _, instance := range otherInstances {
						err = scaleDBInstance(ctx, rdsClient, instance, action.Scale.Replicas.InstanceClass)
						if err != nil {
							msg := fmt.Sprintf("unable to scale cluster %q", cluster.Identifier)
							logger.Error(err, msg)
							errorsList = append(errorsList, msg)
							r.Recorder.Eventf(obj, corev1.EventTypeWarning, "ScaleClusterFailed", msg)
							continue
						}
						err = waitForDBInstanceAvailable(ctx, rdsClient, instance)
						if err != nil {
							msg := fmt.Sprintf("unable to scale cluster %q", cluster.Identifier)
							logger.Error(err, msg)
							errorsList = append(errorsList, msg)
							r.Recorder.Eventf(obj, corev1.EventTypeWarning, "ScaleClusterFailed", msg)
							continue
						}
					}
				}
			}

		default:
			logger.Info("Unknown command", "command", action.Command)
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

func getClusterInfo(ctx context.Context, client *rds.Client, clusterIdentifier string) (*rdsTypes.DBCluster, error) {
	input := &rds.DescribeDBClustersInput{
		DBClusterIdentifier: &clusterIdentifier,
	}
	result, err := client.DescribeDBClusters(ctx, input)
	if err != nil {
		return nil, err
	}
	if len(result.DBClusters) == 0 {
		return nil, fmt.Errorf("no cluster found with identifier: %s", clusterIdentifier)
	}
	return &result.DBClusters[0], nil
}

func scaleDBInstance(ctx context.Context, client *rds.Client, instanceIdentifier, newInstanceClass string) error {
	applyNow := true
	input := &rds.ModifyDBInstanceInput{
		DBInstanceIdentifier: &instanceIdentifier,
		DBInstanceClass:      &newInstanceClass,
		ApplyImmediately:     &applyNow,
	}
	_, err := client.ModifyDBInstance(ctx, input)
	return err
}

func waitForDBInstanceAvailable(ctx context.Context, client *rds.Client, instanceIdentifier string) error {
	waiter := rds.NewDBInstanceAvailableWaiter(client)
	input := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: &instanceIdentifier,
	}
	return waiter.Wait(ctx, input, 30*time.Minute)
}

func failoverDBCluster(ctx context.Context, client *rds.Client, clusterIdentifier, targetInstance string) error {
	input := &rds.FailoverDBClusterInput{
		DBClusterIdentifier:        &clusterIdentifier,
		TargetDBInstanceIdentifier: &targetInstance,
	}
	_, err := client.FailoverDBCluster(ctx, input)
	return err
}

// SetupWithManager sets up the controller with the Manager.
func (r *AwsRdsAuroraClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.AwsRdsAuroraCluster{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
