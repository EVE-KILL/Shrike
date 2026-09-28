package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
)

const (
	roamMaxPilots     = 256
	roamMaxNameLength = 64
	roamMaxInputBytes = 16 << 10
	roamMaxWindow     = 72 * time.Hour
	roamMaxKillmails  = 2000
	roamEngagementGap = 30 * time.Minute
)

type roamReportCreateBody struct {
	NamesText string    `json:"names_text" maxLength:"16384" doc:"Character names copied from the in-game fleet window, one per line."`
	StartTime time.Time `json:"start_time" doc:"Start of the roam in EVE time (UTC)."`
	EndTime   time.Time `json:"end_time" doc:"End of the roam in EVE time (UTC)."`
}

type roamPilot struct {
	CharacterID        int32      `json:"character_id"`
	Name               string     `json:"name"`
	CorporationID      int32      `json:"corporation_id"`
	CorporationName    string     `json:"corporation_name"`
	CorporationTicker  string     `json:"corporation_ticker"`
	AllianceID         int32      `json:"alliance_id"`
	AllianceName       string     `json:"alliance_name"`
	AllianceTicker     string     `json:"alliance_ticker"`
	KillParticipations int        `json:"kill_participations"`
	FinalBlows         int        `json:"final_blows"`
	Losses             int        `json:"losses"`
	Engagements        int        `json:"engagements"`
	DamageDone         int64      `json:"damage_done"`
	Ships              []roamShip `json:"ships"`
}

type roamShip struct {
	ShipTypeID    int32  `json:"ship_type_id"`
	ShipName      string `json:"ship_name"`
	ShipGroupID   int32  `json:"ship_group_id"`
	ShipGroupName string `json:"ship_group_name"`
	Killmails     int    `json:"killmails"`
	Losses        int    `json:"losses"`
	DamageDone    int64  `json:"damage_done"`
}

type roamKillmail struct {
	KillmailID             int32     `json:"killmail_id"`
	KillmailTime           time.Time `json:"killmail_time"`
	SolarSystemID          int32     `json:"solar_system_id"`
	SolarSystemName        string    `json:"solar_system_name"`
	RegionID               int32     `json:"region_id"`
	RegionName             string    `json:"region_name"`
	VictimCharacterID      int32     `json:"victim_character_id"`
	VictimName             string    `json:"victim_name"`
	VictimCorporationID    int32     `json:"victim_corporation_id"`
	VictimCorporationName  string    `json:"victim_corporation_name"`
	VictimAllianceID       int32     `json:"victim_alliance_id"`
	VictimAllianceName     string    `json:"victim_alliance_name"`
	VictimShipTypeID       int32     `json:"victim_ship_type_id"`
	VictimShipName         string    `json:"victim_ship_name"`
	VictimShipGroupID      int32     `json:"victim_ship_group_id"`
	VictimShipGroupName    string    `json:"victim_ship_group_name"`
	TotalValue             float64   `json:"total_value"`
	AttackerCount          int32     `json:"attacker_count"`
	Role                   string    `json:"role"`
	FriendlyFire           bool      `json:"friendly_fire"`
	FleetAttackerIDs       []int32   `json:"fleet_attacker_ids"`
	FleetFinalBlow         bool      `json:"fleet_final_blow"`
	FinalBlowCharacterName string    `json:"final_blow_character_name"`
	FinalBlowCorpName      string    `json:"final_blow_corporation_name"`
}

type roamAttacker struct {
	KillmailID    int32
	CharacterID   int32
	DamageDone    int64
	FinalBlow     bool
	ShipTypeID    int32
	ShipName      string
	ShipGroupID   int32
	ShipGroupName string
}

type roamFinalBlow struct {
	KillmailID      int32
	CharacterName   string
	CorporationName string
}

