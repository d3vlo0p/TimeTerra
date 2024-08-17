package cron

import (
	"context"
	"time"

	"github.com/d3vlo0p/TimeTerra/monitoring"
	"github.com/go-logr/logr"
	cron "github.com/robfig/cron/v3"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ScheduleCron struct {
	// map of schedule => map of action => map of resource => list of cron ids
	m      map[string]map[string]map[string]int
	c      *cron.Cron
	logger logr.Logger
}

func New() *ScheduleCron {
	return &ScheduleCron{
		m: make(map[string]map[string]map[string]int),
		c: cron.New(),
	}
}

// start only if the manager is the leader
func (sm *ScheduleCron) NeedLeaderElection() bool {
	return true
}

func (sm *ScheduleCron) Start(ctx context.Context) error {
	sm.logger = log.FromContext(ctx)
	sm.c.Start()

	// stop internal cron when ctx is done
	stopSignal := make(chan struct{})
	go func() {
		<-ctx.Done()
		sm.logger.Info("Shutting down cron jobs with timeout of 1 minute")
		cronCtx, cancel := context.WithTimeout(sm.c.Stop(), 1*time.Minute)
		defer cancel()
		<-cronCtx.Done()
		sm.logger.Info("stopped cron")
		close(stopSignal)
	}()
	sm.logger.Info("started cron")

	// wait for stop signal
	<-stopSignal
	return nil
}

func (sm *ScheduleCron) Get(entry int) cron.Entry {
	return sm.c.Entry(cron.EntryID(entry))
}

func (sm *ScheduleCron) Add(schedule string, action string, resource string, spec string, cmd func()) (int, error) {
	id, err := sm.c.AddFunc(spec, cmd)
	if err != nil {
		return 0, err
	}

	if _, ok := sm.m[schedule]; !ok {
		sm.m[schedule] = make(map[string]map[string]int)
	}
	if _, ok := sm.m[schedule][action]; !ok {
		sm.m[schedule][action] = make(map[string]int)
	}
	sm.m[schedule][action][resource] = int(id)
	sm.logger.Info("added cron", "schedule", schedule, "action", action, "resource", resource, "spec", spec, "id", sm.m[schedule][action][resource])
	monitoring.TimeterraScheduledCronJobs.WithLabelValues(schedule, action).Inc()

	return int(id), nil
}

func (sm *ScheduleCron) Remove(schedule string, action string, resource string) {
	if _, ok := sm.m[schedule]; !ok {
		return
	}
	if _, ok := sm.m[schedule][action]; !ok {
		return
	}
	if _, ok := sm.m[schedule][action][resource]; !ok {
		return
	}
	sm.c.Remove(cron.EntryID(sm.m[schedule][action][resource]))
	delete(sm.m[schedule][action], resource)
	sm.logger.Info("removed cron", "schedule", schedule, "action", action, "resource", resource)
	monitoring.TimeterraScheduledCronJobs.WithLabelValues(schedule, action).Dec()
}

func (sm *ScheduleCron) GetActions(schedule string) map[string]map[string]int {
	if _, ok := sm.m[schedule]; !ok {
		sm.logger.Info("no actions for schedule", "schedule", schedule)
		return make(map[string]map[string]int)
	}
	return sm.m[schedule]
}

func (sm *ScheduleCron) GetActionIds(schedule string, action string) []int {
	if _, ok := sm.m[schedule]; !ok {
		sm.logger.Info("no actions for schedule", "schedule", schedule)
		return []int{}
	}
	if _, ok := sm.m[schedule][action]; !ok {
		sm.logger.Info("no actions for schedule", "schedule", schedule, "action", action)
		return []int{}
	}
	var ids []int
	for _, v := range sm.m[schedule][action] {
		ids = append(ids, v)
	}
	return ids
}

func (sm *ScheduleCron) GetActionsOfResource(schedule string, resource string) map[string]int {
	if _, ok := sm.m[schedule]; !ok {
		sm.logger.Info("no actions for schedule", "schedule", schedule)
		return make(map[string]int)
	}
	resourceMap := make(map[string]int)
	for action, v := range sm.m[schedule] {
		resourceMap[action] = v[resource]
	}
	return resourceMap
}

func (sm *ScheduleCron) RemoveActionsOfResourceFromNonCurrentSchedule(currentSchedule string, resource string) []string {
	actionsRemoved := make([]string, 0)
	for schedule, actions := range sm.m {
		if schedule != currentSchedule {
			for action, resources := range actions {
				if _, ok := resources[resource]; ok {
					sm.Remove(schedule, action, resource)
					actionsRemoved = append(actionsRemoved, action)
				}
			}
		}
	}
	return actionsRemoved
}

func (sm *ScheduleCron) RemoveResource(resource string) {
	for schedule, actions := range sm.m {
		for action, resources := range actions {
			if _, ok := resources[resource]; ok {
				sm.Remove(schedule, action, resource)
			}
		}
	}
}

func (sm *ScheduleCron) IsValidCron(spec string) bool {
	_, err := cron.ParseStandard(spec)
	return err == nil
}

func (sm *ScheduleCron) UpdateCronSpec(schedule string, action string, resource string, spec string) bool {
	if _, ok := sm.m[schedule]; !ok {
		return false
	}
	if _, ok := sm.m[schedule][action]; !ok {
		return false
	}
	if _, ok := sm.m[schedule][action][resource]; !ok {
		return false
	}
	old := sm.Get(sm.m[schedule][action][resource])
	if old.ID == 0 {
		return false
	}
	_, err := cron.ParseStandard(spec)
	if err != nil {
		return false
	}
	// replace old cron with new one
	newJob := old.Job
	new, err := sm.c.AddJob(spec, newJob)
	if err != nil {
		return false
	}
	sm.m[schedule][action][resource] = int(new)
	sm.c.Remove(old.ID)
	return true
}
