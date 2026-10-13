package main

import (
	"context"
	"net/url"
	"strconv"

	"github.com/eve-kill/shrike/internal/api"
)

type remoteStatistics struct{ origin string }

type remoteRankings struct{ origin string }

func (r remoteRankings) Rankings(ctx context.Context, entity, window string) ([]api.RankingEntry, error) {
	query := url.Values{"section": {"eve-kill"}, "entityType": {entity}, "window": {window}, "limit": {"50"}}
	result, err := fixtureRead[struct {
		Entries []api.RankingEntry `json:"entries"`
	}](ctx, r.origin, "/api/stats/rankings?"+query.Encode())
	return result.Entries, err
}

func (r remoteStatistics) StatisticsSite(ctx context.Context, host string) (api.SiteConfigurationResponse, error) {
	return fixtureRead[api.SiteConfigurationResponse](ctx, r.origin, "/api/site", host)
}
func (r remoteStatistics) StatisticsBoardsCached(context.Context, api.SiteConfigurationResponse, []api.StatisticsQuery) map[int][]api.StatisticsEntry {
	return nil
}
func (r remoteStatistics) StatisticsBoard(ctx context.Context, site api.SiteConfigurationResponse, query api.StatisticsQuery) ([]api.StatisticsEntry, error) {
	values := url.Values{"limit": {"10"}, "days": {strconv.FormatFloat(query.Days, 'g', -1, 64)}}
	path, host := "/api/stats", "localhost"
	if query.DataType != "" {
		values.Set("dataType", query.DataType)
	} else {
		path = "/api/stats/rankings"
		values.Set("section", query.Section)
		values.Set("entityType", query.EntityType)
		if query.Rank != "" {
			values.Set("rank", query.Rank)
		}
		if query.Direction != "" {
			values.Set("direction", query.Direction)
		}
	}
	if site.Domain != nil {
		path = "/api/custom/stats"
		host = site.Domain.Subdomain + ".localhost"
	}
	result, err := fixtureRead[struct {
		Entries []api.StatisticsEntry `json:"entries"`
	}](ctx, r.origin, path+"?"+values.Encode(), host)
	return result.Entries, err
}