type roamEngagement struct {
	Number          int            `json:"number"`
	SolarSystemID   int32          `json:"solar_system_id"`
	SolarSystemName string         `json:"solar_system_name"`
	RegionID        int32          `json:"region_id"`
	RegionName      string         `json:"region_name"`
	StartTime       time.Time      `json:"start_time"`
	EndTime         time.Time      `json:"end_time"`
	Kills           int            `json:"kills"`
	Losses          int            `json:"losses"`
	IskDestroyed    float64        `json:"isk_destroyed"`
	IskLost         float64        `json:"isk_lost"`
	PilotIDs        []int32        `json:"pilot_ids"`
	Killmails       []roamKillmail `json:"killmails"`
}

type roamSystem struct {
	SolarSystemID   int32     `json:"solar_system_id"`
	SolarSystemName string    `json:"solar_system_name"`
	RegionID        int32     `json:"region_id"`
	RegionName      string    `json:"region_name"`
	FirstSeen       time.Time `json:"first_seen"`
	LastSeen        time.Time `json:"last_seen"`
	Engagements     int       `json:"engagements"`
	Kills           int       `json:"kills"`
	Losses          int       `json:"losses"`
	IskDestroyed    float64   `json:"isk_destroyed"`
	IskLost         float64   `json:"isk_lost"`
}

type roamTarget struct {
	CorporationID   int32   `json:"corporation_id"`
	CorporationName string  `json:"corporation_name"`
	AllianceID      int32   `json:"alliance_id"`
	AllianceName    string  `json:"alliance_name"`
	Kills           int     `json:"kills"`
	IskDestroyed    float64 `json:"isk_destroyed"`
}

type roamSummary struct {
	Kills                  int     `json:"kills"`
	Losses                 int     `json:"losses"`
	FriendlyFireLosses     int     `json:"friendly_fire_losses"`
	IskDestroyed           float64 `json:"isk_destroyed"`
	IskLost                float64 `json:"isk_lost"`
	Efficiency             float64 `json:"efficiency"`
	Engagements            int     `json:"engagements"`
	Systems                int     `json:"systems"`
	Regions                int     `json:"regions"`
	ActivePilots           int     `json:"active_pilots"`
	CoordinatedKills       int     `json:"coordinated_kills"`
	UniqueCharacterTargets int     `json:"unique_character_targets"`
}

type roamReport struct {
	ID          string           `json:"id"`
	CreatedAt   time.Time        `json:"created_at"`
	StartTime   time.Time        `json:"start_time"`
	EndTime     time.Time        `json:"end_time"`
	InputCount  int              `json:"input_count"`
	Roster      []roamPilot      `json:"roster"`
	Unresolved  []string         `json:"unresolved"`
	Truncated   bool             `json:"truncated"`
	Summary     roamSummary      `json:"summary"`
	Systems     []roamSystem     `json:"systems"`
	Targets     []roamTarget     `json:"targets"`
	Engagements []roamEngagement `json:"engagements"`
}

