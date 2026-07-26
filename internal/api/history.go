package api

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

var historyDatePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func registerHistoryRoutes(a huma.API, opts Options) {
	registerLegacy(a, huma.Operation{
		OperationID: "history",
		Method:      http.MethodGet,
		Path:        "/history",
		Summary:     "Daily killmail counts",
		Tags:        []string{"history"},
	}, func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		query := `
			SELECT to_char(date, 'YYYY-MM-DD') AS date, count
			FROM kills_daily_count
			WHERE type = 'latest'`
		args := []any{}
		if rawYear := req.Query.Get("year"); rawYear != "" {
			year, err := strconv.ParseFloat(rawYear, 64)
			if err == nil && year != 0 {
				query += ` AND date >= $1::date AND date < $2::date`
				args = append(args,
					fmt.Sprintf("%v-01-01", year),
					fmt.Sprintf("%v-01-01", year+1),
				)
			}
		}
		query += ` ORDER BY date DESC`

		rows, err := queryMaps(ctx, opts.DB, query, args...)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{"data": rows}), nil
	})

	registerLegacy(a, huma.Operation{
		OperationID: "history-latest",
		Method:      http.MethodGet,
		Path:        "/history/latest",
		Summary:     "Latest 10,000 killmail IDs and hashes",
		Tags:        []string{"history"},
	}, func(ctx context.Context, _ *legacyRequest) (legacyPayload, error) {
		rows, err := queryMaps(ctx, opts.DB, `
			SELECT fq.killmail_id, k.killmail_hash
			FROM feed_queue fq
			INNER JOIN killmails k ON k.killmail_id = fq.killmail_id
			ORDER BY fq.seq DESC
			LIMIT 10000`)
		if err != nil {
			return legacyPayload{}, err
		}
		data := make(map[string]any, len(rows))
		for _, row := range rows {
			data[fmt.Sprint(row["killmail_id"])] = row["killmail_hash"]
		}
		return jsonPayload(map[string]any{"data": data}), nil
	})

	registerLegacy(a, huma.Operation{
		OperationID: "history-date",
		Method:      http.MethodGet,
		Path:        "/history/{date}",
		Summary:     "Killmail IDs and hashes for a UTC date",
		Tags:        []string{"history"},
		Parameters: []*huma.Param{{
			Name:     "date",
			In:       "path",
			Required: true,
			Schema:   &huma.Schema{Type: huma.TypeString},
		}},
	}, func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		date := req.Param("date")
		if !historyDatePattern.MatchString(date) {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid date format, use YYYY-MM-DD")
		}
		start, err := time.Parse("2006-01-02", date)
		if err != nil {
			// The TypeScript route only validates the textual shape. Postgres
			// rejects impossible calendar dates, which its handler maps to 500.
			return legacyPayload{}, err
		}
		rows, err := queryMaps(ctx, opts.DB, `
			SELECT killmail_id, killmail_hash
			FROM killmails
			WHERE killmail_time >= $1 AND killmail_time <= $2
			ORDER BY killmail_id`,
			start.UTC(),
			start.UTC().Add(24*time.Hour-time.Millisecond),
		)
		if err != nil {
			return legacyPayload{}, err
		}
		data := make(map[string]any, len(rows))
		for _, row := range rows {
			data[fmt.Sprint(row["killmail_id"])] = row["killmail_hash"]
		}
		return jsonPayload(map[string]any{"data": data}), nil
	})
}
