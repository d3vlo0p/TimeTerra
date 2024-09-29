package controller

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	v1alpha1 "github.com/d3vlo0p/TimeTerra/api/v1alpha1"
	"github.com/d3vlo0p/TimeTerra/internal/cron"
	"github.com/d3vlo0p/TimeTerra/monitoring"
	"github.com/d3vlo0p/TimeTerra/notification"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Reconciler interface {
	// client.Client
	Get(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) error
	client.StatusClient
	GetScheduleService() *cron.ScheduleService
	GetNotificationService() *notification.NotificationService
	GetRecorder() record.EventRecorder
	SetConditions(obj client.Object, conditions []metav1.Condition)
}

type ReconcileResource interface {
	client.Object
	GetSchedule() string
	GetStatus() v1alpha1.Status
	IsActive() bool
}

func reconcileResource[Action Activable](
	ctx context.Context,
	r Reconciler,
	obj ReconcileResource,
	resourceName string,
	actions map[string]Action,
	job func(ctx context.Context, logger logr.Logger, key types.NamespacedName, actionName string) (JobResult, JobMetadata),
) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("resource", resourceName)
	recorder := r.GetRecorder()
	scheduleName := obj.GetSchedule()
	scheduleService := r.GetScheduleService()
	notificationService := r.GetNotificationService()

	conditions := obj.GetStatus().Conditions
	if conditions == nil {
		conditions = make([]metav1.Condition, 0)
	}

	if !obj.IsActive() {
		disableResource(r.GetScheduleService(), &conditions, resourceName)
		addToConditions(&conditions, metav1.Condition{
			LastTransitionTime: metav1.Now(),
			Type:               "Ready",
			Status:             metav1.ConditionFalse,
			Reason:             "Disabled",
		})
		recorder.Eventf(obj, corev1.EventTypeNormal, "Disabled", "Resource %q is not active", obj.GetName())
		return ctrl.Result{}, updateStatus(ctx, r, obj, conditions, nil)
	}

	schedule := &v1alpha1.Schedule{}
	err := r.Get(ctx, client.ObjectKey{Name: scheduleName}, schedule)
	if err != nil {
		logger.Error(err, "Failed to get Schedule resource. Re-running reconcile.", "schedule", scheduleName)
		addToConditions(&conditions, metav1.Condition{
			LastTransitionTime: metav1.Now(),
			Type:               "Ready",
			Status:             metav1.ConditionFalse,
			Reason:             "ScheduleNotExist",
			Message:            "Schedule specified in spec.schedule do not exist",
		})
		recorder.Eventf(obj, corev1.EventTypeWarning, "ScheduleNotExist", "Schedule %q not exist", scheduleName)
		return ctrl.Result{}, updateStatus(ctx, r, obj, conditions, err)
	}

	removedActions := scheduleService.RemoveActionsOfResourceFromNonCurrentSchedule(scheduleName, resourceName)
	if len(removedActions) > 0 {
		logger.Info("Schedule has been changed, removing cron jobs of previous schedule")
		for _, action := range removedActions {
			removeFromConditions(&conditions, conditionTypeForAction(action))
		}
	}

	scheduledActions := make([]string, 0)
	for action, id := range scheduleService.GetActionsOfResource(scheduleName, resourceName) {
		entry := scheduleService.Get(id)
		if entry.Valid() {
			// delete scheduled action if action is not defined in instance spec
			if _, found := actions[action]; !found {
				logger.Info("Action is no more defined in spec, removing it from cron", "action", action)
				scheduleService.Remove(scheduleName, action, resourceName)
				removeFromConditions(&conditions, conditionTypeForAction(action))
			} else {
				scheduledActions = append(scheduledActions, action)
			}
		}
	}

	for actionName, action := range actions {
		actionName := actionName
		if !action.IsActive() {
			logger.Info("Action not active, removing it from cron", "action", actionName)
			scheduleService.Remove(scheduleName, actionName, resourceName)
			addToConditions(&conditions, metav1.Condition{
				LastTransitionTime: metav1.Now(),
				Type:               conditionTypeForAction(actionName),
				Status:             metav1.ConditionFalse,
				Reason:             "Disabled",
			})
			continue
		}
		if !contains(scheduledActions, actionName) {
			scheduleAction, found := schedule.Spec.Actions[actionName]
			if !found {
				logger.Info("The action is not specified in the referenced schedule", "action", actionName, "schedule", scheduleName)
				addToConditions(&conditions, metav1.Condition{
					LastTransitionTime: metav1.Now(),
					Type:               conditionTypeForAction(actionName),
					Status:             metav1.ConditionFalse,
					Reason:             "MissingAction",
					Message:            fmt.Sprintf("action not found in %q", scheduleName),
				})
				recorder.Eventf(obj, corev1.EventTypeWarning, "MissingAction", "action %q not found in %q", actionName, scheduleName)
				continue
			}
			logger.Info("The action is not scheduled, let's schedule it", "action", actionName, "schedule", scheduleName, "cron", scheduleAction.Cron)
			_, err := scheduleService.Add(scheduleName, actionName, resourceName, scheduleAction.Cron, func() {
				// retrive schedule for cheking if it is active
				// we do the check here because the check is simpler, and it avoids us having to delete and create objects in cron schedule
				innerLogger := logger.WithValues("action", actionName, "schedule", scheduleName)
				start := time.Now()
				s := &v1alpha1.Schedule{}
				err := r.Get(ctx, client.ObjectKey{Name: scheduleName}, s)
				if err != nil {
					innerLogger.Error(err, "Cron run, failed to get schedule")
					monitoring.TimeterraActionLatency.WithLabelValues(scheduleName, actionName, resourceName, JobResultError.String()).Observe(time.Since(start).Seconds())
					return
				}

				if !s.Spec.IsActive() {
					innerLogger.Info("The schedule is not active, skipping execution")
					monitoring.TimeterraActionLatency.WithLabelValues(scheduleName, actionName, resourceName, JobResultSkipped.String()).Observe(time.Since(start).Seconds())
					return
				}

				if a, ok := s.Spec.Actions[actionName]; !ok || !a.IsActive() {
					innerLogger.Info("The action is not active, skipping execution")
					monitoring.TimeterraActionLatency.WithLabelValues(scheduleName, actionName, resourceName, JobResultSkipped.String()).Observe(time.Since(start).Seconds())
					return
				}

				//Logic for managing time periods
				//Inactive periods have priority over active ones, no active periods means always active.
				//Truncate time to minute to reflect cron precision
				now := start.Truncate(time.Minute)
				if len(s.Spec.ActivePeriods) > 0 {
					active := false
					// check if current date is inside an Active Period, if not then skip exec
					for _, p := range s.Spec.ActivePeriods {
						if (now.After(p.Start.Time) && now.Before(p.End.Time)) || now.Equal(p.Start.Time) || now.Equal(p.End.Time) {
							active = true
							break
						}
					}
					if !active {
						innerLogger.Info("The execution job action is outside an active period, skipping execution")
						monitoring.TimeterraActionLatency.WithLabelValues(scheduleName, actionName, resourceName, JobResultSkipped.String()).Observe(time.Since(start).Seconds())
						return
					}
				}
				if len(s.Spec.InactivePeriods) > 0 {
					active := true
					// check if current date is inside an Inactive Period, if it is then skip exec
					for _, p := range s.Spec.InactivePeriods {
						if (now.After(p.Start.Time) && now.Before(p.End.Time)) || now.Equal(p.Start.Time) || now.Equal(p.End.Time) {
							active = false
							break
						}
					}
					if !active {
						innerLogger.Info("The execution job action  is inside an inactive period, skipping execution")
						monitoring.TimeterraActionLatency.WithLabelValues(scheduleName, actionName, resourceName, JobResultSkipped.String()).Observe(time.Since(start).Seconds())
						return
					}
				}

				innerLogger.Info("Scheduled job action is starting")
				status, metadata := job(ctx, innerLogger, client.ObjectKey{Name: obj.GetName(), Namespace: obj.GetNamespace()}, actionName)
				monitoring.TimeterraActionLatency.WithLabelValues(scheduleName, actionName, resourceName, status.String()).Observe(time.Since(start).Seconds())
				notificationService.Send(notification.NotificationBody{
					Schedule: scheduleName,
					Action:   actionName,
					Resource: resourceName,
					Status:   status.String(),
					Metadata: metadata,
				})
			})
			if err != nil {
				logger.Error(err, "failed to add cron job")
				addToConditions(&conditions, metav1.Condition{
					LastTransitionTime: metav1.Now(),
					Type:               conditionTypeForAction(actionName),
					Status:             metav1.ConditionFalse,
					Reason:             "Error",
					Message:            err.Error(),
				})
				recorder.Eventf(obj, corev1.EventTypeWarning, "FailedToSchedule", "Failed to schedule %q", actionName)
				return ctrl.Result{}, updateStatus(ctx, r, obj, conditions, err)
			}

			logger.Info("action is scheduled", "action", actionName)
			addToConditions(&conditions, metav1.Condition{
				LastTransitionTime: metav1.Now(),
				Type:               conditionTypeForAction(actionName),
				Status:             metav1.ConditionTrue,
				Reason:             "Active",
			})
		}
	}

	addToConditions(&conditions, metav1.Condition{
		LastTransitionTime: metav1.Now(),
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             "Active",
	})
	return ctrl.Result{}, updateStatus(ctx, r, obj, conditions, nil)
}

func updateStatus(ctx context.Context, r Reconciler, res ReconcileResource, conditions []metav1.Condition, err error) error {
	r.SetConditions(res, conditions)
	e := r.Status().Update(ctx, res)
	if e != nil {
		return errors.Join(e, err)
	}
	return err
}

func addToConditions(conditions *[]metav1.Condition, condition metav1.Condition) {
	meta.SetStatusCondition(conditions, condition)
}

func removeFromConditions(conditions *[]metav1.Condition, conditionType string) {
	meta.RemoveStatusCondition(conditions, conditionType)
}

func disableResource(scheduleService *cron.ScheduleService, conditions *[]metav1.Condition, resourceName string) {
	scheduleService.RemoveResource(resourceName)
	// remove condition if Type starts with "Action"
	if len(*conditions) > 0 {
		newConditions := make([]metav1.Condition, 0)
		for _, c := range *conditions {
			if !strings.HasPrefix(c.Type, "Action") {
				newConditions = append(newConditions, c)
			}
		}
		*conditions = newConditions
	}
}