func registerRoamReportRoutes(a huma.API, opts Options) {
	registerLegacyJSON(a, huma.Operation{
		OperationID: "roam-report-create",
		Method:      http.MethodPost,
		Path:        "/tools/roam-report",
		Summary:     "Create a shareable roam report from fleet character names",
		Tags:        []string{"tools", "battles"},
	}, roamMaxInputBytes+1024, func(ctx context.Context, _ *legacyRequest, body *roamReportCreateBody) (legacyPayload, error) {
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		primary, err := mutationDatabase(opts)
		if err != nil {
			return legacyPayload{}, err
		}
		idBytes := make([]byte, 16)
		if _, err := rand.Read(idBytes); err != nil {
			return legacyPayload{}, err
		}
		report, err := prepareRoamReport(ctx, opts, primary, body, hex.EncodeToString(idBytes), time.Now().UTC())
		if err != nil {
			return legacyPayload{}, err
		}
		encoded, err := json.Marshal(report)
		if err != nil {
			return legacyPayload{}, err
		}
		_, err = primary.Exec(ctx, `INSERT INTO roam_reports (id, start_time, end_time, report) VALUES ($1, $2, $3, $4::jsonb)`, report.ID, report.StartTime, report.EndTime, string(encoded))
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{"id": report.ID}), nil
	})

	registerLegacyJSON(a, huma.Operation{
		OperationID: "roam-report-update",
		Method:      http.MethodPut,
		Path:        "/tools/roam-report/{id}",
		Summary:     "Update the pilots or time range of a saved roam report",
		Tags:        []string{"tools", "battles"},
	}, roamMaxInputBytes+1024, func(ctx context.Context, req *legacyRequest, body *roamReportCreateBody) (legacyPayload, error) {
		id := req.Param("id")
		if !validRoamID(id) {
			return legacyPayload{}, apiError(http.StatusNotFound, "Roam report not found")
		}
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		primary, err := mutationDatabase(opts)
		if err != nil {
			return legacyPayload{}, err
		}
		var raw []byte
		if err := primary.QueryRow(ctx, `SELECT report FROM roam_reports WHERE id = $1`, id).Scan(&raw); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return legacyPayload{}, apiError(http.StatusNotFound, "Roam report not found")
			}
			return legacyPayload{}, err
		}
		var previous roamReport
		if err := json.Unmarshal(raw, &previous); err != nil {
			return legacyPayload{}, fmt.Errorf("decode roam report: %w", err)
		}
		report, err := prepareRoamReport(ctx, opts, primary, body, id, previous.CreatedAt)
		if err != nil {
			return legacyPayload{}, err
		}
		encoded, err := json.Marshal(report)
		if err != nil {
			return legacyPayload{}, err
		}
		if _, err := primary.Exec(ctx, `UPDATE roam_reports SET start_time = $2, end_time = $3, report = $4::jsonb WHERE id = $1`,
			id, report.StartTime, report.EndTime, string(encoded)); err != nil {
			return legacyPayload{}, err
		}
		payload := jsonPayload(report)
		payload.Headers = http.Header{"Cache-Control": []string{"no-store"}}
		return payload, nil
	})

	registerLegacy(a, huma.Operation{
		OperationID: "roam-report-get",
		Method:      http.MethodGet,
		Path:        "/tools/roam-report/{id}",
		Summary:     "Get a saved roam report",
		Tags:        []string{"tools", "battles"},
	}, func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		id := req.Param("id")
		if !validRoamID(id) {
			return legacyPayload{}, apiError(http.StatusNotFound, "Roam report not found")
		}
		db := opts.DB
		if opts.Primary != nil {
			db = opts.Primary
		}
		if db == nil {
			return legacyPayload{}, apiError(http.StatusServiceUnavailable, "API database is not configured")
		}
		var raw []byte
		if err := db.QueryRow(ctx, `SELECT report FROM roam_reports WHERE id = $1`, id).Scan(&raw); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return legacyPayload{}, apiError(http.StatusNotFound, "Roam report not found")
			}
			return legacyPayload{}, err
		}
		var report roamReport
		if err := json.Unmarshal(raw, &report); err != nil {
			return legacyPayload{}, fmt.Errorf("decode roam report: %w", err)
		}
		if total := report.Summary.IskDestroyed + report.Summary.IskLost; total > 0 {
			report.Summary.Efficiency = math.Round(report.Summary.IskDestroyed/total*10000) / 100
		} else {
			report.Summary.Efficiency = 0
		}
		if len(report.Roster) > 0 && report.Roster[0].Ships == nil {
			killIDs := make([]int32, 0)
			for _, engagement := range report.Engagements {
				for _, kill := range engagement.Killmails {
					killIDs = append(killIDs, kill.KillmailID)
				}
			}
			pilotIDs := make([]int32, 0, len(report.Roster))
			for _, pilot := range report.Roster {
				pilotIDs = append(pilotIDs, pilot.CharacterID)
			}
			var attackers []roamAttacker
			if len(killIDs) > 0 {
				var queryErr error
				attackers, queryErr = loadRoamAttackers(ctx, db, killIDs, pilotIDs)
				if queryErr != nil {
					return legacyPayload{}, queryErr
				}
			}
			addRoamShips(&report, attackers)
		}
		payload := jsonPayload(report)
		payload.Headers = http.Header{"Cache-Control": []string{"no-store"}}
		return payload, nil
	})

	registerLegacy(a, huma.Operation{
		OperationID: "roam-report-killlist",
		Method:      http.MethodGet,
		Path:        "/tools/roam-report/{id}/killlist",
		Summary:     "Paginated killlist for the killmails saved in a roam report",
		Tags:        []string{"tools", "battles"},
	}, func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		id := req.Param("id")
		if !validRoamID(id) {
			return legacyPayload{}, apiError(http.StatusNotFound, "Roam report not found")
		}
		role := req.Query.Get("role")
		if role == "" {
			role = "kills"
		}
		if role != "kills" && role != "losses" && role != "all" {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid role")
		}
		db := primaryDatabase(opts)
		if db == nil {
			return legacyPayload{}, apiError(http.StatusServiceUnavailable, "API database is not configured")
		}
		var raw []byte
		if err := db.QueryRow(ctx, `SELECT report FROM roam_reports WHERE id = $1`, id).Scan(&raw); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return legacyPayload{}, apiError(http.StatusNotFound, "Roam report not found")
			}
			return legacyPayload{}, err
		}
		var report roamReport
		if err := json.Unmarshal(raw, &report); err != nil {
			return legacyPayload{}, fmt.Errorf("decode roam report: %w", err)
		}
		killIDs := make([]int32, 0)
		for _, engagement := range report.Engagements {
			for _, kill := range engagement.Killmails {
				if role == "all" || role == "kills" && kill.Role == "kill" || role == "losses" && kill.Role == "loss" {
					killIDs = append(killIDs, kill.KillmailID)
				}
			}
		}
		if len(killIDs) == 0 {
			payload := jsonPayload(map[string]any{"kills": []any{}, "hasMore": false, "cursor": nil, "totalPages": 1})
			payload.Headers = http.Header{"Cache-Control": []string{"no-store"}}
			return payload, nil
		}
		payload, err := loadConflictKilllist(ctx, db, req.Query,
			[]string{"k.killmail_id = ANY($1::int[])"}, []any{killIDs})
		if err != nil {
			return legacyPayload{}, err
		}
		payload.Headers = http.Header{"Cache-Control": []string{"no-store"}}
		return payload, nil
	})
}

