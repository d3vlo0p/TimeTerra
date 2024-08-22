package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/credentials"
	v1alpha1 "github.com/d3vlo0p/TimeTerra/api/v1alpha1"
	sc "github.com/d3vlo0p/TimeTerra/internal/cron"
	"github.com/d3vlo0p/TimeTerra/monitoring"
	"github.com/d3vlo0p/TimeTerra/notification"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func ResourceName(kind string, name string) string {
	return fmt.Sprintf("%s:%s", kind, name)
}

func ConditionTypeForAction(name string) string {
	return fmt.Sprintf("Action-%s", name)
}

func contains(actions []string, action string) bool {
	for _, a := range actions {
		if a == action {
			return true
		}
	}
	return false
}

func disableResource(cron *sc.ScheduleService, conditions *[]metav1.Condition, resourceName string) {
	cron.RemoveResource(resourceName)
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
	addToConditions(conditions, metav1.Condition{
		LastTransitionTime: metav1.Now(),
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		Reason:             "Disabled",
		Message:            "This Resource target is disabled",
	})
}

type Activable interface {
	IsActive() bool
}

type JobResult string

const (
	JobResultSuccess JobResult = "Success"
	JobResultFailure JobResult = "Failure"
	JobResultError   JobResult = "Error"
	JobResultSkipped JobResult = "Skipped"
)

func (jr JobResult) String() string {
	return string(jr)
}

type JobMetadata map[string]any

func reconcileResource[ActionType Activable](
	ctx context.Context,
	req ctrl.Request,
	cli client.Client,
	cron *sc.ScheduleService,
	notificationService *notification.NotificationService,
	instanceActions map[string]ActionType,
	scheduleName string,
	resourceName string,
	job func(ctx context.Context, key types.NamespacedName, actionName string) (JobResult, JobMetadata),
	conditions *[]metav1.Condition,
) error {
	logger := log.FromContext(ctx)

	schedule := &v1alpha1.Schedule{}
	err := cli.Get(ctx, client.ObjectKey{Name: scheduleName}, schedule)
	if err != nil {
		if errors.IsNotFound(err) {
			// return an error to force reconciliation, this is to fix the fact that if you add the missing schedule later, it fixes itself
			logger.Info("Schedule resource not found. Re-running reconcile.")
			addToConditions(conditions, metav1.Condition{
				LastTransitionTime: metav1.Now(),
				Type:               "Ready",
				Status:             metav1.ConditionFalse,
				Reason:             "Error",
				Message:            fmt.Sprintf("Schedule %s not found", scheduleName),
			})
		} else {
			logger.Info("Failed to get Schedule resource. Re-running reconcile.")
		}
		return err
	}

	removedActions := cron.RemoveActionsOfResourceFromNonCurrentSchedule(scheduleName, resourceName)
	if len(removedActions) > 0 {
		logger.Info("Schedule has been changed, removed cron jobs of previous schedule")
		for _, action := range removedActions {
			removeFromConditions(conditions, ConditionTypeForAction(action))
		}
	}

	scheduledActions := make([]string, 0)
	for action, id := range cron.GetActionsOfResource(scheduleName, resourceName) {
		entry := cron.Get(id)
		if entry.Valid() {
			// delete scheduled action if action is not defined in instance spec
			if _, found := instanceActions[action]; !found {
				logger.Info(fmt.Sprintf("Action %q is no more defined in spec, removing it from cron", action))
				cron.Remove(scheduleName, action, resourceName)
				removeFromConditions(conditions, ConditionTypeForAction(action))
			} else {
				scheduledActions = append(scheduledActions, action)
			}
		}
	}

	for actionName, action := range instanceActions {
		actionName := actionName
		if !action.IsActive() {
			logger.Info(fmt.Sprintf("Action %q is not active, removing it from cron", actionName))
			cron.Remove(scheduleName, actionName, resourceName)
			addToConditions(conditions, metav1.Condition{
				LastTransitionTime: metav1.Now(),
				Type:               ConditionTypeForAction(actionName),
				Status:             metav1.ConditionFalse,
				Reason:             "Disabled",
				Message:            "This Action target is disabled",
			})
			continue
		}
		if !contains(scheduledActions, actionName) {
			scheduleAction, found := schedule.Spec.Actions[actionName]
			if !found {
				logger.Info(fmt.Sprintf("action %q is not defined in schedule %s", actionName, scheduleName))
				addToConditions(conditions, metav1.Condition{
					LastTransitionTime: metav1.Now(),
					Type:               "Ready",
					Status:             metav1.ConditionFalse,
					Reason:             "MissingAction",
					Message:            fmt.Sprintf("action %s not found in %s", actionName, scheduleName),
				})
				continue
			}
			logger.Info(fmt.Sprintf("Action %q is not scheduled, scheduling it with %q", actionName, scheduleAction.Cron))
			_, err := cron.Add(scheduleName, actionName, resourceName, scheduleAction.Cron, func() {
				// retrive schedule for cheking if it is active
				// we do the check here because the check is simpler, and it avoids us having to delete and create objects in cron schedule
				start := time.Now()
				s := &v1alpha1.Schedule{}
				err := cli.Get(ctx, client.ObjectKey{Name: scheduleName}, s)
				if err != nil {
					logger.Error(err, "Cron run, failed to get schedule", "name", scheduleName)
					monitoring.TimeterraActionLatency.WithLabelValues(scheduleName, actionName, resourceName, JobResultError.String()).Observe(time.Since(start).Seconds())
					return
				}

				if !s.Spec.IsActive() {
					logger.Info(fmt.Sprintf("Achedule %q is not active, skipping execution", scheduleName))
					monitoring.TimeterraActionLatency.WithLabelValues(scheduleName, actionName, resourceName, JobResultSkipped.String()).Observe(time.Since(start).Seconds())
					return
				}

				if a, ok := s.Spec.Actions[actionName]; !ok || !a.IsActive() {
					logger.Info(fmt.Sprintf("Action %q is not active, skipping execution", actionName))
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
						logger.Info(fmt.Sprintf("Scheduled action %q is outside an active period, skipping execution", actionName))
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
						logger.Info(fmt.Sprintf("Scheduled action %q is inside an inactive period, skipping execution", actionName))
						monitoring.TimeterraActionLatency.WithLabelValues(scheduleName, actionName, resourceName, JobResultSkipped.String()).Observe(time.Since(start).Seconds())
						return
					}
				}

				logger.Info(fmt.Sprintf("Scheduled action %q is starting for resource %q", actionName, resourceName))
				status, metadata := job(ctx, req.NamespacedName, actionName)
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
				return err
			}
			logger.Info(fmt.Sprintf("action %q is scheduled", actionName))
			addToConditions(conditions, metav1.Condition{
				LastTransitionTime: metav1.Now(),
				Type:               ConditionTypeForAction(actionName),
				Status:             metav1.ConditionTrue,
				Reason:             "Scheduled",
				Message:            fmt.Sprintf("action %s is scheduled", actionName),
			})
		}
	}
	addToConditions(conditions, metav1.Condition{
		LastTransitionTime: metav1.Now(),
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             "Ready",
	})
	return nil
}

