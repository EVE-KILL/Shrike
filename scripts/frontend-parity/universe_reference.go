package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/eve-kill/shrike/internal/api"
	"github.com/eve-kill/shrike/internal/frontend"
)

// Same public DTOs and query semantics as HomepageReader, without a database
// connection or credentials. Only the loopback comparison facade uses this.
type remoteUniverse struct{ origin string }

func fixtureFound[T any](ctx context.Context, origin, path string) (T, bool, error) {
	data, err := fixtureRead[T](ctx, origin, path)
	var failure *fixtureHTTPError
	if errors.As(err, &failure) && failure.status == http.StatusNotFound {
		return data, false, nil
	}
	return data, err == nil, err
}
func (r remoteUniverse) UniverseSystem(ctx context.Context, id int64) (api.UniverseSystemData, bool, error) {
	return fixtureFound[api.UniverseSystemData](ctx, r.origin, "/api/universe/systems/"+strconv.FormatInt(id, 10))
}
func (r remoteUniverse) UniverseRegion(ctx context.Context, id int64) (api.UniverseRegionData, bool, error) {
	return fixtureFound[api.UniverseRegionData](ctx, r.origin, "/api/universe/regions/"+strconv.FormatInt(id, 10))
}
func (r remoteUniverse) UniverseConstellation(ctx context.Context, id int64) (api.UniverseConstellationData, bool, error) {
	return fixtureFound[api.UniverseConstellationData](ctx, r.origin, "/api/universe/constellations/"+strconv.FormatInt(id, 10))
}
func universeCursorQuery(limit int, after *int64) url.Values {
	query := url.Values{"limit": {strconv.Itoa(limit)}}
	if after != nil {
		query.Set("after", strconv.FormatInt(*after, 10))
	}
	return query
}
func (r remoteUniverse) UniverseLocationKills(ctx context.Context, kind string, id int64, limit int, after *int64) (api.KilllistPage, error) {
	return fixtureRead[api.KilllistPage](ctx, r.origin, "/api/"+kind+"/"+strconv.FormatInt(id, 10)+"/killlist?"+universeCursorQuery(limit, after).Encode())
}
func (r remoteUniverse) UniverseLocationValuable(ctx context.Context, kind string, id int64, category string) ([]api.ValuableKill, error) {
	query := url.Values{"limit": {"8"}, "days": {"7"}}
	if category != "kills" {
		query.Set("dataType", "most_valuable_"+category)
	}
	result, err := fixtureRead[struct {
		Entries []api.ValuableKill `json:"entries"`
	}](ctx, r.origin, "/api/"+kind+"/"+strconv.FormatInt(id, 10)+"/most-valuable?"+query.Encode())
	return result.Entries, err
}
func (r remoteUniverse) UniverseGroup(ctx context.Context, id int64) (api.UniverseGroupData, bool, error) {
	return fixtureFound[api.UniverseGroupData](ctx, r.origin, "/api/universe/groups/"+strconv.FormatInt(id, 10))
}
func (r remoteUniverse) UniverseGroupKills(ctx context.Context, id int64, limit int, after *int64) (api.KilllistPage, error) {
	filters, _ := json.Marshal(map[string]any{"entities": map[string]any{"victim": []map[string]any{{"id": id, "type": "shipgroup"}}}, "timeRange": map[string]any{"preset": "90d"}})
	query := universeCursorQuery(limit, after)
	query.Set("filters", string(filters))
	return fixtureRead[api.KilllistPage](ctx, r.origin, "/api/killlist/advanced?"+query.Encode())
}
func (r remoteUniverse) UniverseItem(ctx context.Context, id int64) (api.UniverseItemData, bool, error) {
	return fixtureFound[api.UniverseItemData](ctx, r.origin, "/api/universe/types/"+strconv.FormatInt(id, 10))
}
func (r remoteUniverse) UniverseItemKills(ctx context.Context, id int64, ship bool, limit int, after *int64) (api.KilllistPage, error) {
	kind := "item"
	if ship {
		kind = "ship"
	}
	return fixtureRead[api.KilllistPage](ctx, r.origin, "/api/"+kind+"/"+strconv.FormatInt(id, 10)+"/killlist?"+universeCursorQuery(limit, after).Encode())
}
func (r remoteUniverse) UniverseMarketOrders(ctx context.Context, id int64, region int32, security string) (api.UniverseMarketOrders, error) {
	query := url.Values{"region_id": {strconv.FormatInt(int64(region), 10)}, "security": {security}, "limit": {"100"}}
	return fixtureRead[api.UniverseMarketOrders](ctx, r.origin, "/api/market/items/"+strconv.FormatInt(id, 10)+"/orders?"+query.Encode())
}
func (r remoteUniverse) UniverseMarketHistory(ctx context.Context, id int64, region int32, days int) (api.UniverseMarketHistory, error) {
	query := url.Values{"region_id": {strconv.FormatInt(int64(region), 10)}, "days": {strconv.Itoa(days)}}
	return fixtureRead[api.UniverseMarketHistory](ctx, r.origin, "/api/market/items/"+strconv.FormatInt(id, 10)+"/history?"+query.Encode())
}
func (r remoteUniverse) UniverseFitFamilies(ctx context.Context, id int64, query url.Values) (api.UniverseFitFamilies, error) {
	return fixtureRead[api.UniverseFitFamilies](ctx, r.origin, "/api/fittings/ships/"+strconv.FormatInt(id, 10)+"/families?"+query.Encode())
}
func (r remoteUniverse) UniverseFitMeta(ctx context.Context, id int64, query url.Values) (api.UniverseFitMeta, error) {
	return fixtureRead[api.UniverseFitMeta](ctx, r.origin, "/api/fittings/ships/"+strconv.FormatInt(id, 10)+"/metadata?"+query.Encode())
}
func (r remoteUniverse) UniverseFitDistributions(ctx context.Context, id int64, filters url.Values) (api.UniverseFitDistributions, error) {
	query := url.Values{}
	for key, values := range filters {
		query[key] = append([]string(nil), values...)
	}
	query.Set("days", "90")
	return fixtureRead[api.UniverseFitDistributions](ctx, r.origin, "/api/fittings/ships/"+strconv.FormatInt(id, 10)+"/distributions?"+query.Encode())
}
func (r remoteUniverse) UniverseMarketTree(ctx context.Context) ([]api.UniverseMarketTreeNode, error) {
	result, err := fixtureRead[struct {
		Groups []api.UniverseMarketTreeNode `json:"groups"`
	}](ctx, r.origin, "/api/market/tree")
	return result.Groups, err
}
func (r remoteUniverse) UniverseMarketGroupItems(ctx context.Context, id int64) ([]api.UniverseMarketGroupItem, error) {
	result, err := fixtureRead[struct {
		Items []api.UniverseMarketGroupItem `json:"items"`
	}](ctx, r.origin, "/api/market/groups/"+strconv.FormatInt(id, 10)+"/items")
	return result.Items, err
}

