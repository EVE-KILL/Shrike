package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/eve-kill/shrike/internal/killtype"
)

const labelsCacheTTL = 15 * time.Minute

type LabelEntry struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	Category      string         `json:"category"`
	Count         int64          `json:"count"`
	ViewURL       string         `json:"view_url"`
	SearchFilters map[string]any `json:"search_filters"`
}

// Labels serves the API and the Go categories page from one catalogue and
// one count query. The response cache is shared with other HTML read models.
func (r *HomepageReader) Labels(ctx context.Context) ([]LabelEntry, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("homepage database is not configured")
	}
	return cachedHomepage(ctx, r, "labels", labelsCacheTTL, func(ctx context.Context) ([]LabelEntry, error) {
		rows, err := queryMaps(ctx, r.db, `
			SELECT type, COALESCE(SUM(count), 0)::bigint AS count
			FROM kills_daily_count
			WHERE type = ANY($1::text[])
			GROUP BY type`, publicLabelIDs())
		if err != nil {
			return nil, err
		}
		counts := make(map[string]int64, len(rows))
		for _, row := range rows {
			counts[stringOrEmpty(row["type"])] = int64OrZero(row["count"])
		}
		labels := make([]LabelEntry, 0, len(killtype.Labels))
		for _, label := range killtype.Labels {
			labels = append(labels, LabelEntry{
				ID: label.ID, Name: label.Name, Description: label.Description,
				Category: label.Category, Count: counts[label.ID],
				ViewURL:       "/kills/" + label.ID,
				SearchFilters: map[string]any{"label": label.ID},
			})
		}
		return labels, nil
	})
}

func registerLabelRoutes(a huma.API, opts Options) {
	registerLegacy(a, huma.Operation{
		OperationID: "killmail-labels",
		Method:      http.MethodGet,
		Path:        "/labels",
		Summary:     "Killmail classification catalogue",
		Description: "Lists the authoritative killmail classifications used by EVE-KILL, with all-time counts and advanced-search filters.",
		Tags:        []string{"killboard", "killmails", "statistics"},
		Responses: map[string]*huma.Response{
			"200": {
				Description: "OK",
				Content: map[string]*huma.MediaType{
					"application/json": {Schema: labelsResponseSchema()},
				},
			},
		},
	}, routeJSONCache(
		opts,
		labelsCacheTTL,
		"public, max-age=300, s-maxage=900, stale-while-revalidate=900",
		labelsHandler(opts),
	))
}

func labelsResponseSchema() *huma.Schema {
	label := responseSchema(map[string]*huma.Schema{
		"id": stringSchema(), "name": stringSchema(),
		"description": stringSchema(), "category": stringSchema(),
		"count": intSchema(), "view_url": stringSchema(),
		"search_filters": openJSONObjectSchema("Canonical advanced-search filter payload for this classification."),
	}, "id", "name", "description", "category", "count", "view_url", "search_filters")
	return responseSchema(map[string]*huma.Schema{
		"labels": arraySchema(label),
	}, "labels")
}

func labelsHandler(opts Options) legacyHandler {
	return func(ctx context.Context, _ *legacyRequest) (legacyPayload, error) {
		labels, err := NewHomepageReader(opts.DB).Labels(ctx)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{"labels": labels}), nil
	}
}

func publicLabelIDs() []string {
	ids := make([]string, 0, len(killtype.Labels))
	for _, label := range killtype.Labels {
		ids = append(ids, label.ID)
	}
	return ids
}
