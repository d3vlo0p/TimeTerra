package cron

import (
	"context"
	"sync"
	"time"

	"github.com/d3vlo0p/TimeTerra/monitoring"
	"github.com/go-logr/logr"
	cron "github.com/robfig/cron/v3"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ScheduleService struct {
	mu sync.RWMutex
	// map of schedule => map of action => map of resource => list of cron ids
	m      map[string]map[string]map[string]int
	c      *cron.Cron
	logger logr.Logger
}

func New() *ScheduleService {
	return &ScheduleService{
		m:      make(map[string]map[string]map[string]int),
		c:      cron.New(),
		logger: log.Log.WithName("cron"),
	}
}

// start only if the manager is the leader
func (s *ScheduleService) NeedLeaderElection() bool {
	return true
}

func (s *ScheduleService) Start(ctx context.Context) error {
	s.logger = log.FromContext(ctx)
	s.c.Start()

	// stop internal cron when ctx is done
	stopSignal := make(chan struct{})
	go func() {
		<-ctx.Done()
		s.logger.Info("Shutting down cron jobs with timeout of 1 minute")
		cronCtx, cancel := context.WithTimeout(s.c.Stop(), 1*time.Minute)
		defer cancel()
		<-cronCtx.Done()
		s.logger.Info("stopped cron")
		close(stopSignal)
	}()
	s.logger.Info("started cron")

	// wait for stop signal
	<-stopSignal
	return nil
}

func (s *ScheduleService) Get(entry int) cron.Entry {
	return s.c.Entry(cron.EntryID(entry))
}

func (s *ScheduleService) Add(schedule string, action string, resource string, spec string, cmd func()) (int, error) {
	id, err := s.c.AddFunc(spec, cmd)
	if err != nil {
		return 0, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.m[schedule]; !ok {
		s.m[schedule] = make(map[string]map[string]int)
	}
	if _, ok := s.m[schedule][action]; !ok {
		s.m[schedule][action] = make(map[string]int)
	}
	s.m[schedule][action][resource] = int(id)
	s.logger.Info("added cron", "schedule", schedule, "action", action, "resource", resource, "spec", spec, "id", s.m[schedule][action][resource])
	monitoring.TimeterraScheduledCronJobs.WithLabelValues(schedule, action).Inc()

	return int(id), nil
}

func (s *ScheduleService) Remove(schedule string, action string, resource string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.removeLocked(schedule, action, resource)
}

func (s *ScheduleService) removeLocked(schedule string, action string, resource string) {
	if _, ok := s.m[schedule]; !ok {
		return
	}
	if _, ok := s.m[schedule][action]; !ok {
		return
	}
	if _, ok := s.m[schedule][action][resource]; !ok {
		return
	}
	s.c.Remove(cron.EntryID(s.m[schedule][action][resource]))
	delete(s.m[schedule][action], resource)
	s.logger.Info("removed cron", "schedule", schedule, "action", action, "resource", resource)
	monitoring.TimeterraScheduledCronJobs.WithLabelValues(schedule, action).Dec()
}

func (s *ScheduleService) GetActions(schedule string) map[string]map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	actions, ok := s.m[schedule]
	if !ok {
		s.logger.Info("no actions for schedule", "schedule", schedule)
		return make(map[string]map[string]int)
	}
	copyMap := make(map[string]map[string]int, len(actions))
	for action, resources := range actions {
		copyMap[action] = make(map[string]int, len(resources))
		for res, id := range resources {
			copyMap[action][res] = id
		}
	}
	return copyMap
}

func (s *ScheduleService) GetActionsOfResource(schedule string, resource string) map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.m[schedule]; !ok {
		s.logger.Info("no actions for schedule", "schedule", schedule)
		return make(map[string]int)
	}
	resourceMap := make(map[string]int)
	for action, v := range s.m[schedule] {
		if id, ok := v[resource]; ok {
			resourceMap[action] = id
		}
	}
	return resourceMap
}

func (s *ScheduleService) RemoveActionsOfResourceFromNonCurrentSchedule(currentSchedule string, resource string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	actionsRemoved := make([]string, 0)
	for schedule, actions := range s.m {
		if schedule != currentSchedule {
			for action, resources := range actions {
				if _, ok := resources[resource]; ok {
					s.removeLocked(schedule, action, resource)
					actionsRemoved = append(actionsRemoved, action)
				}
			}
		}
	}
	return actionsRemoved
}

func (s *ScheduleService) RemoveResource(resource string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for schedule, actions := range s.m {
		for action, resources := range actions {
			if _, ok := resources[resource]; ok {
				s.removeLocked(schedule, action, resource)
			}
		}
	}
}

func (s *ScheduleService) IsValidCron(spec string) bool {
	_, err := cron.ParseStandard(spec)
	return err == nil
}

func (s *ScheduleService) UpdateCronSpec(schedule string, action string, resource string, spec string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.m[schedule]; !ok {
		return false
	}
	if _, ok := s.m[schedule][action]; !ok {
		return false
	}
	if _, ok := s.m[schedule][action][resource]; !ok {
		return false
	}
	old := s.Get(s.m[schedule][action][resource])
	if old.ID == 0 {
		return false
	}
	_, err := cron.ParseStandard(spec)
	if err != nil {
		return false
	}
	// replace old cron with new one
	newJob := old.Job
	new, err := s.c.AddJob(spec, newJob)
	if err != nil {
		return false
	}
	s.m[schedule][action][resource] = int(new)
	s.c.Remove(old.ID)
	return true
}
