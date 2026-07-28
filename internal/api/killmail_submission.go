package api

import (
	"context"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

const killmailSubmissionBodyLimit = 1 << 20

var esiKillmailURLPattern = regexp.MustCompile(
	`(?i)https?://esi\.evetech\.net/(?:latest/|v[0-9]+/)?killmails/([0-9]+)/([a-f0-9]{20,64})/?`,
)

type submittedKillmail struct {
	ID   int64
	Hash string
}

// killmailSubmissionBody is both the decode target and the documented schema.
// Either form is accepted; text wins when both arrive.
type killmailSubmissionBody struct {
	Text  *string  `json:"text,omitempty" doc:"Free text containing ESI killmail links, one per line. Takes precedence over links."`
	Links []string `json:"links,omitempty" nullable:"false" doc:"ESI killmail links. Joined with newlines and parsed the same way as text."`
}

type killmailSubmissionDispatcher func(
	context.Context,
	[]submittedKillmail,
) error

func registerKillmailSubmissionRoute(a huma.API, opts Options) {
	registerLegacyJSON(a, huma.Operation{
		OperationID: "killmail-submit",
		Method:      http.MethodPost,
		Path:        "/killmail/post",
		Summary:     "Submit ESI killmail links",
		Description: "Queues valid ESI killmail links that are not already stored.",
		Tags:        []string{"killmails"},
	}, killmailSubmissionBodyLimit,
		killmailSubmissionHandler(opts, newKillmailSubmissionDispatcher(opts)))
}

func newKillmailSubmissionDispatcher(opts Options) killmailSubmissionDispatcher {
	pool, ok := opts.DB.(*pgxpool.Pool)
	if !ok || pool == nil {
		// Focused tests and development configurations may have a read-only
		// database facade. This matches the site's old no-dispatch development
		// behavior: valid submissions are still acknowledged.
		return nil
	}
	client, err := queue.New(queue.Options{Pool: pool})
	if err != nil {
		return func(context.Context, []submittedKillmail) error { return err }
	}
	return func(ctx context.Context, killmails []submittedKillmail) error {
		batch := make([]river.JobArgs, 0, len(killmails))
		for _, killmail := range killmails {
			batch = append(batch, queue.KillmailArgs{
				KillmailID: killmail.ID, KillmailHash: killmail.Hash,
			})
		}
		_, err := queue.DispatchMany(ctx, client, batch, queue.Live)
		return err
	}
}

func killmailSubmissionHandler(
	opts Options,
	dispatch killmailSubmissionDispatcher,
) bodyHandler[killmailSubmissionBody] {
	return func(
		ctx context.Context, req *legacyRequest, body *killmailSubmissionBody,
	) (legacyPayload, error) {
		text := ""
		if body.Text != nil {
			text = *body.Text
		} else if body.Links != nil {
			text = strings.Join(body.Links, "\n")
		}
		text = strings.TrimSpace(text)
		if text == "" {
			return legacyPayload{}, apiError(
				http.StatusBadRequest, "No text provided",
			)
		}

		parsed := parseSubmittedKillmails(text)
		if len(parsed) == 0 {
			return jsonPayload(map[string]any{
				"accepted": 0, "rejected": 0, "existing": 0, "total": 0,
				"killmails":   []int64{},
				"existingIds": []int64{},
				"message":     "No valid ESI killmail links found",
			}), nil
		}

		ids := make([]int64, 0, len(parsed))
		for _, killmail := range parsed {
			ids = append(ids, killmail.ID)
		}
		rows, err := queryMaps(ctx, opts.DB, `
			SELECT killmail_id
			FROM killmails
			WHERE killmail_id = ANY($1::bigint[])`,
			ids,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		existingSet := make(map[int64]struct{}, len(rows))
		for _, row := range rows {
			if id, ok := int64Value(row["killmail_id"]); ok {
				existingSet[id] = struct{}{}
			}
		}

		newKillmails := make([]submittedKillmail, 0, len(parsed)-len(existingSet))
		existingIDs := make([]int64, 0, len(existingSet))
		for _, killmail := range parsed {
			if _, exists := existingSet[killmail.ID]; exists {
				existingIDs = append(existingIDs, killmail.ID)
				continue
			}
			newKillmails = append(newKillmails, killmail)
		}

		accepted := len(newKillmails)
		if dispatch != nil && len(newKillmails) > 0 {
			if err := dispatch(ctx, newKillmails); err != nil {
				// Submission is a best-effort convenience API. Match the
				// existing contract by reporting dispatch failures per batch
				// instead of turning otherwise valid links into an HTTP 500.
				accepted = 0
			}
		}
		return jsonPayload(map[string]any{
			"accepted":    accepted,
			"rejected":    len(newKillmails) - accepted,
			"existing":    len(existingSet),
			"total":       len(parsed),
			"killmails":   ids,
			"existingIds": existingIDs,
		}), nil
	}
}

func parseSubmittedKillmails(text string) []submittedKillmail {
	matches := esiKillmailURLPattern.FindAllStringSubmatch(text, -1)
	result := make([]submittedKillmail, 0, len(matches))
	seen := make(map[int64]struct{}, len(matches))
	for _, match := range matches {
		id, err := strconv.ParseInt(match[1], 10, 64)
		if err != nil {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, submittedKillmail{
			ID: id, Hash: strings.ToLower(match[2]),
		})
	}
	return result
}
