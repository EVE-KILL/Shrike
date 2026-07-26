package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

const (
	scanBodyLimit       = 8 << 20
	localScanMaxNames   = 4096
	localScanMaxNameLen = 64
)

// registerScanRoutes exposes a consolidated scan API while retaining the
// current Nuxt paths during the frontend migration.
func registerScanRoutes(a huma.API, opts Options) {
	auth := newAuthService(opts)
	required := []map[string][]string{{"eveSession": {}}}

	for _, route := range []struct {
		id, method, path, summary string
		security                  []map[string][]string
		handler                   legacyHandler
	}{
		{
			id: "dscan-analyze", method: http.MethodPost,
			path: "/scans/dscan/analyze", summary: "Analyze a directional scan",
			handler: analyzeDirectionalScanHandler(opts),
		},
		{
			id: "dscan-analyze-legacy", method: http.MethodPost,
			path: "/tools/dscan", summary: "Analyze a directional scan",
			handler: analyzeDirectionalScanHandler(opts),
		},
		{
			id: "dscan-save", method: http.MethodPost,
			path: "/scans/dscan", summary: "Save a directional scan",
			security: required, handler: saveScanHandler(opts, auth, "dscan"),
		},
		{
			id: "dscan-save-legacy", method: http.MethodPost,
			path: "/tools/dscan/save", summary: "Save a directional scan",
			security: required, handler: saveScanHandler(opts, auth, "dscan"),
		},
		{
			id: "dscan-get", method: http.MethodGet,
			path: "/scans/dscan/{hash}", summary: "Get a saved directional scan",
			handler: getScanHandler(opts, "dscan"),
		},
		{
			id: "dscan-get-legacy", method: http.MethodGet,
			path: "/tools/dscan/{hash}", summary: "Get a saved directional scan",
			handler: getScanHandler(opts, "dscan"),
		},
		{
			id: "localscan-analyze", method: http.MethodPost,
			path: "/scans/local/analyze", summary: "Analyze a local character scan",
			handler: analyzeLocalScanHandler(opts),
		},
		{
			id: "localscan-analyze-legacy", method: http.MethodPost,
			path: "/tools/localscan", summary: "Analyze a local character scan",
			handler: analyzeLocalScanHandler(opts),
		},
		{
			id: "localscan-save", method: http.MethodPost,
			path: "/scans/local", summary: "Save a local character scan",
			security: required, handler: saveScanHandler(opts, auth, "localscan"),
		},
		{
			id: "localscan-save-legacy", method: http.MethodPost,
			path: "/tools/localscan/save", summary: "Save a local character scan",
			security: required, handler: saveScanHandler(opts, auth, "localscan"),
		},
		{
			id: "localscan-get", method: http.MethodGet,
			path: "/scans/local/{hash}", summary: "Get a saved local character scan",
			handler: getScanHandler(opts, "localscan"),
		},
		{
			id: "localscan-get-legacy", method: http.MethodGet,
			path: "/tools/localscan/{hash}", summary: "Get a saved local character scan",
			handler: getScanHandler(opts, "localscan"),
		},
	} {
		handler := route.handler
		if route.method == http.MethodGet {
			handler = routeJSONCache(
				opts, 24*time.Hour, "public, max-age=86400", handler,
			)
		}
		registerLegacy(a, huma.Operation{
			OperationID: route.id,
			Method:      route.method,
			Path:        route.path,
			Summary:     route.summary,
			Tags:        []string{"scans", "tools"},
			Security:    route.security,
		}, handler)
	}
}

func decodeScanBody(req *legacyRequest) (any, error) {
	decoder := json.NewDecoder(io.LimitReader(req.Body, scanBodyLimit+1))
	decoder.UseNumber()
	var body any
	if err := decoder.Decode(&body); err != nil {
		return nil, apiError(http.StatusBadRequest, "Invalid JSON body")
	}
	var extra any
	err := decoder.Decode(&extra)
	if !errors.Is(err, io.EOF) {
		return nil, apiError(http.StatusBadRequest, "Invalid JSON body")
	}
	return body, nil
}

func analyzeDirectionalScanHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		raw, err := decodeScanBody(req)
		if err != nil {
			return legacyPayload{}, err
		}
		body, ok := raw.(map[string]any)
		dscan, isString := body["dscan"].(string)
		if !ok || !isString || dscan == "" {
			return legacyPayload{}, apiError(
				http.StatusBadRequest, "No D-Scan data provided",
			)
		}

		typeCounts := make(map[string]int64)
		typeOrder := make([]string, 0)
		for _, line := range strings.Split(dscan, "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			fields := splitDirectionalScanLine(line)
			if len(fields) < 3 {
				continue
			}
			typeName := strings.TrimSpace(fields[2])
			if typeName == "" {
				continue
			}
			if _, exists := typeCounts[typeName]; !exists {
				typeOrder = append(typeOrder, typeName)
			}
			typeCounts[typeName]++
		}
		if len(typeOrder) == 0 {
			return legacyPayload{}, apiError(
				http.StatusBadRequest, "No valid D-Scan entries found",
			)
		}

		rows, err := queryMaps(ctx, opts.DB, `
			SELECT t.type_id, t.name AS type_name,
			       t.group_id, g.name AS group_name,
			       t.category_id, c.name AS category_name
			FROM inv_types t
			LEFT JOIN inv_groups g ON g.group_id = t.group_id
			LEFT JOIN inv_categories c ON c.category_id = t.category_id
			WHERE t.name = ANY($1::text[])`, typeOrder)
		if err != nil {
			return legacyPayload{}, err
		}
		lookup := make(map[string]map[string]any, len(rows))
		for _, row := range rows {
			name, _ := stringValue(row["type_name"])
			lookup[name] = row
		}

		grouped := make(map[string]any)
		var total int64
		for _, typeName := range typeOrder {
			count := typeCounts[typeName]
			row := lookup[typeName]
			categoryName := "Unknown"
			groupName := "Unknown"
			var categoryID, groupID, typeID any
			if row != nil {
				if value, ok := stringValue(row["category_name"]); ok {
					categoryName = value
				}
				if value, ok := stringValue(row["group_name"]); ok {
					groupName = value
				}
				categoryID, groupID, typeID =
					row["category_id"], row["group_id"], row["type_id"]
			}
			category, _ := grouped[categoryName].(map[string]any)
			if category == nil {
				category = map[string]any{
					"categoryId": categoryID,
					"groups":     map[string]any{},
				}
				grouped[categoryName] = category
			}
			groups := category["groups"].(map[string]any)
			group, _ := groups[groupName].(map[string]any)
			if group == nil {
				group = map[string]any{
					"groupId": groupID,
					"types":   []map[string]any{},
				}
				groups[groupName] = group
			}
			types := group["types"].([]map[string]any)
			group["types"] = append(types, map[string]any{
				"typeName": typeName, "typeId": typeID, "count": count,
			})
			total += count
		}
		for _, categoryValue := range grouped {
			category := categoryValue.(map[string]any)
			for _, groupValue := range category["groups"].(map[string]any) {
				group := groupValue.(map[string]any)
				types := group["types"].([]map[string]any)
				sort.SliceStable(types, func(i, j int) bool {
					return int64OrZero(types[i]["count"]) >
						int64OrZero(types[j]["count"])
				})
			}
		}
		return jsonPayload(map[string]any{
			"grouped": grouped, "totalCount": total,
			"uniqueTypes": len(typeOrder),
		}), nil
	}
}

func splitDirectionalScanLine(line string) []string {
	if strings.Contains(line, "\t") {
		return strings.Split(line, "\t")
	}
	return splitFourSpaces(line)
}

func splitFourSpaces(line string) []string {
	fields := make([]string, 0, 4)
	start := 0
	for index := 0; index < len(line); {
		if line[index] != ' ' && line[index] != '\t' {
			index++
			continue
		}
		end := index
		for end < len(line) && (line[end] == ' ' || line[end] == '\t') {
			end++
		}
		if end-index >= 4 {
			fields = append(fields, line[start:index])
			start = end
		}
		index = end
	}
	fields = append(fields, line[start:])
	return fields
}

func analyzeLocalScanHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		raw, err := decodeScanBody(req)
		if err != nil {
			return legacyPayload{}, err
		}
		values, ok := raw.([]any)
		if !ok || len(values) == 0 {
			return legacyPayload{}, apiError(
				http.StatusBadRequest, "No character names provided",
			)
		}
		if len(values) > localScanMaxNames {
			return legacyPayload{}, apiError(
				http.StatusBadRequest, "Too many names. Maximum is 4096",
			)
		}
		validNames := make([]string, 0, len(values))
		for _, value := range values {
			name, ok := value.(string)
			if !ok {
				continue
			}
			name = strings.TrimSpace(name)
			if name != "" && len([]rune(name)) <= localScanMaxNameLen {
				validNames = append(validNames, name)
			}
		}
		if len(validNames) == 0 {
			return legacyPayload{}, apiError(
				http.StatusBadRequest, "No valid character names after filtering",
			)
		}

		rows, err := queryMaps(ctx, opts.DB, `
			SELECT ch.character_id, ch.name,
			       ch.corporation_id, ch.alliance_id,
			       corp.name AS corporation_name,
			       corp.ticker AS corporation_ticker,
			       ally.name AS alliance_name,
			       ally.ticker AS alliance_ticker,
			       COUNT(DISTINCT ka.killmail_id)::bigint AS kills
			FROM characters ch
			LEFT JOIN corporations corp
			  ON corp.corporation_id = ch.corporation_id
			LEFT JOIN alliances ally
			  ON ally.alliance_id = ch.alliance_id
			LEFT JOIN killmail_attackers ka
			  ON ka.character_id = ch.character_id
			 AND ka.killmail_time >= NOW() - INTERVAL '7 days'
			WHERE ch.name = ANY($1::text[])
			GROUP BY ch.character_id, ch.name,
			         ch.corporation_id, ch.alliance_id,
			         corp.name, corp.ticker, ally.name, ally.ticker`,
			validNames)
		if err != nil {
			return legacyPayload{}, err
		}
		if len(rows) == 0 {
			return legacyPayload{}, apiError(
				http.StatusNotFound, "No characters found",
			)
		}

		resolved := make(map[string]bool, len(rows))
		alliances := make(map[string]any)
		corporations := make(map[string]any)
		dangerous := 0
		for _, row := range rows {
			name, _ := stringValue(row["name"])
			resolved[name] = true
			characterID := int64OrZero(row["character_id"])
			kills := int64OrZero(row["kills"])
			if kills > 5 {
				dangerous++
			}
			character := map[string]any{
				"characterId": characterID, "name": name, "kills": kills,
			}
			corporationID := int64OrZero(row["corporation_id"])
			allianceID := int64OrZero(row["alliance_id"])
			if allianceID > 0 {
				key := strconv.FormatInt(allianceID, 10)
				alliance, _ := alliances[key].(map[string]any)
				if alliance == nil {
					alliance = map[string]any{
						"name":         valueOrFallback(row["alliance_name"], "Unknown Alliance"),
						"ticker":       valueOrFallback(row["alliance_ticker"], "???"),
						"corporations": map[string]any{},
					}
					alliances[key] = alliance
				}
				if corporationID > 0 {
					corporationKey := strconv.FormatInt(corporationID, 10)
					corps := alliance["corporations"].(map[string]any)
					corporation, _ := corps[corporationKey].(map[string]any)
					if corporation == nil {
						corporation = map[string]any{
							"name": valueOrFallback(
								row["corporation_name"], "Unknown Corporation",
							),
							"ticker": valueOrFallback(
								row["corporation_ticker"], "???",
							),
							"characters": []map[string]any{},
						}
						corps[corporationKey] = corporation
					}
					corporation["characters"] = append(
						corporation["characters"].([]map[string]any), character,
					)
				}
			} else if corporationID > 0 {
				key := strconv.FormatInt(corporationID, 10)
				corporation, _ := corporations[key].(map[string]any)
				if corporation == nil {
					corporation = map[string]any{
						"name": valueOrFallback(
							row["corporation_name"], "Unknown Corporation",
						),
						"ticker": valueOrFallback(
							row["corporation_ticker"], "???",
						),
						"characters": []map[string]any{},
					}
					corporations[key] = corporation
				}
				corporation["characters"] = append(
					corporation["characters"].([]map[string]any), character,
				)
			}
		}
		unresolved := make([]string, 0)
		for _, name := range validNames {
			if !resolved[name] {
				unresolved = append(unresolved, name)
			}
		}
		return jsonPayload(map[string]any{
			"alliances": alliances, "corporations": corporations,
			"unresolved": unresolved, "totalCharacters": len(rows),
			"totalDangerous": dangerous,
		}), nil
	}
}