// Other entity methods delegate to an existing fixture reader (or its normal
// nil-database error boundary); location battle cards always use real public GETs.
type remoteLocationEntities struct {
	frontend.EntityReader
	origin string
}

func (r remoteLocationEntities) EntityBattles(ctx context.Context, kind string, id int64, page int) (api.EntityBattlesData, error) {
	query := url.Values{"page": {strconv.Itoa(page)}, "limit": {"50"}, kind + "Id": {strconv.FormatInt(id, 10)}}
	return fixtureRead[api.EntityBattlesData](ctx, r.origin, "/api/conflicts/battles?"+query.Encode())
}

func universeReferenceRenderer(reference, renderer http.Handler, scripts map[string][]byte) http.Handler {
	// These original modules already guard their own page markers. One entry
	// preserves their original cooperation on item-market and fitting fragments.
	scripts["explorer.js"] = []byte("import './market.js';import './universe.js';import './system.js';")
	paths := regexp.MustCompile(`^/(?:market(?:/.*)?|(?:item|group|system|region|constellation)/[0-9]+(?:/[a-z]+)?)$`)
	return componentReferenceRenderer(reference, renderer, scripts, "universe", regexp.MustCompile(`src="/assets/[^/"<>]+/(?:market|universe|system)\.js"`), func(path string) bool {
		if paths.MatchString(path) {
			return true
		}
		for _, prefix := range []string{"/_fragments/market/", "/_fragments/item/", "/_fragments/group/", "/_fragments/system/", "/_fragments/location/"} {
			if strings.HasPrefix(path, prefix) {
				return true
			}
		}
		return false
	}, []string{"market-browser", "market-search", "item-market", "universe", "item-fits", "location"}, "explorer.js")
}
