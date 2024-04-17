package cron

import (
	cron "github.com/robfig/cron/v3"
	ctrl "sigs.k8s.io/controller-runtime"
)

var log = ctrl.Log.WithName("schedule_cron")

type ScheduleCron struct {
	// map of schedule => map of action => map of resource => list of cron ids
	m map[string]map[string]map[string]int
	c *cron.Cron
}

func New() *ScheduleCron {
	return &ScheduleCron{
		m: make(map[string]map[string]map[string]int),
		c: cron.New(),
	}
}

func (sm *ScheduleCron) Start() {
	sm.c.Start()
}

func (sm *ScheduleCron) Stop() {
	sm.c.Stop()
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
	log.Info("added cron", "schedule", schedule, "action", action, "resource", resource, "spec", spec, "id", sm.m[schedule][action][resource])

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
	log.Info("removed cron", "schedule", schedule, "action", action, "resource", resource)
}

func (sm *ScheduleCron) GetActions(schedule string) map[string]map[string]int {
	if _, ok := sm.m[schedule]; !ok {
		log.Info("no actions for schedule", "schedule", schedule)
		return make(map[string]map[string]int)
	}
	return sm.m[schedule]
}

func (sm *ScheduleCron) GetActionIds(schedule string, action string) []int {
	if _, ok := sm.m[schedule]; !ok {
		log.Info("no actions for schedule", "schedule", schedule)
		return []int{}
	}
	if _, ok := sm.m[schedule][action]; !ok {
		log.Info("no actions for schedule", "schedule", schedule, "action", action)
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
		log.Info("no actions for schedule", "schedule", schedule)
		return make(map[string]int)
	}
	resourceMap := make(map[string]int)
	for action, v := range sm.m[schedule] {
		resourceMap[action] = v[resource]
	}
	return resourceMap
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

func (sm *ScheduleCron) UpdateCronSpec(id int, spec string) bool {
	entry := sm.Get(id)
	if entry.ID == 0 {
		return false
	}
	schedule, err := cron.ParseStandard(spec)
	if err != nil {
		return false
	}
	entry.Schedule = schedule
	return true
}
