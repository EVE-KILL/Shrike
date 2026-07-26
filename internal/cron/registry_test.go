package cron

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/jobs"
)

func noop(context.Context) (string, error) { return "", nil }

// A cron registered under a name nothing schedules would never run, and would
// look exactly like one that runs and does nothing.
func TestRegisterRejectsAnUndeclaredCron(t *testing.T) {
	r := NewRegistry()

	err := r.Register("sovereignty", noop)
	if err != nil {
		t.Fatalf("registering a declared cron failed: %v", err)
	}

	err = r.Register("soverignty", noop) // misspelled
	if err == nil {
		t.Fatal("a cron name absent from the registry was accepted — it would be " +
			"scheduled by nothing and never run")
	}
	if !strings.Contains(err.Error(), "soverignty") {
		t.Errorf("the error does not name the offending cron: %v", err)
	}
}

// Two implementations for one name means one of them silently never runs.
func TestRegisterRejectsADuplicate(t *testing.T) {
	r := NewRegistry()
	if err := r.Register("wars", noop); err != nil {
		t.Fatal(err)
	}
	if err := r.Register("wars", noop); err == nil {
		t.Error("registering the same cron twice was accepted — one implementation " +
			"would be silently discarded")
	}
}

// The one hyphenated legacy name has to keep resolving, or a runbook entry
// stops working after the port.
func TestLookupResolvesTheLegacyName(t *testing.T) {
	r := NewRegistry()
	if err := r.Register("system_activity", noop); err != nil {
		t.Fatal(err)
	}

	if _, ok := r.Lookup("system-activity"); !ok {
		t.Error("the legacy hyphenated name no longer resolves")
	}
	if _, ok := r.Lookup("system_activity"); !ok {
		t.Error("the canonical name does not resolve")
	}
}

// The gap between declared and implemented is the port's progress bar, and it
// has to be reportable rather than guessed at.
func TestUnimplementedReportsTheGap(t *testing.T) {
	r := NewRegistry()
	r.MustRegister("sovereignty", noop)
	r.MustRegister("insurance", noop)

	implemented := r.Implemented()
	if len(implemented) != 2 {
		t.Errorf("Implemented() = %v, want two entries", implemented)
	}

	unimplemented := r.Unimplemented()
	if want := len(jobs.Crons) - 2; len(unimplemented) != want {
		t.Errorf("Unimplemented() has %d entries, want %d", len(unimplemented), want)
	}
	for _, name := range unimplemented {
		if name == "sovereignty" || name == "insurance" {
			t.Errorf("%s is implemented but reported as missing", name)
		}
	}

	// Together they must account for every declared cron — otherwise the report
	// is not a report, and a job could fall through both lists.
	if len(implemented)+len(unimplemented) != len(jobs.Crons) {
		t.Errorf("implemented (%d) + unimplemented (%d) != declared (%d): a cron is "+
			"in neither list", len(implemented), len(unimplemented), len(jobs.Crons))
	}
}

// Only implemented crons get scheduled. Scheduling the rest would insert a job
// every interval that fails every time.
func TestPeriodicJobsCoverOnlyWhatIsImplemented(t *testing.T) {
	r := NewRegistry()
	r.MustRegister("sovereignty", noop)
	r.MustRegister("insurance", noop)
	r.MustRegister("status_update", noop)

	periodic, err := r.PeriodicJobs()
	if err != nil {
		t.Fatal(err)
	}
	if len(periodic) != 3 {
		t.Errorf("built %d periodic jobs for 3 implementations", len(periodic))
	}
}

// Each spec carries its own cron's name and schedule. This is the assertion
// that a loop variable was not captured by reference, which would give every
// schedule the last cron's name and silently run one job thirty-two times.
func TestSpecsCarryTheirOwnCron(t *testing.T) {
	r := NewRegistry()
	for _, name := range []string{"sovereignty", "insurance", "status_update"} {
		r.MustRegister(name, noop)
	}

	specs, err := r.Specs()
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 3 {
		t.Fatalf("built %d specs for 3 implementations", len(specs))
	}

	byName := map[string]Spec{}
	for _, s := range specs {
		if _, dup := byName[s.Name]; dup {
			t.Fatalf("two specs are both named %q — a loop variable was captured "+
				"by reference and every schedule runs the same cron", s.Name)
		}
		byName[s.Name] = s
	}

	// The schedules must be each cron's own, not one shared value: sovereignty
	// is 6h, insurance 1d, status_update 1s.
	want := map[string]time.Duration{
		"sovereignty":   6 * time.Hour,
		"insurance":     24 * time.Hour,
		"status_update": time.Second,
	}
	for name, d := range want {
		s, ok := byName[name]
		if !ok {
			t.Errorf("no spec for %q", name)
			continue
		}
		if s.Schedule.Interval() != d {
			t.Errorf("%s scheduled every %v, want %v", name, s.Schedule.Interval(), d)
		}
	}

	// RunOnStart is per cron too, and status_update is the one of these three
	// that has it.
	if !byName["status_update"].RunOnStart {
		t.Error("status_update lost its RunOnStart flag")
	}
	if byName["insurance"].RunOnStart {
		t.Error("insurance gained a RunOnStart flag it was never declared with")
	}
}

// The River jobs are built from the specs, so the count must agree.
func TestPeriodicJobsMatchTheSpecs(t *testing.T) {
	r := NewRegistry()
	for _, name := range []string{"sovereignty", "insurance", "wars"} {
		r.MustRegister(name, noop)
	}

	specs, err := r.Specs()
	if err != nil {
		t.Fatal(err)
	}
	periodic, err := r.PeriodicJobs()
	if err != nil {
		t.Fatal(err)
	}
	if len(periodic) != len(specs) {
		t.Errorf("built %d periodic jobs from %d specs", len(periodic), len(specs))
	}
}

// RunOnce is the operator's escape hatch and must say clearly which of the two
// possible "no" answers applies.
func TestRunOnceDistinguishesUndeclaredFromUnimplemented(t *testing.T) {
	r := NewRegistry()
	ctx := context.Background()

	_, err := RunOnce(ctx, r, "not_a_real_cron")
	if err == nil || !strings.Contains(err.Error(), "not declared") {
		t.Errorf("running an unknown cron said %v, want that it is not declared", err)
	}

	_, err = RunOnce(ctx, r, "sovereignty")
	if err == nil || !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("running a declared but unported cron said %v, want that it is "+
			"declared but not implemented", err)
	}
}

// The report and the elapsed time come back, so a run that did nothing is
// distinguishable from one that did work.
func TestRunOnceReturnsTheReport(t *testing.T) {
	r := NewRegistry()
	r.MustRegister("analyze", func(context.Context) (string, error) {
		return "12 tables analyzed", nil
	})

	run, err := RunOnce(context.Background(), r, "analyze")
	if err != nil {
		t.Fatal(err)
	}
	if run.Report != "12 tables analyzed" {
		t.Errorf("report = %q", run.Report)
	}
	if run.Name != "analyze" {
		t.Errorf("name = %q", run.Name)
	}
}

// A failing cron reports its failure rather than swallowing it.
func TestRunOnceSurfacesFailure(t *testing.T) {
	boom := errors.New("EVE Ref is down")
	r := NewRegistry()
	r.MustRegister("insurance", func(context.Context) (string, error) { return "", boom })

	run, err := RunOnce(context.Background(), r, "insurance")
	if !errors.Is(err, boom) {
		t.Errorf("RunOnce returned %v, want the handler's error", err)
	}
	if !errors.Is(run.Err, boom) {
		t.Errorf("Run.Err = %v, want the handler's error", run.Err)
	}
}
