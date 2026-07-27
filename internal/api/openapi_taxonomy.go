package api

import (
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

// Tags are derived from the URL rather than chosen per route.
//
// Hand-picked tags drifted: /admin/campaigns carried "campaigns" and filed
// itself next to the public campaign endpoints, while system and region data
// scattered across "killmails" and "entities". A reader looking for universe
// data had to already know which tag someone had picked for it.
//
// The rule is the one the previous documentation used: an endpoint belongs to
// its first URL segment. Everything under /universe is universe data, and the
// address bar and the sidebar agree.
//
// Two segments are access markers rather than resources. /admin and /me are
// about who may call an endpoint, not what it operates on, so the resource
// comes from the segment after them and the access level becomes a second tag.
// /admin/campaigns then sits with the other campaign routes and still says it
// is administrative.

const (
	tagAdmin   = "admin"
	tagAccount = "account"

	// adminTagPrefix namespaces the administrative half of a resource so it
	// can sit under Administration while the public half stays with its peers.
	adminTagPrefix = "admin-"
)

// resourceAliases folds a raw first segment onto the tag that owns it.
// Singular and plural spellings of the same resource must not become two
// sidebar entries, and a handful of one-endpoint segments belong to a larger
// neighbour rather than to a section of their own.
var resourceAliases = map[string]string{
	"alliance": "alliances", "battle": "battles", "campaign": "campaigns",
	"character": "characters", "constellation": "constellations",
	"coalition": "coalitions", "corporation": "corporations",
	"entity": "entities", "fit": "fittings", "fits": "fittings",
	"item": "items", "killmail": "killmails", "region": "regions",
	"scan": "scans", "ship": "ships", "system": "systems", "war": "wars",

	"__sitemap__":     "sitemap",
	"campaign-prizes": "campaigns",
	"comment-reports": "comments",
	"conflicts":       "battles",
	"esi-entities":    "esi",
	"esi-logs":        "esi",
	"faction":         "faction-war",
	"faction-wars":    "faction-war",
	"graph":           "stats",
	"killlist":        "killmails",
	"kills":           "killmails",
	"location":        "universe",
	"matchup":         "ships",
	"overview":        "admin",
	"prices":          "market",
	"resolve":         "search",
	"tools":           "scans",
}

func resourceTag(segment string) string {
	if alias, ok := resourceAliases[segment]; ok {
		return alias
	}
	return segment
}

// classifyOperation returns the resource tag for a path and the access marker
// implied by its prefix, which is empty for anonymous routes.
func classifyOperation(path string) (resource string, access string) {
	segments := strings.FieldsFunc(path, func(r rune) bool { return r == '/' })
	if len(segments) == 0 {
		return "index", ""
	}

	switch segments[0] {
	case "admin":
		if len(segments) == 1 {
			return tagAdmin, tagAdmin
		}
		// Administrative routes get their own tag rather than joining the
		// public tag for the same resource. A tag belongs to exactly one
		// group, so sharing "campaigns" would have dragged /admin/campaigns
		// into Conflicts alongside the endpoints anyone can read.
		resource := resourceTag(segments[1])
		if resource == tagAdmin {
			return tagAdmin, tagAdmin
		}
		return adminTagPrefix + resource, tagAdmin
	case "me", "user":
		// Every /me and /user route is one surface: the signed-in character's
		// own settings, boards, tokens, and notifications. Splitting it by
		// sub-resource produced fifteen tags of one or two endpoints each.
		return tagAccount, tagAccount
	}
	return resourceTag(segments[0]), ""
}

// applyOperationTags replaces the tags on every operation.
//
// It runs after registration and overwrites rather than merges, because the
// point is that one rule decides placement. A route keeping a hand-written tag
// would be exactly the drift this removes.
func applyOperationTags(document *huma.OpenAPI) {
	for path, item := range document.Paths {
		resource, pathAccess := classifyOperation(path)
		for _, operation := range operationsOf(item) {
			access := pathAccess
			// A route under a public prefix can still demand a session. The
			// declared security requirement is the authority on that, not the
			// URL: /campaigns is public to read and signed-in to create.
			if access == "" && requiresSession(operation) {
				access = tagAccount
			}
			tags := []string{resource}
			if access != "" && access != resource {
				tags = append(tags, access)
			}
			operation.Tags = tags
		}
	}
}

// requiresSession reports whether every declared security option needs the
// session scheme. An operation listing an empty option alongside it is
// readable anonymously and stays public.
func requiresSession(operation *huma.Operation) bool {
	if len(operation.Security) == 0 {
		return false
	}
	for _, option := range operation.Security {
		if len(option) == 0 {
			return false
		}
	}
	return true
}

func operationsOf(item *huma.PathItem) []*huma.Operation {
	all := []*huma.Operation{
		item.Get, item.Put, item.Post, item.Delete,
		item.Options, item.Head, item.Patch, item.Trace,
	}
	live := all[:0]
	for _, operation := range all {
		if operation != nil {
			live = append(live, operation)
		}
	}
	return live
}
