package main

import (
	"context"
	"net/url"

	"github.com/eve-kill/shrike/internal/api"
)

// The facade reads the retained fixture API; it never needs another database.
type remoteAdvancedSearch struct{ origin string }

func (r remoteAdvancedSearch) Labels(ctx context.Context) ([]api.LabelEntry, error) {
	result, err := fixtureRead[struct {
		Labels []api.LabelEntry `json:"labels"`
	}](ctx, r.origin, "/api/labels")
	return result.Labels, err
}
func (r remoteAdvancedSearch) AdvancedSearch(ctx context.Context, query url.Values) (api.AdvancedSearchResult, error) {
	return fixtureRead[api.AdvancedSearchResult](ctx, r.origin, "/api/killlist/advanced?"+query.Encode())
}
