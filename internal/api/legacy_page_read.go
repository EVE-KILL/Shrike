package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type LegacyArchiveStats struct {
	Killmails    int64 `json:"killmails"`
	Characters   int64 `json:"characters"`
	Corporations int64 `json:"corporations"`
	Alliances    int64 `json:"alliances"`
}

func (r *HomepageReader) LegacyArchiveStats(ctx context.Context) (LegacyArchiveStats, error) {
	return cachedHomepage(ctx, r, "legacy:stats", time.Hour, func(ctx context.Context) (LegacyArchiveStats, error) {
		return loadLegacyArchiveStats(ctx, r.db)
	})
}

func loadLegacyArchiveStats(ctx context.Context, db Database) (LegacyArchiveStats, error) {
	row, err := queryMap(ctx, db, `
		SELECT COUNT(*)::int AS killmails,
			(COUNT(DISTINCT victim_name) FILTER (WHERE victim_name IS NOT NULL))::int AS characters,
			(COUNT(DISTINCT victim_corp) FILTER (WHERE victim_corp IS NOT NULL))::int AS corporations,
			(COUNT(DISTINCT victim_alliance) FILTER (WHERE victim_alliance IS NOT NULL))::int AS alliances
		FROM old_killmails`)
	if err != nil {
		return LegacyArchiveStats{}, err
	}
	return LegacyArchiveStats{
		Killmails: int64OrZero(row["killmails"]), Characters: int64OrZero(row["characters"]),
		Corporations: int64OrZero(row["corporations"]), Alliances: int64OrZero(row["alliances"]),
	}, nil
}

func (r *HomepageReader) LegacyArchiveTop(ctx context.Context, kind string, year int) ([]TopEntry, error) {
	if _, ok := legacyTopSpecs[kind]; !ok {
		return nil, fmt.Errorf("invalid legacy top list %q", kind)
	}
	if year != 0 && (year < 2003 || year > 2007) {
		return nil, fmt.Errorf("invalid legacy year")
	}
	return cachedHomepage(ctx, r, fmt.Sprintf("legacy:top:%s:%d", kind, year), time.Hour, func(ctx context.Context) ([]TopEntry, error) {
		values := url.Values{"dataType": {kind}, "limit": {"10"}}
		if year != 0 {
			values.Set("year", strconv.Itoa(year))
		}
		payload, err := legacyArchiveTopHandler(Options{DB: r.db})(ctx, &legacyRequest{Query: values})
		if err != nil {
			return nil, err
		}
		var result struct {
			Entries []TopEntry `json:"entries"`
		}
		if err := decodeLegacyPagePayload(payload, &result); err != nil {
			return nil, err
		}
		return result.Entries, nil
	})
}

func (r *HomepageReader) LegacyArchiveKills(ctx context.Context, values url.Values) (KilllistPage, error) {
	payload, err := legacyArchiveKillsHandler(Options{DB: r.db})(ctx, &legacyRequest{Query: values})
	if err != nil {
		return KilllistPage{}, err
	}
	return decodeKilllistPayload(payload)
}

type LegacyArchiveKillData struct {
	Kill      LegacyArchiveKill       `json:"kill"`
	Attackers []LegacyArchiveAttacker `json:"attackers"`
	Items     []LegacyArchiveItem     `json:"items"`
}

type LegacyArchiveKill struct {
	KillmailID          int64     `json:"killmail_id"`
	KillmailTime        time.Time `json:"killmail_time"`
	RawText             string    `json:"raw_text"`
	TotalValue          float64   `json:"total_value"`
	Security            *float64  `json:"security"`
	SystemName          string    `json:"system_name"`
	VictimName          string    `json:"victim_name"`
	VictimCorp          string    `json:"victim_corp"`
	VictimAlliance      string    `json:"victim_alliance"`
	VictimShip          string    `json:"victim_ship"`
	VictimShipTypeID    *int64    `json:"victim_ship_type_id"`
	VictimCharacterID   *int64    `json:"victim_character_id"`
	VictimCorporationID *int64    `json:"victim_corporation_id"`
	VictimAllianceID    *int64    `json:"victim_alliance_id"`
}

type LegacyArchiveAttacker struct {
	Name          string `json:"name"`
	Corp          string `json:"corp"`
	Alliance      string `json:"alliance"`
	Ship          string `json:"ship"`
	Weapon        string `json:"weapon"`
	FinalBlow     bool   `json:"final_blow"`
	CharacterID   *int64 `json:"character_id"`
	CorporationID *int64 `json:"corporation_id"`
	AllianceID    *int64 `json:"alliance_id"`
	ShipTypeID    *int64 `json:"ship_type_id"`
	WeaponTypeID  *int64 `json:"weapon_type_id"`
}

type LegacyArchiveItem struct {
	ID       int64   `json:"id"`
	Name     string  `json:"name"`
	Location *string `json:"location"`
	TypeID   *int64  `json:"type_id"`
	Quantity int64   `json:"quantity"`
}

func (r *HomepageReader) LegacyArchiveKill(ctx context.Context, id int64) (LegacyArchiveKillData, bool, error) {
	payload, err := legacyArchiveKillHandler(Options{DB: r.db})(ctx, &legacyRequest{Params: map[string]string{"id": strconv.FormatInt(id, 10)}})
	if err != nil {
		if apiErr, ok := err.(*legacyAPIError); ok && apiErr.Status == http.StatusNotFound {
			return LegacyArchiveKillData{}, false, nil
		}
		return LegacyArchiveKillData{}, false, err
	}
	var result LegacyArchiveKillData
	if err := decodeLegacyPagePayload(payload, &result); err != nil {
		return LegacyArchiveKillData{}, false, err
	}
	return result, true, nil
}

func decodeLegacyPagePayload(payload legacyPayload, target any) error {
	encoded, err := json.Marshal(payload.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(encoded, target)
}
