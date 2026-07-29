package mcpserver

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
)

type ForensicsEvidence struct {
	AttackerCount    *int64   `json:"attacker_count,omitempty"`
	TypicalThreshold *int     `json:"typical_threshold,omitempty"`
	HullClass        *string  `json:"hull_class,omitempty"`
	CapTimeSeconds   *int64   `json:"cap_time_s,omitempty"`
	CapStable        *bool    `json:"cap_stable,omitempty"`
	ActualEHP        *int64   `json:"actual_ehp,omitempty"`
	TypicalEHP       *int64   `json:"typical_ehp,omitempty"`
	AlignTimeSeconds *float64 `json:"align_time_s,omitempty"`
	FamilyHash       *string  `json:"family_hash,omitempty"`
	TopFamilyCount   *int     `json:"top_family_count,omitempty"`
}

type ForensicsFinding struct {
	Code     string            `json:"code"`
	Severity string            `json:"severity" enum:"info,warn,critical"`
	Message  string            `json:"message"`
	Evidence ForensicsEvidence `json:"evidence"`
}

type ForensicsShip struct {
	TypeID int64   `json:"type_id"`
	Name   *string `json:"name"`
	Group  *string `json:"group"`
}

type ForensicsSystem struct {
	ID       int64    `json:"id"`
	Name     *string  `json:"name"`
	Security *float64 `json:"security"`
}

type KillmailForensicsOutput struct {
	KillmailID    int64              `json:"killmail_id"`
	URL           string             `json:"url"`
	KillTime      *time.Time         `json:"kill_time"`
	VictimShip    ForensicsShip      `json:"victim_ship"`
	System        ForensicsSystem    `json:"system"`
	AttackerCount int64              `json:"attacker_count"`
	TotalValue    float64            `json:"total_value"`
	FitHash       *string            `json:"fit_hash"`
	DogmaStats    *HullStats         `json:"dogma_stats"`
	Findings      []ForensicsFinding `json:"findings"`
	FindingCount  int                `json:"finding_count"`
}

var (
	frigateGroups       = int64Set(25, 324, 831, 893, 834, 830, 1283, 237)
	destroyerGroups     = int64Set(420, 541, 1305)
	cruiserGroups       = int64Set(26, 358, 832, 833, 894, 963, 906)
	battlecruiserGroups = int64Set(419, 540)
	battleshipGroups    = int64Set(27, 898, 900)
	capitalGroups       = int64Set(547, 485, 659, 30, 1538, 883)
	industrialGroups    = int64Set(28, 380, 513, 902, 941, 1202)
)

func registerForensicsTool(registry *Registry) error {
	return addTool(registry, ToolDefinition{
		Name: "killmail_forensics", Title: "Analyze why a ship died",
		Description: "Run a victim fit through the dogma engine and produce rule-based findings about numbers, capacitor, tank, align time, and doctrine match.",
	}, func(ctx context.Context, input KillmailInput) (KillmailForensicsOutput, error) {
		return killmailForensics(ctx, registry.deps, input)
	})
}

