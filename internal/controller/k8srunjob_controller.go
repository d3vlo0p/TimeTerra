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

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	batchv1 "k8s.io/api/batch/v1"
)

// K8sRunJobReconciler reconciles a K8sRunJob object
type K8sRunJobReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	Cron                *cron.ScheduleCron
	NotificationService *notification.NotificationService
	Recorder            record.EventRecorder
}

//+kubebuilder:rbac:groups=core.timeterra.d3vlo0p.dev,resources=k8srunjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.timeterra.d3vlo0p.dev,resources=k8srunjobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.timeterra.d3vlo0p.dev,resources=k8srunjobs/finalizers,verbs=update
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=create
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the K8sRunJob object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *K8sRunJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info(fmt.Sprintf("reconciling object %#q", req.NamespacedName))

	resourceName := ResourceName("K8sRunJob", req.Name)
	instance := &corev1alpha1.K8sRunJob{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("RunJob resource not found. object must has been deleted.")
			r.Cron.RemoveResource(resourceName)
			return ctrl.Result{}, nil
		}
		logger.Info("Failed to get RunJob resource. Re-running reconcile.")
		return ctrl.Result{}, err
	}

	if instance.Status.Conditions == nil {
		instance.Status.Conditions = make([]metav1.Condition, 0)
	}
	if instance.Spec.Enabled != nil && !*instance.Spec.Enabled {
		disableResource(r.Cron, &instance.Status.Conditions, resourceName)
	} else {
		err := reconcileResource(ctx, req, r.Client, r.Cron, r.NotificationService, instance.Spec.Actions, instance.Spec.Schedule, resourceName, r.runJob, &instance.Status.Conditions)
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

func (r *K8sRunJobReconciler) runJob(ctx context.Context, key types.NamespacedName, actionName string) JobResult {
	logger := log.FromContext(ctx)
	start := time.Now()
	obj := &corev1alpha1.K8sRunJob{}
	err := r.Get(ctx, key, obj)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Error(err, "RunJob resource not found. object must has been deleted.")
			return JobResultError
		}
		logger.Error(err, "Failed to get RunJob resource.")
		return JobResultError
	}

	action, ok := obj.Spec.Actions[actionName]
	if !ok {
		logger.Info("Action not found")
		return JobResultError
	}

	actionType := ConditionTypeForAction(actionName)
	errorsList := make([]string, 0)

	namespaces := obj.Spec.Namespaces
	if len(namespaces) == 0 {
		namespaces = []string{obj.Namespace}
	}

	//generate string suffix date with format YYYYMMDDHHmm
	suffix := time.Now().Format("200601021504")
	jobName := fmt.Sprintf("%s-%s-%s", obj.Name, actionName, suffix)

	//set default TTL
	if action.Job.TTLSecondsAfterFinished == nil {
		defaultTTLSeconds := new(int32)
		*defaultTTLSeconds = 86400
		action.Job.TTLSecondsAfterFinished = defaultTTLSeconds
	}

	for _, namespace := range namespaces {
		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      jobName,
				Namespace: namespace,
				Labels: map[string]string{
					"timeterra.d3vlo0p.dev/action":   actionName,
					"timeterra.d3vlo0p.dev/schedule": obj.Spec.Schedule,
				},
			},
			Spec: action.Job,
		}

		err = ctrl.SetControllerReference(obj, job, r.Scheme)
		if err != nil {
			msg := fmt.Sprintf("Failed to set controller reference: %s", err)
			errorsList = append(errorsList, msg)
			logger.Error(err, msg)
			r.Recorder.Eventf(obj, "Warning", "Failed", msg)
			continue
		}

		err = r.Create(ctx, job)
		if err != nil {
			msg := fmt.Sprintf("Failed to create job: %s", err)
			errorsList = append(errorsList, msg)
			logger.Error(err, msg)
			r.Recorder.Eventf(obj, "Warning", "Failed", msg)
			continue
		}

		r.Recorder.Eventf(obj, "Normal", "Success", fmt.Sprintf("Job %q created in namespace %q", jobName, namespace))
		logger.Info("Job created", "Job", jobName, "Namespace", namespace)
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
func (r *K8sRunJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.K8sRunJob{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
