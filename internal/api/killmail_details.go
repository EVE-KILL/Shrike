package api

import (
	"context"
	"net/http"
	"sort"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

const killmailCacheTTL = 5 * time.Minute

func registerKillmailDetailRoutes(a huma.API, opts Options) {
	cache := func(handler legacyHandler) legacyHandler {
		return routeJSONCache(
			opts,
			killmailCacheTTL,
			"public, max-age=60, s-maxage=300, stale-while-revalidate=60",
			handler,
		)
	}

	// /killmails/{id} is the canonical detail operation already registered by
	// the established catalogue. This singular path keeps the current Nuxt caller
	// working until web/ moves to that shared contract.
	registerLegacy(a, killmailIDOperation(
		"killmail-detail-legacy",
		"/killmail/{id}",
		"Full killmail detail",
	), cache(killmailDetailHandler(opts)))

	exists := killmailExistsHandler(opts)
	registerLegacy(a, killmailIDOperation(
		"killmail-exists",
		"/killmails/{id}/exists",
		"Check whether a killmail exists",
	), exists)
	registerLegacy(a, killmailIDOperation(
		"killmail-exists-legacy",
		"/killmail/{id}/exists",
		"Check whether a killmail exists",
	), exists)

	editorFit := cache(killmailEditorFitHandler(opts))
	registerLegacy(a, killmailIDOperation(
		"killmail-editor-fit",
		"/killmails/{id}/editor-fit",
		"UI-ready fitting extracted from a killmail",
	), editorFit)
	registerLegacy(a, killmailIDOperation(
		"killmail-editor-fit-legacy",
		"/killmail/{id}/fit",
		"UI-ready fitting extracted from a killmail",
	), editorFit)

	siblings := cache(killmailSiblingsHandler(opts))
	registerLegacy(a, killmailIDOperation(
		"killmail-siblings",
		"/killmails/{id}/siblings",
		"Related losses for the same victim",
	), siblings)
	registerLegacy(a, killmailIDOperation(
		"killmail-siblings-legacy",
		"/killmail/{id}/siblings",
		"Related losses for the same victim",
	), siblings)
}

func killmailDetailHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		id, err := parseID(req.Param("id"))
		if err != nil {
			return legacyPayload{}, err
		}
		body, found, err := loadKillmailDetail(ctx, opts.DB, id)
		if err != nil {
			return legacyPayload{}, err
		}
		if !found {
			return legacyPayload{}, apiError(http.StatusNotFound, "Killmail not found")
		}
		return jsonPayload(body), nil
	}
}

func killmailExistsHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		id, err := parseID(req.Param("id"))
		if err != nil {
			return legacyPayload{}, apiError(
				http.StatusBadRequest, "Invalid killmail id",
			)
		}
		row, err := queryMap(ctx, opts.DB, `
			SELECT EXISTS (
				SELECT 1 FROM killmails WHERE killmail_id = $1
			) AS exists`, id)
		if err != nil {
			return legacyPayload{}, err
		}
		exists, _ := row["exists"].(bool)
		return jsonPayload(map[string]any{"exists": exists}), nil
	}
}

func killmailEditorFitHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		id, err := parseID(req.Param("id"))
		if err != nil {
			return legacyPayload{}, apiError(
				http.StatusBadRequest, "Invalid killmail id",
			)
		}

		killmail, err := queryMap(ctx, opts.DB, `
			SELECT k.victim_ship_type_id AS ship_type_id, t.name AS ship_name
			FROM killmails k
			LEFT JOIN inv_types t ON t.type_id = k.victim_ship_type_id
			WHERE k.killmail_id = $1
			LIMIT 1`, id)
		if err != nil {
			return legacyPayload{}, err
		}
		if killmail == nil {
			return legacyPayload{}, apiError(http.StatusNotFound, "Killmail not found")
		}
		shipTypeID, ok := int64Value(killmail["ship_type_id"])
		if !ok {
			return legacyPayload{}, apiError(
				http.StatusNotFound, "Killmail has no victim ship",
			)
		}
		shipName, ok := stringValue(killmail["ship_name"])
		if !ok || shipName == "" {
			shipName = "Killmail Fit"
		}

		rows, err := queryMaps(ctx, opts.DB, `
			SELECT i.item_index, i.type_id, i.flag_id,
			       COALESCE(i.quantity_dropped, 0) AS quantity_dropped,
			       COALESCE(i.quantity_destroyed, 0) AS quantity_destroyed,
			       t.name AS type_name, g.category_id
			FROM killmail_items i
			LEFT JOIN inv_types t ON t.type_id = i.type_id
			LEFT JOIN inv_groups g ON g.group_id = t.group_id
			WHERE i.killmail_id = $1
			  AND i.parent_index IS NULL
			  AND (
			    i.flag_id BETWEEN 11 AND 34
			    OR i.flag_id BETWEEN 92 AND 99
			    OR i.flag_id BETWEEN 125 AND 132
			    OR i.flag_id = 87
			  )
			ORDER BY i.item_index`, id)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(buildKillmailEditorFit(shipTypeID, shipName, rows)), nil
	}
}

type editorFitModule struct {
	SlotGroup    int    `json:"slot_group"`
	Ordinal      int    `json:"ordinal"`
	TypeID       int64  `json:"type_id"`
	Name         any    `json:"name"`
	ChargeTypeID *int64 `json:"charge_type_id"`
}

