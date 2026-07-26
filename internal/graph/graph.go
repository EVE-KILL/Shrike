// Package graph maintains the relationship graph in Memgraph.
//
// Postgres answers "what happened"; this answers "who flies with whom". Those
// are different shapes of question — the second is a traversal over an
// arbitrary number of hops, which relational joins do badly and a graph
// database does natively — which is why a second store exists at all.
//
// Everything here is derived: the graph can be dropped and rebuilt from the
// killmails at any time. That is what makes the retention policy safe, and why
// a failure to write an edge is not worth failing a killmail over.
package graph

import (
	"context"
	"fmt"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// EdgeTypes are the relationships stored. Listed because the purge iterates
// them and a type missing from the list is never pruned.
var EdgeTypes = []string{"FLEW_WITH", "KILLED", "OPERATED_IN", "MEMBER_OF", "ALLIED_WITH"}

// RetentionDays bounds how far back relationships are kept.
//
// Ninety days: the graph answers questions about who is flying together now,
// and an edge from a year ago describes a fleet that no longer exists.
const RetentionDays = 90

// The TypeScript maintenance job deliberately uses a smaller orphan batch
// than its edge batch.
const orphanPurgeBatch = 5_000

// Client wraps a Memgraph connection.
type Client struct {
	driver neo4j.DriverWithContext
}

// Connect opens a Memgraph connection.
//
// Memgraph speaks the Bolt protocol, so Neo4j's driver is the client — the two
// are wire-compatible for everything used here.
func Connect(ctx context.Context, url string) (*Client, error) {
	// No authentication: Memgraph runs inside the cluster and is not exposed.
	d, err := neo4j.NewDriverWithContext(url, neo4j.NoAuth())
	if err != nil {
		return nil, fmt.Errorf("open memgraph driver: %w", err)
	}
	if err := d.VerifyConnectivity(ctx); err != nil {
		_ = d.Close(ctx)
		return nil, fmt.Errorf("connect to memgraph at %s: %w", url, err)
	}
	return &Client{driver: d}, nil
}

// Close releases the connection.
func (c *Client) Close(ctx context.Context) error {
	if c == nil || c.driver == nil {
		return nil
	}
	return c.driver.Close(ctx)
}

// Read runs one read-only Cypher query and returns each record as a map.
// Public API intelligence endpoints use this for relationship lookups while
// keeping the Neo4j/Memgraph driver out of the HTTP package.
func (c *Client) Read(
	ctx context.Context,
	cypher string,
	params map[string]any,
) ([]map[string]any, error) {
	if c == nil || c.driver == nil {
		return nil, fmt.Errorf("memgraph is not configured")
	}
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode: neo4j.AccessModeRead,
	})
	defer session.Close(ctx) //nolint:errcheck

	result, err := session.Run(ctx, cypher, params)
	if err != nil {
		return nil, err
	}
	rows := []map[string]any{}
	for result.Next(ctx) {
		record := result.Record()
		row := make(map[string]any, len(record.Keys))
		for i, key := range record.Keys {
			row[key] = record.Values[i]
		}
		rows = append(rows, row)
	}
	if err := result.Err(); err != nil {
		return nil, err
	}
	return rows, nil
}

// Clear removes every node and relationship from the derived graph.
func (c *Client) Clear(ctx context.Context) error {
	if c == nil || c.driver == nil {
		return nil
	}
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx) //nolint:errcheck

	_, err := session.Run(ctx, `MATCH (n) DETACH DELETE n`, nil)
	if err != nil {
		return fmt.Errorf("clear graph: %w", err)
	}
	return nil
}

// Character is one participant in a killmail, as the graph stores them.
type Character struct {
	ID            int64
	CorporationID int64
	AllianceID    int64

	LastFCSeen        time.Time
	LastSuperKill     time.Time
	LastBlopsSeen     time.Time
	LastCapitalKill   time.Time
	LastLogisticsSeen time.Time
}

// KilledEdge is one attacker-victim relationship.
type KilledEdge struct {
	AttackerID   int64
	VictimID     int64
	IskDestroyed float64
	FinalBlows   int64
}