func validRoamID(id string) bool {
	return len(id) == 32 && strings.Trim(id, "0123456789abcdef") == ""
}

func prepareRoamReport(ctx context.Context, opts Options, primary Database, body *roamReportCreateBody, id string, createdAt time.Time) (roamReport, error) {
	names, err := parseRoamNames(body.NamesText)
	if err != nil {
		return roamReport{}, apiError(http.StatusBadRequest, err.Error())
	}
	if err := validateRoamWindow(body.StartTime, body.EndTime, time.Now()); err != nil {
		return roamReport{}, apiError(http.StatusBadRequest, err.Error())
	}
	roster, unresolved, err := loadRoamRoster(ctx, primary, names)
	if err != nil {
		return roamReport{}, err
	}
	if len(roster) == 0 {
		return roamReport{}, apiError(http.StatusBadRequest, "No character names were found. Check the fleet list and try again.")
	}
	ids := make([]int32, 0, len(roster))
	for _, pilot := range roster {
		ids = append(ids, pilot.CharacterID)
	}
	killmails, err := loadRoamKillmails(ctx, opts.DB, ids, body.StartTime, body.EndTime)
	if err != nil {
		return roamReport{}, err
	}
	truncated := len(killmails) > roamMaxKillmails
	if truncated {
		killmails = killmails[:roamMaxKillmails]
	}
	var attackers []roamAttacker
	var finalBlows []roamFinalBlow
	if len(killmails) > 0 {
		killIDs := make([]int32, 0, len(killmails))
		for _, km := range killmails {
			killIDs = append(killIDs, km.KillmailID)
		}
		attackers, err = loadRoamAttackers(ctx, opts.DB, killIDs, ids)
		if err != nil {
			return roamReport{}, err
		}
		finalBlows, err = loadRoamFinalBlows(ctx, opts.DB, killIDs)
		if err != nil {
			return roamReport{}, err
		}
	}
	report := buildRoamReport(id, createdAt, body.StartTime.UTC(), body.EndTime.UTC(), names, roster, unresolved, killmails, attackers, finalBlows)
	report.Truncated = truncated
	addRoamShips(&report, attackers)
	return report, nil
}

