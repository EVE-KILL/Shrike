package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/eve-kill/shrike/internal/killtype"
)

const advancedMaximumIDsPerList = 15

var advancedHashPattern = regexp.MustCompile(`(?i)^[a-f0-9]{64}$`)

type advancedEntity struct {
	ID      int64  `json:"id"`
	Type    string `json:"type"`
	Name    string `json:"name,omitempty"`
	Exclude bool   `json:"exclude,omitempty"`
}

type advancedItemFilter struct {
	TypeID  *int64 `json:"typeId,omitempty"`
	GroupID *int64 `json:"groupId,omitempty"`
	Name    string `json:"name,omitempty"`
	Slot    string `json:"slot,omitempty"`
	Side    string `json:"side,omitempty"`
}

type advancedTimeRange struct {
	Preset string `json:"preset,omitempty"`
	From   string `json:"from,omitempty"`
	To     string `json:"to,omitempty"`
}

type advancedFilters struct {
	Label    string `json:"label,omitempty"`
	Entities *struct {
		Victim   []advancedEntity `json:"victim,omitempty"`
		Attacker []advancedEntity `json:"attacker,omitempty"`
		Both     []advancedEntity `json:"both,omitempty"`
	} `json:"entities,omitempty"`
	Items    []advancedItemFilter `json:"items,omitempty"`
	Location *struct {
		SecurityTypes   []string `json:"securityTypes,omitempty"`
		SystemID        int64    `json:"systemId,omitempty"`
		RegionID        int64    `json:"regionId,omitempty"`
		ConstellationID int64    `json:"constellationId,omitempty"`
	} `json:"location,omitempty"`
	TimeRange     *advancedTimeRange `json:"timeRange,omitempty"`
	AttackerCount string             `json:"attackerCount,omitempty"`
	AttackerType  string             `json:"attackerType,omitempty"`
	ISKValue      string             `json:"iskValue,omitempty"`
	ISKMin        *float64           `json:"iskMin,omitempty"`
	ISKMax        *float64           `json:"iskMax,omitempty"`
	ShipCategory  string             `json:"shipCategory,omitempty"`
	TechLevel     string             `json:"techLevel,omitempty"`
	Sort          *struct {
		Field     string `json:"field,omitempty"`
		Direction string `json:"direction,omitempty"`
	} `json:"sort,omitempty"`
}

type advancedKilllistQuery struct {
	Where     []string
	Args      []any
	Sort      string
	Direction string
	Limit     int
	View      string
	Dedup     string
}

func (q *advancedKilllistQuery) bind(value any) string {
	q.Args = append(q.Args, value)
	return fmt.Sprintf("$%d", len(q.Args))
}

func (q advancedKilllistQuery) whereSQL() string {
	if len(q.Where) == 0 {
		return "TRUE"
	}
	return strings.Join(q.Where, " AND ")
}

func advancedKilllistHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		query, err := parseAdvancedKilllistQuery(req, time.Now().UTC())
		if err != nil {
			return legacyPayload{}, err
		}
		switch query.View {
		case "fits":
			return loadAdvancedFits(ctx, opts.DB, query)
		default:
			return loadAdvancedKills(ctx, opts.DB, query)
		}
	}
}

