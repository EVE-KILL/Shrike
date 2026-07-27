package cron

import (
	"context"
	"errors"
	"testing"

	"github.com/eve-kill/shrike/internal/queue"
	"github.com/riverqueue/river"
)

func TestWorkerReportsCronLifecycleInOrder(t *testing.T) {
	registry := NewRegistry()
	var events []string
	registry.MustRegister("analyze", func(context.Context) (string, error) {
		events = append(events, "handler")
		return "planner statistics updated", nil
	})

	var completed Run
	worker := &Worker{
		Registry: registry,
		OnStart: func(name string) {
			events = append(events, "start:"+name)
		},
		OnRun: func(run Run) {
			events = append(events, "finish:"+run.Name)
			completed = run
		},
	}

	err := worker.Work(context.Background(), &river.Job[queue.CronArgs]{
		Args: queue.CronArgs{Name: "analyze"},
	})
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"start:analyze", "handler", "finish:analyze"}
	if len(events) != len(want) {
		t.Fatalf("events = %v, want %v", events, want)
	}
	for i := range want {
		if events[i] != want[i] {
			t.Errorf("event %d = %q, want %q", i, events[i], want[i])
		}
	}
	if completed.Report != "planner statistics updated" {
		t.Errorf("report = %q", completed.Report)
	}
}

func TestWorkerReportsFailedCron(t *testing.T) {
	boom := errors.New("EVE Ref is unavailable")
	registry := NewRegistry()
	registry.MustRegister("insurance", func(context.Context) (string, error) {
		return "download failed", boom
	})

	var completed Run
	worker := &Worker{
		Registry: registry,
		OnRun:    func(run Run) { completed = run },
	}

	err := worker.Work(context.Background(), &river.Job[queue.CronArgs]{
		Args: queue.CronArgs{Name: "insurance"},
	})
	if !errors.Is(err, boom) {
		t.Fatalf("Work returned %v, want %v", err, boom)
	}
	if !errors.Is(completed.Err, boom) {
		t.Errorf("reported error = %v, want %v", completed.Err, boom)
	}
	if completed.Report != "download failed" {
		t.Errorf("report = %q", completed.Report)
	}
}

func TestWorkerDoesNotReportInvalidCronAsStarted(t *testing.T) {
	started := false
	finished := false
	worker := &Worker{
		Registry: NewRegistry(),
		OnStart:  func(string) { started = true },
		OnRun:    func(Run) { finished = true },
	}

	err := worker.Work(context.Background(), &river.Job[queue.CronArgs]{
		Args: queue.CronArgs{Name: "not_a_cron"},
	})
	var cancelErr *river.JobCancelError
	if !errors.As(err, &cancelErr) {
		t.Fatalf("Work returned %v, want JobCancelError", err)
	}
	if started || finished {
		t.Fatalf("invalid cron emitted lifecycle events: started=%v finished=%v", started, finished)
	}
}
