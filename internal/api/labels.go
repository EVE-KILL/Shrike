package api

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/eve-kill/shrike/internal/killtype"
)

const labelsCacheTTL = 15 * time.Minute

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
		rows, err := queryMaps(ctx, opts.DB, `
			SELECT type, COALESCE(SUM(count), 0)::bigint AS count
			FROM kills_daily_count
			WHERE type = ANY($1::text[])
			GROUP BY type`, publicLabelIDs())
		if err != nil {
			return legacyPayload{}, err
		}
		counts := make(map[string]int64, len(rows))
		for _, row := range rows {
			counts[stringOrEmpty(row["type"])] = int64OrZero(row["count"])
		}
		labels := make([]map[string]any, 0, len(killtype.Labels))
		for _, label := range killtype.Labels {
			labels = append(labels, map[string]any{
				"id": label.ID, "name": label.Name,
				"description": label.Description, "category": label.Category,
				"count": counts[label.ID], "view_url": "/kills/" + label.ID,
				"search_filters": map[string]any{"label": label.ID},
			})
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
