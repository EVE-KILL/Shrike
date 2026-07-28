package api

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

const (
	searchMaxWindowDays       = 7
	searchMaxIDsPerField      = 15
	searchMaxFilterCategories = 3
)

type killmailSearchInput struct {
	From             time.Time
	To               time.Time
	Limit            int
	After            *int64
	SystemIDs        []int32
	ConstellationIDs []int32
	RegionIDs        []int32
	CharacterIDs     []int32
	CorporationIDs   []int32
	AllianceIDs      []int32
}

func registerKillmailSearchRoute(a huma.API, opts Options) {
	registerLegacy(a, documentJSONBody[killmailSearchDocument](a, huma.Operation{
		OperationID: "killmail-search",
		Method:      http.MethodPost,
		Path:        "/killmails/search",
		Summary:     "Bulk killmail search",
		Tags:        []string{"killmails"},
	}), func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		input, err := parseKillmailSearch(req)
		if err != nil {
			return legacyPayload{}, err
		}
		ids, err := queryKillmailSearch(ctx, opts.DB, input)
		if err != nil {
			return legacyPayload{}, err
		}
		hasMore := len(ids) > input.Limit
		if hasMore {
			ids = ids[:input.Limit]
		}
		if len(ids) == 0 {
			return jsonPayload(map[string]any{
				"data": []any{},
				"pagination": map[string]any{
					"hasMore": false, "cursor": nil,
				},
			}), nil
		}
		intIDs := make([]int32, 0, len(ids))
		for _, id := range ids {
			intIDs = append(intIDs, int32(id))
		}
		data, err := loadKillmailsESI(ctx, opts.DB, intIDs)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{
			"data": data,
			"pagination": map[string]any{
				"hasMore": hasMore, "cursor": ids[len(ids)-1],
			},
		}), nil
	})
}

// killmailSearchBody is the runtime compatibility decode target. Fields stay
// raw because the parser coerces the way the TypeScript API did. Its concrete
// OpenAPI wire shape is killmailSearchDocument.
type killmailSearchBody struct {
	From  json.RawMessage `json:"from" doc:"Window start. A date (YYYY-MM-DD) or an ISO 8601 timestamp, as a string."`
	To    json.RawMessage `json:"to" doc:"Window end. A date (YYYY-MM-DD) or an ISO 8601 timestamp, as a string."`
	Limit json.RawMessage `json:"limit,omitempty" doc:"Maximum killmails to return, capped at 100."`
	After json.RawMessage `json:"after,omitempty" doc:"Cursor: return killmails after this identifier."`

	SystemIDs        json.RawMessage `json:"system_ids,omitempty" doc:"Restrict to these solar systems."`
	ConstellationIDs json.RawMessage `json:"constellation_ids,omitempty" doc:"Restrict to these constellations."`
	RegionIDs        json.RawMessage `json:"region_ids,omitempty" doc:"Restrict to these regions."`
	CharacterIDs     json.RawMessage `json:"character_ids,omitempty" doc:"Restrict to killmails involving these characters."`
	CorporationIDs   json.RawMessage `json:"corporation_ids,omitempty" doc:"Restrict to killmails involving these corporations."`
	AllianceIDs      json.RawMessage `json:"alliance_ids,omitempty" doc:"Restrict to killmails involving these alliances."`
}

// idField resolves a filter name to its raw value so the loop below can stay
// table-driven.
func (b *killmailSearchBody) idField(name string) json.RawMessage {
	switch name {
	case "system_ids":
		return b.SystemIDs
	case "constellation_ids":
		return b.ConstellationIDs
	case "region_ids":
		return b.RegionIDs
	case "character_ids":
		return b.CharacterIDs
	case "corporation_ids":
		return b.CorporationIDs
	case "alliance_ids":
		return b.AllianceIDs
	}
	return nil
}