func parseAdvancedKilllistQuery(
	req *legacyRequest,
	now time.Time,
) (advancedKilllistQuery, error) {
	query := advancedKilllistQuery{
		Limit: boundedQueryInt(req, "limit", 50, 1, 100),
		View:  strings.TrimSpace(req.Query.Get("view")),
		Dedup: strings.TrimSpace(req.Query.Get("dedup")),
	}
	if query.View == "" {
		query.View = "kills"
	}
	if query.Dedup == "" {
		query.Dedup = "none"
	}

	fitHash := strings.TrimSpace(req.Query.Get("fitHash"))
	familyHash := strings.TrimSpace(req.Query.Get("familyHash"))
	if fitHash != "" && !advancedHashPattern.MatchString(fitHash) {
		return query, apiError(http.StatusBadRequest, "Invalid fitHash")
	}
	if familyHash != "" && !advancedHashPattern.MatchString(familyHash) {
		return query, apiError(http.StatusBadRequest, "Invalid familyHash")
	}

	filters, err := decodeAdvancedFilters(req.Query.Get("filters"))
	if err != nil {
		return query, err
	}
	if err := validateAdvancedListSizes(filters); err != nil {
		return query, err
	}
	if filters.Label != "" {
		predicate, ok := killtype.Predicates()[filters.Label]
		if !ok || filters.Label == "latest" {
			return query, apiError(http.StatusBadRequest, "Invalid label")
		}
		query.Where = append(query.Where, predicate)
	}

	if fitHash != "" {
		arg := query.bind(fitHash)
		query.Where = append(query.Where,
			"k.killmail_id IN (SELECT killmail_id FROM killmail_fittings WHERE fit_hash = "+arg+")",
		)
	} else if familyHash != "" {
		arg := query.bind(familyHash)
		query.Where = append(query.Where, `
			k.killmail_id IN (
				SELECT fitting.killmail_id
				FROM killmail_fittings fitting
				JOIN fittings family ON family.fit_hash = fitting.fit_hash
				WHERE family.family_hash = `+arg+`
			)`)
	}

	since, until, err := advancedTimeBounds(filters.TimeRange, now)
	if err != nil {
		return query, err
	}
	query.Where = append(query.Where,
		"k.killmail_time >= "+query.bind(since))
	if until != nil {
		query.Where = append(query.Where,
			"k.killmail_time <= "+query.bind(*until))
	}

	addAdvancedLocationFilters(&query, filters)
	addAdvancedValueFilters(&query, filters)
	addAdvancedCombatFilters(&query, filters)
	if err := addAdvancedEntityFilters(&query, filters); err != nil {
		return query, err
	}
	if err := addAdvancedItemFilters(&query, filters); err != nil {
		return query, err
	}

	query.Sort = "k.killmail_time"
	if filters.Sort != nil {
		switch filters.Sort.Field {
		case "total_value":
			query.Sort = "COALESCE(k.total_value, 0)"
		case "attacker_count":
			query.Sort = "COALESCE(k.attacker_count, 0)"
		}
	}
	query.Direction = "DESC"
	if filters.Sort != nil &&
		strings.EqualFold(filters.Sort.Direction, "asc") {
		query.Direction = "ASC"
	}

	after, err := optionalPositiveInt64(req.Query.Get("after"))
	if err != nil {
		return query, apiError(http.StatusBadRequest, "Invalid after")
	}
	if after != nil {
		operator := "<"
		if query.Direction == "ASC" {
			operator = ">"
		}
		arg := query.bind(*after)
		query.Where = append(query.Where, fmt.Sprintf(
			`(%s, k.killmail_id) %s (
				SELECT %s, killmail_id FROM killmails
				WHERE killmail_id = %s
			)`,
			query.Sort, operator,
			advancedCursorColumn(query.Sort), arg,
		))
	}
	return query, nil
}

func advancedCursorColumn(column string) string {
	switch column {
	case "COALESCE(k.total_value, 0)":
		return "COALESCE(total_value, 0)"
	case "COALESCE(k.attacker_count, 0)":
		return "COALESCE(attacker_count, 0)"
	default:
		return "killmail_time"
	}
}

