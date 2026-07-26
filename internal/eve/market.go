package eve

import (
	"context"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Market group paths.
//
// A killlist row carries `/market/ships/battleships/marauders` so the client
// can link a hull to its market page. Building that means walking the market
// group tree from a leaf to the root, which would be several queries per
// killmail — so the whole tree is materialised once at load and every lookup
// after that is a map read.
//
// The tree is SDE-static: about a thousand rows that change only when CCP ships
// a new expansion, by which point the process has been redeployed anyway.

// MarketPaths maps a market group id to its slug path.
type MarketPaths map[int32]string

// marketPathDepthCap guards against a cycle in corrupt SDE data. EVE's real
// tree is five or six deep.
const marketPathDepthCap = 16

// LoadMarketPaths materialises the path for every market group.
func LoadMarketPaths(ctx context.Context, pool *pgxpool.Pool) (MarketPaths, error) {
	type node struct {
		name   string
		parent int32
	}

	rows, err := pool.Query(ctx,
		`SELECT market_group_id, coalesce(name, ''), coalesce(parent_group_id, 0) FROM inv_market_groups`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byID := make(map[int32]node, 1024)
	for rows.Next() {
		var id int32
		var n node
		if err := rows.Scan(&id, &n.name, &n.parent); err != nil {
			return nil, err
		}
		byID[id] = n
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make(MarketPaths, len(byID))
	for id := range byID {
		var segments []string
		seen := make(map[int32]bool, marketPathDepthCap)

		for cur, depth := id, 0; cur != 0 && !seen[cur] && depth < marketPathDepthCap; depth++ {
			seen[cur] = true
			n, ok := byID[cur]
			if !ok {
				break
			}
			// Prepended: the walk goes leaf to root, the path reads root to leaf.
			segments = append([]string{Slugify(n.name)}, segments...)
			cur = n.parent
		}
		out[id] = "/market/" + strings.Join(segments, "/")
	}
	return out, nil
}

// Path returns the market path for a group, or "" when there is none.
//
// An empty string rather than a sentinel because the caller writes it into a
// nullable JSON field, where absent and empty mean the same thing.
func (m MarketPaths) Path(marketGroupID int32) string {
	if marketGroupID == 0 {
		return ""
	}
	return m[marketGroupID]
}

// PathForType resolves a type id's market path through its group.
func (m MarketPaths) PathForType(c *Cache, typeID int32) string {
	if typeID == 0 || c == nil {
		return ""
	}
	t, ok := c.Type(typeID)
	if !ok {
		return ""
	}
	return m.Path(t.MarketGroupID)
}

var slugNonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify renders a market group name as a URL segment.
//
// Matches the TypeScript implementation exactly, including replacing `&` with
// "and" before stripping punctuation — so "Ammunition & Charges" becomes
// "ammunition-and-charges" rather than "ammunition-charges". The paths are
// user-visible URLs, so a difference here is a broken link.
func Slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "&", "and")
	s = slugNonAlnum.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}
