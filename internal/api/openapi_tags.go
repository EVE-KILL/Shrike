package api

import (
	"fmt"
	"sort"

	"github.com/danielgtaylor/huma/v2"
)

// The rendered reference groups 43 tags into sections.
//
// Without this the sidebar is one flat alphabetical list, which puts /health
// next to /admin and gives a reader no way to tell the public read API from
// the signed-in surface. Scalar reads `x-tagGroups` for the section headings
// and the top-level `tags` array for ordering and per-tag prose.
//
// Every tag an operation uses must appear in exactly one group. A tag that is
// missing from every group vanishes from the sidebar rather than falling back
// to the flat list, so TestOpenAPITagGroupsCoverEveryTag enforces the mapping.

type tagGroup struct {
	Name string
	Tags []string
}

// tagGroups is ordered. Sections render top to bottom, and the tags inside a
// section render in the order given here, so the most-asked-for endpoints sit
// closest to the top of the page.
var tagGroups = []tagGroup{
	{"Killboard", []string{
		"killmails", "characters", "corporations", "alliances", "entities",
		"coalitions", "stats", "history", "items", "ships",
	}},
	{"Conflicts", []string{"battles", "wars", "campaigns", "faction-war"}},
	{"Universe", []string{
		"universe", "systems", "regions", "constellations", "map",
	}},
	{"Static data", []string{"sde", "market"}},
	{"Community", []string{"fittings", "comments", "blog", "scans", "boards"}},
	{"Account", []string{
		"account", "auth", "announcements", "notifications", "esi", "wallet",
		"users",
	}},
	{"Administration", []string{"admin", "moderation", "domains"}},
	{"Platform", []string{
		"health", "feed", "images", "sitemap", "site", "search", "legacy",
		"backgrounds",
	}},
}

// tagDescriptions is the one-line prose under each sidebar heading. A tag with
// no entry still renders; it just carries no description.
var tagDescriptions = map[string]string{
	"account":        "The signed-in character: preferences, boards, saved descriptions, and dismissals.",
	"admin":          "Operator-only administration. Requires an administrator session.",
	"alliances":      "Alliance listings, detail, member corporations, and kill and loss pages.",
	"announcements":  "Site announcements, and the per-character record of which were dismissed.",
	"auth":           "EVE Single Sign-On: the login flow, the session, and token inspection.",
	"backgrounds":    "Background images used by the site chrome.",
	"battles":        "Detected battles, their reports, compositions, timelines, and killmails.",
	"blog":           "Blog posts, and the drafting and preview endpoints behind them.",
	"boards":         "Killboards a character can manage.",
	"campaigns":      "Campaign definitions, their killmails, standings, and prize payouts.",
	"characters":     "Character listings, detail, statistics, and kill and loss pages.",
	"comments":       "Comment threads, reactions, and the report queue.",
	"corporations":   "Corporation listings, detail, members, and kill and loss pages.",
	"domains":        "Custom-domain killboards: configuration, theming, and assets.",
	"entities":       "Character, corporation, and alliance profile pages, and the panels on them.",
	"esi":            "EVE Swagger Interface (ESI) token state and request logs for the signed-in character.",
	"faction-war":    "Faction warfare matchups, systems, members, and intelligence.",
	"feed":           "The live killmail feed, by poll or by server-sent events.",
	"fittings":       "Ship fittings: the catalog, search, ratings, and trends.",
	"health":         "Liveness probe. Checks that the API can reach Postgres.",
	"history":        "Daily killmail totals, by date.",
	"images":         "Character, corporation, alliance, and type images, plus social cards.",
	"killmails":      "Killmails: listings, detail, ESI form, and the fitting they carried.",
	"map":            "Region and constellation map data.",
	"market":         "Market groups and bulk type prices.",
	"moderation":     "The moderation queue and its decisions.",
	"notifications":  "Comment replies addressed to the signed-in character.",
	"scans":          "Directional and local scan parsing and analysis.",
	"sde":            "The EVE Static Data Export: types, groups, systems, stations, and more.",
	"search":         "Full-text entity search, and exact name resolution.",
	"ships":          "Ship-level data: matchups and the fittings flown.",
	"site":           "Runtime site configuration served to the frontend.",
	"sitemap":        "Sitemap sources consumed by the frontend.",
	"stats":          "Batch statistics for several entities in one request.",
	"universe":       "Systems, constellations, and regions, and the kills inside them.",
	"wallet":         "Campaign prize wallets and their payout authorization.",
	"coalitions":     "Coalition-level aggregates across the alliances that make one up.",
	"constellations": "Constellations, and the killmails recorded inside them.",
	"items":          "Item-level killboard views: what a type destroyed and lost.",
	"legacy":         "Compatibility endpoints for the previous EVE-KILL site.",
	"regions":        "Regions, and the killmails recorded inside them.",
	"systems":        "Solar systems, and the killmails recorded inside them.",
	"users":          "Administrator management of user accounts and their roles.",
	"wars":           "Wars, their dashboards, participants, and killmails.",
}

// applyTagMetadata attaches the ordering, prose, and grouping to the document.
// It runs after every route is registered, because the tag list it validates
// against is whatever those routes declared.
func applyTagMetadata(document *huma.OpenAPI) {
	document.Tags = make([]*huma.Tag, 0, len(tagDescriptions))
	for _, group := range tagGroups {
		for _, name := range group.Tags {
			document.Tags = append(document.Tags, &huma.Tag{
				Name:        name,
				Description: tagDescriptions[name],
			})
		}
	}

	groups := make([]map[string]any, 0, len(tagGroups))
	for _, group := range tagGroups {
		groups = append(groups, map[string]any{
			"name": group.Name,
			"tags": group.Tags,
		})
	}
	if document.Extensions == nil {
		document.Extensions = map[string]any{}
	}
	document.Extensions["x-tagGroups"] = groups
}

// documentTags returns every tag any operation declares, sorted.
func documentTags(document *huma.OpenAPI) []string {
	seen := map[string]bool{}
	for _, item := range document.Paths {
		for _, operation := range []*huma.Operation{
			item.Get, item.Put, item.Post, item.Delete,
			item.Options, item.Head, item.Patch, item.Trace,
		} {
			if operation == nil {
				continue
			}
			for _, tag := range operation.Tags {
				seen[tag] = true
			}
		}
	}
	tags := make([]string, 0, len(seen))
	for tag := range seen {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return tags
}

// checkTagGroups reports tags that are used but ungrouped, grouped more than
// once, or grouped but unused. The first case is the one that costs a reader
// an endpoint they cannot find.
func checkTagGroups(used []string) error {
	usedSet := map[string]bool{}
	for _, tag := range used {
		usedSet[tag] = true
	}

	groupedIn := map[string][]string{}
	for _, group := range tagGroups {
		for _, tag := range group.Tags {
			groupedIn[tag] = append(groupedIn[tag], group.Name)
		}
	}

	var problems []string
	for _, tag := range used {
		if len(groupedIn[tag]) == 0 {
			problems = append(problems, fmt.Sprintf(
				"tag %q is used by an operation but is in no group", tag))
		}
	}
	for tag, groups := range groupedIn {
		if len(groups) > 1 {
			problems = append(problems, fmt.Sprintf(
				"tag %q is in %d groups: %v", tag, len(groups), groups))
		}
		if !usedSet[tag] {
			problems = append(problems, fmt.Sprintf(
				"tag %q is grouped but no operation uses it", tag))
		}
	}
	sort.Strings(problems)
	if len(problems) > 0 {
		return fmt.Errorf("%v", problems)
	}
	return nil
}