func decodeAdvancedFilters(raw string) (advancedFilters, error) {
	if strings.TrimSpace(raw) == "" {
		return advancedFilters{}, nil
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	var filters advancedFilters
	if err := decoder.Decode(&filters); err != nil {
		return advancedFilters{}, apiError(
			http.StatusBadRequest, "Invalid filters JSON",
		)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return advancedFilters{}, apiError(
			http.StatusBadRequest, "Invalid filters JSON",
		)
	}
	return filters, nil
}

func validateAdvancedListSizes(filters advancedFilters) error {
	lists := []struct {
		Name string
		Size int
	}{{
		Name: "items", Size: len(filters.Items),
	}}
	if filters.Entities != nil {
		lists = append(lists,
			struct {
				Name string
				Size int
			}{"entities.victim", len(filters.Entities.Victim)},
			struct {
				Name string
				Size int
			}{"entities.attacker", len(filters.Entities.Attacker)},
			struct {
				Name string
				Size int
			}{"entities.both", len(filters.Entities.Both)},
		)
	}
	for _, list := range lists {
		if list.Size > advancedMaximumIDsPerList {
			return apiError(http.StatusBadRequest, fmt.Sprintf(
				"%s exceeds %d entries",
				list.Name, advancedMaximumIDsPerList,
			))
		}
	}
	return nil
}

func advancedTimeBounds(
	input *advancedTimeRange,
	now time.Time,
) (time.Time, *time.Time, error) {
	now = now.UTC()
	if input == nil {
		return now.AddDate(0, 0, -30), nil, nil
	}
	today := time.Date(
		now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC,
	)
	if input.Preset != "" {
		switch input.Preset {
		case "today":
			return today, nil, nil
		case "yesterday":
			start := today.AddDate(0, 0, -1)
			return start, &today, nil
		case "24h":
			return now.Add(-24 * time.Hour), nil, nil
		case "7d":
			return now.AddDate(0, 0, -7), nil, nil
		case "30d":
			return now.AddDate(0, 0, -30), nil, nil
		case "90d":
			return now.AddDate(0, 0, -90), nil, nil
		case "thisWeek":
			daysSinceMonday := (int(today.Weekday()) + 6) % 7
			return today.AddDate(0, 0, -daysSinceMonday), nil, nil
		case "thisMonth":
			return time.Date(
				now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC,
			), nil, nil
		default:
			return now.AddDate(0, 0, -30), nil, nil
		}
	}
	from, err := parseAdvancedTime(input.From)
	if err != nil {
		return time.Time{}, nil, apiError(
			http.StatusBadRequest, "Invalid timeRange.from",
		)
	}
	to, err := parseAdvancedTime(input.To)
	if err != nil {
		return time.Time{}, nil, apiError(
			http.StatusBadRequest, "Invalid timeRange.to",
		)
	}
	if to.Before(from) {
		return time.Time{}, nil, apiError(
			http.StatusBadRequest, "timeRange.from must be before timeRange.to",
		)
	}
	return from, &to, nil
}

func parseAdvancedTime(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	for _, layout := range []string{
		time.RFC3339Nano,
		"2006-01-02T15:04",
		"2006-01-02T15:04:05",
		"2006-01-02",
	} {
		if value, err := time.Parse(layout, raw); err == nil {
			return value.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid time")
}

func addAdvancedLocationFilters(
	query *advancedKilllistQuery,
	filters advancedFilters,
) {
	if filters.Location == nil {
		return
	}
	switch {
	case filters.Location.SystemID != 0:
		query.Where = append(query.Where,
			"k.solar_system_id = "+query.bind(filters.Location.SystemID))
	case filters.Location.ConstellationID != 0:
		query.Where = append(query.Where,
			`k.solar_system_id IN (
				SELECT solar_system_id FROM solar_systems
				WHERE constellation_id = `+
				query.bind(filters.Location.ConstellationID)+
				")")
	case filters.Location.RegionID != 0:
		query.Where = append(query.Where,
			"k.region_id = "+query.bind(filters.Location.RegionID))
	}
	security := map[string]string{
		"highsec": `k.solar_system_id IN (
			SELECT solar_system_id FROM solar_systems WHERE security >= 0.45
		)`,
		"lowsec": `k.solar_system_id IN (
			SELECT solar_system_id FROM solar_systems
			WHERE security > 0.0 AND security < 0.45
		)`,
		"nullsec": `k.solar_system_id IN (
			SELECT solar_system_id FROM solar_systems
			WHERE security <= 0.0 AND region_id < 11000000
		)`,
		"wspace":  "k.region_id >= 11000001 AND k.region_id <= 11000033",
		"abyssal": "k.region_id >= 12000001 AND k.region_id <= 12000005",
		"pochven": "k.region_id = 10000070",
	}
	parts := []string{}
	for _, kind := range filters.Location.SecurityTypes {
		if condition := security[kind]; condition != "" {
			parts = append(parts, "("+condition+")")
		}
	}
	if len(parts) > 0 {
		query.Where = append(query.Where,
			"("+strings.Join(parts, " OR ")+")")
	}
}

func addAdvancedValueFilters(
	query *advancedKilllistQuery,
	filters advancedFilters,
) {
	if filters.ISKMin != nil || filters.ISKMax != nil {
		if filters.ISKMin != nil &&
			!math.IsNaN(*filters.ISKMin) &&
			!math.IsInf(*filters.ISKMin, 0) {
			query.Where = append(query.Where,
				"k.total_value >= "+query.bind(*filters.ISKMin))
		}
		if filters.ISKMax != nil &&
			!math.IsNaN(*filters.ISKMax) &&
			!math.IsInf(*filters.ISKMax, 0) {
			query.Where = append(query.Where,
				"k.total_value <= "+query.bind(*filters.ISKMax))
		}
	} else {
		thresholds := map[string]float64{
			"1b": 1e9, "5b": 5e9, "10b": 1e10,
			"50b": 5e10, "100b": 1e11,
		}
		if threshold := thresholds[filters.ISKValue]; threshold > 0 {
			query.Where = append(query.Where,
				"k.total_value >= "+query.bind(threshold))
		}
	}
}

func addAdvancedCombatFilters(
	query *advancedKilllistQuery,
	filters advancedFilters,
) {
	switch filters.AttackerCount {
	case "solo":
		query.Where = append(query.Where, "k.is_solo = true")
	default:
		if before, ok := strings.CutSuffix(filters.AttackerCount, "+"); ok {
			raw := before
			if count, err := strconv.ParseInt(raw, 10, 32); err == nil {
				query.Where = append(query.Where,
					"k.attacker_count >= "+query.bind(count))
			}
		}
	}
	switch filters.AttackerType {
	case "npc":
		query.Where = append(query.Where, "k.is_npc = true")
	case "ganked":
		query.Where = append(query.Where,
			"k.attacker_count >= 10",
			`k.solar_system_id IN (
				SELECT solar_system_id FROM solar_systems
				WHERE security >= 0.45
			)`,
		)
	}

	shipGroups := map[string][]int32{
		"frigates":       {25, 324, 830, 831, 834, 893, 1283, 1527},
		"destroyers":     {420, 1305, 1534},
		"cruisers":       {26, 358, 832, 833, 906, 894, 963, 1972},
		"battlecruisers": {419, 1201, 540},
		"battleships":    {27, 898, 900},
		"capitals":       {547, 485, 1538, 883},
		"supercarriers":  {659},
		"titans":         {30},
		"freighters":     {513, 902},
		"supercapitals":  {659, 30},
	}
	if filters.ShipCategory == "citadels" {
		query.Where = append(query.Where, `k.victim_ship_group_id IN (
			SELECT group_id FROM inv_groups WHERE category_id = 65
		)`)
	} else if groups := shipGroups[filters.ShipCategory]; len(groups) > 0 {
		query.Where = append(query.Where,
			"k.victim_ship_group_id = ANY("+
				query.bind(groups)+"::int[])")
	}

	meta := map[string]string{
		"t1":      "COALESCE(meta_group_id, 1) = 1",
		"t2":      "meta_group_id = 2",
		"t3":      "meta_group_id = 14",
		"faction": "meta_group_id = 4",
	}
	if condition := meta[filters.TechLevel]; condition != "" {
		query.Where = append(query.Where,
			"k.victim_ship_type_id IN (SELECT type_id FROM inv_types WHERE "+
				condition+")")
	}
}

func addAdvancedEntityFilters(
	query *advancedKilllistQuery,
	filters advancedFilters,
) error {
	if filters.Entities == nil {
		return nil
	}
	victimIncludes := []string{}
	victimExcludes := []string{}
	attackerIncludes := []string{}
	attackerExcludes := []string{}

	for _, entity := range filters.Entities.Victim {
		condition, ok := advancedVictimEntityCondition(query, entity)
		if !ok {
			continue
		}
		if entity.Exclude {
			victimExcludes = append(victimExcludes, condition)
		} else {
			victimIncludes = append(victimIncludes, condition)
		}
	}
	for _, entity := range filters.Entities.Attacker {
		condition, ok := advancedAttackerEntityCondition(query, entity)
		if !ok {
			continue
		}
		if entity.Exclude {
			attackerExcludes = append(attackerExcludes, condition)
		} else {
			attackerIncludes = append(attackerIncludes, condition)
		}
	}
	for _, entity := range filters.Entities.Both {
		// The "both" bucket means either side; exclusion is only a supported
		// semantic in the explicit victim/attacker buckets.
		entity.Exclude = false
		if condition, ok := advancedVictimEntityCondition(query, entity); ok {
			victimIncludes = append(victimIncludes, condition)
		}
		if condition, ok := advancedAttackerEntityCondition(query, entity); ok {
			attackerIncludes = append(attackerIncludes, condition)
		}
	}

	query.Where = append(query.Where, victimExcludes...)
	if len(attackerIncludes) > 0 {
		attacker := `EXISTS (
			SELECT 1 FROM killmail_attackers attacker
			WHERE attacker.killmail_id = k.killmail_id
			  AND (` + strings.Join(attackerIncludes, " OR ") + `)
		)`
		if len(victimIncludes) > 0 {
			victimIncludes = append(victimIncludes, attacker)
			query.Where = append(query.Where,
				"("+strings.Join(victimIncludes, " OR ")+")")
		} else {
			query.Where = append(query.Where, attacker)
		}
	} else if len(victimIncludes) == 1 {
		query.Where = append(query.Where, victimIncludes[0])
	} else if len(victimIncludes) > 1 {
		// This preserves the original victim-only contract: multiple explicit
		// victim filters narrow the result rather than broadening it.
		query.Where = append(query.Where,
			"("+strings.Join(victimIncludes, " AND ")+")")
	}
	if len(attackerExcludes) > 0 {
		query.Where = append(query.Where, `NOT EXISTS (
			SELECT 1 FROM killmail_attackers attacker
			WHERE attacker.killmail_id = k.killmail_id
			  AND (`+strings.Join(attackerExcludes, " OR ")+`)
		)`)
	}
	return nil
}

func advancedVictimEntityCondition(
	query *advancedKilllistQuery,
	entity advancedEntity,
) (string, bool) {
	if entity.ID == 0 {
		return "", false
	}
	column := map[string]string{
		"character":   "k.victim_character_id",
		"corporation": "k.victim_corporation_id",
		"alliance":    "k.victim_alliance_id",
		"ship":        "k.victim_ship_type_id",
		"shipgroup":   "k.victim_ship_group_id",
		"faction":     "k.victim_faction_id",
	}[entity.Type]
	if column == "" {
		return "", false
	}
	operator := "="
	if entity.Exclude {
		operator = "<>"
	}
	return column + " " + operator + " " + query.bind(entity.ID), true
}

func advancedAttackerEntityCondition(
	query *advancedKilllistQuery,
	entity advancedEntity,
) (string, bool) {
	if entity.ID == 0 {
		return "", false
	}
	column := map[string]string{
		"character":   "attacker.character_id",
		"corporation": "attacker.corporation_id",
		"alliance":    "attacker.alliance_id",
		"ship":        "attacker.ship_type_id",
		"shipgroup":   "attacker.ship_group_id",
		"weapon":      "attacker.weapon_type_id",
		"faction":     "attacker.faction_id",
	}[entity.Type]
	if column == "" {
		return "", false
	}
	return column + " = " + query.bind(entity.ID), true
}

func addAdvancedItemFilters(
	query *advancedKilllistQuery,
	filters advancedFilters,
) error {
	for _, item := range filters.Items {
		var victimMatch, attackerMatch string
		switch {
		case item.TypeID != nil && *item.TypeID != 0:
			arg := query.bind(*item.TypeID)
			victimMatch = "items.type_id = " + arg
			attackerMatch = "attacker.weapon_type_id = " + arg
		case item.GroupID != nil && *item.GroupID != 0:
			arg := query.bind(*item.GroupID)
			victimMatch = `items.type_id IN (
				SELECT type_id FROM inv_types WHERE group_id = ` + arg + `
			)`
			attackerMatch = `attacker.weapon_type_id IN (
				SELECT type_id FROM inv_types WHERE group_id = ` + arg + `
			)`
		default:
			continue
		}

		slot := item.Slot
		if slot == "" {
			slot = "any"
		}
		side := item.Side
		if side == "" {
			side = "victim"
		}
		fitted := `items.parent_index IS NULL AND (
			items.flag_id BETWEEN 11 AND 34
			OR items.flag_id BETWEEN 92 AND 99
			OR items.flag_id BETWEEN 125 AND 132
			OR items.flag_id = 87
		)`
		slotCondition := ""
		switch slot {
		case "fitted":
			slotCondition = " AND (" + fitted + ")"
		case "cargo":
			slotCondition = " AND NOT (" + fitted + ")"
		}

		conditions := []string{}
		if side == "victim" || side == "either" {
			conditions = append(conditions, `EXISTS (
				SELECT 1 FROM killmail_items items
				WHERE items.killmail_id = k.killmail_id
				  AND `+victimMatch+slotCondition+`
			)`)
		}
		if side == "attacker" || side == "either" {
			conditions = append(conditions, `EXISTS (
				SELECT 1 FROM killmail_attackers attacker
				WHERE attacker.killmail_id = k.killmail_id
				  AND `+attackerMatch+`
			)`)
		}
		if len(conditions) == 1 {
			query.Where = append(query.Where, conditions[0])
		} else if len(conditions) > 1 {
			query.Where = append(query.Where,
				"("+strings.Join(conditions, " OR ")+")")
		}
	}
	return nil
}

func loadAdvancedKills(
	ctx context.Context,
	db Database,
	query advancedKilllistQuery,
) (legacyPayload, error) {
	args := append([]any(nil), query.Args...)
	args = append(args, query.Limit+1)
	rows, err := queryMaps(ctx, db,
		campaignKilllistSelect+
			" WHERE "+query.whereSQL()+
			fmt.Sprintf(
				" ORDER BY %s %s, k.killmail_id %s LIMIT $%d",
				query.Sort, query.Direction, query.Direction, len(args),
			),
		args...,
	)
	if err != nil {
		return legacyPayload{}, err
	}
	rows, hasMore, cursor, err := finishUniverseKilllist(
		ctx, db, rows, query.Limit,
	)
	if err != nil {
		return legacyPayload{}, err
	}
	return jsonPayload(map[string]any{
		"kills": rows, "hasMore": hasMore, "cursor": cursor,
	}), nil
}

func loadAdvancedFits(
	ctx context.Context,
	db Database,
	query advancedKilllistQuery,
) (legacyPayload, error) {
	if query.Dedup == "exact" || query.Dedup == "family" {
		return loadAdvancedGroupedFits(ctx, db, query)
	}
	return loadAdvancedIndividualFits(ctx, db, query)
}

func loadAdvancedGroupedFits(
	ctx context.Context,
	db Database,
	query advancedKilllistQuery,
) (legacyPayload, error) {
	useFamily := query.Dedup == "family"
	hash := "fitting.fit_hash"
	representative := "fitting.fit_hash"
	join := ""
	if useFamily {
		hash = "family.family_hash"
		representative = "MIN(fitting.fit_hash)"
		join = " JOIN fittings family ON family.fit_hash = fitting.fit_hash"
	}
	args := append([]any(nil), query.Args...)
	args = append(args, min(query.Limit, 100))
	rows, err := queryMaps(ctx, db, fmt.Sprintf(`
		SELECT %s AS hash,
		       %s AS representative_fit_hash,
		       COUNT(*)::int AS count,
		       MAX(k.killmail_id) AS killmail_id,
		       MAX(k.killmail_time) AS killmail_time,
		       MAX(k.victim_ship_type_id) AS victim_ship_type_id,
		       AVG(k.total_value)::double precision AS avg_value,
		       ROUND(AVG(k.attacker_count))::int AS avg_attackers
		FROM killmails k
		JOIN killmail_fittings fitting
		  ON fitting.killmail_id = k.killmail_id
		%s
		WHERE %s
		GROUP BY %s
		ORDER BY count DESC
		LIMIT $%d`,
		hash, representative, join, query.whereSQL(), hash, len(args),
	), args...)
	if err != nil {
		return legacyPayload{}, err
	}
	if len(rows) == 0 {
		return jsonPayload(map[string]any{
			"fits":    []map[string]any{},
			"hasMore": false, "cursor": nil,
		}), nil
	}

	hashes := []string{}
	shipSet := map[int32]struct{}{}
	for _, row := range rows {
		if value := stringOrEmpty(row["representative_fit_hash"]); value != "" {
			hashes = append(hashes, value)
		}
		if id := int32From(row["victim_ship_type_id"]); id != 0 {
			shipSet[id] = struct{}{}
		}
	}
	shipIDs := make([]int32, 0, len(shipSet))
	for id := range shipSet {
		shipIDs = append(shipIDs, id)
	}
	itemRows, err := queryMaps(ctx, db, `
		SELECT item.fit_hash, item.slot_group, item.ordinal,
		       item.type_id, item.charge_type_id, item.quantity,
		       type.name AS type_name, group_data.category_id
		FROM fitting_items item
		LEFT JOIN inv_types type ON type.type_id = item.type_id
		LEFT JOIN inv_groups group_data ON group_data.group_id = type.group_id
		WHERE item.fit_hash = ANY($1::text[])
		ORDER BY item.fit_hash, item.slot_group, item.ordinal`, hashes)
	if err != nil {
		return legacyPayload{}, err
	}
	shipRows, err := queryMaps(ctx, db, `
		SELECT type_id, name FROM inv_types
		WHERE type_id = ANY($1::int[])`, shipIDs)
	if err != nil {
		return legacyPayload{}, err
	}
	shipNames := map[int64]string{}
	for _, row := range shipRows {
		shipNames[int64OrZero(row["type_id"])] = stringOrEmpty(row["name"])
	}
	itemsByHash := map[string][]map[string]any{}
	for _, row := range itemRows {
		key := stringOrEmpty(row["fit_hash"])
		itemsByHash[key] = append(itemsByHash[key], row)
	}

	fits := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		fitHash := stringOrEmpty(row["representative_fit_hash"])
		modules := []map[string]any{}
		drones := []map[string]any{}
		for _, item := range itemsByHash[fitHash] {
			slot := int64OrZero(item["slot_group"])
			category := int64OrZero(item["category_id"])
			name := stringOrEmpty(item["type_name"])
			if name == "" {
				name = "Unknown"
			}
			if slot >= 1 && slot <= 5 {
				modules = append(modules, map[string]any{
					"slot_group":     slot,
					"type_id":        item["type_id"],
					"name":           name,
					"charge_type_id": item["charge_type_id"],
				})
			}
			if slot == 6 || category == 18 {
				quantity, ok := int64Value(item["quantity"])
				if !ok {
					quantity = 1
				}
				drones = append(drones, map[string]any{
					"type_id": item["type_id"],
					"name":    name, "quantity": quantity,
				})
			}
		}
		shipID := int64OrZero(row["victim_ship_type_id"])
		shipName := shipNames[shipID]
		if shipName == "" {
			shipName = "Unknown"
		}
		fits = append(fits, map[string]any{
			"killmail_id":         row["killmail_id"],
			"killmail_time":       row["killmail_time"],
			"victim_ship_type_id": shipID,
			"victim_ship_name":    shipName,
			"total_value":         float64OrZero(row["avg_value"]),
			"attacker_count":      int64OrZero(row["avg_attackers"]),
			"modules":             modules, "drones": drones,
			"count":    int64OrZero(row["count"]),
			"fit_hash": fitHash, "hash": row["hash"],
			"dedup_mode": query.Dedup,
		})
	}
	return jsonPayload(map[string]any{
		"fits": fits, "hasMore": false, "cursor": nil,
	}), nil
}

func loadAdvancedIndividualFits(
	ctx context.Context,
	db Database,
	query advancedKilllistQuery,
) (legacyPayload, error) {
	limit := min(query.Limit, 100)
	args := append([]any(nil), query.Args...)
	args = append(args, limit+1)
	rows, err := queryMaps(ctx, db,
		campaignKilllistSelect+
			" WHERE "+query.whereSQL()+
			fmt.Sprintf(
				" ORDER BY %s %s, k.killmail_id %s LIMIT $%d",
				query.Sort, query.Direction, query.Direction, len(args),
			),
		args...,
	)
	if err != nil {
		return legacyPayload{}, err
	}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	if len(rows) == 0 {
		return jsonPayload(map[string]any{
			"fits":    []map[string]any{},
			"hasMore": false, "cursor": nil,
		}), nil
	}
	ids := make([]int32, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, int32(int64OrZero(row["killmail_id"])))
	}
	itemRows, err := queryMaps(ctx, db, `
		SELECT item.killmail_id, item.type_id, item.flag_id,
		       item.quantity_dropped, item.quantity_destroyed,
		       type.name AS type_name, group_data.category_id
		FROM killmail_items item
		LEFT JOIN inv_types type ON type.type_id = item.type_id
		LEFT JOIN inv_groups group_data ON group_data.group_id = type.group_id
		WHERE item.killmail_id = ANY($1::int[])
		  AND item.parent_index IS NULL
		  AND (
		    item.flag_id BETWEEN 11 AND 34
		    OR item.flag_id BETWEEN 92 AND 99
		    OR item.flag_id BETWEEN 125 AND 132
		    OR item.flag_id = 87
		  )
		ORDER BY item.killmail_id, item.item_index`, ids)
	if err != nil {
		return legacyPayload{}, err
	}
	itemsByKill := map[int64][]map[string]any{}
	for _, item := range itemRows {
		id := int64OrZero(item["killmail_id"])
		itemsByKill[id] = append(itemsByKill[id], item)
	}

	fits := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		id := int64OrZero(row["killmail_id"])
		modules, drones := advancedFitItems(itemsByKill[id])
		shipName := stringOrEmpty(row["ship_name"])
		if shipName == "" {
			shipName = "Unknown"
		}
		fits = append(fits, map[string]any{
			"killmail_id":         id,
			"killmail_hash":       row["killmail_hash"],
			"killmail_time":       row["killmail_time"],
			"victim_ship_type_id": row["ship_type_id"],
			"victim_ship_name":    shipName,
			"total_value":         float64OrZero(row["total_value"]),
			"attacker_count":      int64OrZero(row["attacker_count"]),
			"modules":             modules, "drones": drones,
			"count": int64(1),
		})
	}
	var cursor any
	if len(fits) > 0 {
		cursor = fits[len(fits)-1]["killmail_id"]
	}
	return jsonPayload(map[string]any{
		"fits": fits, "hasMore": hasMore, "cursor": cursor,
	}), nil
}