func parseRoamNames(text string) ([]string, error) {
	if len(text) > roamMaxInputBytes {
		return nil, fmt.Errorf("fleet list is too large")
	}
	seen := make(map[string]struct{})
	names := make([]string, 0)
	for line := range strings.SplitSeq(text, "\n") {
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}
		if utf8.RuneCountInString(name) > roamMaxNameLength {
			return nil, fmt.Errorf("character names must be at most %d characters", roamMaxNameLength)
		}
		key := strings.ToLower(name)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		names = append(names, name)
		if len(names) > roamMaxPilots {
			return nil, fmt.Errorf("maximum %d characters per report", roamMaxPilots)
		}
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("paste at least one character name")
	}
	return names, nil
}

func validateRoamWindow(start, end, now time.Time) error {
	if start.IsZero() || end.IsZero() {
		return fmt.Errorf("choose a start and end time")
	}
	if !end.After(start) {
		return fmt.Errorf("end time must be after start time")
	}
	if end.Sub(start) > roamMaxWindow {
		return fmt.Errorf("maximum roam window is 72 hours")
	}
	if end.After(now.Add(5 * time.Minute)) {
		return fmt.Errorf("end time cannot be in the future")
	}
	return nil
}

func loadRoamRoster(ctx context.Context, db Database, names []string) ([]roamPilot, []string, error) {
	rows, err := db.Query(ctx, `
		SELECT c.character_id, c.name,
		       COALESCE(c.corporation_id, 0), COALESCE(co.name, ''), COALESCE(co.ticker, ''),
		       COALESCE(c.alliance_id, 0), COALESCE(al.name, ''), COALESCE(al.ticker, '')
		FROM characters c
		LEFT JOIN corporations co ON co.corporation_id = c.corporation_id
		LEFT JOIN alliances al ON al.alliance_id = c.alliance_id
		WHERE c.name = ANY($1::text[]) AND c.deleted IS NOT TRUE`, names)
	if err != nil {
		return nil, nil, err
	}
	found, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (roamPilot, error) {
		var pilot roamPilot
		err := row.Scan(&pilot.CharacterID, &pilot.Name,
			&pilot.CorporationID, &pilot.CorporationName, &pilot.CorporationTicker,
			&pilot.AllianceID, &pilot.AllianceName, &pilot.AllianceTicker)
		return pilot, err
	})
	if err != nil {
		return nil, nil, err
	}
	byName := make(map[string]roamPilot, len(found))
	for _, pilot := range found {
		byName[strings.ToLower(pilot.Name)] = pilot
	}
	roster := make([]roamPilot, 0, len(found))
	unresolved := make([]string, 0)
	for _, name := range names {
		if pilot, ok := byName[strings.ToLower(name)]; ok {
			roster = append(roster, pilot)
		} else {
			unresolved = append(unresolved, name)
		}
	}
	return roster, unresolved, nil
}

