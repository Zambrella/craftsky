package app

import "sync"

// dependencyCleanup owns resources in construction order and releases them in
// reverse dependency order. close is safe to call repeatedly so both partial
// construction failures and the normal process shutdown path can share it.
type dependencyCleanup struct {
	once  sync.Once
	steps []func()
}

func (cleanup *dependencyCleanup) add(step func()) {
	if cleanup == nil || step == nil {
		return
	}
	cleanup.steps = append(cleanup.steps, step)
}

func (cleanup *dependencyCleanup) close() {
	if cleanup == nil {
		return
	}
	cleanup.once.Do(func() {
		for index := len(cleanup.steps) - 1; index >= 0; index-- {
			cleanup.steps[index]()
		}
	})
}