func killmailForensics(ctx context.Context, deps Dependencies, input KillmailInput) (KillmailForensicsOutput, error) {
	if input.KillmailID <= 0 {
		return KillmailForensicsOutput{}, fmt.Errorf("invalid killmail_id")
	}
	killmailRows, err := queryMaps(ctx, deps.DB, `
		SELECT killmail.killmail_id, killmail.killmail_time, killmail.total_value,
		       killmail.attacker_count, killmail.is_solo, killmail.victim_ship_type_id,
		       ship.name AS ship_name, ship.group_id AS ship_group_id, ship_group.name AS ship_group_name,
		       killmail.solar_system_id, system.system_name, system.security
		FROM killmails killmail LEFT JOIN inv_types ship ON ship.type_id = killmail.victim_ship_type_id
		LEFT JOIN inv_groups ship_group ON ship_group.group_id = ship.group_id
		LEFT JOIN solar_systems system ON system.solar_system_id = killmail.solar_system_id
		WHERE killmail.killmail_id = $1 LIMIT 1`, input.KillmailID)
	if err != nil {
		return KillmailForensicsOutput{}, err
	}
	killmail := firstMap(killmailRows)
	if killmail == nil {
		return KillmailForensicsOutput{}, fmt.Errorf("killmail %d not found", input.KillmailID)
	}
	fittingRows, err := queryMaps(ctx, deps.DB, `
		SELECT fitting.fit_hash, fitting.ship_type_id, fit.family_hash
		FROM killmail_fittings fitting JOIN fittings fit ON fit.fit_hash = fitting.fit_hash
		WHERE fitting.killmail_id = $1 LIMIT 1`, input.KillmailID)
	if err != nil {
		return KillmailForensicsOutput{}, err
	}
	fitting := firstMap(fittingRows)
	groupID, attackers := valueInt64(killmail["ship_group_id"]), valueInt64(killmail["attacker_count"])
	findings := []ForensicsFinding{}
	threshold := outnumberThreshold(groupID)
	if attackers >= int64(threshold) {
		hullClass := nullableString(killmail["ship_group_name"])
		findings = append(findings, ForensicsFinding{
			Code: "outnumbered", Severity: "warn",
			Message:  fmt.Sprintf("Victim faced %d attackers in a %s — beyond the typical engagement size for this hull class.", attackers, pointerString(hullClass, "ship")),
			Evidence: ForensicsEvidence{AttackerCount: &attackers, TypicalThreshold: &threshold, HullClass: hullClass},
		})
	}
	var hull *HullStats
	var fitHash *string
	if fitting == nil {
		findings = append(findings, ForensicsFinding{Code: "no_fit_available", Severity: "info", Message: "No extracted fit is available for this killmail.", Evidence: ForensicsEvidence{}})
	} else {
		hash := valueString(fitting["fit_hash"])
		fitHash = &hash
		items, loadErr := loadFittingItems(ctx, deps, hash)
		if loadErr != nil {
			return KillmailForensicsOutput{}, loadErr
		}
		stats, evaluateErr := evaluateDogma(ctx, fittingItemsToEsf(valueInt64(fitting["ship_type_id"]), items), false)
		if evaluateErr != nil {
			findings = append(findings, ForensicsFinding{Code: "dogma_unavailable", Severity: "info", Message: "Could not compute fit stats: " + evaluateErr.Error(), Evidence: ForensicsEvidence{}})
		} else {
			hull = &stats
			findings = append(findings, dogmaForensicsFindings(stats, groupID, nullableString(killmail["ship_group_name"]))...)
		}
		doctrineFinding, doctrineErr := doctrineMatchFinding(ctx, deps, fitting, killmail)
		if doctrineErr != nil {
			return KillmailForensicsOutput{}, doctrineErr
		}
		if doctrineFinding != nil {
			findings = append(findings, *doctrineFinding)
		}
	}
	if valueBool(killmail["is_solo"]) && attackers == 1 {
		findings = append(findings, ForensicsFinding{Code: "solo_loss", Severity: "info", Message: "Lost to a solo attacker — a 1v1 outplay or a poor engagement choice.", Evidence: ForensicsEvidence{}})
	}
	return KillmailForensicsOutput{
		KillmailID: input.KillmailID, URL: killmailURL(deps.BaseURL, input.KillmailID),
		KillTime:      nullableTime(killmail["killmail_time"]),
		VictimShip:    ForensicsShip{TypeID: valueInt64(killmail["victim_ship_type_id"]), Name: nullableString(killmail["ship_name"]), Group: nullableString(killmail["ship_group_name"])},
		System:        ForensicsSystem{ID: valueInt64(killmail["solar_system_id"]), Name: nullableString(killmail["system_name"]), Security: nullableFloat64(killmail["security"])},
		AttackerCount: attackers, TotalValue: valueFloat64(killmail["total_value"]), FitHash: fitHash,
		DogmaStats: hull, Findings: findings, FindingCount: len(findings),
	}, nil
}

