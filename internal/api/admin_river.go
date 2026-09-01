package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/eve-kill/shrike/internal/jobs"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

const adminRiverBodyLimit = 16 << 10

var riverJobStates = map[string]bool{
	"available": true, "cancelled": true, "completed": true,
	"discarded": true, "pending": true, "retryable": true,
	"running": true, "scheduled": true,
}

type adminRiverQueueActionBody struct {
	Action string `json:"action" enum:"pause,resume" doc:"Queue control action."`
}

type adminRiverClearBody struct {
	States []string `json:"states" minItems:"1" doc:"Final or waiting states to delete. Running jobs cannot be cleared."`
	Limit  int      `json:"limit,omitempty" minimum:"1" maximum:"10000" default:"1000"`
}

type adminRiverJobActionBody struct {
	Action string `json:"action" enum:"cancel,retry,delete" doc:"Job control action."`
}

func (s *adminService) riverOverviewHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		if _, err := s.requireAdmin(ctx, req, false); err != nil {
			return legacyPayload{}, err
		}
		if s.opts.PrimaryPool == nil {
			return legacyPayload{}, apiError(http.StatusServiceUnavailable, "River database unavailable")
		}

		depths, err := queue.Depths(ctx, s.opts.PrimaryPool)
		if err != nil {
			return legacyPayload{}, err
		}
		client, err := queue.New(queue.Options{Pool: s.opts.PrimaryPool})
		if err != nil {
			return legacyPayload{}, err
		}
		listed, err := client.QueueList(ctx, river.NewQueueListParams().First(10000))
		if err != nil {
			return legacyPayload{}, err
		}
		live := make(map[string]*rivertype.Queue, len(listed.Queues))
		for _, q := range listed.Queues {
			live[q.Name] = q
		}

		declared := map[string]jobs.Queue{}
		for _, q := range jobs.Queues {
			declared[q.Name] = q
		}
		queues := make([]map[string]any, 0, len(depths))
		for _, d := range depths {
			item := map[string]any{"name": d.Queue, "depth": d, "cron": d.Queue == queue.CronQueue}
			if spec, ok := declared[d.Queue]; ok {
				item["concurrency"] = spec.Concurrency
				item["max_attempts"] = spec.Retries
				item["description"] = spec.Description
				item["external"] = spec.ConsumerElsewhere
			} else if d.Queue == queue.CronQueue {
				item["concurrency"] = queue.CronConcurrency
				item["description"] = "Scheduled cron jobs"
			}
			if q := live[d.Queue]; q != nil {
				item["paused_at"] = q.PausedAt
				item["worker_updated_at"] = q.UpdatedAt
				item["worker_active"] = time.Since(q.UpdatedAt) < 2*time.Minute
			}
			queues = append(queues, item)
		}
		return accountNoStorePayload(map[string]any{"queues": queues}), nil
	}
}

