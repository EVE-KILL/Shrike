package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

// Test-only reader: lets the latest renderer use a read-only preview API,
// including the fixed-size viewport fixture, without creating a database.
type remoteFactionWar struct{ origin string }

type fixtureHTTPError struct {
	path   string
	status int
}

func (e *fixtureHTTPError) Error() string {
	return fmt.Sprintf("fixture API %s: HTTP %d", e.path, e.status)
}

func fixtureRead[T any](ctx context.Context, origin, path string, host ...string) (T, error) {
	var result T
	req, err := http.NewRequestWithContext(ctx, "GET", origin+path, nil)
	if err != nil {
		return result, err
	}
	if len(host) > 0 {
		req.Host = host[0]
	}
	client := http.Client{Timeout: 35 * time.Second}
	response, err := client.Do(req)
	if err != nil {
		return result, err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != 200 {
		return result, &fixtureHTTPError{path: path, status: response.StatusCode}
	}
	err = json.NewDecoder(response.Body).Decode(&result)
	return result, err
}
func (r remoteFactionWar) FactionWarDashboard(ctx context.Context, slug string) (api.FactionWarData, error) {
	return fixtureRead[api.FactionWarData](ctx, r.origin, "/api/faction-war/"+slug+"/dashboard")
}
func (r remoteFactionWar) FactionWarOverview(ctx context.Context, slug, period string) (api.FactionWarOverview, error) {
	return fixtureRead[api.FactionWarOverview](ctx, r.origin, "/api/faction-war/"+slug+"/overview?period="+url.QueryEscape(period))
}
func (r remoteFactionWar) FactionWarSystems(ctx context.Context, slug string) (api.FactionWarSystems, error) {
	return fixtureRead[api.FactionWarSystems](ctx, r.origin, "/api/faction-war/"+slug+"/systems")
}
func (r remoteFactionWar) FactionWarMembers(ctx context.Context, slug string, q url.Values) (api.WarMembersData, error) {
	return fixtureRead[api.WarMembersData](ctx, r.origin, "/api/faction-war/"+slug+"/members?"+q.Encode())
}
func (r remoteFactionWar) FactionWarIntel(ctx context.Context, slug string) (api.WarIntelData, error) {
	return fixtureRead[api.WarIntelData](ctx, r.origin, "/api/faction-war/"+slug+"/intel")
}
func (r remoteFactionWar) FactionWarKilllist(ctx context.Context, slug, side string, page, limit int, after *int64) (api.KilllistPage, error) {
	factions := []string{"500001", "500004"}
	if slug == "amarr-vs-minmatar" {
		factions = []string{"500003", "500002"}
	}
	selected := factions[0] + "," + factions[1]
	switch side {
	case "side1":
		selected = factions[1]
	case "side2":
		selected = factions[0]
	}
	q := url.Values{"type": {"latest"}, "limit": {strconv.Itoa(limit)}, "victimFactions": {selected}}
	if after != nil {
		q.Set("after", strconv.FormatInt(*after, 10))
	}
	return fixtureRead[api.KilllistPage](ctx, r.origin, "/api/killlist?"+q.Encode())
}
