package main

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/eve-kill/shrike/internal/api"
)

// Public GET-only data for current-renderer war comparisons; no credentials or
// writes are forwarded. Custom-board scoping is tested separately.
type remoteWars struct{ origin string }

func (r remoteWars) WarDirectory(ctx context.Context, q url.Values, _ string) (api.WarDirectoryData, error) {
	return fixtureRead[api.WarDirectoryData](ctx, r.origin, "/api/conflicts/wars?"+q.Encode())
}
func (r remoteWars) WarOverview(ctx context.Context) (api.WarOverviewData, error) {
	return fixtureRead[api.WarOverviewData](ctx, r.origin, "/api/wars/stats")
}
func (r remoteWars) FactionWars(ctx context.Context) (api.FactionWarsData, error) {
	return fixtureRead[api.FactionWarsData](ctx, r.origin, "/api/faction-wars")
}
func (r remoteWars) WarDetail(ctx context.Context, id int64) (api.WarDetailData, bool, error) {
	data, err := fixtureRead[api.WarDetailData](ctx, r.origin, "/api/war/"+strconv.FormatInt(id, 10))
	var status *fixtureHTTPError
	if errors.As(err, &status) && status.status == http.StatusNotFound {
		return data, false, nil
	}
	return data, err == nil, err
}
func (r remoteWars) WarKilllist(ctx context.Context, id int64, q url.Values) (api.KilllistPage, error) {
	return fixtureRead[api.KilllistPage](ctx, r.origin, "/api/war/"+strconv.FormatInt(id, 10)+"/killlist?"+q.Encode())
}
func (r remoteWars) WarStats(ctx context.Context, id int64) (api.WarStatsData, error) {
	return fixtureRead[api.WarStatsData](ctx, r.origin, "/api/war/"+strconv.FormatInt(id, 10)+"/stats")
}
func (r remoteWars) WarMembers(ctx context.Context, id int64, q url.Values) (api.WarMembersData, error) {
	return fixtureRead[api.WarMembersData](ctx, r.origin, "/api/war/"+strconv.FormatInt(id, 10)+"/members?"+q.Encode())
}
func (r remoteWars) WarIntel(ctx context.Context, id int64) (api.WarIntelData, error) {
	return fixtureRead[api.WarIntelData](ctx, r.origin, "/api/war/"+strconv.FormatInt(id, 10)+"/intel")
}
