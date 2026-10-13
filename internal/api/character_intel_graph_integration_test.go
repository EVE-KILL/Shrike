package api

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/eve-kill/shrike/internal/graph"
)

type plannedIntelGraph struct {
	*graph.Client
	queries []string
}

func (g *plannedIntelGraph) Read(ctx context.Context, query string, params map[string]any) ([]map[string]any, error) {
	g.queries = append(g.queries, query)
	return g.Client.Read(ctx, query, params)
}

func TestCharacterGraphIntelAnchorsLookupsAndRetainsResults(t *testing.T) {
	origin := os.Getenv("TEST_MEMGRAPH_URL")
	if origin == "" {
		t.Skip("TEST_MEMGRAPH_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	client, err := graph.Connect(ctx, origin)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close(context.Background()) })
	if err := client.Clear(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Clear(context.Background()) })
	if err := client.EnsureSchema(ctx); err != nil {
		t.Fatal(err)
	}
	_, err = client.Read(ctx, `
		CREATE (c:Character {id: 7, corporation_id: 70, alliance_id: 700}),
		       (a:Character {id: 8, corporation_id: 80, alliance_id: 800}),
		       (b:Character {id: 9, corporation_id: 80, alliance_id: 800}),
		       (d:Character {id: 10, corporation_id: 90, alliance_id: 900}),
		       (e:Character {id: 11, corporation_id: 100}),
		       (c)-[:FLEW_WITH {weight: 9}]->(a),
		       (b)-[:FLEW_WITH {weight: 7}]->(c),
		       (c)-[:FLEW_WITH {weight: 5}]->(d),
		       (e)-[:FLEW_WITH {weight: 2}]->(c)`, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Unrelated alliance-bearing pilots must never become the traversal's
	// starting set. They made the old plan expand unrelated fleet edges.
	_, err = client.Read(ctx, `UNWIND range(1000,1999) AS id
		CREATE (:Character {id:id, alliance_id:800})`, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, batch := range []bool{false, true} {
		name := "single"
		if batch {
			name = "batch"
		}
		t.Run(name, func(t *testing.T) {
			g := &plannedIntelGraph{Client: client}
			var got graphIntel
			if batch {
				rows := loadGraphIntelBatch(ctx, g, []int32{7, 42})
				got = rows[7]
				if rows[42].BridgeScore != 0 || len(rows[42].FleetPartners) != 0 || rows[42].Timestamps == nil {
					t.Errorf("missing character result = %#v", rows[42])
				}
			} else {
				got = loadGraphIntel(ctx, g, 7)
			}
			if len(got.FleetPartners) != 4 || got.FleetPartners[0]["id"] != int64(8) ||
				got.BridgeScore != 2 || len(got.GroupsFlownWith) != 2 ||
				got.GroupsFlownWith[0]["alliance_id"] != int64(800) ||
				got.GroupsFlownWith[0]["shared_partners"] != int64(2) {
				t.Fatalf("fleet partners or alliance counts changed: %#v", got)
			}
			params := map[string]any{"id": int64(7), "ids": []int64{7, 42}}
			for _, query := range g.queries {
				if !strings.Contains(query, "FLEW_WITH") {
					continue
				}
				plan, err := client.Read(ctx, "EXPLAIN "+query, params)
				if err != nil {
					t.Fatal(err)
				}
				indexed := false
				for _, row := range plan {
					step, _ := row["QUERY PLAN"].(string)
					if strings.Contains(step, "ScanAllByLabelProperties") && strings.Contains(step, "c :Character {id}") {
						indexed = true
					}
					if strings.Contains(step, "ScanAll (c)") || strings.Contains(step, "ScanAllByLabelProperties (p") {
						t.Errorf("fleet traversal starts outside the requested pilot: %s", step)
					}
				}
				if !indexed {
					t.Errorf("fleet traversal lacks indexed pilot lookup: %#v", plan)
				}
			}
		})
	}
}
