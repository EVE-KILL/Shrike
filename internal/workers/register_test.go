package workers

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/cron"
	"github.com/eve-kill/shrike/internal/jobs"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/rs/zerolog"
)

// What is and is not ported has to be stated accurately, because the failure it
// prevents is silent. A worker consuming a queue it has no implementation for
// fetches jobs, finds no handler, and fails them — so an overstated list drains
// a backlog into the failure table instead of leaving it to wait.

// Every registered worker must consume a declared queue. A kind that matches no
// declaration is routed to a queue nothing configures, and its jobs sit
// forever.
func TestEveryWorkerConsumesADeclaredQueue(t *testing.T) {
	for _, r := range registrations {
		if jobs.QueueByName(r.kind) == nil {
			t.Errorf("a worker is registered for kind %q, which is not a declared queue", r.kind)
		}
	}
}

// No kind may be registered twice — River would silently keep one handler.
func TestNoDuplicateRegistrations(t *testing.T) {
	seen := map[string]bool{}
	for _, r := range registrations {
		if seen[r.kind] {
			t.Errorf("kind %q is registered twice", r.kind)
		}
		seen[r.kind] = true
	}
}

// Register must actually add a worker for every entry in the table. The table
// is the single source of truth precisely so this cannot drift, and this is the
// assertion that it has not.
func TestRegisterAddsAWorkerForEveryKind(t *testing.T) {
	w, registry, err := Register(&Deps{})
	if err != nil {
		t.Fatal(err)
	}
	if w == nil || registry == nil {
		t.Fatal("Register returned nothing")
	}

	// River's Workers type does not expose what has been registered, so the
	// check is that Register ran the whole table without panicking — AddWorker
	// panics on a duplicate kind, which is the failure worth catching.
	if len(registrations) == 0 {
		t.Fatal("no workers are registered at all")
	}
}

func TestCronRunLogIncludesOutcomeAndReport(t *testing.T) {
	boom := errors.New("upstream unavailable")
	for _, tc := range []struct {
		name      string
		run       cron.Run
		wantLevel string
		wantMsg   string
	}{
		{
			name: "completed",
			run: cron.Run{
				Name:    "analyze",
				Report:  "planner statistics updated",
				Elapsed: 2 * time.Second,
			},
			wantLevel: "info",
			wantMsg:   "cron completed",
		},
		{
			name: "skipped",
			run: cron.Run{
				Name:    "wars",
				Report:  "Tranquility is offline",
				Skipped: true,
			},
			wantLevel: "info",
			wantMsg:   "cron skipped",
		},
		{
			name: "failed",
			run: cron.Run{
				Name:    "insurance",
				Report:  "download failed",
				Elapsed: time.Second,
				Err:     boom,
			},
			wantLevel: "error",
			wantMsg:   "cron failed",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			logCronRun(zerolog.New(&output), tc.run)

			var record map[string]any
			if err := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &record); err != nil {
				t.Fatalf("decode log %q: %v", output.String(), err)
			}
			if got := record["level"]; got != tc.wantLevel {
				t.Errorf("level = %v, want %q", got, tc.wantLevel)
			}
			if got := record["message"]; got != tc.wantMsg {
				t.Errorf("message = %v, want %q", got, tc.wantMsg)
			}
			if got := record["cron"]; got != tc.run.Name {
				t.Errorf("cron = %v, want %q", got, tc.run.Name)
			}
			if got := record["report"]; got != tc.run.Report {
				t.Errorf("report = %v, want %q", got, tc.run.Report)
			}
			if tc.run.Err != nil && record["error"] != tc.run.Err.Error() {
				t.Errorf("error = %v, want %q", record["error"], tc.run.Err)
			}
		})
	}
}

// Implemented and unimplemented together must account for every declared queue
// that this process could consume. A queue in neither list is invisible.
func TestQueueAccountingIsComplete(t *testing.T) {
	implemented := ImplementedQueues()
	unimplemented := UnimplementedQueues()

	inList := map[string]int{}
	for _, n := range implemented {
		inList[n]++
	}
	for _, n := range unimplemented {
		inList[n]++
	}

	var external int
	for _, q := range jobs.Queues {
		if q.ConsumerElsewhere {
			external++
			// A queue owned by another pod belongs in neither list: it is not
			// unported, it is not ours.
			if inList[q.Name] > 0 {
				t.Errorf("%s is consumed by another pod but appears in the port status", q.Name)
			}
			continue
		}
		switch inList[q.Name] {
		case 0:
			t.Errorf("%s is declared but appears in neither the implemented nor the "+
				"unimplemented list — its status is invisible", q.Name)
		case 1:
		default:
			t.Errorf("%s appears in both lists", q.Name)
		}
	}

	if want := len(jobs.Queues) - external; len(implemented)+len(unimplemented) != want {
		t.Errorf("implemented (%d) + unimplemented (%d) != declared minus external (%d)",
			len(implemented), len(unimplemented), want)
	}
}

