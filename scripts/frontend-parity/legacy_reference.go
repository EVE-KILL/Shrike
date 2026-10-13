package main

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/eve-kill/shrike/internal/api"
)

type remoteLegacy struct{ origin string }

func (r remoteLegacy) LegacyArchiveStats(ctx context.Context) (api.LegacyArchiveStats, error) {
	return fixtureRead[api.LegacyArchiveStats](ctx, r.origin, "/api/legacy/stats")
}
func (r remoteLegacy) LegacyArchiveTop(ctx context.Context, kind string, year int) ([]api.TopEntry, error) {
	query := url.Values{"dataType": {kind}, "limit": {"10"}}
	if year != 0 {
		query.Set("year", strconv.Itoa(year))
	}
	result, err := fixtureRead[struct {
		Entries []api.TopEntry `json:"entries"`
	}](ctx, r.origin, "/api/legacy/top?"+query.Encode())
	return result.Entries, err
}
func (r remoteLegacy) LegacyArchiveKills(ctx context.Context, query url.Values) (api.KilllistPage, error) {
	return fixtureRead[api.KilllistPage](ctx, r.origin, "/api/legacy/kills?"+query.Encode())
}
func (r remoteLegacy) LegacyArchiveKill(ctx context.Context, id int64) (api.LegacyArchiveKillData, bool, error) {
	data, err := fixtureRead[api.LegacyArchiveKillData](ctx, r.origin, "/api/legacy/kill/"+strconv.FormatInt(id, 10))
	var failure *fixtureHTTPError
	if errors.As(err, &failure) && failure.status == http.StatusNotFound {
		return data, false, nil
	}
	return data, err == nil, err
}

func legacyReferenceRenderer(reference, renderer http.Handler, scripts map[string][]byte) http.Handler {
	return componentReferenceRenderer(reference, renderer, scripts, "legacy", regexp.MustCompile(`src="/assets/[^/"<>]+/legacy\.js"`), func(path string) bool {
		return path == "/legacy" || strings.HasPrefix(path, "/_fragments/legacy/")
	}, []string{"legacy"}, "legacy.js")
}
