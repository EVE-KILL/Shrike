// Package cron schedules the recurring jobs.
//
// It replaces the TypeScript CronRunner, which was a process holding a
// setInterval per job. That arrangement had two properties worth losing: only
// one process could ever run it, because a second one would double-run every
// job, and a run that failed was simply gone — there was no record and no
// retry. Scheduling through River instead gives leader election, so any number
// of pods can be started and exactly one of them schedules, and it puts every
// run in a table with its arguments, its attempts and its errors.
//
// What is deliberately preserved is the schedule semantics: fixed intervals
// aligned to wall-clock boundaries, not cron expressions.
package cron

import (
	"time"

	"github.com/eve-kill/shrike/internal/jobs"
)

// BoundarySchedule fires on wall-clock boundaries measured from the Unix epoch.
//
// This reproduces msUntilNextBoundary() from backend/src/cron/CronRunner.ts, and
// it is the reason the schedules are intervals rather than cron expressions. A
// `1h` job fires at the top of every UTC hour and a `1d` job at UTC midnight,
// without any persisted state and without drifting: the next fire time is a
// function of the clock alone, so restarting a pod does not shift a daily job
// to whenever the pod happened to come up.
//
// The alternative — River's own PeriodicInterval — measures from when the
// scheduler started, which would make every deploy silently move every job.
type BoundarySchedule struct {
	interval time.Duration
}

// Every returns a schedule firing on boundaries of the given interval.
func Every(interval time.Duration) BoundarySchedule {
	return BoundarySchedule{interval: interval}
}

// Next returns the first boundary strictly after current.
//
// Strictly after matters: called at exactly a boundary — which is precisely
// when it is called, having just fired — returning that same instant would
// schedule a job to run immediately and forever.
func (s BoundarySchedule) Next(current time.Time) time.Time {
	interval := s.interval.Milliseconds()
	if interval <= 0 {
		// A zero interval would divide by zero and a negative one would run
		// backwards. Neither can be scheduled, so fall back to something valid
		// and let the registry validation report the real problem.
		return current.Add(time.Minute)
	}

	ms := current.UnixMilli()
	// ceil((ms+1)/interval) * interval, which for positive intervals is the
	// next multiple strictly greater than ms.
	next := (ms/interval + 1) * interval
	return time.UnixMilli(next).UTC()
}

// Interval reports the configured interval.
func (s BoundarySchedule) Interval() time.Duration { return s.interval }

// ScheduleFor resolves a cron's declared schedule string.
func ScheduleFor(c jobs.Cron) (BoundarySchedule, error) {
	d, err := jobs.ParseSchedule(c.Schedule)
	if err != nil {
		return BoundarySchedule{}, err
	}
	return Every(d), nil
}
