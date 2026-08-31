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
	"sync"
	"time"
	"uuid"

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

// Each edge type gets several small transactions per maintenance run. One
// batch per day cannot keep pace with the production edge creation rate, while
// one enormous transaction can exhaust Memgraph's transactional memory.
const purgeBatchesPerEdgeType = 50

// Client wraps a Memgraph connection.
type Client struct {
	driver neo4j.DriverWithContext

	// Memgraph's MERGE can race when concurrent transactions first observe the
	// same node or relationship. Uniqueness constraints correctly reject the
	// duplicate node at commit, but relationships cannot be constrained and may
	// be duplicated. Shrike deliberately has one graph consumer deployment, so
	// serializing its short write transactions gives MERGE deterministic
	// semantics without reducing concurrency for the other River queues.
	ingestMu sync.Mutex
}

// nodeLabels is the complete set of node labels keyed by an external EVE id.
// Keeping this list next to Connect makes schema drift visible when a new node
// type is introduced.
var nodeLabels = []string{"Character", "Corporation", "Alliance", "SolarSystem", "Killmail"}

var nodeIndexes = map[string][]string{
	"Character":   {"corporation_id", "alliance_id"},
	"Corporation": {"alliance_id"},
	"Killmail":    {"killmail_time"},
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

// EnsureSchema installs uniqueness constraints for id lookups and concurrency
// safety, plus secondary indexes for the actual affiliation and retention scan
// predicates. Avoid a separate id index: Memgraph backs a uniqueness
// constraint with its own lookup structure, so both would duplicate memory and
// write overhead. Equivalent schema creation is a no-op.
//
// This is deliberately explicit rather than part of Connect: adding an index
// across a production graph is an operational change, and a uniqueness
// constraint must report pre-existing duplicate nodes without taking every
// graph-backed API offline.
func (c *Client) EnsureSchema(ctx context.Context) error {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx) //nolint:errcheck

	// Check every label before making any schema changes. Otherwise constraints
	// for early labels could be installed before a later duplicate makes the
	// command fail, leaving a confusing half-applied schema.
	for _, label := range nodeLabels {
		result, err := session.Run(ctx, fmt.Sprintf(`
			MATCH (n:%s)
			WITH n.id AS id, count(n) AS copies
			WHERE copies > 1
			RETURN count(*) AS cnt`, label), nil)
		if err != nil {
			return fmt.Errorf("inspect memgraph ids for %s: %w", label, err)
		}
		duplicates := countFrom(ctx, result)
		if duplicates > 0 {
			return fmt.Errorf("cannot constrain %s ids: %d duplicate ids exist; clean rebuild required", label, duplicates)
		}
	}

	for _, label := range nodeLabels {
		statements := []string{
			fmt.Sprintf("CREATE CONSTRAINT ON (n:%s) ASSERT n.id IS UNIQUE", label),
		}
		for _, property := range nodeIndexes[label] {
			statements = append(statements, fmt.Sprintf("CREATE INDEX ON :%s(%s)", label, property))
		}
		for _, statement := range statements {
			result, err := session.Run(ctx, statement, nil)
			if err != nil {
				return fmt.Errorf("initialize memgraph schema for %s: %w", label, err)
			}
			if _, err := result.Consume(ctx); err != nil {
				return fmt.Errorf("initialize memgraph schema for %s: %w", label, err)
			}
		}
	}
	// These old declarations are node indexes (and have no matching nodes), not
	// relationship indexes. Their names look plausible in SHOW INDEX INFO, which
	// allowed the mistake to survive unnoticed.
	for _, statement := range []string{
		"DROP INDEX ON :FLEW_WITH(weight)",
		"DROP INDEX ON :KILLED(weight)",
	} {
		result, err := session.Run(ctx, statement, nil)
		if err != nil {
			return fmt.Errorf("remove obsolete memgraph index: %w", err)
		}
		if _, err := result.Consume(ctx); err != nil {
			return fmt.Errorf("remove obsolete memgraph index: %w", err)
		}
	}
	for _, edgeType := range EdgeTypes {
		result, err := session.Run(ctx,
			fmt.Sprintf("CREATE EDGE INDEX ON :%s(last_seen)", edgeType), nil)
		if err != nil {
			return fmt.Errorf("initialize memgraph schema for %s: %w", edgeType, err)
		}
		if _, err := result.Consume(ctx); err != nil {
			return fmt.Errorf("initialize memgraph schema for %s: %w", edgeType, err)
		}
	}
	return nil
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
	c.ingestMu.Lock()
	defer c.ingestMu.Unlock()

	session := c.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx) //nolint:errcheck // best-effort close

	// All mutations for one killmail share a transaction. The marker makes the
	// operation idempotent across duplicate jobs, retries, and ambiguous commit
	// outcomes; a partial ingest is rolled back rather than retried on top of
	// already-incremented counters.
	token := uuid.New().String()
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		fresh, err := claimKillmail(ctx, tx, km.KillmailID, token)
		if err != nil || !fresh {
			return nil, err
		}
		if err := ingest(ctx, tx, km); err != nil {
			return nil, err
		}
		return nil, finishKillmail(ctx, tx, km.KillmailID, token, km.KillmailTime)
	})
	if err != nil {
		return fmt.Errorf("ingest killmail %d: %w", km.KillmailID, err)
	}
	return nil
}