func parseKillmailSearch(req *legacyRequest) (killmailSearchInput, error) {
	body, decodeErr := decodeJSONBody[killmailSearchBody](req, defaultBodyLimit)
	if err := decodeErr; err != nil {
		return killmailSearchInput{}, apiError(
			http.StatusBadRequest, "Invalid JSON body",
		)
	}
	if body == nil {
		return killmailSearchInput{}, apiError(http.StatusBadRequest, "Body required")
	}
	fromRaw, fromOK := rawJSONField(body.From)
	toRaw, toOK := rawJSONField(body.To)
	if !fromOK || !toOK || !jsTruthy(fromRaw) || !jsTruthy(toRaw) {
		return killmailSearchInput{}, apiError(
			http.StatusBadRequest, "from and to are required",
		)
	}
	from, err := searchTimestamp(fromRaw, "from")
	if err != nil {
		return killmailSearchInput{}, err
	}
	to, err := searchTimestamp(toRaw, "to")
	if err != nil {
		return killmailSearchInput{}, err
	}
	if !from.Before(to) {
		return killmailSearchInput{}, apiError(
			http.StatusBadRequest, "from must be before to",
		)
	}
	if to.Sub(from) > searchMaxWindowDays*24*time.Hour {
		return killmailSearchInput{}, apiError(
			http.StatusBadRequest, "Time window cannot exceed 7 days",
		)
	}
	input := killmailSearchInput{From: from, To: to, Limit: 100}
	if raw, exists := rawJSONField(body.Limit); exists {
		if n, ok := jsNumber(raw); ok && n > 0 {
			input.Limit = min(100, int(math.Floor(n)))
		}
	}
	if raw, exists := rawJSONField(body.After); exists && raw != nil {
		id, err := searchPositiveID(raw, "after")
		if err != nil {
			return killmailSearchInput{}, err
		}
		input.After = &id
	}
	for _, field := range []struct {
		name   string
		target *[]int32
	}{
		{"system_ids", &input.SystemIDs},
		{"constellation_ids", &input.ConstellationIDs},
		{"region_ids", &input.RegionIDs},
		{"character_ids", &input.CharacterIDs},
		{"corporation_ids", &input.CorporationIDs},
		{"alliance_ids", &input.AllianceIDs},
	} {
		ids, err := searchIDArray(rawJSONValue(body.idField(field.name)), field.name)
		if err != nil {
			return killmailSearchInput{}, err
		}
		*field.target = ids
	}
	active := 0
	for _, ids := range [][]int32{
		input.SystemIDs, input.ConstellationIDs, input.RegionIDs,
		input.CharacterIDs, input.CorporationIDs, input.AllianceIDs,
	} {
		if len(ids) > 0 {
			active++
		}
	}
	if active > searchMaxFilterCategories {
		return killmailSearchInput{}, apiError(
			http.StatusBadRequest,
			"Cannot combine more than 3 filter categories per request",
		)
	}
	return input, nil
}

func searchTimestamp(value any, field string) (time.Time, error) {
	raw, ok := value.(string)
	if !ok {
		return time.Time{}, apiError(
			http.StatusBadRequest, field+" must be a string",
		)
	}
	if len(raw) == 10 {
		if parsed, err := time.Parse("2006-01-02", raw); err == nil {
			return parsed, nil
		}
	}
	for _, layout := range []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	} {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, apiError(
		http.StatusBadRequest,
		field+" is not a valid ISO8601 timestamp",
	)
}

func searchPositiveID(value any, field string) (int64, error) {
	number, ok := jsNumber(value)
	if !ok || math.Trunc(number) != number || number <= 0 {
		return 0, apiError(
			http.StatusBadRequest, field+" must be a positive integer",
		)
	}
	return int64(number), nil
}

func searchIDArray(value any, field string) ([]int32, error) {
	if value == nil {
		return []int32{}, nil
	}
	values, ok := value.([]any)
	if !ok {
		return nil, apiError(http.StatusBadRequest, field+" must be an array")
	}
	if len(values) > searchMaxIDsPerField {
		return nil, apiError(
			http.StatusBadRequest,
			fmt.Sprintf("%s exceeds %d entries", field, searchMaxIDsPerField),
		)
	}
	seen := map[int32]struct{}{}
	result := make([]int32, 0, len(values))
	for _, value := range values {
		number, ok := jsNumber(value)
		if !ok || math.Trunc(number) != number || number <= 0 {
			return nil, apiError(
				http.StatusBadRequest,
				fmt.Sprintf("%s contains invalid id: %v", field, value),
			)
		}
		id := int32(number)
		if _, exists := seen[id]; !exists {
			seen[id] = struct{}{}
			result = append(result, id)
		}
	}
	return result, nil
}

