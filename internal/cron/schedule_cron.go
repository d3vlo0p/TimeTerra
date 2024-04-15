package cron

import (
	cron "github.com/robfig/cron/v3"
)

// shedule -> action -> list of cron based on
type ScheduleCron struct {
	m map[string]map[string][]int
	c *cron.Cron
}

func New() *ScheduleCron {
	return &ScheduleCron{
		m: make(map[string]map[string][]int),
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

func (sm *ScheduleCron) Add(schedule string, action string, spec string, cmd func()) (int, error) {
	id, err := sm.c.AddFunc(spec, cmd)
	if err != nil {
		return 0, err
	}

	if _, ok := sm.m[schedule]; !ok {
		sm.m[schedule] = make(map[string][]int)
	}
	if _, ok := sm.m[schedule][action]; !ok {
		sm.m[schedule][action] = make([]int, 0)
	}
	sm.m[schedule][action] = append(sm.m[schedule][action], int(id))

	return int(id), nil
}

func (sm *ScheduleCron) Remove(schedule string, action string, id int) {
	if _, ok := sm.m[schedule]; !ok {
		return
	}
	if _, ok := sm.m[schedule][action]; !ok {
		return
	}
	for i, v := range sm.m[schedule][action] {
		if v == id {
			sm.m[schedule][action] = append(sm.m[schedule][action][:i], sm.m[schedule][action][i+1:]...)
			break
		}
	}
	sm.c.Remove(cron.EntryID(id))
}

func (sm *ScheduleCron) GetActionList(schedule string) map[string][]int {
	if _, ok := sm.m[schedule]; !ok {
		return make(map[string][]int)
	}
	return sm.m[schedule]
}

func (sm *ScheduleCron) ListId(schedule string, action string) []int {
	if _, ok := sm.m[schedule]; !ok {
		return []int{}
	}
	if _, ok := sm.m[schedule][action]; !ok {
		return []int{}
	}
	return sm.m[schedule][action]
}

func (sm *ScheduleCron) IsValidCron(spec string) bool {
	_, err := cron.ParseStandard(spec)
	return err == nil
}