type queryRunner interface {
	Run(context.Context, string, map[string]any) (neo4j.ResultWithContext, error)
}

func claimKillmail(ctx context.Context, tx queryRunner, id int64, token string) (bool, error) {
	if id <= 0 {
		return false, fmt.Errorf("killmail id must be positive")
	}
	result, err := tx.Run(ctx, `
		MERGE (k:Killmail {id: $id})
		ON CREATE SET k.ingest_token = $token, k.ingested = false
		RETURN k.ingest_token = $token AND NOT k.ingested AS fresh`, map[string]any{
		"id": id, "token": token,
	})
	if err != nil {
		return false, fmt.Errorf("claim killmail: %w", err)
	}
	if !result.Next(ctx) {
		if err := result.Err(); err != nil {
			return false, fmt.Errorf("claim killmail: %w", err)
		}
		return false, fmt.Errorf("claim killmail returned no result")
	}
	fresh, ok := result.Record().Values[0].(bool)
	if !ok {
		return false, fmt.Errorf("claim killmail returned %T, want boolean", result.Record().Values[0])
	}
	return fresh, nil
}

func finishKillmail(ctx context.Context, tx queryRunner, id int64, token string, eventAt time.Time) error {
	result, err := tx.Run(ctx, `
		MATCH (k:Killmail {id: $id})
		WHERE k.ingest_token = $token AND NOT k.ingested
		SET k.ingested = true, k.ingested_at = $ingested_at,
		    k.killmail_time = $killmail_time
		REMOVE k.ingest_token
		RETURN count(k) AS finished`, map[string]any{
		"id": id, "token": token,
		"ingested_at": isoTimestamp(time.Now()), "killmail_time": isoTimestamp(eventAt),
	})
	if err != nil {
		return fmt.Errorf("finish killmail: %w", err)
	}
	if !result.Next(ctx) {
		if err := result.Err(); err != nil {
			return fmt.Errorf("finish killmail: %w", err)
		}
		return fmt.Errorf("finish killmail returned no result")
	}
	finished, ok := result.Record().Values[0].(int64)
	if !ok || finished != 1 {
		return fmt.Errorf("finish killmail updated %v markers, want 1", result.Record().Values[0])
	}
	return nil
}

