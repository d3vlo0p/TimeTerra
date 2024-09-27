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
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	v1alpha1 "github.com/d3vlo0p/TimeTerra/api/v1alpha1"
	"github.com/d3vlo0p/TimeTerra/internal/cron"
	"github.com/d3vlo0p/TimeTerra/notification"
)

// AwsEc2InstanceReconciler reconciles a AwsEc2Instance object
type AwsEc2InstanceReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	Cron                *cron.ScheduleService
	NotificationService *notification.NotificationService
	Recorder            record.EventRecorder
	OperatorNamespace   string
}

//+kubebuilder:rbac:groups=timeterra.d3vlo0p.dev,resources=awsec2instances,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=timeterra.d3vlo0p.dev,resources=awsec2instances/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=timeterra.d3vlo0p.dev,resources=awsec2instances/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the ec2 closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AwsEc2Instance object against the actual ec2 state, and then
// perform operations to make the ec2 state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *AwsEc2InstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info(fmt.Sprintf("reconciling object %#q", req.NamespacedName))

	resourceName := resourceName("v1alpha1.AwsEc2Instance", req.Name)
	instance := &v1alpha1.AwsEc2Instance{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Resource not found. object must has been deleted.")
			r.Cron.RemoveResource(resourceName)
			return ctrl.Result{}, nil
		}
		logger.Info("Failed to get resource. Re-running reconcile.")
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
		r.startStopInstances,
	)
}

func (r *AwsEc2InstanceReconciler) GetScheduleService() *cron.ScheduleService {
	return r.Cron
}

func (r *AwsEc2InstanceReconciler) GetNotificationService() *notification.NotificationService {
	return r.NotificationService
}

func (r *AwsEc2InstanceReconciler) GetRecorder() record.EventRecorder {
	return r.Recorder
}

func (r *AwsEc2InstanceReconciler) SetConditions(obj client.Object, conditions []metav1.Condition) {
	obj.(*v1alpha1.AwsEc2Instance).Status.Conditions = conditions
}

func (r *AwsEc2InstanceReconciler) startStopInstances(ctx context.Context, key types.NamespacedName, actionName string) (JobResult, JobMetadata) {
	metadata := JobMetadata{}
	logger := log.FromContext(ctx)
	start := time.Now()
	obj := &v1alpha1.AwsEc2Instance{}
	err := r.Get(ctx, key, obj)
	if err != nil {
		logger.Error(err, "Failed to get AwsEc2Instance resource.")
		metadata["error"] = err.Error()
		return JobResultError, metadata
	}
	action, ok := obj.Spec.Actions[actionName]
	if !ok {
		logger.Info(fmt.Sprintf("Action %q not found in AWSEc2Instance resource.", actionName))
		metadata["error"] = fmt.Sprintf("Action %q not found in AWSEc2Instance resource.", actionName)
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

	actionType := conditionTypeForAction(actionName)
	errorsList := make([]string, 0)
	ec2Client := ec2.NewFromConfig(cfg)
	for _, ec2Instance := range obj.Spec.Instances {
		opts := func(o *ec2.Options) {
			o.Region = ec2Instance.Region
			if obj.Spec.ServiceEndpoint != nil {
				o.BaseEndpoint = obj.Spec.ServiceEndpoint
			}
		}

		switch action.Command {

		case v1alpha1.AwsEc2InstanceCommandStart:
			_, err := ec2Client.StartInstances(ctx, &ec2.StartInstancesInput{
				InstanceIds: []string{ec2Instance.Id},
			}, opts)
			if err != nil {
				msg := fmt.Sprintf("unable to start ec2 %q", ec2Instance.Id)
				logger.Error(err, msg)
				errorsList = append(errorsList, msg)
				r.Recorder.Eventf(obj, corev1.EventTypeWarning, "StartEc2Failed", msg)
				continue
			}

			r.Recorder.Eventf(obj, corev1.EventTypeNormal, "StartEc2Succeeded", "Ec2 %q is starting", ec2Instance.Id)
			logger.Info("Ec2 is starting", "identifier", ec2Instance.Id)

		case v1alpha1.AwsEc2InstanceCommandStop, v1alpha1.AwsEc2InstanceCommandHibernate:
			hibernate := action.Command == v1alpha1.AwsEc2InstanceCommandHibernate
			_, err := ec2Client.StopInstances(ctx, &ec2.StopInstancesInput{
				InstanceIds: []string{ec2Instance.Id},
				Hibernate:   &hibernate,
				Force:       action.Force,
			}, opts)
			if err != nil {
				msg := fmt.Sprintf("unable to stop ec2 %q", ec2Instance.Id)
				logger.Error(err, msg)
				errorsList = append(errorsList, msg)
				r.Recorder.Eventf(obj, corev1.EventTypeWarning, "StopEc2Failed", msg)
				continue
			}

			r.Recorder.Eventf(obj, corev1.EventTypeNormal, "StopEc2Succeeded", "Ec2 %q is stopping", ec2Instance.Id)
			logger.Info("Ec2 is stopping", "identifier", ec2Instance.Id)
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
func (r *AwsEc2InstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.AwsEc2Instance{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
