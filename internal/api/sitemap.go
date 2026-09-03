package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

const (
	sitemapCacheTTL            = time.Hour
	sitemapCanonicalPath       = "/sitemap/{kind}"
	sitemapCompatibilityPrefix = "/__sitemap__/"
)

type sitemapSpec struct {
	Kind            string
	Query           string
	LocationPrefix  string
	ChangeFrequency string
	Priority        float64
	HasLastModified bool
	StringID        bool
}

var sitemapSpecs = []sitemapSpec{
	{
		Kind: "alliances",
		Query: `
			SELECT alliance_id AS id, updated_at AS lastmod
			FROM alliances
			WHERE updated_at > now() - interval '30 days'
			ORDER BY updated_at DESC
			LIMIT 5000`,
		LocationPrefix:  "/alliance/",
		ChangeFrequency: "daily",
		Priority:        0.7,
		HasLastModified: true,
	},
	{
		Kind: "battles",
		Query: `
			SELECT battle_id AS id, start_time AS lastmod
			FROM battles
			ORDER BY start_time DESC
			LIMIT 5000`,
		LocationPrefix:  "/battle/",
		ChangeFrequency: "weekly",
		Priority:        0.5,
		HasLastModified: true,
	},
	{
		Kind: "characters",
		Query: `
			SELECT character_id AS id, updated_at AS lastmod
			FROM characters
			WHERE last_active > now() - interval '30 days'
			ORDER BY last_active DESC
			LIMIT 50000`,
		LocationPrefix:  "/character/",
		ChangeFrequency: "daily",
		Priority:        0.7,
		HasLastModified: true,
	},
	{
		Kind: "coalitions",
		Query: `
			SELECT slug AS id, updated_at AS lastmod
			FROM coalitions
			ORDER BY updated_at DESC, slug ASC`,
		LocationPrefix:  "/coalitions/",
		ChangeFrequency: "daily",
		Priority:        0.6,
		HasLastModified: true,
		StringID:        true,
	},
	{
		Kind: "corporations",
		Query: `
			SELECT corporation_id AS id, updated_at AS lastmod
			FROM corporations
			WHERE updated_at > now() - interval '30 days'
			ORDER BY updated_at DESC
			LIMIT 10000`,
		LocationPrefix:  "/corporation/",
		ChangeFrequency: "daily",
		Priority:        0.7,
		HasLastModified: true,
	},
	{
		Kind: "items",
		Query: `
			SELECT t.type_id AS id
			FROM inv_types t
			INNER JOIN inv_groups g ON g.group_id = t.group_id
			WHERE t.published IS TRUE
			  AND g.category_id <> 6
			  AND t.market_group_id IS NOT NULL
			ORDER BY t.type_id ASC
			LIMIT 50000`,
		LocationPrefix:  "/item/",
		ChangeFrequency: "monthly",
		Priority:        0.4,
	},
	{
		Kind: "kills",
		Query: `
			SELECT killmail_id AS id, killmail_time AS lastmod
			FROM killmails
			WHERE killmail_time > now() - interval '30 days'
			ORDER BY total_value DESC
			LIMIT 50000`,
		LocationPrefix:  "/kill/",
		ChangeFrequency: "monthly",
		Priority:        0.6,
		HasLastModified: true,
	},
	{
		Kind: "regions",
		Query: `
			SELECT region_id AS id
			FROM regions
			ORDER BY region_id ASC`,
		LocationPrefix:  "/region/",
		ChangeFrequency: "daily",
		Priority:        0.6,
	},
	{
		Kind: "ships",
		Query: `
			SELECT t.type_id AS id
			FROM inv_types t
			INNER JOIN inv_groups g ON g.group_id = t.group_id
			WHERE g.category_id = 6
			  AND t.published IS TRUE
			ORDER BY t.type_id ASC`,
		LocationPrefix:  "/item/",
		ChangeFrequency: "weekly",
		Priority:        0.6,
	},
	{
		Kind: "systems",
		Query: `
			SELECT solar_system_id AS id
			FROM solar_systems
			ORDER BY solar_system_id ASC`,
		LocationPrefix:  "/system/",
		ChangeFrequency: "daily",
		Priority:        0.6,
	},
	{
		Kind: "wars",
		Query: `
			SELECT war_id AS id
			FROM wars
			ORDER BY war_id DESC
			LIMIT 5000`,
		LocationPrefix:  "/war/",
		ChangeFrequency: "weekly",
		Priority:        0.5,
	},
}

func registerSitemapRoutes(a huma.API, opts Options) {
	cache := func(handler legacyHandler) legacyHandler {
		return routeJSONCache(
			opts,
			sitemapCacheTTL,
			"public, max-age=300, s-maxage=3600, stale-while-revalidate=3600",
			handler,
		)
	}

	registerLegacy(a, huma.Operation{
		OperationID: "sitemap",
		Method:      http.MethodGet,
		Path:        sitemapCanonicalPath,
		Summary:     "URLs for a sitemap category",
		Tags:        []string{"sitemap"},
	}, cache(sitemapHandler(opts, "")))

	for _, spec := range sitemapSpecs {
		registerLegacy(a, huma.Operation{
			OperationID: "sitemap-" + spec.Kind + "-compat",
			Method:      http.MethodGet,
			Path:        sitemapCompatibilityPrefix + spec.Kind,
			Summary:     "URLs for the " + spec.Kind + " sitemap",
			Tags:        []string{"sitemap"},
		}, cache(sitemapHandler(opts, spec.Kind)))
	}
}

func sitemapHandler(opts Options, fixedKind string) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		kind := fixedKind
		if kind == "" {
			kind = req.Param("kind")
		}
		spec, err := resolveSitemapSpec(kind)
		if err != nil {
			return legacyPayload{}, err
		}

		rows, err := queryMaps(ctx, opts.DB, spec.Query)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(buildSitemapEntries(spec, rows)), nil
	}
}

func resolveSitemapSpec(kind string) (sitemapSpec, error) {
	for _, spec := range sitemapSpecs {
		if kind == spec.Kind {
			return spec, nil
		}
	}
	return sitemapSpec{}, apiError(
		http.StatusBadRequest,
		fmt.Sprintf("Unknown sitemap kind: %s", kind),
	)
}

func buildSitemapEntries(
	spec sitemapSpec,
	rows []map[string]any,
) []map[string]any {
	entries := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		var location string
		if spec.StringID {
			id, ok := row["id"].(string)
			if !ok || !coalitionSlugPattern.MatchString(id) {
				continue
			}
			location = spec.LocationPrefix + id
		} else {
			id, ok := int64Value(row["id"])
			if !ok {
				continue
			}
			location = spec.LocationPrefix + strconv.FormatInt(id, 10)
		}
		entry := map[string]any{
			"loc":        location,
			"changefreq": spec.ChangeFrequency,
			"priority":   spec.Priority,
		}
		if spec.HasLastModified && row["lastmod"] != nil {
			entry["lastmod"] = row["lastmod"]
		}
		entries = append(entries, entry)
	}
	return entries
}