func (s *adminService) riverJobsHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		if _, err := s.requireAdmin(ctx, req, false); err != nil {
			return legacyPayload{}, err
		}
		limit := adminBoundedNumber(req.Query.Get("limit"), 50, 1, 200)
		beforeID, _ := strconv.ParseInt(req.Query.Get("before_id"), 10, 64)
		queueName := strings.TrimSpace(req.Query.Get("queue"))
		state := strings.TrimSpace(req.Query.Get("state"))
		if state != "" && !riverJobStates[state] {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid River job state")
		}

		where := []string{"TRUE"}
		args := []any{}
		add := func(clause string, value any) {
			args = append(args, value)
			where = append(where, fmt.Sprintf(clause, len(args)))
		}
		if queueName != "" {
			add("queue = $%d", queueName)
		}
		if state != "" {
			add("state = $%d::river_job_state", state)
		}
		if beforeID > 0 {
			add("id < $%d", beforeID)
		}
		args = append(args, limit+1)
		rows, err := s.opts.DB.Query(ctx, `SELECT id, state::text, attempt, max_attempts,
			attempted_at, attempted_by, created_at, finalized_at, scheduled_at,
			args, to_json(errors), kind, metadata, priority, queue, tags
			FROM river_job WHERE `+strings.Join(where, " AND ")+`
			ORDER BY id DESC LIMIT $`+strconv.Itoa(len(args)), args...)
		if err != nil {
			return legacyPayload{}, err
		}
		defer rows.Close()
		items := make([]map[string]any, 0, limit)
		var next int64
		for rows.Next() {
			var id int64
			var state string
			var attempt, maxAttempts, priority int
			var attemptedAt, finalizedAt *time.Time
			var createdAt, scheduledAt time.Time
			var attemptedBy, tags []string
			var rawArgs, rawErrors, metadata []byte
			var kind, queueName string
			if err := rows.Scan(&id, &state, &attempt, &maxAttempts, &attemptedAt, &attemptedBy,
				&createdAt, &finalizedAt, &scheduledAt, &rawArgs, &rawErrors, &kind, &metadata,
				&priority, &queueName, &tags); err != nil {
				return legacyPayload{}, err
			}
			if len(items) == limit {
				next = id
				break
			}
			items = append(items, riverJobMap(id, state, attempt, maxAttempts, priority, attemptedAt,
				finalizedAt, createdAt, scheduledAt, attemptedBy, tags, rawArgs, rawErrors, metadata, kind, queueName))
		}
		if err := rows.Err(); err != nil {
			return legacyPayload{}, err
		}
		return accountNoStorePayload(map[string]any{"jobs": items, "next_before_id": next}), nil
	}
}

func riverJobMap(id int64, state string, attempt, maxAttempts, priority int,
	attemptedAt, finalizedAt *time.Time, createdAt, scheduledAt time.Time,
	attemptedBy, tags []string, rawArgs, rawErrors, metadata []byte, kind, queueName string) map[string]any {
	decode := func(raw []byte, fallback any) any {
		var value any
		if len(raw) == 0 || json.Unmarshal(raw, &value) != nil {
			return fallback
		}
		return value
	}
	meta := decode(metadata, map[string]any{})
	var output any
	if m, ok := meta.(map[string]any); ok {
		output = m["output"]
	}
	return map[string]any{"id": id, "state": state, "attempt": attempt, "max_attempts": maxAttempts,
		"attempted_at": attemptedAt, "attempted_by": attemptedBy, "created_at": createdAt,
		"finalized_at": finalizedAt, "scheduled_at": scheduledAt, "args": decode(rawArgs, map[string]any{}),
		"errors": decode(rawErrors, []any{}), "kind": kind, "metadata": meta, "output": output,
		"priority": priority, "queue": queueName, "tags": tags}
}

func (s *adminService) riverJobHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		if _, err := s.requireAdmin(ctx, req, false); err != nil {
			return legacyPayload{}, err
		}
		id, err := strconv.ParseInt(req.Param("id"), 10, 64)
		if err != nil || id <= 0 {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid River job ID")
		}
		client, err := s.riverClient()
		if err != nil {
			return legacyPayload{}, err
		}
		job, err := client.JobGet(ctx, id)
		if err != nil {
			return legacyPayload{}, riverAdminError(err)
		}
		return accountNoStorePayload(map[string]any{"job": riverRowMap(job)}), nil
	}
}

func (s *adminService) riverJobActionHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		if _, err := s.requireAdmin(ctx, req, true); err != nil {
			return legacyPayload{}, err
		}
		id, err := strconv.ParseInt(req.Param("id"), 10, 64)
		if err != nil || id <= 0 {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid River job ID")
		}
		body, err := decodeJSONBody[adminRiverJobActionBody](req, adminRiverBodyLimit)
		if err != nil {
			return legacyPayload{}, err
		}
		client, err := s.riverClient()
		if err != nil {
			return legacyPayload{}, err
		}
		var job *rivertype.JobRow
		switch body.Action {
		case "cancel":
			job, err = client.JobCancel(ctx, id)
		case "retry":
			job, err = client.JobRetry(ctx, id)
		case "delete":
			job, err = client.JobDelete(ctx, id)
		default:
			return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid River job action")
		}
		if err != nil {
			return legacyPayload{}, riverAdminError(err)
		}
		return accountNoStorePayload(map[string]any{"job": riverRowMap(job), "action": body.Action}), nil
	}
}

