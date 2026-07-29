package mcpserver

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
)

type DoctrineDetectInput struct {
	Entity             StringOrInt64 `json:"entity"`
	Type               *EntityType   `json:"type,omitempty" enum:"character,corporation,alliance"`
	Since              *string       `json:"since,omitempty"`
	Until              *string       `json:"until,omitempty"`
	MinClusterSize     int           `json:"min_cluster_size,omitempty" default:"5" minimum:"2"`
	IncludeRookieShips bool          `json:"include_rookie_ships,omitempty" default:"false"`
	Limit              int           `json:"limit,omitempty" default:"10" minimum:"1" maximum:"30"`
}

type DoctrineShip struct {
	TypeID int64   `json:"type_id"`
	Name   *string `json:"name"`
	Group  *string `json:"group,omitempty"`
}

type DoctrineExample struct {
	KillmailID int64    `json:"killmail_id"`
	URL        string   `json:"url"`
	Modules    []string `json:"modules,omitempty"`
}

type DoctrineCluster struct {
	FamilyHash     string          `json:"family_hash"`
	Ship           DoctrineShip    `json:"ship"`
	Signature      string          `json:"signature,omitempty"`
	Losses         int64           `json:"losses"`
	ISKLost        float64         `json:"isk_lost"`
	AverageISKLoss float64         `json:"avg_isk_per_loss"`
	Example        DoctrineExample `json:"example_killmail"`
	FirstLoss      *time.Time      `json:"first_loss,omitempty"`
	LastLoss       *time.Time      `json:"last_loss,omitempty"`
}

type DoctrineDetectOutput struct {
	Entity   Entity            `json:"entity"`
	Window   TimeWindow        `json:"window"`
	Count    int               `json:"count"`
	Clusters []DoctrineCluster `json:"clusters"`
	Notes    string            `json:"notes,omitempty"`
}

type MetaPulseInput struct {
	RegionID           *int64  `json:"region_id,omitempty"`
	ShipCategory       string  `json:"ship_category,omitempty" enum:"all,frigate,destroyer,cruiser,battlecruiser,battleship,capital,supercap,subcap" default:"all"`
	Since              *string `json:"since,omitempty"`
	Until              *string `json:"until,omitempty"`
	MinClusterSize     int     `json:"min_cluster_size,omitempty" default:"10" minimum:"1"`
	IncludeRookieShips bool    `json:"include_rookie_ships,omitempty" default:"false"`
	Limit              int     `json:"limit,omitempty" default:"15" minimum:"1" maximum:"30"`
}

type MetaPulseOutput struct {
	Window       TimeWindow        `json:"window"`
	RegionID     *int64            `json:"region_id"`
	ShipCategory string            `json:"ship_category"`
	Count        int               `json:"count"`
	Clusters     []DoctrineCluster `json:"clusters"`
}

type doctrineItem struct {
	slotGroup           int
	typeID              int64
	typeName, groupName *string
}

var shipCategoryGroups = map[string][]int64{
	"all": {}, "frigate": {25, 324, 831, 893, 834, 830, 1283},
	"destroyer": {420, 541, 1305}, "cruiser": {26, 358, 832, 833, 894, 963, 906},
	"battlecruiser": {419, 540}, "battleship": {27, 898, 900},
	"capital": {547, 485, 659, 30, 1538, 883, 4594}, "supercap": {30, 659},
	"subcap": {25, 420, 26, 419, 27, 324, 831, 893, 834, 830, 541, 358, 832, 833, 894, 540, 898, 1283, 1305},
}

func registerDoctrineTools(registry *Registry) error {
	if err := addTool(registry, ToolDefinition{
		Name: "doctrine_detect", Title: "Detect fit doctrines",
		Description: "Identify dominant victim-fit families for a character, corporation, or alliance from losses in a time window.",
	}, func(ctx context.Context, input DoctrineDetectInput) (DoctrineDetectOutput, error) {
		return doctrineDetect(ctx, registry.deps, input)
	}); err != nil {
		return err
	}
	return addTool(registry, ToolDefinition{
		Name: "meta_pulse", Title: "Scan the current fitting meta",
		Description: "Find the most frequently lost fit families globally or in a region, optionally filtered by hull category.",
	}, func(ctx context.Context, input MetaPulseInput) (MetaPulseOutput, error) {
		return metaPulse(ctx, registry.deps, input)
	})
}

