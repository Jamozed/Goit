// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package cron

import (
	"log"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Jamozed/Goit/src/util"
)

type Cron struct {
	jobs    []Job
	stop    chan struct{}
	update  chan struct{}
	running atomic.Bool
	mutex   sync.Mutex
	lastId  uint64
	waiter  sync.WaitGroup
}

type Job struct {
	Id         uint64
	Rid        int64
	Schedule   Schedule
	Next, Last time.Time
	fn         func()
}

const maxDuration time.Duration = 1<<63 - 1

func New() *Cron {
	return &Cron{
		jobs:   []Job{},
		stop:   make(chan struct{}),
		update: make(chan struct{}),
	}
}

func (c *Cron) Start() {
	c.mutex.Lock()
	util.Debugln("[cron.Start] Cron mutex lock")
	defer c.mutex.Unlock()
	defer util.Debugln("[cron.Start] Cron mutex unlock")

	if !c.running.CompareAndSwap(false, true) {
		return
	}

	for _, job := range c.jobs {
		job.Next = job.Schedule.Next(time.Now().UTC())
	}

	go func() {
		for {
			c.mutex.Lock()
			util.Debugln("[cron.run] Cron mutex lock")

			var timer *time.Timer

			if len(c.jobs) == 0 {
				timer = time.NewTimer(maxDuration)
			} else {
				timer = time.NewTimer(c.jobs[0].Next.Sub(time.Now().UTC()))
			}

			c.mutex.Unlock()
			util.Debugln("[cron.run] Cron mutex unlock")

			select {
			case now := <-timer.C:
				now = now.UTC()
				log.Println("[cron] timer expired")

				c.mutex.Lock()
				util.Debugln("[cron.now] Cron mutex lock")

				tmp := c.jobs[:0]
				for _, job := range c.jobs {
					if job.Next.After(now) || job.Next.IsZero() {
						tmp = append(tmp, job)
						continue
					}

					log.Println("[cron] running job", job.Id, job.Rid)

					j := job
					c.waiter.Add(1)
					go func() {
						defer c.waiter.Done()
						j.fn()
					}()

					if !job.Schedule.IsImmediate() {
						job.Next = job.Schedule.Next(now)
						job.Last = now
						tmp = append(tmp, job)
					}
				}

				c.jobs = tmp

				util.Debugln("[cron.now] Cron mutex unlock")
				c.mutex.Unlock()

				c._update()

			case <-c.stop:
				timer.Stop()

				c.mutex.Lock()
				util.Debugln("[cron.stop] Cron mutex lock")

				c.waiter.Wait()
				c.running.Store(false)

				util.Debugln("[cron.stop] Cron mutex unlock")
				c.mutex.Unlock()

				return

			case <-c.update:
				c._update()
			}
		}
	}()
}

func (c *Cron) Stop() {
	if !c.running.Load() {
		return
	}

	close(c.stop)
}

func (c *Cron) Update() {
	if !c.running.Load() {
		return
	}

	c.update <- struct{}{}
}

func (c *Cron) _update() {
	c.mutex.Lock()
	util.Debugln("[cron.Update] Cron mutex lock")
	defer c.mutex.Unlock()
	defer util.Debugln("[cron.Update] Cron mutex unlock")

	now := time.Now().UTC()
	slices.SortFunc(c.jobs, func(a, b Job) int {
		return a.Schedule.Next(now).Compare(b.Schedule.Next(now))
	})
}

func (c *Cron) Jobs() []Job {
	c.mutex.Lock()
	util.Debugln("[cron.Jobs] Cron mutex lock")
	defer c.mutex.Unlock()
	defer util.Debugln("[cron.Jobs] Cron mutex unlock")

	jobs := make([]Job, len(c.jobs))
	copy(jobs, c.jobs)
	return jobs
}

func (c *Cron) Add(rid int64, schedule Schedule, fn func()) uint64 {
	c.mutex.Lock()
	util.Debugln("[cron.Add] Cron mutex lock")
	defer c.mutex.Unlock()
	defer util.Debugln("[cron.Add] Cron mutex unlock")

	c.lastId += 1

	job := Job{Id: c.lastId, Rid: rid, Schedule: schedule, fn: fn}
	job.Next = job.Schedule.Next(time.Now().UTC())
	c.jobs = append(c.jobs, job)

	log.Println("[cron] added job", job.Id, "for", job.Rid)
	return job.Id
}

func (c *Cron) RemoveFor(rid int64) {
	c.mutex.Lock()
	util.Debugln("[cron.RemoveFor] Cron mutex lock")
	defer c.mutex.Unlock()
	defer util.Debugln("[cron.RemoveFor] Cron mutex unlock")

	tmp := c.jobs[:0]
	for _, job := range c.jobs {
		if job.Rid != rid {
			tmp = append(tmp, job)
		} else {
			log.Println("[cron] removing job", job.Id, "for", job.Rid)
		}
	}

	c.jobs = tmp
}