func jsNumber(value any) (float64, bool) {
	switch v := value.(type) {
	case json.Number:
		number, err := v.Float64()
		return number, err == nil && !math.IsInf(number, 0) && !math.IsNaN(number)
	case string:
		number, err := numberFromString(v)
		return number, err == nil
	case bool:
		if v {
			return 1, true
		}
		return 0, true
	case nil:
		return 0, true
	default:
		return 0, false
	}
}

func numberFromString(raw string) (float64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	return strconv.ParseFloat(raw, 64)
}

func jsTruthy(value any) bool {
	switch v := value.(type) {
	case nil:
		return false
	case string:
		return v != ""
	case bool:
		return v
	case json.Number:
		n, err := v.Float64()
		return err == nil && n != 0 && !math.IsNaN(n)
	default:
		return true
	}
}

func queryKillmailSearch(
	ctx context.Context,
	db Database,
	input killmailSearchInput,
) ([]int64, error) {
	hasEntity := len(input.CharacterIDs)+len(input.CorporationIDs)+
		len(input.AllianceIDs) > 0
	if !hasEntity {
		return queryKillmailSearchLocations(ctx, db, input)
	}
	var cursorTime *time.Time
	if input.After != nil {
		row, err := queryMap(ctx, db,
			`SELECT killmail_time FROM killmails
			 WHERE killmail_id = $1 LIMIT 1`, *input.After)
		if err != nil {
			return nil, err
		}
		if row != nil {
			if value, ok := row["killmail_time"].(time.Time); ok {
				cursorTime = &value
			}
		}
	}
	return queryKillmailSearchEntities(ctx, db, input, cursorTime)
}

func queryKillmailSearchLocations(
	ctx context.Context,
	db Database,
	input killmailSearchInput,
) ([]int64, error) {
	args := []any{input.From, input.To}
	where := []string{"killmail_time >= $1", "killmail_time <= $2"}
	appendSearchLocationFilters(&where, &args, input, "")
	if input.After != nil {
		args = append(args, *input.After)
		where = append(where, fmt.Sprintf("killmail_id < $%d", len(args)))
	}
	args = append(args, input.Limit+1)
	rows, err := queryMaps(ctx, db, `
		SELECT killmail_id FROM killmails
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY killmail_id DESC
		LIMIT $`+fmt.Sprintf("%d", len(args)), args...)
	return searchKillmailIDs(rows, err)
}

