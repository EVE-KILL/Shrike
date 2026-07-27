package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

// resolveBody is both the decode target and the documented request schema.
type resolveBody struct {
	Names []string `json:"names" minItems:"1" maxItems:"100" doc:"Exact entity names to resolve. Matching is case-sensitive and exact; use /search for fuzzy lookup."`
	Type  string   `json:"type,omitempty" enum:"character,corporation,alliance" default:"character" doc:"Which entity table to resolve against."`
}

func registerResolveRoute(a huma.API, opts Options) {
	registerLegacyJSON(a, huma.Operation{
		OperationID: "resolve",
		Method:      http.MethodPost,
		Path:        "/resolve",
		Summary:     "Resolve exact entity names to IDs",
		Tags:        []string{"search"},
	}, defaultBodyLimit, func(
		ctx context.Context, req *legacyRequest, body *resolveBody,
	) (legacyPayload, error) {
		if len(body.Names) == 0 {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Missing names array")
		}
		if len(body.Names) > 100 {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Maximum 100 names per request")
		}
		entityType := body.Type
		if entityType == "" {
			entityType = "character"
		}

		var idColumn, table string
		switch entityType {
		case "character":
			idColumn, table = "character_id", "characters"
		case "corporation":
			idColumn, table = "corporation_id", "corporations"
		case "alliance":
			idColumn, table = "alliance_id", "alliances"
		default:
			return legacyPayload{}, apiError(http.StatusBadRequest, "type must be character, corporation, or alliance")
		}

		rows, err := queryMaps(ctx, opts.DB,
			`SELECT `+idColumn+` AS id, name
			 FROM `+table+`
			 WHERE name = ANY($1::text[])
			   AND deleted IS NOT TRUE`,
			body.Names,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		found := make(map[string]map[string]any, len(rows))
		for _, row := range rows {
			name, _ := row["name"].(string)
			found[strings.ToLower(name)] = row
		}

		results := make([]map[string]any, 0, len(body.Names))
		resolved := 0
		for _, name := range body.Names {
			match := found[strings.ToLower(name)]
			if match == nil {
				results = append(results, map[string]any{
					"name": name, "id": nil, "resolved_name": nil,
				})
				continue
			}
			resolved++
			results = append(results, map[string]any{
				"name":          name,
				"id":            match["id"],
				"resolved_name": match["name"],
			})
		}
		return jsonPayload(map[string]any{
			"type":       entityType,
			"results":    results,
			"resolved":   resolved,
			"unresolved": len(results) - resolved,
		}), nil
	})
}