func loadRoamKillmails(ctx context.Context, db Database, ids []int32, start, end time.Time) ([]roamKillmail, error) {
	rows, err := db.Query(ctx, `
		WITH matched AS (
		    SELECT a.killmail_id FROM killmail_attackers a
		    WHERE a.character_id = ANY($1::int[]) AND a.killmail_time >= $2 AND a.killmail_time < $3
		    UNION
		    SELECT k.killmail_id FROM killmails k
		    WHERE k.victim_character_id = ANY($1::int[]) AND k.killmail_time >= $2 AND k.killmail_time < $3
		)
		SELECT k.killmail_id, k.killmail_time, k.solar_system_id,
		       COALESCE(s.system_name, ''), COALESCE(k.region_id, s.region_id, 0), COALESCE(r.name, ''),
		       COALESCE(k.victim_character_id, 0), COALESCE(vc.name, ''),
		       COALESCE(k.victim_corporation_id, 0), COALESCE(co.name, ''),
		       COALESCE(k.victim_alliance_id, 0), COALESCE(al.name, ''),
		       COALESCE(k.victim_ship_type_id, 0), COALESCE(ship.name, ''),
		       COALESCE(k.victim_ship_group_id, ship.group_id, 0), COALESCE(ship_group.name, ''),
		       COALESCE(k.total_value, 0), COALESCE(k.attacker_count, 0)
		FROM matched m
		JOIN killmails k ON k.killmail_id = m.killmail_id
		LEFT JOIN solar_systems s ON s.solar_system_id = k.solar_system_id
		LEFT JOIN regions r ON r.region_id = COALESCE(k.region_id, s.region_id)
		LEFT JOIN characters vc ON vc.character_id = k.victim_character_id
		LEFT JOIN corporations co ON co.corporation_id = k.victim_corporation_id
		LEFT JOIN alliances al ON al.alliance_id = k.victim_alliance_id
		LEFT JOIN inv_types ship ON ship.type_id = k.victim_ship_type_id
		LEFT JOIN inv_groups ship_group ON ship_group.group_id = COALESCE(k.victim_ship_group_id, ship.group_id)
		WHERE k.killmail_time >= $2 AND k.killmail_time < $3 AND k.is_npc IS NOT TRUE
		ORDER BY k.killmail_time, k.killmail_id
		LIMIT $4`, ids, start, end, roamMaxKillmails+1)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (roamKillmail, error) {
		var km roamKillmail
		err := row.Scan(&km.KillmailID, &km.KillmailTime, &km.SolarSystemID,
			&km.SolarSystemName, &km.RegionID, &km.RegionName,
			&km.VictimCharacterID, &km.VictimName,
			&km.VictimCorporationID, &km.VictimCorporationName,
			&km.VictimAllianceID, &km.VictimAllianceName,
			&km.VictimShipTypeID, &km.VictimShipName, &km.VictimShipGroupID, &km.VictimShipGroupName,
			&km.TotalValue, &km.AttackerCount)
		if math.IsNaN(km.TotalValue) || math.IsInf(km.TotalValue, 0) || km.TotalValue < 0 {
			km.TotalValue = 0
		}
		return km, err
	})
}

func loadRoamAttackers(ctx context.Context, db Database, killIDs, pilotIDs []int32) ([]roamAttacker, error) {
	rows, err := db.Query(ctx, `
		SELECT a.killmail_id, a.character_id, COALESCE(a.damage_done, 0), COALESCE(a.final_blow, false),
		       COALESCE(a.ship_type_id, 0), COALESCE(ship.name, ''),
		       COALESCE(a.ship_group_id, ship.group_id, 0), COALESCE(ship_group.name, '')
		FROM killmail_attackers a
		LEFT JOIN inv_types ship ON ship.type_id = a.ship_type_id
		LEFT JOIN inv_groups ship_group ON ship_group.group_id = COALESCE(a.ship_group_id, ship.group_id)
		WHERE a.killmail_id = ANY($1::int[]) AND a.character_id = ANY($2::int[])
		ORDER BY a.killmail_id, a.attacker_index`, killIDs, pilotIDs)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (roamAttacker, error) {
		var attacker roamAttacker
		err := row.Scan(&attacker.KillmailID, &attacker.CharacterID, &attacker.DamageDone, &attacker.FinalBlow,
			&attacker.ShipTypeID, &attacker.ShipName, &attacker.ShipGroupID, &attacker.ShipGroupName)
		return attacker, err
	})
}

func loadRoamFinalBlows(ctx context.Context, db Database, killIDs []int32) ([]roamFinalBlow, error) {
	rows, err := db.Query(ctx, `
		SELECT a.killmail_id, COALESCE(c.name, ''), COALESCE(co.name, '')
		FROM killmail_attackers a
		LEFT JOIN characters c ON c.character_id = a.character_id
		LEFT JOIN corporations co ON co.corporation_id = a.corporation_id
		WHERE a.killmail_id = ANY($1::int[]) AND a.final_blow IS TRUE`, killIDs)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (roamFinalBlow, error) {
		var blow roamFinalBlow
		err := row.Scan(&blow.KillmailID, &blow.CharacterName, &blow.CorporationName)
		return blow, err
	})
}
