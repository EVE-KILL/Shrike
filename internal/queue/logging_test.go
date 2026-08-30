package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/rs/zerolog"
)

func TestJobLogMiddlewareReportsStartAndCompletion(t *testing.T) {
	var output bytes.Buffer
	middleware := newJobLogMiddleware(zerolog.New(&output))
	job := testJobRow()

	if err := middleware.Work(context.Background(), job, func(context.Context) error {
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	records := logRecords(t, output.String())
	if len(records) != 2 {
		t.Fatalf("got %d log records, want start and completion: %s", len(records), output.String())
	}
	assertLogRecord(t, records[0], "info", "job started")
	assertLogRecord(t, records[1], "info", "job completed")
	if got := records[0]["job_id"]; got != float64(job.ID) {
		t.Errorf("job_id = %v, want %d", got, job.ID)
	}
	if got := records[0]["queue"]; got != job.Queue {
		t.Errorf("queue = %v, want %s", got, job.Queue)
	}
	if _, ok := records[1]["duration"]; !ok {
		t.Error("completion log has no duration")
	}
}

func TestJobLogMiddlewareReportsRetryAndPermanentFailure(t *testing.T) {
	boom := errors.New("database unavailable")
	for _, tc := range []struct {
		name        string
		attempt     int
		wantLevel   string
		wantMessage string
	}{
		{
			name:        "retry",
			attempt:     1,
			wantLevel:   "warn",
			wantMessage: "job failed; retry scheduled",
		},
		{
			name:        "permanent",
			attempt:     3,
			wantLevel:   "error",
			wantMessage: "job failed permanently",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			middleware := newJobLogMiddleware(zerolog.New(&output))
			job := testJobRow()
			job.Attempt = tc.attempt

			err := middleware.Work(context.Background(), job, func(context.Context) error {
				return boom
			})
			if !errors.Is(err, boom) {
				t.Fatalf("Work returned %v, want %v", err, boom)
			}

			records := logRecords(t, output.String())
			if len(records) != 2 {
				t.Fatalf("got %d log records, want start and finish: %s", len(records), output.String())
			}
			assertLogRecord(t, records[1], tc.wantLevel, tc.wantMessage)
			if got := records[1]["error"]; got != boom.Error() {
				t.Errorf("error = %v, want %q", got, boom)
			}
		})
	}
}

func TestJobLogMiddlewareReportsSnoozeWithoutCallingItFailure(t *testing.T) {
	var output bytes.Buffer
	middleware := newJobLogMiddleware(zerolog.New(&output))

	err := middleware.Work(context.Background(), testJobRow(), func(context.Context) error {
		return river.JobSnooze(5 * time.Minute)
	})
	if _, ok := errors.AsType[*river.JobSnoozeError](err); !ok {
		t.Fatalf("Work returned %v, want JobSnoozeError", err)
	}

	records := logRecords(t, output.String())
	assertLogRecord(t, records[1], "info", "job snoozed")
}

func TestJobLogMiddlewareLeavesCronLoggingToCronWorker(t *testing.T) {
	var output bytes.Buffer
	middleware := newJobLogMiddleware(zerolog.New(&output))
	job := testJobRow()
	job.Queue = CronQueue

	called := false
	if err := middleware.Work(context.Background(), job, func(context.Context) error {
		called = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("cron job did not reach its worker")
	}
	if output.Len() != 0 {
		t.Fatalf("generic middleware duplicated cron logging: %s", output.String())
	}
}

func testJobRow() *rivertype.JobRow {
	return &rivertype.JobRow{
		ID:          42,
		Attempt:     1,
		Kind:        "killmail",
		Queue:       "killmail",
		MaxAttempts: 3,
	}
}

func logRecords(t *testing.T, output string) []map[string]any {
	t.Helper()
	var records []map[string]any
	for line := range strings.SplitSeq(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("decode log %q: %v", line, err)
		}
		records = append(records, record)
	}
	return records
}

func assertLogRecord(t *testing.T, record map[string]any, level, message string) {
	t.Helper()
	if got := record["level"]; got != level {
		t.Errorf("level = %v, want %q", got, level)
	}
	if got := record["message"]; got != message {
		t.Errorf("message = %v, want %q", got, message)
	}
}