// FlewWithEdge is a pair who appeared on the same side of one kill.
//
// Stored with the lower id first so the pair is one edge rather than two: a
// flew-with relationship is symmetric, and storing both directions would double
// every weight.
type FlewWithEdge struct {
	Lo, Hi int64
}

// Killmail is what one ingest writes.
type Killmail struct {
	KillmailID    int64
	KillmailTime  time.Time
	SolarSystemID int64

	Characters []Character
	Killed     []KilledEdge
	FlewWith   []FlewWithEdge
}

// Ingest writes one killmail's relationships.
//
// Every edge uses max-wins on last_seen rather than overwriting it. Killmails
// arrive out of order — the repair cron backfills days-old kills alongside the
// live feed — and an older one must not drag an edge's last_seen backwards.
func (c *Client) Ingest(ctx context.Context, km Killmail) error {
	if c == nil || c.driver == nil {
		return nil
	}

	session := c.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx) //nolint:errcheck // best-effort close

	at := isoTimestamp(km.KillmailTime)

	// Nodes before edges: a MATCH on a node that does not exist silently
	// matches nothing, so the edge would be dropped without an error.
	if len(km.Characters) > 0 {
		chars := make([]map[string]any, 0, len(km.Characters))
		corps := map[int64]bool{}
		alliances := map[int64]bool{}

		for _, ch := range km.Characters {
			chars = append(chars, map[string]any{
				"id":                ch.ID,
				"corp":              nullable(ch.CorporationID),
				"alliance":          nullable(ch.AllianceID),
				"last_fc_seen":      nullableTime(ch.LastFCSeen),
				"last_super_kill":   nullableTime(ch.LastSuperKill),
				"last_blops_seen":   nullableTime(ch.LastBlopsSeen),
				"last_capital_kill": nullableTime(ch.LastCapitalKill),
				"last_logi_seen":    nullableTime(ch.LastLogisticsSeen),
			})
			if ch.CorporationID != 0 {
				corps[ch.CorporationID] = true
			}
			if ch.AllianceID != 0 {
				alliances[ch.AllianceID] = true
			}
		}

		if _, err := session.Run(ctx, `
            UNWIND $chars AS c
            MERGE (n:Character {id: c.id})
            SET n.corporation_id = c.corp,
                n.alliance_id = c.alliance,
                n.last_seen = CASE WHEN $time > coalesce(n.last_seen, '') THEN $time ELSE n.last_seen END,
                n.last_fc_seen = CASE
                    WHEN c.last_fc_seen IS NOT NULL AND c.last_fc_seen > coalesce(n.last_fc_seen, '')
                    THEN c.last_fc_seen ELSE n.last_fc_seen END,
                n.last_super_kill = CASE
                    WHEN c.last_super_kill IS NOT NULL AND c.last_super_kill > coalesce(n.last_super_kill, '')
                    THEN c.last_super_kill ELSE n.last_super_kill END,
                n.last_blops_seen = CASE
                    WHEN c.last_blops_seen IS NOT NULL AND c.last_blops_seen > coalesce(n.last_blops_seen, '')
                    THEN c.last_blops_seen ELSE n.last_blops_seen END,
                n.last_capital_kill = CASE
                    WHEN c.last_capital_kill IS NOT NULL AND c.last_capital_kill > coalesce(n.last_capital_kill, '')
                    THEN c.last_capital_kill ELSE n.last_capital_kill END,
                n.last_logi_seen = CASE
                    WHEN c.last_logi_seen IS NOT NULL AND c.last_logi_seen > coalesce(n.last_logi_seen, '')
                    THEN c.last_logi_seen ELSE n.last_logi_seen END`,
			map[string]any{"chars": chars, "time": at}); err != nil {
			return fmt.Errorf("merge character nodes: %w", err)
		}

		if err := mergeIDs(ctx, session, "Corporation", corps); err != nil {
			return err
		}
		if err := mergeIDs(ctx, session, "Alliance", alliances); err != nil {
			return err
		}
	}

	if km.SolarSystemID != 0 {
		if _, err := session.Run(ctx, `MERGE (:SolarSystem {id: $id})`,
			map[string]any{"id": km.SolarSystemID}); err != nil {
			return fmt.Errorf("merge solar system: %w", err)
		}
	}

	if len(km.FlewWith) > 0 {
		items := make([]map[string]any, 0, len(km.FlewWith))
		for _, e := range km.FlewWith {
			items = append(items, map[string]any{"lo": e.Lo, "hi": e.Hi})
		}
		if _, err := session.Run(ctx, `
            UNWIND $items AS e
            MATCH (a:Character {id: e.lo}), (b:Character {id: e.hi})
            MERGE (a)-[r:FLEW_WITH]->(b)
            ON CREATE SET r.weight = 1, r.first_seen = $time, r.last_seen = $time
            ON MATCH SET r.weight = r.weight + 1,
                r.last_seen = CASE WHEN $time > r.last_seen THEN $time ELSE r.last_seen END`,
			map[string]any{"items": items, "time": at}); err != nil {
			return fmt.Errorf("merge flew-with edges: %w", err)
		}
	}

	if len(km.Killed) > 0 {
		items := make([]map[string]any, 0, len(km.Killed))
		for _, e := range km.Killed {
			items = append(items, map[string]any{
				"atk": e.AttackerID, "vic": e.VictimID,
				"isk": e.IskDestroyed, "fb": e.FinalBlows,
			})
		}
		if _, err := session.Run(ctx, `
            UNWIND $items AS e
            MATCH (a:Character {id: e.atk}), (v:Character {id: e.vic})
            MERGE (a)-[r:KILLED]->(v)
            ON CREATE SET r.weight = 1, r.isk_destroyed = e.isk, r.final_blows = e.fb,
                r.first_seen = $time, r.last_seen = $time
            ON MATCH SET r.weight = r.weight + 1,
                r.isk_destroyed = r.isk_destroyed + e.isk,
                r.final_blows = r.final_blows + e.fb,
                r.last_seen = CASE WHEN $time > r.last_seen THEN $time ELSE r.last_seen END`,
			map[string]any{"items": items, "time": at}); err != nil {
			return fmt.Errorf("merge killed edges: %w", err)
		}
	}

	if km.SolarSystemID != 0 && len(km.Characters) > 0 {
		ids := make([]int64, 0, len(km.Characters))
		for _, ch := range km.Characters {
			ids = append(ids, ch.ID)
		}
		if _, err := session.Run(ctx, `
            UNWIND $ids AS cid
            MATCH (c:Character {id: cid}), (s:SolarSystem {id: $sys})
            MERGE (c)-[r:OPERATED_IN]->(s)
            ON CREATE SET r.weight = 1, r.last_seen = $time
            ON MATCH SET r.weight = r.weight + 1,
                r.last_seen = CASE WHEN $time > r.last_seen THEN $time ELSE r.last_seen END`,
			map[string]any{"ids": ids, "sys": km.SolarSystemID, "time": at}); err != nil {
			return fmt.Errorf("merge operated-in edges: %w", err)
		}
	}

	if len(km.Characters) > 0 {
		members := make([]map[string]any, 0, len(km.Characters))
		allied := make([]map[string]any, 0, len(km.Characters))
		seenAllied := make(map[[2]int64]bool)
		for _, ch := range km.Characters {
			if ch.CorporationID == 0 {
				continue
			}
			members = append(members, map[string]any{
				"char": ch.ID,
				"corp": ch.CorporationID,
			})
			if ch.AllianceID == 0 {
				continue
			}
			key := [2]int64{ch.CorporationID, ch.AllianceID}
			if seenAllied[key] {
				continue
			}
			seenAllied[key] = true
			allied = append(allied, map[string]any{
				"corp":     ch.CorporationID,
				"alliance": ch.AllianceID,
			})
		}

		if len(members) > 0 {
			if _, err := session.Run(ctx, `
                UNWIND $items AS e
                MATCH (c:Character {id: e.char}), (corp:Corporation {id: e.corp})
                MERGE (c)-[r:MEMBER_OF]->(corp)
                ON CREATE SET r.first_seen = $time, r.last_seen = $time
                ON MATCH SET r.last_seen =
                    CASE WHEN $time > r.last_seen THEN $time ELSE r.last_seen END`,
				map[string]any{"items": members, "time": at}); err != nil {
				return fmt.Errorf("merge member-of edges: %w", err)
			}
		}

		if len(allied) > 0 {
			if _, err := session.Run(ctx, `
                UNWIND $items AS e
                MATCH (corp:Corporation {id: e.corp}), (a:Alliance {id: e.alliance})
                MERGE (corp)-[r:ALLIED_WITH]->(a)
                ON CREATE SET r.first_seen = $time, r.last_seen = $time
                ON MATCH SET r.last_seen =
                    CASE WHEN $time > r.last_seen THEN $time ELSE r.last_seen END`,
				map[string]any{"items": allied, "time": at}); err != nil {
				return fmt.Errorf("merge allied-with edges: %w", err)
			}
		}
	}

	return nil
}