func doctrineDetect(ctx context.Context, deps Dependencies, input DoctrineDetectInput) (DoctrineDetectOutput, error) {
	entity, err := resolveEntity(ctx, deps, input.Entity, input.Type)
	if err != nil {
		return DoctrineDetectOutput{}, err
	}
	if entity == nil || organizationVictimColumns[entity.Type] == "" {
		return DoctrineDetectOutput{}, fmt.Errorf("entity must be a character, corporation, or alliance")
	}
	since, until, err := parseVSWindow(input.Since, input.Until, 30)
	if err != nil {
		return DoctrineDetectOutput{}, err
	}
	minimum, limit := input.MinClusterSize, input.Limit
	if minimum == 0 {
		minimum = 5
	}
	if limit == 0 {
		limit = 10
	}
	rookieFilter := "AND ship.group_id <> 237"
	if input.IncludeRookieShips {
		rookieFilter = ""
	}
	entityFilter := "fitting." + organizationVictimColumns[entity.Type] + " = $1"
	if entity.Type == EntityCharacter {
		entityFilter = "fitting.killmail_id IN (SELECT killmail_id FROM killmails WHERE victim_character_id = $1 AND killmail_time >= $2 AND killmail_time <= $3)"
	}
	rows, err := queryMaps(ctx, deps.DB, fmt.Sprintf(`
		SELECT fit.family_hash, fit.ship_type_id, ship.name AS ship_name, COUNT(*)::bigint AS losses,
		       COALESCE(SUM(killmail.total_value), 0)::double precision AS isk_lost,
		       MAX(fitting.kill_time) AS latest_loss, MIN(fitting.kill_time) AS earliest_loss,
		       (ARRAY_AGG(fitting.killmail_id ORDER BY fitting.kill_time DESC))[1] AS example_killmail_id
		FROM killmail_fittings fitting JOIN fittings fit ON fit.fit_hash = fitting.fit_hash
		JOIN killmails killmail ON killmail.killmail_id = fitting.killmail_id
		LEFT JOIN inv_types ship ON ship.type_id = fit.ship_type_id
		WHERE %s AND fitting.kill_time >= $2 AND fitting.kill_time <= $3 %s
		GROUP BY fit.family_hash, fit.ship_type_id, ship.name
		HAVING COUNT(*) >= $4 ORDER BY losses DESC LIMIT $5`, entityFilter, rookieFilter),
		entity.ID, since, until, minimum, clamp(limit, 1, 30))
	if err != nil {
		return DoctrineDetectOutput{}, err
	}
	output := DoctrineDetectOutput{
		Entity: entity.Public(deps.BaseURL),
		Window: TimeWindow{Since: since.Format(time.RFC3339Nano), Until: until.Format(time.RFC3339Nano)},
		Count:  len(rows), Clusters: []DoctrineCluster{},
	}
	if len(rows) == 0 {
		output.Notes = fmt.Sprintf("No doctrine clusters found (min_cluster_size=%d). Try lowering min_cluster_size or widening the window.", minimum)
		return output, nil
	}
	exampleIDs := make([]int64, 0, len(rows))
	for _, row := range rows {
		exampleIDs = append(exampleIDs, valueInt64(row["example_killmail_id"]))
	}
	items, err := loadDoctrineItems(ctx, deps, exampleIDs)
	if err != nil {
		return DoctrineDetectOutput{}, err
	}
	for _, row := range rows {
		exampleID, losses := valueInt64(row["example_killmail_id"]), valueInt64(row["losses"])
		isk := valueFloat64(row["isk_lost"])
		shipName := nullableString(row["ship_name"])
		output.Clusters = append(output.Clusters, DoctrineCluster{
			FamilyHash: valueString(row["family_hash"]), Ship: DoctrineShip{TypeID: valueInt64(row["ship_type_id"]), Name: shipName},
			Signature: buildDoctrineSignature(shipName, items[exampleID]), Losses: losses, ISKLost: isk,
			AverageISKLoss: math.Round(isk / math.Max(1, float64(losses))),
			Example:        DoctrineExample{KillmailID: exampleID, URL: killmailURL(deps.BaseURL, exampleID), Modules: renderDoctrineItems(items[exampleID])},
			FirstLoss:      nullableTime(row["earliest_loss"]), LastLoss: nullableTime(row["latest_loss"]),
		})
	}
	return output, nil
}

