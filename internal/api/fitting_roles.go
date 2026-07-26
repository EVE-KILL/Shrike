package api

import (
	"context"
	"strings"
)

type fittingRoleDefinition struct {
	ID, Label, Icon, Description, Category string
	GroupIDs                               []int32
	NameIncludes, NameExcludes             []string
	SlotGroups                             []int16
}

var fittingRoleDefinitions = []fittingRoleDefinition{
	{"armor-rep", "Armor Repairer", "lucide:wrench", "Local armor repair modules (any size/meta).", "tank", []int32{62}, nil, nil, []int16{3}},
	{"shield-booster", "Shield Booster", "lucide:shield", "Local shield boosters (any size/meta).", "tank", []int32{40}, nil, nil, []int16{2}},
	{"damage-control", "Damage Control", "lucide:shield-check", "", "tank", []int32{60}, nil, nil, []int16{3}},
	{"shield-extender", "Shield Extender", "lucide:shield-plus", "", "tank", []int32{295}, nil, nil, []int16{2}},
	{"armor-plate", "Armor Plate", "lucide:square-stack", "", "tank", []int32{329}, nil, nil, []int16{3}},
	{"remote-armor-rep", "Remote Armor Rep", "lucide:heart-handshake", "Remote armor repairers (logi).", "tank", []int32{325}, nil, nil, []int16{1}},
	{"remote-shield-booster", "Remote Shield Booster", "lucide:heart-handshake", "", "tank", []int32{41}, nil, nil, []int16{1}},

	{"pulse-laser", "Pulse Laser", "lucide:zap", "", "weapon", []int32{53}, []string{"pulse laser"}, nil, []int16{1}},
	{"beam-laser", "Beam Laser", "lucide:zap", "", "weapon", []int32{53}, []string{"beam laser"}, nil, []int16{1}},
	{"energy-weapon", "Energy Weapon (any)", "lucide:zap", "Any pulse or beam laser turret.", "weapon", []int32{53}, nil, nil, []int16{1}},
	{"blaster", "Blaster", "lucide:flame", "", "weapon", []int32{74}, []string{"blaster"}, nil, []int16{1}},
	{"railgun", "Railgun", "lucide:crosshair", "", "weapon", []int32{74}, []string{"railgun"}, nil, []int16{1}},
	{"hybrid-weapon", "Hybrid Weapon (any)", "lucide:crosshair", "Any blaster or railgun turret.", "weapon", []int32{74}, nil, nil, []int16{1}},
	{"autocannon", "Autocannon", "lucide:rotate-cw", "", "weapon", []int32{55}, []string{"autocannon"}, nil, []int16{1}},
	{"artillery", "Artillery", "lucide:crosshair", "", "weapon", []int32{55}, []string{"artillery"}, nil, []int16{1}},
	{"projectile-weapon", "Projectile Weapon (any)", "lucide:crosshair", "Any autocannon or artillery turret.", "weapon", []int32{55}, nil, nil, []int16{1}},
	{"rocket-launcher", "Rocket Launcher", "lucide:rocket", "", "weapon", []int32{507}, nil, nil, []int16{1}},
	{"light-missile", "Light Missile Launcher", "lucide:rocket", "", "weapon", []int32{509}, nil, nil, []int16{1}},
	{"heavy-assault-missile", "Heavy Assault Missile (HAM)", "lucide:rocket", "", "weapon", []int32{510}, nil, nil, []int16{1}},
	{"heavy-missile", "Heavy Missile Launcher", "lucide:rocket", "", "weapon", []int32{506}, nil, nil, []int16{1}},
	{"torpedo-launcher", "Torpedo Launcher", "lucide:rocket", "", "weapon", []int32{511}, nil, nil, []int16{1}},
	{"cruise-missile", "Cruise Missile Launcher", "lucide:rocket", "", "weapon", []int32{508}, nil, nil, []int16{1}},
	{"rapid-light-missile", "Rapid Light Missile", "lucide:rocket", "", "weapon", []int32{771}, nil, nil, []int16{1}},
	{"missile-launcher", "Missile Launcher (any)", "lucide:rocket", "Any missile launcher regardless of size.", "weapon", []int32{506, 507, 508, 509, 510, 511, 524, 771, 1419, 1418, 1420, 1422}, nil, nil, []int16{1}},

	{"mwd", "Microwarpdrive", "lucide:zap", "", "prop", []int32{46}, []string{"microwarpdrive", "10mn afterburner ii"}, []string{"afterburner"}, []int16{2}},
	{"afterburner", "Afterburner", "lucide:wind", "", "prop", []int32{46}, []string{"afterburner"}, nil, []int16{2}},
	{"propmod", "Propulsion Module (any)", "lucide:wind", "Any MWD or AB.", "prop", []int32{46}, nil, nil, []int16{2}},
	{"warp-scrambler", "Warp Scrambler", "lucide:circle-slash", "Warp Scrambler (2pt) — shuts MWDs off.", "tackle", []int32{52}, []string{"scrambler"}, nil, []int16{2}},
	{"warp-disruptor", "Warp Disruptor (Point)", "lucide:circle-dot", "Warp Disruptor (1pt long-range).", "tackle", []int32{52}, []string{"disruptor"}, nil, []int16{2}},
	{"stasis-web", "Stasis Webifier", "lucide:spider", "", "tackle", []int32{65}, nil, nil, []int16{2}},

	{"ecm", "ECM", "lucide:waves", "Electronic Counter Measures jammers.", "ewar", []int32{201}, nil, nil, []int16{2}},
	{"sensor-damp", "Sensor Dampener", "lucide:eye-off", "", "ewar", []int32{208}, nil, nil, []int16{2}},
	{"tracking-disruptor", "Tracking Disruptor", "lucide:crosshair", "", "ewar", []int32{291}, nil, nil, []int16{2}},
	{"target-painter", "Target Painter", "lucide:target", "", "ewar", []int32{209}, nil, nil, []int16{2}},
	{"energy-neutralizer", "Energy Neutralizer (Neut)", "lucide:battery-low", "", "ewar", []int32{71}, nil, nil, []int16{1}},
	{"energy-nosferatu", "Nosferatu (NOS)", "lucide:battery", "", "ewar", []int32{70}, nil, nil, []int16{1}},
	{"smartbomb", "Smartbomb", "lucide:bomb", "", "ewar", []int32{72}, nil, nil, []int16{1}},

	{"cap-booster", "Capacitor Booster", "lucide:battery-charging", "", "utility", []int32{76}, []string{"cap booster", "capacitor booster"}, nil, []int16{2}},
	{"cloak", "Cloaking Device", "lucide:eye-off", "", "utility", []int32{330}, nil, nil, []int16{1}},
	{"mjd", "Micro Jump Drive", "lucide:zap", "", "utility", []int32{1404}, nil, nil, []int16{2}},
	{"probe-launcher", "Probe Launcher", "lucide:radar", "", "utility", []int32{481}, nil, nil, []int16{1}},
}

