package api

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/eve-kill/shrike/internal/eve"
)

const maxBulkPriceTypes = 256

type marketTreeNode struct {
	ID       int64             `json:"id"`
	Name     string            `json:"name"`
	Slug     string            `json:"slug"`
	ParentID *int64            `json:"parent_id"`
	HasTypes bool              `json:"has_types"`
	IconID   *int64            `json:"icon_id"`
	Children []*marketTreeNode `json:"children"`
}

func registerPrivateMarketRoutes(a huma.API, opts Options) {
	longCache := privateJSONCache(
		opts,
		6*time.Hour,
		"public, max-age=3600, s-maxage=21600, stale-while-revalidate=3600",
		marketTreeHandler(opts),
	)
	registerLegacy(a, huma.Operation{
		OperationID: "frontend-market-tree",
		Method:      http.MethodGet,
		Path:        "/api/market/tree",
		Summary:     "Market browser hierarchy",
		Tags:        []string{"market"},
	}, longCache)

	registerLegacy(a, huma.Operation{
		OperationID: "frontend-market-group-items",
		Method:      http.MethodGet,
		Path:        "/api/market/groups/{id}/items",
		Summary:     "Published items in a market group",
		Tags:        []string{"market"},
	}, privateJSONCache(
		opts,
		6*time.Hour,
		"public, max-age=3600, s-maxage=21600, stale-while-revalidate=3600",
		marketGroupItemsHandler(opts),
	))

	registerLegacy(a, huma.Operation{
		OperationID: "frontend-bulk-prices",
		Method:      http.MethodGet,
		Path:        "/api/prices/bulk",
		Summary:     "Latest prices for a set of item types",
		Tags:        []string{"market"},
	}, privateJSONCacheBy(
		opts,
		5*time.Minute,
		"public, max-age=60, s-maxage=300, stale-while-revalidate=60",
		bulkPriceCacheKey,
		bulkPricesHandler(opts),
	))
}

func marketTreeHandler(opts Options) legacyHandler {
	return func(ctx context.Context, _ *legacyRequest) (legacyPayload, error) {
		rows, err := queryMaps(ctx, opts.DB, `
			SELECT market_group_id, parent_group_id, name, has_types, icon_id
			FROM inv_market_groups
			ORDER BY name ASC`)
		if err != nil {
			return legacyPayload{}, err
		}

		return jsonPayload(map[string]any{"groups": buildMarketTree(rows)}), nil
	}
}

func buildMarketTree(rows []map[string]any) []*marketTreeNode {
	nodes := make(map[int64]*marketTreeNode, len(rows))
	order := make([]int64, 0, len(rows))
	for _, row := range rows {
		id, ok := int64Value(row["market_group_id"])
		if !ok {
			continue
		}
		name, _ := stringValue(row["name"])
		var parentID *int64
		if value, ok := int64Value(row["parent_group_id"]); ok {
			parentID = &value
		}
		var iconID *int64
		if value, ok := int64Value(row["icon_id"]); ok {
			iconID = &value
		}
		hasTypes, _ := row["has_types"].(bool)
		nodes[id] = &marketTreeNode{
			ID: id, Name: name, Slug: eve.Slugify(name),
			ParentID: parentID, HasTypes: hasTypes, IconID: iconID,
			Children: []*marketTreeNode{},
		}
		order = append(order, id)
	}

	roots := make([]*marketTreeNode, 0)
	for _, id := range order {
		node := nodes[id]
		if node.ParentID == nil {
			roots = append(roots, node)
			continue
		}
		if parent := nodes[*node.ParentID]; parent != nil {
			parent.Children = append(parent.Children, node)
		}
	}
	var sortNodes func([]*marketTreeNode)
	sortNodes = func(list []*marketTreeNode) {
		sort.SliceStable(list, func(i, j int) bool {
			return strings.ToLower(list[i].Name) < strings.ToLower(list[j].Name)
		})
		for _, node := range list {
			sortNodes(node.Children)
		}
	}
	sortNodes(roots)
	return roots
}

func marketGroupItemsHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		id, err := parseID(req.Param("id"))
		if err != nil || id > pgInt4Max {
			return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid ID")
		}
		rows, err := queryMaps(ctx, opts.DB, `
			SELECT t.type_id, t.name, t.group_id, g.category_id,
			       t.meta_group_id, (g.category_id = 6) AS is_ship
			FROM inv_types t
			LEFT JOIN inv_groups g ON g.group_id = t.group_id
			WHERE t.market_group_id = $1 AND t.published IS TRUE
			ORDER BY t.name ASC`, id)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{"items": rows}), nil
	}
}

func parseBulkPriceIDs(raw string) ([]int32, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	seen := make(map[int32]bool)
	ids := make([]int32, 0)
	for part := range strings.SplitSeq(raw, ",") {
		value, err := strconv.ParseInt(strings.TrimSpace(part), 10, 32)
		if err != nil || value <= 0 {
			continue
		}
		id := int32(value)
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	if len(ids) > maxBulkPriceTypes {
		return nil, apiError(http.StatusBadRequest,
			"too many type_ids (max 256)")
	}
	return ids, nil
}

func bulkPriceCacheKey(req *legacyRequest) string {
	ids, err := parseBulkPriceIDs(req.Query.Get("types"))
	if err != nil {
		url := req.Huma.URL()
		return url.RequestURI()
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.FormatInt(int64(id), 10)
	}
	return "/api/prices/bulk?types=" + strings.Join(parts, ",")
}

func bulkPricesHandler(opts Options) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		ids, err := parseBulkPriceIDs(req.Query.Get("types"))
		if err != nil {
			return legacyPayload{}, err
		}
		if len(ids) == 0 {
			return jsonPayload(map[string]any{"prices": map[string]float64{}}), nil
		}

		prices, err := loadFittingPrices(ctx, opts.DB, ids)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{"prices": prices}), nil
	}
}
