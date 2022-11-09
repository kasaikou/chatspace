package scheduler

import (
	"sync"
	"time"
)

type Scheduler struct {
	lock      sync.Mutex
	schedules []schedule
}

type schedule struct {
	unix int64
	call func()
}

func NewScheduler() *Scheduler {
	scheduler := &Scheduler{
		schedules: []schedule{},
	}

	go func() {
		for {
			st := time.Now()
			exited := func() bool {
				scheduler.lock.Lock()
				defer scheduler.lock.Unlock()
				if scheduler.schedules == nil {
					return true
				}

				schedules := scheduler.schedules

				for i := range schedules {
					for j := i; j-1 > -1; j-- {
						if schedules[j].unix < schedules[j-1].unix {
							schedules[j], schedules[j-1] = schedules[j-1], schedules[j]
						}
					}
				}

				current := time.Now().Unix()

				for len(schedules) > 0 {
					if schedules[0].unix < current {
						schedules[0].call()
						schedules = schedules[1:]
					} else {
						break
					}
				}

				scheduler.schedules = schedules
				return false
			}()
			if exited {
				return
			}

			lap := time.Since(st)
			if lap < time.Second {
				time.Sleep(time.Second - lap)
			}
		}
	}()

	return scheduler
}

func (s *Scheduler) CreateEventAt(at time.Time, call func()) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.schedules == nil {
		panic("scheduler was closed")
	}

	s.schedules = append(s.schedules, schedule{
		unix: at.Unix(),
		call: call,
	})
}

func (s *Scheduler) CreateEventAfter(duration time.Duration, call func()) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.schedules == nil {
		panic("scheduler was closed")
	}

	s.schedules = append(s.schedules, schedule{
		unix: time.Now().Add(duration).Unix(),
		call: call,
	})
}

func (s *Scheduler) Close() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.schedules = nil
	return nil
}