func dogmaForensicsFindings(hull HullStats, groupID int64, groupName *string) []ForensicsFinding {
	output := []ForensicsFinding{}
	if hull.CapDepletesIn != nil && *hull.CapDepletesIn > 0 && *hull.CapDepletesIn < 120 {
		seconds := int64(math.Round(*hull.CapDepletesIn))
		stable := false
		severity := "warn"
		if seconds < 60 {
			severity = "critical"
		}
		output = append(output, ForensicsFinding{Code: "cap_too_small", Severity: severity, Message: fmt.Sprintf("Fit runs out of capacitor in %ds with all modules running.", seconds), Evidence: ForensicsEvidence{CapTimeSeconds: &seconds, CapStable: &stable}})
	}
	expected := expectedEHP(groupID)
	if hull.EHP != nil && expected > 0 && *hull.EHP < float64(expected)*0.6 {
		actual := int64(math.Round(*hull.EHP))
		output = append(output, ForensicsFinding{Code: "under_tanked", Severity: "warn", Message: fmt.Sprintf("Fit has only %d EHP, well below the typical %d EHP for this hull class.", actual, expected), Evidence: ForensicsEvidence{ActualEHP: &actual, TypicalEHP: &expected}})
	}
	if hull.AlignTime != nil && *hull.AlignTime > 10 && groupName != nil {
		name := strings.ToLower(*groupName)
		if strings.Contains(name, "industrial") || strings.Contains(name, "freighter") || strings.Contains(name, "hauler") || strings.Contains(name, "transport") {
			align := math.Round(*hull.AlignTime*10) / 10
			output = append(output, ForensicsFinding{Code: "slow_align", Severity: "info", Message: fmt.Sprintf("%.1fs align time — the victim could not warp quickly.", align), Evidence: ForensicsEvidence{AlignTimeSeconds: &align}})
		}
	}
	return output
}

func doctrineMatchFinding(ctx context.Context, deps Dependencies, fitting, killmail map[string]any) (*ForensicsFinding, error) {
	killTime := nullableTime(killmail["killmail_time"])
	if killTime == nil {
		return nil, nil
	}
	rows, err := queryMaps(ctx, deps.DB, `
		SELECT fit.family_hash, COUNT(*)::bigint AS count
		FROM killmail_fittings fitting JOIN fittings fit ON fit.fit_hash = fitting.fit_hash
		WHERE fitting.ship_type_id = $1 AND fitting.kill_time >= $2 AND fitting.kill_time < $3
		GROUP BY fit.family_hash ORDER BY count DESC LIMIT 10`,
		valueInt64(fitting["ship_type_id"]), killTime.Add(-30*24*time.Hour), *killTime)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	family := valueString(fitting["family_hash"])
	for _, row := range rows {
		if valueString(row["family_hash"]) == family {
			return &ForensicsFinding{Code: "doctrine_match", Severity: "info", Message: fmt.Sprintf("Fit matches a top-10 doctrine for %s in the preceding 30 days.", pointerString(nullableString(killmail["ship_name"]), "this hull")), Evidence: ForensicsEvidence{FamilyHash: &family}}, nil
		}
	}
	count := len(rows)
	return &ForensicsFinding{Code: "non_meta_fit", Severity: "info", Message: fmt.Sprintf("Fit is not in the top-10 doctrines for %s.", pointerString(nullableString(killmail["ship_name"]), "this hull")), Evidence: ForensicsEvidence{FamilyHash: &family, TopFamilyCount: &count}}, nil
}

func outnumberThreshold(groupID int64) int {
	switch {
	case frigateGroups[groupID]:
		return 6
	case destroyerGroups[groupID]:
		return 8
	case cruiserGroups[groupID]:
		return 12
	case battlecruiserGroups[groupID]:
		return 16
	case battleshipGroups[groupID]:
		return 20
	case capitalGroups[groupID]:
		return 50
	case industrialGroups[groupID]:
		return 3
	default:
		return 10
	}
}

func expectedEHP(groupID int64) int64 {
	switch {
	case frigateGroups[groupID]:
		return 8_000
	case destroyerGroups[groupID]:
		return 12_000
	case cruiserGroups[groupID]:
		return 35_000
	case battlecruiserGroups[groupID]:
		return 70_000
	case battleshipGroups[groupID]:
		return 130_000
	case capitalGroups[groupID]:
		return 1_000_000
	default:
		return 0
	}
}

func int64Set(values ...int64) map[int64]bool {
	output := make(map[int64]bool, len(values))
	for _, value := range values {
		output[value] = true
	}
	return output
}

func pointerString(value *string, fallback string) string {
	if value == nil || *value == "" {
		return fallback
	}
	return *value
}
