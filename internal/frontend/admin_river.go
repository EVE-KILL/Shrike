package frontend

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/eve-kill/shrike/internal/queue"
)

var adminRiverStates = []string{"available", "running", "scheduled", "retryable", "discarded", "cancelled", "completed"}

func adminRiverDepthCount(depth queue.Depth, state string) int64 {
	switch state {
	case "available":
		return depth.Available
	case "running":
		return depth.Running
	case "scheduled":
		return depth.Scheduled
	case "retryable":
		return depth.Retryable
	case "discarded":
		return depth.Discarded
	case "cancelled":
		return depth.Cancelled
	case "completed":
		return depth.Completed
	}
	return 0
}

type adminRiverQueue struct {
	Name        string      `json:"name"`
	Depth       queue.Depth `json:"depth"`
	Cron        bool        `json:"cron"`
	Concurrency *int        `json:"concurrency"`
	Description string      `json:"description"`
	Paused      *time.Time  `json:"paused_at"`
	Updated     *time.Time  `json:"worker_updated_at"`
	Active      bool        `json:"worker_active"`
}
type adminRiverJob struct {
	ID          int64           `json:"id"`
	Queue       string          `json:"queue"`
	Kind        string          `json:"kind"`
	State       string          `json:"state"`
	Attempt     int             `json:"attempt"`
	MaxAttempts int             `json:"max_attempts"`
	Priority    int             `json:"priority"`
	Created     *time.Time      `json:"created_at"`
	Scheduled   *time.Time      `json:"scheduled_at"`
	Finalized   *time.Time      `json:"finalized_at"`
	Workers     []string        `json:"attempted_by"`
	Args        json.RawMessage `json:"args"`
	Errors      json.RawMessage `json:"errors"`
	Output      json.RawMessage `json:"output"`
	Metadata    json.RawMessage `json:"metadata"`
}
type adminRiverQuery struct {
	Queue, State string
	Before       int64
}
type adminRiverQueues struct {
	Rows   []adminRiverQueue `json:"queues"`
	Failed bool              `json:"-"`
}
type adminRiverJobs struct {
	Rows   []adminRiverJob `json:"jobs"`
	Before int64           `json:"next_before_id"`
	Failed bool            `json:"-"`
}
type adminRiverData struct {
	Queues adminRiverQueues
	Jobs   adminRiverJobs
	Query  adminRiverQuery
}

func adminRiverSelection(query url.Values) adminRiverQuery {
	result := adminRiverQuery{Queue: strings.TrimSpace(query.Get("queue"))}
	for _, state := range adminRiverStates {
		if query.Get("state") == state {
			result.State = state
			break
		}
	}
	if before, err := strconv.ParseInt(query.Get("before_id"), 10, 64); err == nil && before > 0 {
		result.Before = before
	}
	return result
}
func (query adminRiverQuery) values() url.Values {
	values := url.Values{"queue": {query.Queue}, "state": {query.State}, "limit": {"100"}}
	if query.Before > 0 {
		values.Set("before_id", strconv.FormatInt(query.Before, 10))
	}
	return values
}
func adminRiverStateClass(state string) string {
	tone := map[string]string{"completed": "text-green-400", "running": "text-blue-400", "discarded": "text-red-400", "retryable": "text-yellow-400", "cancelled": "text-gray-500", "available": "text-cyan-400", "scheduled": "text-purple-400"}[state]
	if tone == "" {
		return "text-gray-400"
	}
	return tone
}
func adminRiverConcurrency(value *int) string {
	if value == nil {
		return "—"
	}
	return strconv.Itoa(*value)
}
func (q adminRiverQueue) pending() int64 {
	return q.Depth.Pending()
}
func adminRiverStamp(value *time.Time) string {
	if value == nil {
		return "—"
	}
	return value.In(time.Local).Format("1/2/2006, 3:04:05 PM")
}
func adminRiverTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}
func (job adminRiverJob) cancellable() bool {
	return job.State == "running" || job.State == "available" || job.State == "scheduled" || job.State == "retryable"
}
func (s *site) loadAdminRiver(r *http.Request, data *adminRiverData) {
	var tasks sync.WaitGroup
	tasks.Add(2)
	go func() {
		defer tasks.Done()
		data.Queues.Failed = s.accountRead(r, "/api/admin/river", &data.Queues) != nil
	}()
	go func() {
		defer tasks.Done()
		data.Jobs.Failed = s.accountRead(r, "/api/admin/river/jobs?"+data.Query.values().Encode(), &data.Jobs) != nil
	}()
	tasks.Wait()
}
func (s *site) adminRiverFragment(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.adminFragmentIdentity(w, r); !ok {
		return
	}
	switch r.PathValue("part") {
	case "queues":
		var data adminRiverQueues
		data.Failed = s.accountRead(r, "/api/admin/river", &data) != nil
		s.renderFragment(w, r, adminRiverQueueResults(data))
	case "jobs":
		var data adminRiverJobs
		data.Failed = s.accountRead(r, "/api/admin/river/jobs?"+adminRiverSelection(r.URL.Query()).values().Encode(), &data) != nil
		s.renderFragment(w, r, adminRiverJobResults(data))
	default:
		http.NotFound(w, r)
	}
}