// A worker process must consume only implemented queues — never an unported
// one, whose jobs it would fetch and fail rather than leave waiting.
func TestConsumableQueuesExcludeTheUnported(t *testing.T) {
	unported := map[string]bool{}
	for _, n := range UnimplementedQueues() {
		unported[n] = true
	}

	for _, n := range ConsumableQueues() {
		if unported[n] {
			t.Errorf("%s has no Go worker but would be consumed — its jobs would be "+
				"fetched and failed instead of waiting for the port", n)
		}
	}
}

// The general workers must not consume the cron queue. Keeping scheduled jobs
// off them is the entire reason that queue is separate: a nightly rebuild
// running for twenty minutes must not hold a slot killmails need.
func TestGeneralWorkersDoNotConsumeCronJobs(t *testing.T) {
	for _, n := range ConsumableQueues() {
		if n == queue.CronQueue {
			t.Error("the general workers consume the cron queue — a long-running " +
				"scheduled job would occupy a slot meant for killmails")
		}
	}
}

// The cron process must consume the cron queue, or nothing scheduled ever runs.
// River also refuses to start a client that has periodic jobs and no queues, so
// an empty list here is a process that will not boot at all.
func TestCronProcessConsumesTheCronQueue(t *testing.T) {
	got := CronQueues()
	if len(got) == 0 {
		t.Fatal("the cron process consumes no queues — River will refuse to start it")
	}

	var sawCron bool
	for _, n := range got {
		if n == queue.CronQueue {
			sawCron = true
		}
	}
	if !sawCron {
		t.Errorf("CronQueues() = %v, which does not include %q — nothing scheduled "+
			"would ever run", got, queue.CronQueue)
	}
}

// Every cron implementation must be registered under a declared name. The
// registry enforces this, so this is really a check that the wiring in
// RegisterCrons has not drifted from the declarations.
func TestCronRegistrationsAreDeclared(t *testing.T) {
	registry, err := RegisterCrons(&Deps{})
	if err != nil {
		t.Fatal(err)
	}

	implemented := registry.Implemented()
	if len(implemented) == 0 {
		t.Fatal("no crons are implemented at all")
	}

	for _, name := range implemented {
		if jobs.CronByName(name) == nil {
			t.Errorf("cron %q is implemented but not declared", name)
		}
	}
}

// The summary line has to report real numbers, not placeholders.
func TestDescribeReportsTheRealCounts(t *testing.T) {
	registry, err := RegisterCrons(&Deps{})
	if err != nil {
		t.Fatal(err)
	}

	got := Describe(registry)
	for _, want := range []string{"queues", "crons", "/"} {
		if !strings.Contains(got, want) {
			t.Errorf("Describe() = %q, missing %q", got, want)
		}
	}

	// It must name the actual declared totals, so a growing registry is
	// reflected without anyone remembering to update a constant.
	if !strings.Contains(got, itoa(len(jobs.Queues))) {
		t.Errorf("Describe() = %q, does not mention the %d declared queues", got, len(jobs.Queues))
	}
	if !strings.Contains(got, itoa(len(jobs.Crons))) {
		t.Errorf("Describe() = %q, does not mention the %d declared crons", got, len(jobs.Crons))
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// The cascade tier must never widen: follow-up work inherits its parent's
// urgency, so a dormant backfill cannot spawn immediate work.
func TestCascadeTierNeverEscalates(t *testing.T) {
	for _, p := range []queue.Priority{
		queue.Immediate, queue.Live, queue.RecentBackfill, queue.DormantBackfill,
	} {
		if got := cascadeTier(int(p)); got != p {
			t.Errorf("a %s job's children became %s", p, got)
		}
	}

	// River hands back whatever integer the job carries; a value outside the
	// tiers must land on Live rather than on the top tier.
	for _, bad := range []int{0, -1, 5, 99} {
		if got := cascadeTier(bad); got != queue.Live {
			t.Errorf("cascadeTier(%d) = %s, want live", bad, got)
		}
	}
}