func mergeIDs(ctx context.Context, session neo4j.SessionWithContext, label string, ids map[int64]bool) error {
	if len(ids) == 0 {
		return nil
	}
	list := make([]int64, 0, len(ids))
	for id := range ids {
		list = append(list, id)
	}
	// The label comes from this package's fixed call sites, never from input.
	if _, err := session.Run(ctx,
		fmt.Sprintf(`UNWIND $ids AS id MERGE (:%s {id: id})`, label),
		map[string]any{"ids": list}); err != nil {
		return fmt.Errorf("merge %s nodes: %w", label, err)
	}
	return nil
}

// PurgeResult reports what a prune removed.
type PurgeResult struct {
	Edges   int64            `json:"edges"`
	Orphans int64            `json:"orphans"`
	ByType  map[string]int64 `json:"by_type"`
}

// Purge removes relationships older than the retention window, and any
// character left with none.
//
// Batched rather than done in one statement. A single DELETE over ninety days
// of edges is a transaction large enough to exhaust Memgraph's memory, and a
// partial prune is harmless — the next run continues where this one stopped.
func (c *Client) Purge(ctx context.Context, batchSize int) (PurgeResult, error) {
	out := PurgeResult{ByType: map[string]int64{}}
	if c == nil || c.driver == nil {
		return out, nil
	}

	session := c.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx) //nolint:errcheck // best-effort close

	cutoff := isoTimestamp(time.Now().UTC().AddDate(0, 0, -RetentionDays))

	for _, edgeType := range EdgeTypes {
		// The edge type comes from the fixed list above.
		res, err := session.Run(ctx, fmt.Sprintf(`
            MATCH ()-[r:%s]->()
            WHERE r.last_seen < $cutoff
            WITH r LIMIT $limit
            DELETE r
            RETURN count(r) AS cnt`, edgeType),
			map[string]any{"cutoff": cutoff, "limit": batchSize})
		if err != nil {
			return out, fmt.Errorf("purge %s edges: %w", edgeType, err)
		}
		n := countFrom(ctx, res)
		out.Edges += n
		if n > 0 {
			out.ByType[edgeType] = n
		}
	}

	// Orphans only after the edges are gone — a character with no relationships
	// left is one whose entire history aged out.
	res, err := session.Run(ctx, `
        MATCH (n:Character)
        WHERE NOT (n)-[]-()
        WITH n LIMIT $limit
        DELETE n
        RETURN count(n) AS cnt`, map[string]any{"limit": orphanPurgeBatch})
	if err != nil {
		return out, fmt.Errorf("purge orphaned characters: %w", err)
	}
	out.Orphans = countFrom(ctx, res)

	return out, nil
}

func countFrom(ctx context.Context, res neo4j.ResultWithContext) int64 {
	if res.Next(ctx) {
		if v, ok := res.Record().Get("cnt"); ok {
			if n, ok := v.(int64); ok {
				return n
			}
		}
	}
	return 0
}

// nullable maps the zero-means-absent convention onto a Cypher null.
func nullable(v int64) any {
	if v == 0 {
		return nil
	}
	return v
}

func nullableTime(v time.Time) any {
	if v.IsZero() {
		return nil
	}
	return isoTimestamp(v)
}

// JavaScript Date.toISOString(), used by the TypeScript graph worker, always
// includes milliseconds. Memgraph stores these timestamps as strings and
// compares them lexicographically, so mixing second-precision and
// millisecond-precision forms can produce the wrong max-wins result.
func isoTimestamp(v time.Time) string {
	return v.UTC().Format("2006-01-02T15:04:05.000Z")
}