func addToConditions(conditions *[]metav1.Condition, condition metav1.Condition) {
	meta.SetStatusCondition(conditions, condition)
}

func removeFromConditions(conditions *[]metav1.Condition, conditionType string) {
	meta.RemoveStatusCondition(conditions, conditionType)
}

func removeMissingActionFromConditions(conditions *[]metav1.Condition, actions []string) {
	// remove from orphanedConditions all current action, so will remain only conditions without actions
	orphanedConditions := *conditions
	for _, action := range actions {
		cond := meta.FindStatusCondition(orphanedConditions, ConditionTypeForAction(action))
		if cond != nil {
			// find cond inside orphanedConditions and remove it
			for _, c := range orphanedConditions {
				if c.Type == cond.Type {
					removeFromConditions(&orphanedConditions, c.Type)
					break
				}
			}
		}
	}
	// remove orphaned conditions
	for _, c := range orphanedConditions {
		removeFromConditions(conditions, c.Type)
	}
}

// func printBytes(b []byte) string {
// 	r := ""
// 	for i := 0; i < len(b); i++ {
// 		r = strings.Join([]string{r, fmt.Sprintf("%d", b[i])}, ", ")
// 	}
// 	return r
// }

func decodeSecret(secret *corev1.Secret) (map[string]string, error) {
	values := make(map[string]string, len(secret.Data))
	for k, v := range secret.Data {
		// log.Log.Info("decoding secret", "key", k, "value", v, "bytes", printBytes(v), "dec", string(v))
		// dst := make([]byte, base64.StdEncoding.DecodedLen(len(v)))
		// n, err := base64.StdEncoding.Decode(dst, v)
		// if err != nil {
		// 	return nil, err
		// }
		// log.Log.Info("decoded secret", "key", k, "dst", dst[:n], "bytes_dst", printBytes(dst[:n]), "v", v, "bytes_v", printBytes(v))
		// values[k] = string(dst[:n])

		// seems that string() does already base64 decoding
		// because if i print "v" in the logs i see the base64 string, but if i try to convert to string, the result is alredy base64 decoded.
		values[k] = string(v)
	}
	// log.Log.Info("decoded secret", "values", values)
	return values, nil
}

func getAwsCredentialProviderFromSecret(secret *corev1.Secret, keysRef *v1alpha1.AwsCredentialsKeysRef) (*credentials.StaticCredentialsProvider, error) {
	values, err := decodeSecret(secret)
	if err != nil {
		return nil, err
	}

	var kAcceccKey, kSecretKey, kSessionKey = "accessKey", "secretKey", "sessionKey"
	// override default keys if specified
	if keysRef != nil {
		kAcceccKey, kSecretKey, kSessionKey = keysRef.AccessKey, keysRef.SecretKey, keysRef.SessionKey
	}

	accessKey, ok := values[kAcceccKey]
	if !ok {
		return nil, fmt.Errorf("failed to get AccessKey from Secret resource, no %q key found", kAcceccKey)
	}
	secretKey, ok := values[kSecretKey]
	if !ok {
		return nil, fmt.Errorf("failed to get SecretKey from Secret resource, no %q key found", kSecretKey)
	}
	sessionKey := ""
	if kSessionKey != "" {
		sessionKey, ok = values[kSessionKey]
		if !ok && keysRef != nil {
			return nil, fmt.Errorf("failed to get SessionKey from Secret resource, no %q key found", kSessionKey)
		}
	}

	r := credentials.NewStaticCredentialsProvider(
		accessKey,
		secretKey,
		sessionKey,
	)

	return &r, nil
}