func queryKillmailSearchEntities(
	ctx context.Context,
	db Database,
	input killmailSearchInput,
	cursorTime *time.Time,
) ([]int64, error) {
	victimArgs := []any{input.From, input.To}
	victimWhere := []string{"killmail_time >= $1", "killmail_time <= $2"}
	victimEntity := searchEntityConditions(
		&victimArgs, "", "victim_",
		input.CharacterIDs, input.CorporationIDs, input.AllianceIDs,
	)
	victimWhere = append(victimWhere, "("+strings.Join(victimEntity, " OR ")+")")
	appendSearchLocationFilters(&victimWhere, &victimArgs, input, "")
	if input.After != nil {
		victimArgs = append(victimArgs, *input.After)
		victimWhere = append(
			victimWhere, fmt.Sprintf("killmail_id < $%d", len(victimArgs)),
		)
	}
	victimArgs = append(victimArgs, input.Limit+1)
	victimSQL := `SELECT killmail_id FROM killmails
		WHERE ` + strings.Join(victimWhere, " AND ") + `
		ORDER BY killmail_id DESC LIMIT $` + fmt.Sprintf("%d", len(victimArgs))

	attackerArgs := []any{input.From, input.To}
	attackerWhere := []string{"a.killmail_time >= $1", "a.killmail_time <= $2"}
	attackerEntity := searchEntityConditions(
		&attackerArgs, "a.", "",
		input.CharacterIDs, input.CorporationIDs, input.AllianceIDs,
	)
	attackerWhere = append(attackerWhere, "("+strings.Join(attackerEntity, " OR ")+")")
	needsJoin := len(input.SystemIDs)+len(input.ConstellationIDs)+len(input.RegionIDs) > 0
	if needsJoin {
		appendSearchLocationFilters(&attackerWhere, &attackerArgs, input, "k.")
	}
	if input.After != nil && cursorTime != nil {
		attackerArgs = append(attackerArgs, *cursorTime, *input.After)
		attackerWhere = append(attackerWhere, fmt.Sprintf(
			"(a.killmail_time, a.killmail_id) < ($%d, $%d)",
			len(attackerArgs)-1, len(attackerArgs),
		))
	} else if input.After != nil {
		attackerArgs = append(attackerArgs, *input.After)
		attackerWhere = append(
			attackerWhere,
			fmt.Sprintf("a.killmail_id < $%d", len(attackerArgs)),
		)
	}
	attackerArgs = append(attackerArgs, input.Limit+1)
	join := ""
	if needsJoin {
		join = " JOIN killmails k ON k.killmail_id = a.killmail_id"
	}
	attackerSQL := `SELECT DISTINCT a.killmail_id, a.killmail_time
		FROM killmail_attackers a` + join + `
		WHERE ` + strings.Join(attackerWhere, " AND ") + `
		ORDER BY a.killmail_time DESC, a.killmail_id DESC
		LIMIT $` + fmt.Sprintf("%d", len(attackerArgs))

	// The two independently-parameterized legs cannot be interpolated into a
	// single pgx statement without renumbering. Keep each leg bounded exactly
	// like the legacy query, then perform its UNION/distinct/order in memory.
	victimRows, err := queryMaps(ctx, db, victimSQL, victimArgs...)
	if err != nil {
		return nil, err
	}
	attackerRows, err := queryMaps(ctx, db, attackerSQL, attackerArgs...)
	if err != nil {
		return nil, err
	}
	seen := map[int64]struct{}{}
	ids := make([]int64, 0, len(victimRows)+len(attackerRows))
	for _, row := range append(victimRows, attackerRows...) {
		id, _ := int64Value(row["killmail_id"])
		if _, exists := seen[id]; !exists {
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	sortInt64Descending(ids)
	if len(ids) > input.Limit+1 {
		ids = ids[:input.Limit+1]
	}
	return ids, nil
}

func searchEntityConditions(
	args *[]any,
	prefix string,
	columnPrefix string,
	characters, corporations, alliances []int32,
) []string {
	result := []string{}
	for _, field := range []struct {
		column string
		ids    []int32
	}{
		{"character_id", characters},
		{"corporation_id", corporations},
		{"alliance_id", alliances},
	} {
		if len(field.ids) > 0 {
			*args = append(*args, field.ids)
			result = append(result, fmt.Sprintf(
				"%s%s = ANY($%d::int[])",
				prefix, columnPrefix+field.column, len(*args),
			))
		}
	}
	return result
}

func appendSearchLocationFilters(
	where *[]string,
	args *[]any,
	input killmailSearchInput,
	prefix string,
) {
	for _, field := range []struct {
		column string
		ids    []int32
	}{
		{"solar_system_id", input.SystemIDs},
		{"constellation_id", input.ConstellationIDs},
		{"region_id", input.RegionIDs},
	} {
		if len(field.ids) > 0 {
			*args = append(*args, field.ids)
			*where = append(*where, fmt.Sprintf(
				"%s%s = ANY($%d::int[])",
				prefix, field.column, len(*args),
			))
		}
	}
}

func searchKillmailIDs(rows []map[string]any, err error) ([]int64, error) {
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		id, _ := int64Value(row["killmail_id"])
		ids = append(ids, id)
	}
	return ids, nil
}

func sortInt64Descending(values []int64) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] > values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}
