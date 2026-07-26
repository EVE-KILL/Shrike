package cron

import (
	"context"
	"fmt"
	"sort"

	"github.com/eve-kill/shrike/internal/jobs"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/riverqueue/river"
)

// RunFunc executes one scheduled job.
//
// The report string is a one-line summary for the operator — "3,412 rows, 2
// systems changed hands" — not a log line. It ends up on the job row, so a run
// that did nothing and a run that did a great deal are distinguishable after
// the fact rather than only while someone is watching.
type RunFunc func(ctx context.Context) (string, error)

// Registry maps cron names to their implementations.
//
// Separate from internal/jobs.Crons, which declares what exists and how often.
// This says what actually runs. The two are checked against each other at
// startup, which is how a cron that was declared but never implemented — or
// implemented under a misspelled name — becomes a startup error rather than a
// job that silently never runs.
type Registry struct {
	handlers map[string]RunFunc
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{handlers: map[string]RunFunc{}}
}

// Register adds an implementation.
//
// Registering a name that is not declared in internal/jobs.Crons is an error:
// it would run on no schedule at all, which is never what was meant.
func (r *Registry) Register(name string, fn RunFunc) error {
	if jobs.CronByName(name) == nil {
		return fmt.Errorf("cron %q is not declared in internal/jobs.Crons", name)
	}
	if _, exists := r.handlers[name]; exists {
		return fmt.Errorf("cron %q is registered twice", name)
	}
	r.handlers[name] = fn
	return nil
}

// MustRegister is Register for wiring that cannot meaningfully continue.
func (r *Registry) MustRegister(name string, fn RunFunc) {
	if err := r.Register(name, fn); err != nil {
		panic(err)
	}
}

// Lookup returns an implementation.
func (r *Registry) Lookup(name string) (RunFunc, bool) {
	// The legacy spelling resolves too, so `cron:run system-activity` keeps
	// working for anyone with it in their fingers or a runbook.
	if c := jobs.CronByName(name); c != nil {
		name = c.Name
	}
	fn, ok := r.handlers[name]
	return fn, ok
}

// Implemented returns the registered names, sorted.
func (r *Registry) Implemented() []string {
	out := make([]string, 0, len(r.handlers))
	for name := range r.handlers {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// Unimplemented returns declared crons with no implementation, sorted.
//
// During the port this is most of them, and surfacing it is the point: a port
// that quietly runs eight of thirty-two jobs looks identical from the outside
// to one that runs all thirty-two, right up until something is noticed missing
// weeks later.
func (r *Registry) Unimplemented() []string {
	var out []string
	for _, c := range jobs.Crons {
		if _, ok := r.handlers[c.Name]; !ok {
			out = append(out, c.Name)
		}
	}
	sort.Strings(out)
	return out
}

// Spec is one scheduled job, resolved.
//
// This exists as a separate step from building River's PeriodicJob because
// River's is opaque: its schedule and constructor are unexported, so once
// built there is no way to ask what it will run. Resolving to a Spec first
// makes the schedule inspectable — which is what `cron:list` prints and what
// the tests assert against — and leaves PeriodicJobs as mechanical translation.
type Spec struct {
	Name        string
	Description string
	Schedule    BoundarySchedule
	RunOnStart  bool
	RequiresTQ  bool
}

// Specs resolves every implemented cron to its schedule, in registry order.
//
// Only implemented crons are included. Scheduling a declared-but-unimplemented
// one would insert a job every interval that fails every time, which fills the
// error log with a fact already known and retries it forever.
func (r *Registry) Specs() ([]Spec, error) {
	var out []Spec

	for _, c := range jobs.Crons {
		if _, ok := r.handlers[c.Name]; !ok {
			continue
		}

		schedule, err := ScheduleFor(c)
		if err != nil {
			return nil, fmt.Errorf("cron %s: %w", c.Name, err)
		}

		out = append(out, Spec{
			Name:        c.Name,
			Description: c.Description,
			Schedule:    schedule,
			RunOnStart:  c.RunOnStart,
			RequiresTQ:  c.RequiresTQ,
		})
	}

	return out, nil
}

// PeriodicJobs builds the River periodic jobs for everything registered.
func (r *Registry) PeriodicJobs() ([]*river.PeriodicJob, error) {
	specs, err := r.Specs()
	if err != nil {
		return nil, err
	}

	out := make([]*river.PeriodicJob, 0, len(specs))
	for _, s := range specs {
		out = append(out, river.NewPeriodicJob(
			s.Schedule,
			func() (river.JobArgs, *river.InsertOpts) {
				return queue.CronArgs{Name: s.Name}, nil
			},
			&river.PeriodicJobOpts{
				ID:         s.Name,
				RunOnStart: s.RunOnStart,
			},
		))
	}
	return out, nil
}