func advancedFitItems(
	rows []map[string]any,
) ([]map[string]any, []map[string]any) {
	chargeByFlag := map[int64]any{}
	for _, row := range rows {
		if int64OrZero(row["category_id"]) == 8 {
			chargeByFlag[int64OrZero(row["flag_id"])] = row["type_id"]
		}
	}
	buckets := map[int64][]map[string]any{}
	droneByType := map[int64]map[string]any{}
	for _, row := range rows {
		flag := int64OrZero(row["flag_id"])
		category := int64OrZero(row["category_id"])
		typeID := int64OrZero(row["type_id"])
		name := stringOrEmpty(row["type_name"])
		if name == "" {
			name = "Unknown"
		}
		slot := advancedSlotGroup(flag)
		if slot >= 1 && slot <= 5 && (category == 7 || category == 32) {
			buckets[slot] = append(buckets[slot], map[string]any{
				"slot_group": slot, "type_id": typeID, "name": name,
				"charge_type_id": chargeByFlag[flag],
			})
		}
		if flag == 87 && category == 18 {
			quantity := int64OrZero(row["quantity_dropped"]) +
				int64OrZero(row["quantity_destroyed"])
			if existing := droneByType[typeID]; existing != nil {
				existing["quantity"] = int64OrZero(existing["quantity"]) + quantity
			} else {
				droneByType[typeID] = map[string]any{
					"type_id": typeID, "name": name, "quantity": quantity,
				}
			}
		}
	}
	modules := []map[string]any{}
	for slot := int64(1); slot <= 5; slot++ {
		sort.Slice(buckets[slot], func(i, j int) bool {
			return int64OrZero(buckets[slot][i]["type_id"]) <
				int64OrZero(buckets[slot][j]["type_id"])
		})
		modules = append(modules, buckets[slot]...)
	}
	droneIDs := make([]int64, 0, len(droneByType))
	for id := range droneByType {
		droneIDs = append(droneIDs, id)
	}
	slices.Sort(droneIDs)
	drones := make([]map[string]any, 0, len(droneIDs))
	for _, id := range droneIDs {
		drones = append(drones, droneByType[id])
	}
	return modules, drones
}

func advancedSlotGroup(flag int64) int64 {
	switch {
	case flag >= 27 && flag <= 34:
		return 1
	case flag >= 19 && flag <= 26:
		return 2
	case flag >= 11 && flag <= 18:
		return 3
	case flag >= 92 && flag <= 99:
		return 4
	case flag >= 125 && flag <= 132:
		return 5
	case flag == 87:
		return 6
	default:
		return 0
	}
}