func valueOrFallback(value any, fallback string) string {
	if text, ok := stringValue(value); ok {
		return text
	}
	return fallback
}

func saveScanHandler(
	opts Options,
	auth *authService,
	scanType string,
) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		if err := requireSameOriginMutation(req.Huma); err != nil {
			return legacyPayload{}, err
		}
		principal, err := auth.requirePrincipal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		raw, err := decodeScanBody(req)
		if err != nil {
			return legacyPayload{}, err
		}
		body, ok := raw.(map[string]any)
		if !ok {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid JSON body")
		}
		result := body["result"]
		if !jsonTruthy(result) {
			message := "Missing names or result"
			if scanType == "dscan" {
				message = "Missing dscan or result"
			}
			return legacyPayload{}, apiError(http.StatusBadRequest, message)
		}

		var normalized string
		if scanType == "dscan" {
			dscan, ok := body["dscan"].(string)
			if !ok || dscan == "" {
				return legacyPayload{}, apiError(
					http.StatusBadRequest, "Missing dscan or result",
				)
			}
			normalized = strings.TrimSpace(dscan)
		} else {
			names, ok := body["names"].([]any)
			if !ok {
				return legacyPayload{}, apiError(
					http.StatusBadRequest, "Missing names or result",
				)
			}
			normalizedNames := make([]string, 0, len(names))
			for _, value := range names {
				name, ok := value.(string)
				if !ok {
					return legacyPayload{}, apiError(
						http.StatusBadRequest, "Missing names or result",
					)
				}
				normalizedNames = append(normalizedNames, name)
			}
			sort.Strings(normalizedNames)
			normalized = strings.Join(normalizedNames, "\n")
		}
		digest := sha256.Sum256([]byte(normalized))
		hash := hex.EncodeToString(digest[:])
		db, err := mutationDatabase(opts)
		if err != nil {
			return legacyPayload{}, err
		}
		if _, err := db.Exec(ctx, `
			INSERT INTO scans (
				hash, scan_type, input, result, character_id
			) VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (hash) DO UPDATE
			SET result = EXCLUDED.result,
			    character_id = EXCLUDED.character_id`,
			hash, scanType, normalized, result, principal.CharacterID,
		); err != nil {
			return legacyPayload{}, err
		}
		return accountNoStorePayload(map[string]any{"hash": hash}), nil
	}
}

func jsonTruthy(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case bool:
		return typed
	case string:
		return typed != ""
	case json.Number:
		number, err := strconv.ParseFloat(string(typed), 64)
		return err != nil || number != 0
	case float64:
		return typed != 0
	default:
		return true
	}
}

func getScanHandler(opts Options, scanType string) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		hash := req.Param("hash")
		if hash == "" {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Missing hash")
		}
		row, err := queryMap(ctx, opts.DB, `
			SELECT result
			FROM scans
			WHERE hash = $1 AND scan_type = $2
			LIMIT 1`, hash, scanType)
		if err != nil {
			return legacyPayload{}, err
		}
		if row == nil {
			return legacyPayload{}, apiError(http.StatusNotFound, "Scan not found")
		}
		return jsonPayload(row["result"]), nil
	}
}