func (s *adminService) riverQueueActionHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		if _, err := s.requireAdmin(ctx, req, true); err != nil {
			return legacyPayload{}, err
		}
		name := strings.TrimSpace(req.Param("name"))
		if name == "" {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid River queue")
		}
		body, err := decodeJSONBody[adminRiverQueueActionBody](req, adminRiverBodyLimit)
		if err != nil {
			return legacyPayload{}, err
		}
		client, err := s.riverClient()
		if err != nil {
			return legacyPayload{}, err
		}
		switch body.Action {
		case "pause":
			err = client.QueuePause(ctx, name, nil)
		case "resume":
			err = client.QueueResume(ctx, name, nil)
		default:
			return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid River queue action")
		}
		if err != nil {
			return legacyPayload{}, err
		}
		return accountNoStorePayload(map[string]any{"queue": name, "action": body.Action}), nil
	}
}

func (s *adminService) riverClearHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		if _, err := s.requireAdmin(ctx, req, true); err != nil {
			return legacyPayload{}, err
		}
		name := strings.TrimSpace(req.Param("name"))
		if name == "" || name == "*" {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Clear requires one explicit queue")
		}
		body, err := decodeJSONBody[adminRiverClearBody](req, adminRiverBodyLimit)
		if err != nil {
			return legacyPayload{}, err
		}
		if body.Limit == 0 {
			body.Limit = 1000
		}
		if body.Limit < 1 || body.Limit > 10000 {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Clear limit must be between 1 and 10000")
		}
		states := make([]rivertype.JobState, 0, len(body.States))
		for _, state := range body.States {
			if !riverJobStates[state] || state == "running" {
				return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid or unsafe clear state")
			}
			states = append(states, rivertype.JobState(state))
		}
		if len(states) == 0 {
			return legacyPayload{}, apiError(http.StatusBadRequest, "At least one state is required")
		}
		client, err := s.riverClient()
		if err != nil {
			return legacyPayload{}, err
		}
		result, err := client.JobDeleteMany(ctx, river.NewJobDeleteManyParams().First(body.Limit).Queues(name).States(states...))
		if err != nil {
			return legacyPayload{}, riverAdminError(err)
		}
		return accountNoStorePayload(map[string]any{"queue": name, "deleted": len(result.Jobs)}), nil
	}
}

func (s *adminService) riverClient() (*queue.Client, error) {
	if s.opts.PrimaryPool == nil {
		return nil, apiError(http.StatusServiceUnavailable, "River database unavailable")
	}
	return queue.New(queue.Options{Pool: s.opts.PrimaryPool})
}

func riverRowMap(job *rivertype.JobRow) map[string]any {
	rawErrors, _ := json.Marshal(job.Errors)
	return riverJobMap(job.ID, string(job.State), job.Attempt, job.MaxAttempts, job.Priority,
		job.AttemptedAt, job.FinalizedAt, job.CreatedAt, job.ScheduledAt, job.AttemptedBy,
		job.Tags, job.EncodedArgs, rawErrors, job.Metadata, job.Kind, job.Queue)
}

func riverAdminError(err error) error {
	switch {
	case errors.Is(err, rivertype.ErrNotFound):
		return apiError(http.StatusNotFound, "River job not found")
	case errors.Is(err, rivertype.ErrJobRunning):
		return apiError(http.StatusConflict, "Running River jobs cannot be deleted")
	default:
		return err
	}
}
