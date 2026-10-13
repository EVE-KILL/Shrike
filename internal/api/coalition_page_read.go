package api

import (
	"context"
	"net/http"
	"time"
)

// CoalitionDirectoryData is the existing public coalition API contract.
// The Go page calls the same handlers as /api/coalitions, so membership,
// combat totals, and edit history have one source of truth.
type CoalitionDirectoryData struct {
	Coalitions      []CoalitionSummary `json:"coalitions"`
	StatsWindowDays int                `json:"stats_window_days"`
}

type CoalitionSummary struct {
	CoalitionID            int64   `json:"coalition_id"`
	Slug                   string  `json:"slug"`
	Name                   string  `json:"name"`
	Description            string  `json:"description"`
	SourceURL              *string `json:"source_url"`
	Revision               int64   `json:"revision"`
	AllianceCount          int64   `json:"alliance_count"`
	MemberCount            int64   `json:"member_count"`
	SystemCount            int64   `json:"system_count"`
	Kills                  int64   `json:"kills"`
	Losses                 int64   `json:"losses"`
	ISKDestroyed           float64 `json:"isk_destroyed"`
	ISKLost                float64 `json:"isk_lost"`
	UpdatedAt              string  `json:"updated_at"`
	UpdatedByCharacterID   *int64  `json:"updated_by_character_id"`
	UpdatedByCharacterName *string `json:"updated_by_character_name"`
}

type CoalitionDetailData struct {
	Coalition       CoalitionSummary    `json:"coalition"`
	Alliances       []CoalitionAlliance `json:"alliances"`
	Edits           []CoalitionEdit     `json:"edits"`
	StatsWindowDays int                 `json:"stats_window_days"`
}

type CoalitionAlliance struct {
	AllianceID       int64  `json:"alliance_id"`
	Name             string `json:"name"`
	Ticker           string `json:"ticker"`
	MemberCount      int64  `json:"member_count"`
	CorporationCount int64  `json:"corporation_count"`
	SystemCount      int64  `json:"system_count"`
}

type CoalitionEdit struct {
	EditID              int64  `json:"edit_id"`
	EditorCharacterID   *int64 `json:"editor_character_id"`
	EditorCharacterName string `json:"editor_character_name"`
	Action              string `json:"action"`
	Summary             string `json:"summary"`
	CreatedAt           string `json:"created_at"`
}

func (r *HomepageReader) CoalitionDirectory(ctx context.Context) (CoalitionDirectoryData, error) {
	return cachedHomepage(ctx, r, "/coalitions", time.Minute, func(ctx context.Context) (CoalitionDirectoryData, error) {
		payload, err := coalitionDirectoryListHandler(Options{DB: r.db})(ctx, &legacyRequest{})
		if err != nil {
			return CoalitionDirectoryData{}, err
		}
		var result CoalitionDirectoryData
		if err := decodeLegacyPagePayload(payload, &result); err != nil {
			return CoalitionDirectoryData{}, err
		}
		return result, nil
	})
}

// CoalitionDirectoryOverview supplies useful SSR content without combat totals.
// The cache key remains under /coalitions so edits invalidate both datasets.
func (r *HomepageReader) CoalitionDirectoryOverview(ctx context.Context) (CoalitionDirectoryData, error) {
	return cachedHomepage(ctx, r, "/coalitions/overview", 5*time.Minute, func(ctx context.Context) (CoalitionDirectoryData, error) {
		rows, err := queryMaps(ctx, r.db, coalitionDirectoryOverviewSQL)
		if err != nil {
			return CoalitionDirectoryData{}, err
		}
		coalitions, err := decodeHomepageRows[CoalitionSummary](rows)
		return CoalitionDirectoryData{Coalitions: coalitions, StatsWindowDays: coalitionStatsDays}, err
	})
}

func (r *HomepageReader) CoalitionDetail(ctx context.Context, slug string) (CoalitionDetailData, bool, error) {
	if !coalitionSlugPattern.MatchString(slug) {
		return CoalitionDetailData{}, false, nil
	}
	type lookup struct {
		Data  CoalitionDetailData
		Found bool
	}
	result, err := cachedHomepage(ctx, r, "/coalitions/"+slug, time.Minute, func(ctx context.Context) (lookup, error) {
		payload, err := coalitionDirectoryDetailHandler(Options{DB: r.db})(ctx, &legacyRequest{Params: map[string]string{"slug": slug}})
		if err != nil {
			if apiErr, ok := err.(*legacyAPIError); ok && apiErr.Status == http.StatusNotFound {
				return lookup{}, nil
			}
			return lookup{}, err
		}
		var data CoalitionDetailData
		if err := decodeLegacyPagePayload(payload, &data); err != nil {
			return lookup{}, err
		}
		return lookup{Data: data, Found: true}, nil
	})
	return result.Data, result.Found, err
}