type resolvedFittingRoles map[string][]int32

func resolveFittingRoles(
	ctx context.Context,
	db Database,
) (resolvedFittingRoles, error) {
	groupSet := make(map[int32]bool)
	for _, role := range fittingRoleDefinitions {
		for _, groupID := range role.GroupIDs {
			groupSet[groupID] = true
		}
	}
	groupIDs := make([]int32, 0, len(groupSet))
	for groupID := range groupSet {
		groupIDs = append(groupIDs, groupID)
	}
	rows, err := queryMaps(ctx, db, `
		SELECT type_id, name, group_id
		FROM inv_types
		WHERE group_id = ANY($1::int[])
		  AND published = TRUE`, groupIDs)
	if err != nil {
		return nil, err
	}
	byGroup := make(map[int64][]map[string]any)
	for _, row := range rows {
		groupID := int64OrZero(row["group_id"])
		byGroup[groupID] = append(byGroup[groupID], row)
	}
	resolved := make(resolvedFittingRoles, len(fittingRoleDefinitions))
	for _, role := range fittingRoleDefinitions {
		ids := make([]int32, 0)
		for _, groupID := range role.GroupIDs {
			for _, row := range byGroup[int64(groupID)] {
				name := strings.ToLower(stringOrEmpty(row["name"]))
				if len(role.NameIncludes) > 0 &&
					!containsAnySubstring(name, role.NameIncludes) {
					continue
				}
				if containsAnySubstring(name, role.NameExcludes) {
					continue
				}
				ids = append(ids, int32(int64OrZero(row["type_id"])))
			}
		}
		resolved[role.ID] = ids
	}
	return resolved, nil
}

func containsAnySubstring(text string, parts []string) bool {
	for _, part := range parts {
		if strings.Contains(text, part) {
			return true
		}
	}
	return false
}

func fittingRolesHandler(opts Options) legacyHandler {
	return func(ctx context.Context, _ *legacyRequest) (legacyPayload, error) {
		resolved, err := resolveFittingRoles(ctx, opts.DB)
		if err != nil {
			return legacyPayload{}, err
		}
		roles := make([]map[string]any, 0, len(fittingRoleDefinitions))
		for _, role := range fittingRoleDefinitions {
			row := map[string]any{
				"id": role.ID, "label": role.Label, "icon": role.Icon,
				"category": role.Category, "typeCount": len(resolved[role.ID]),
			}
			// JSON.stringify omits an undefined optional property. Preserve that
			// distinction rather than returning an empty description.
			if role.Description != "" {
				row["description"] = role.Description
			}
			roles = append(roles, row)
		}
		return jsonPayload(map[string]any{"roles": roles}), nil
	}
}

func fittingRoleByID(id string) (fittingRoleDefinition, bool) {
	for _, role := range fittingRoleDefinitions {
		if role.ID == id {
			return role, true
		}
	}
	return fittingRoleDefinition{}, false
}