type editorFitDrone struct {
	TypeID   int64 `json:"type_id"`
	Name     any   `json:"name"`
	Quantity int64 `json:"quantity"`
}

type pendingEditorModule struct {
	TypeID       int64
	Name         any
	ChargeTypeID *int64
}

func buildKillmailEditorFit(
	shipTypeID int64,
	shipName string,
	rows []map[string]any,
) map[string]any {
	charges := make(map[int64]int64)
	for _, row := range rows {
		categoryID, _ := int64Value(row["category_id"])
		if categoryID != 8 {
			continue
		}
		flagID, flagOK := int64Value(row["flag_id"])
		typeID, typeOK := int64Value(row["type_id"])
		if flagOK && typeOK {
			charges[flagID] = typeID
		}
	}

	buckets := make(map[int][]pendingEditorModule)
	dronesByType := make(map[int64]*editorFitDrone)
	for _, row := range rows {
		flagID, flagOK := int64Value(row["flag_id"])
		typeID, typeOK := int64Value(row["type_id"])
		categoryID, categoryOK := int64Value(row["category_id"])
		if !flagOK || !typeOK || !categoryOK {
			continue
		}
		name := row["type_name"]

		if flagID == 87 && categoryID == 18 {
			dropped, _ := int64Value(row["quantity_dropped"])
			destroyed, _ := int64Value(row["quantity_destroyed"])
			drone := dronesByType[typeID]
			if drone == nil {
				drone = &editorFitDrone{TypeID: typeID, Name: name}
				dronesByType[typeID] = drone
			}
			drone.Quantity += dropped + destroyed
			continue
		}

		slot := editorSlotGroup(flagID)
		if slot == 0 || (categoryID != 7 && categoryID != 32) {
			continue
		}
		var charge *int64
		if value, found := charges[flagID]; found {
			copy := value
			charge = &copy
		}
		buckets[slot] = append(buckets[slot], pendingEditorModule{
			TypeID: typeID, Name: name, ChargeTypeID: charge,
		})
	}

	modules := make([]editorFitModule, 0)
	for slot := 1; slot <= 5; slot++ {
		bucket := buckets[slot]
		sort.SliceStable(bucket, func(i, j int) bool {
			if bucket[i].TypeID != bucket[j].TypeID {
				return bucket[i].TypeID < bucket[j].TypeID
			}
			return optionalInt64(bucket[i].ChargeTypeID) <
				optionalInt64(bucket[j].ChargeTypeID)
		})
		for ordinal, module := range bucket {
			modules = append(modules, editorFitModule{
				SlotGroup: slot, Ordinal: ordinal,
				TypeID: module.TypeID, Name: module.Name,
				ChargeTypeID: module.ChargeTypeID,
			})
		}
	}

	droneIDs := make([]int64, 0, len(dronesByType))
	for typeID := range dronesByType {
		droneIDs = append(droneIDs, typeID)
	}
	sort.Slice(droneIDs, func(i, j int) bool { return droneIDs[i] < droneIDs[j] })
	drones := make([]editorFitDrone, 0, len(droneIDs))
	for _, typeID := range droneIDs {
		drones = append(drones, *dronesByType[typeID])
	}

	return map[string]any{
		"shipTypeId": shipTypeID,
		"name":       shipName,
		"modules":    modules,
		"drones":     drones,
	}
}

func optionalInt64(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func editorSlotGroup(flagID int64) int {
	switch {
	case flagID >= 27 && flagID <= 34:
		return 1
	case flagID >= 19 && flagID <= 26:
		return 2
	case flagID >= 11 && flagID <= 18:
		return 3
	case flagID >= 92 && flagID <= 99:
		return 4
	case flagID >= 125 && flagID <= 132:
		return 5
	case flagID == 87:
		return 6
	default:
		return 0
	}
}

func killmailSiblingsHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		id, err := parseID(req.Param("id"))
		if err != nil {
			return legacyPayload{}, apiError(
				http.StatusBadRequest, "Invalid killmail id",
			)
		}
		header, err := queryMap(ctx, opts.DB, `
			SELECT victim_character_id, victim_ship_type_id,
			       solar_system_id, killmail_time
			FROM killmails
			WHERE killmail_id = $1
			LIMIT 1`, id)
		if err != nil {
			return legacyPayload{}, err
		}
		if header == nil || header["victim_character_id"] == nil {
			return jsonPayload(map[string]any{"siblings": []any{}}), nil
		}

		rows, err := queryMaps(ctx, opts.DB, `
			SELECT k.killmail_id,
			       k.victim_ship_type_id AS ship_type_id,
			       k.victim_ship_group_id AS ship_group_id,
			       t.name AS ship_name,
			       COALESCE(k.total_value, 0) AS total_value,
			       k.killmail_time
			FROM killmails k
			LEFT JOIN inv_types t ON t.type_id = k.victim_ship_type_id
			WHERE k.victim_character_id = $1
			  AND k.solar_system_id = $2
			  AND k.killmail_time >= $3::timestamptz - interval '1 hour'
			  AND k.killmail_time <= $3::timestamptz + interval '1 hour'
			  AND k.killmail_id <> $4
			  AND k.victim_ship_type_id <> COALESCE($5, 0)
			ORDER BY k.killmail_time DESC, k.killmail_id DESC
			LIMIT 10`,
			header["victim_character_id"],
			header["solar_system_id"],
			header["killmail_time"],
			id,
			header["victim_ship_type_id"],
		)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{"siblings": nonNilRows(rows)}), nil
	}
}