func metaPulse(ctx context.Context, deps Dependencies, input MetaPulseInput) (MetaPulseOutput, error) {
	since, until, err := parseVSWindow(input.Since, input.Until, 7)
	if err != nil {
		return MetaPulseOutput{}, err
	}
	category := input.ShipCategory
	if category == "" {
		category = "all"
	}
	groups, ok := shipCategoryGroups[category]
	if !ok {
		return MetaPulseOutput{}, fmt.Errorf("invalid ship_category")
	}
	minimum, limit := input.MinClusterSize, input.Limit
	if minimum == 0 {
		minimum = 10
	}
	if limit == 0 {
		limit = 15
	}
	filters, args := []string{"fitting.kill_time >= $1", "fitting.kill_time <= $2"}, []any{since, until}
	if input.RegionID != nil {
		args = append(args, *input.RegionID)
		filters = append(filters, fmt.Sprintf("killmail.region_id = $%d", len(args)))
	}
	if len(groups) > 0 {
		args = append(args, groups)
		filters = append(filters, fmt.Sprintf("ship.group_id = ANY($%d)", len(args)))
	}
	if !input.IncludeRookieShips {
		filters = append(filters, "ship.group_id <> 237")
	}
	args = append(args, minimum)
	minimumParameter := len(args)
	args = append(args, clamp(limit, 1, 30))
	rows, err := queryMaps(ctx, deps.DB, fmt.Sprintf(`
		SELECT fit.family_hash, fit.ship_type_id, ship.name AS ship_name, ship_group.name AS group_name,
		       COUNT(*)::bigint AS losses, COALESCE(SUM(killmail.total_value), 0)::double precision AS isk_lost,
		       (ARRAY_AGG(fitting.killmail_id ORDER BY fitting.kill_time DESC))[1] AS example_killmail_id
		FROM killmail_fittings fitting JOIN fittings fit ON fit.fit_hash = fitting.fit_hash
		JOIN killmails killmail ON killmail.killmail_id = fitting.killmail_id
		LEFT JOIN inv_types ship ON ship.type_id = fit.ship_type_id
		LEFT JOIN inv_groups ship_group ON ship_group.group_id = ship.group_id
		WHERE %s GROUP BY fit.family_hash, fit.ship_type_id, ship.name, ship_group.name
		HAVING COUNT(*) >= $%d ORDER BY losses DESC LIMIT $%d`,
		strings.Join(filters, " AND "), minimumParameter, len(args)), args...)
	if err != nil {
		return MetaPulseOutput{}, err
	}
	output := MetaPulseOutput{
		Window:   TimeWindow{Since: since.Format(time.RFC3339Nano), Until: until.Format(time.RFC3339Nano)},
		RegionID: input.RegionID, ShipCategory: category, Count: len(rows), Clusters: []DoctrineCluster{},
	}
	for _, row := range rows {
		exampleID, losses, isk := valueInt64(row["example_killmail_id"]), valueInt64(row["losses"]), valueFloat64(row["isk_lost"])
		output.Clusters = append(output.Clusters, DoctrineCluster{
			FamilyHash: valueString(row["family_hash"]),
			Ship:       DoctrineShip{TypeID: valueInt64(row["ship_type_id"]), Name: nullableString(row["ship_name"]), Group: nullableString(row["group_name"])},
			Losses:     losses, ISKLost: isk, AverageISKLoss: math.Round(isk / math.Max(1, float64(losses))),
			Example: DoctrineExample{KillmailID: exampleID, URL: killmailURL(deps.BaseURL, exampleID)},
		})
	}
	return output, nil
}