func ingest(ctx context.Context, tx queryRunner, km Killmail) error {

	at := isoTimestamp(km.KillmailTime)

	// Nodes before edges: a MATCH on a node that does not exist silently
	// matches nothing, so the edge would be dropped without an error.
	if len(km.Characters) > 0 {
		chars := make([]map[string]any, 0, len(km.Characters))
		corps := map[int64]int64{}
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
				corps[ch.CorporationID] = ch.AllianceID
			}
			if ch.AllianceID != 0 {
				alliances[ch.AllianceID] = true
			}
		}

		if _, err := tx.Run(ctx, `
            UNWIND $chars AS c
            MERGE (n:Character {id: c.id})
			SET n.corporation_id = CASE
			        WHEN $time > coalesce(n.last_seen, '') OR
			             ($time = n.last_seen AND $killmail_id > coalesce(n.last_seen_killmail_id, 0))
			        THEN c.corp
			        ELSE n.corporation_id END,
			    n.alliance_id = CASE
			        WHEN $time > coalesce(n.last_seen, '') OR
			             ($time = n.last_seen AND $killmail_id > coalesce(n.last_seen_killmail_id, 0))
			        THEN c.alliance
			        ELSE n.alliance_id END,
			    n.last_seen_killmail_id = CASE
			        WHEN $time > coalesce(n.last_seen, '') OR
			             ($time = n.last_seen AND $killmail_id > coalesce(n.last_seen_killmail_id, 0))
			        THEN $killmail_id ELSE n.last_seen_killmail_id END,
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
			map[string]any{"chars": chars, "time": at, "killmail_id": km.KillmailID}); err != nil {
			return fmt.Errorf("merge character nodes: %w", err)
		}

		if err := mergeCorporations(ctx, tx, corps, at, km.KillmailID); err != nil {
			return err
		}
		if err := mergeIDs(ctx, tx, "Alliance", alliances); err != nil {
			return err
		}
	}

	if km.SolarSystemID != 0 {
		if _, err := tx.Run(ctx, `MERGE (:SolarSystem {id: $id})`,
			map[string]any{"id": km.SolarSystemID}); err != nil {
			return fmt.Errorf("merge solar system: %w", err)
		}
	}

	if len(km.FlewWith) > 0 {
		items := make([]map[string]any, 0, len(km.FlewWith))
		for _, e := range km.FlewWith {
			items = append(items, map[string]any{"lo": e.Lo, "hi": e.Hi})
		}
		if _, err := tx.Run(ctx, `
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
		if _, err := tx.Run(ctx, `
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
		if _, err := tx.Run(ctx, `
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
			if _, err := tx.Run(ctx, `
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
			if _, err := tx.Run(ctx, `
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

func mergeCorporations(
	ctx context.Context,
	tx queryRunner,
	corporations map[int64]int64,
	at string,
	killmailID int64,
) error {
	if len(corporations) == 0 {
		return nil
	}
	items := make([]map[string]any, 0, len(corporations))
	for corporationID, allianceID := range corporations {
		items = append(items, map[string]any{
			"id": corporationID, "alliance": nullable(allianceID),
		})
	}
	_, err := tx.Run(ctx, `
		UNWIND $items AS item
		MERGE (corp:Corporation {id: item.id})
		SET corp.alliance_id = CASE
		        WHEN $time > coalesce(corp.last_seen, '') OR
		             ($time = corp.last_seen AND $killmail_id > coalesce(corp.last_seen_killmail_id, 0))
		        THEN item.alliance ELSE corp.alliance_id END,
		    corp.last_seen_killmail_id = CASE
		        WHEN $time > coalesce(corp.last_seen, '') OR
		             ($time = corp.last_seen AND $killmail_id > coalesce(corp.last_seen_killmail_id, 0))
		        THEN $killmail_id ELSE corp.last_seen_killmail_id END,
		    corp.last_seen = CASE
		        WHEN $time > coalesce(corp.last_seen, '') THEN $time
		        ELSE corp.last_seen END`, map[string]any{
		"items": items, "time": at, "killmail_id": killmailID,
	})
	if err != nil {
		return fmt.Errorf("merge corporation nodes: %w", err)
	}
	return nil
}

func mergeIDs(ctx context.Context, tx queryRunner, label string, ids map[int64]bool) error {
	if len(ids) == 0 {
		return nil
	}
	list := make([]int64, 0, len(ids))
	for id := range ids {
		list = append(list, id)
	}
	// The label comes from this package's fixed call sites, never from input.
	if _, err := tx.Run(ctx,
		fmt.Sprintf(`UNWIND $ids AS id MERGE (:%s {id: id})`, label),
		map[string]any{"ids": list}); err != nil {
		return fmt.Errorf("merge %s nodes: %w", label, err)
	}
	return nil
}

// PurgeResult reports what a prune removed.
type PurgeResult struct {
	Edges     int64            `json:"edges"`
	Orphans   int64            `json:"orphans"`
	Killmails int64            `json:"killmails"`
	ByType    map[string]int64 `json:"by_type"`
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
	if batchSize < 1 {
		return out, fmt.Errorf("purge batch size must be positive")
	}

	session := c.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx) //nolint:errcheck // best-effort close

	cutoff := isoTimestamp(time.Now().UTC().AddDate(0, 0, -RetentionDays))

	for _, edgeType := range EdgeTypes {
		for range purgeBatchesPerEdgeType {
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
			out.ByType[edgeType] += n
			if n < int64(batchSize) {
				break
			}
		}
		if out.ByType[edgeType] == 0 {
			delete(out.ByType, edgeType)
		}
	}

	res, err := session.Run(ctx, `
		MATCH (k:Killmail)
		WHERE k.killmail_time < $cutoff
		WITH k LIMIT $limit
		DELETE k
		RETURN count(k) AS cnt`, map[string]any{
		"cutoff": cutoff, "limit": batchSize,
	})
	if err != nil {
		return out, fmt.Errorf("purge killmail markers: %w", err)
	}
	out.Killmails = countFrom(ctx, res)

	// Orphans only after the edges are gone. Killmail markers are intentionally
	// excluded because they have their own event-time retention above.
	res, err = session.Run(ctx, `
		MATCH (n)
		WHERE (n:Character OR n:Corporation OR n:Alliance OR n:SolarSystem)
		  AND NOT (n)-[]-()
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