func loadDoctrineItems(ctx context.Context, deps Dependencies, killmailIDs []int64) (map[int64][]doctrineItem, error) {
	rows, err := queryMaps(ctx, deps.DB, `
		SELECT fitting.killmail_id, item.slot_group, item.ordinal, item.type_id,
		       type.name AS type_name, item_group.name AS group_name
		FROM killmail_fittings fitting JOIN fitting_items item ON item.fit_hash = fitting.fit_hash
		LEFT JOIN inv_types type ON type.type_id = item.type_id
		LEFT JOIN inv_groups item_group ON item_group.group_id = type.group_id
		WHERE fitting.killmail_id = ANY($1) ORDER BY fitting.killmail_id, item.slot_group, item.ordinal`, killmailIDs)
	if err != nil {
		return nil, err
	}
	output := map[int64][]doctrineItem{}
	for _, row := range rows {
		id := valueInt64(row["killmail_id"])
		output[id] = append(output[id], doctrineItem{slotGroup: int(valueInt64(row["slot_group"])), typeID: valueInt64(row["type_id"]), typeName: nullableString(row["type_name"]), groupName: nullableString(row["group_name"])})
	}
	return output, nil
}

func buildDoctrineSignature(shipName *string, items []doctrineItem) string {
	parts := []string{"Unknown"}
	if shipName != nil {
		parts[0] = *shipName
	}
	if weapon := detectDoctrineWeapon(items); weapon != "" {
		parts = append(parts, weapon)
	}
	if tank := detectDoctrineTank(items); tank != "" {
		parts = append(parts, tank)
	}
	if propulsion := detectDoctrinePropulsion(items); propulsion != "" {
		parts = append(parts, propulsion)
	}
	return strings.Join(parts, " / ")
}

func detectDoctrineTank(items []doctrineItem) string {
	groups := doctrineGroupNames(items)
	switch {
	case strings.Contains(groups, "shield extender"):
		return "Shield Buffer"
	case strings.Contains(groups, "shield booster"):
		return "Shield Active"
	case strings.Contains(groups, "armor plating") || strings.Contains(groups, "armor plate"):
		return "Armor Buffer"
	case strings.Contains(groups, "armor repair"):
		return "Armor Active"
	}
	return ""
}

func detectDoctrinePropulsion(items []doctrineItem) string {
	groups := doctrineGroupNames(items)
	if strings.Contains(groups, "microwarp") {
		return "MWD"
	}
	if strings.Contains(groups, "afterburner") {
		return "AB"
	}
	if strings.Contains(groups, "micro jump drive") {
		return "MJD"
	}
	return ""
}

func detectDoctrineWeapon(items []doctrineItem) string {
	for _, item := range items {
		if item.slotGroup != 1 || item.groupName == nil {
			continue
		}
		group := strings.ToLower(*item.groupName)
		switch {
		case strings.Contains(group, "energy weapon"):
			return "Laser"
		case strings.Contains(group, "hybrid weapon") && strings.Contains(group, "rail"):
			return "Rail"
		case strings.Contains(group, "hybrid weapon"):
			return "Blaster"
		case strings.Contains(group, "projectile weapon") && strings.Contains(group, "artillery"):
			return "Artillery"
		case strings.Contains(group, "projectile weapon"):
			return "Autocannon"
		case strings.Contains(group, "missile launcher") && strings.Contains(group, "heavy assault"):
			return "HAM"
		case strings.Contains(group, "missile launcher") && strings.Contains(group, "light"):
			return "LML"
		case strings.Contains(group, "missile launcher") && strings.Contains(group, "rocket"):
			return "Rocket"
		case strings.Contains(group, "missile launcher") && strings.Contains(group, "torpedo"):
			return "Torp"
		case strings.Contains(group, "missile launcher") && strings.Contains(group, "cruise"):
			return "Cruise"
		case strings.Contains(group, "missile launcher") && strings.Contains(group, "rapid"):
			return "RLML"
		case strings.Contains(group, "missile launcher"):
			return "Missile"
		}
	}
	return ""
}

func doctrineGroupNames(items []doctrineItem) string {
	values := make([]string, 0, len(items))
	for _, item := range items {
		if item.groupName != nil {
			values = append(values, strings.ToLower(*item.groupName))
		}
	}
	return strings.Join(values, " ")
}

func renderDoctrineItems(items []doctrineItem) []string {
	labels := map[int]string{1: "high", 2: "med", 3: "low", 4: "rig", 5: "sub", 6: "drone"}
	output := []string{}
	for _, item := range items {
		label, ok := labels[item.slotGroup]
		if !ok {
			continue
		}
		name := fmt.Sprintf("TypeID_%d", item.typeID)
		if item.typeName != nil {
			name = *item.typeName
		}
		output = append(output, fmt.Sprintf("[%s] %s", label, name))
	}
	return output
}
