package workers

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/eve-kill/shrike/internal/graph"
	"github.com/eve-kill/shrike/internal/killmail"
	"github.com/eve-kill/shrike/internal/queue"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

// GraphIngestWorker records one killmail's relationships.
type GraphIngestWorker struct {
	river.WorkerDefaults[queue.GraphIngestArgs]
	Deps *Deps
}

const (
	graphNPCCharacterMax = 4_000_000
	graphNPCCorpMin      = 1_000_000
	graphNPCCorpMax      = 2_000_000

	graphSmartbombGroup    = 72
	graphMonitorGroup      = 1972
	graphSupercarrierGroup = 659
	graphBlackOpsGroup     = 898

	graphKillwhoreThreshold = 0.05
)

var graphWhitelistedShipGroups = map[int32]bool{
	1972: true, // Monitor
	832:  true, // logistics cruiser
	1527: true, // logistics frigate
	1534: true, // command destroyer
	540:  true, // command ship
	1538: true, // force auxiliary
	547:  true, // carrier
	659:  true, // supercarrier
	30:   true, // titan
	485:  true, // dreadnought
	898:  true, // black ops
}

var graphCapitalShipGroups = map[int32]bool{
	547:  true,
	30:   true,
	485:  true,
	1538: true,
}

var graphLogisticsShipGroups = map[int32]bool{
	832:  true,
	1527: true,
}

var graphSmartbombTypes struct {
	sync.RWMutex
	ids map[int32]bool
}

func (w *GraphIngestWorker) Work(ctx context.Context, job *river.Job[queue.GraphIngestArgs]) error {
	if w.Deps.Graph == nil {
		// No Memgraph configured. The graph is entirely derived and can be
		// rebuilt from the killmails, so this is a degraded mode rather than a
		// failure.
		return nil
	}

	p, err := killmail.Load(ctx, w.Deps.Pool, job.Args.KillmailID)
	if errors.Is(err, killmail.ErrNotStored) {
		return nil
	}
	if err != nil {
		return err
	}
	if p.Killmail.IsNPC {
		return nil
	}

	smartbombs, err := loadGraphSmartbombTypes(ctx, w.Deps.Pool)
	if err != nil {
		return err
	}
	km, ok := buildGraphKillmail(p, smartbombs)
	if !ok {
		return nil
	}
	return w.Deps.Graph.Ingest(ctx, km)
}

func buildGraphKillmail(p *killmail.Parsed, smartbombs map[int32]bool) (graph.Killmail, bool) {
	if p == nil || p.Killmail.IsNPC {
		return graph.Killmail{}, false
	}

	km := graph.Killmail{
		KillmailID:    p.Killmail.KillmailID,
		KillmailTime:  p.Killmail.KillmailTime,
		SolarSystemID: int64(p.Killmail.SolarSystemID),
	}

	var totalDamage int64
	for _, attacker := range p.Attackers {
		totalDamage += int64(attacker.DamageDone)
	}

	playerAttackers := make([]killmail.Attacker, 0, len(p.Attackers))
	for _, attacker := range p.Attackers {
		if !graphPlayerCharacter(attacker.CharacterID) ||
			graphNPCCorporation(attacker.CorporationID) ||
			smartbombs[attacker.WeaponTypeID] {
			continue
		}
		if !graphWhitelistedShipGroups[attacker.ShipGroupID] &&
			totalDamage > 0 &&
			float64(attacker.DamageDone)/float64(totalDamage) < graphKillwhoreThreshold {
			continue
		}
		playerAttackers = append(playerAttackers, attacker)
	}
	if len(playerAttackers) == 0 {
		return graph.Killmail{}, false
	}

	characters := make(map[int32]graph.Character)
	addCharacter := func(id, corp, alliance, shipGroup int32) {
		if !graphPlayerCharacter(id) || graphNPCCorporation(corp) {
			return
		}
		ch, exists := characters[id]
		if !exists {
			ch = graph.Character{
				ID:            int64(id),
				CorporationID: int64(corp),
				AllianceID:    int64(alliance),
			}
		}
		tagGraphArchetypes(&ch, shipGroup, p.Killmail.KillmailTime)
		characters[id] = ch
	}

	victim := p.Killmail.VictimCharacterID
	victimEligible := graphPlayerCharacter(victim) &&
		!graphNPCCorporation(p.Killmail.VictimCorporationID)
	if victimEligible {
		addCharacter(
			victim,
			p.Killmail.VictimCorporationID,
			p.Killmail.VictimAllianceID,
			p.Killmail.VictimShipGroupID,
		)
	}

	for _, attacker := range playerAttackers {
		addCharacter(
			attacker.CharacterID,
			attacker.CorporationID,
			attacker.AllianceID,
			attacker.ShipGroupID,
		)
		if victimEligible && attacker.CharacterID != victim {
			var finalBlows int64
			if attacker.FinalBlow {
				finalBlows = 1
			}
			km.Killed = append(km.Killed, graph.KilledEdge{
				AttackerID:   int64(attacker.CharacterID),
				VictimID:     int64(victim),
				IskDestroyed: p.Killmail.TotalValue,
				FinalBlows:   finalBlows,
			})
		}
	}

	km.Characters = make([]graph.Character, 0, len(characters))
	for _, ch := range characters {
		km.Characters = append(km.Characters, ch)
	}
	km.FlewWith = flewWithAttackerPairs(playerAttackers, p.Killmail.VictimAllianceID)
	return km, true
}

// flewWithPairs builds the symmetric pairs among a kill's attackers.
//
// Each pair is stored once with the lower id first: the relationship is
// symmetric, and storing both directions would double every weight.
func flewWithPairs(ids []int32) []graph.FlewWithEdge {
	unique := map[int32]bool{}
	for _, id := range ids {
		if id != 0 {
			unique[id] = true
		}
	}
	if len(unique) < 2 {
		return nil
	}

	list := make([]int32, 0, len(unique))
	for id := range unique {
		list = append(list, id)
	}
	// Sorted so the pairs are deterministic and always ordered low-high.
	slices.Sort(list)

	out := make([]graph.FlewWithEdge, 0, len(list)*(len(list)-1)/2)
	for i := range list {
		for j := i + 1; j < len(list); j++ {
			out = append(out, graph.FlewWithEdge{Lo: int64(list[i]), Hi: int64(list[j])})
		}
	}
	return out
}

func flewWithAttackerPairs(
	attackers []killmail.Attacker,
	victimAllianceID int32,
) []graph.FlewWithEdge {
	ids := make([]int32, 0, len(attackers))
	seen := make(map[int32]bool)
	for _, attacker := range attackers {
		if victimAllianceID != 0 && attacker.AllianceID == victimAllianceID {
			continue
		}
		if !seen[attacker.CharacterID] {
			seen[attacker.CharacterID] = true
			ids = append(ids, attacker.CharacterID)
		}
	}
	return flewWithPairs(ids)
}

func tagGraphArchetypes(ch *graph.Character, shipGroup int32, at time.Time) {
	switch shipGroup {
	case graphMonitorGroup:
		ch.LastFCSeen = at
	case graphSupercarrierGroup:
		ch.LastSuperKill = at
	case graphBlackOpsGroup:
		ch.LastBlopsSeen = at
	}
	if graphCapitalShipGroups[shipGroup] {
		ch.LastCapitalKill = at
	}
	if graphLogisticsShipGroups[shipGroup] {
		ch.LastLogisticsSeen = at
	}
}

func graphPlayerCharacter(id int32) bool {
	return id >= graphNPCCharacterMax
}

func graphNPCCorporation(id int32) bool {
	return id >= graphNPCCorpMin && id <= graphNPCCorpMax
}

func loadGraphSmartbombTypes(ctx context.Context, pool *pgxpool.Pool) (map[int32]bool, error) {
	graphSmartbombTypes.RLock()
	cached := graphSmartbombTypes.ids
	graphSmartbombTypes.RUnlock()
	if cached != nil {
		return cached, nil
	}

	rows, err := pool.Query(ctx, `SELECT type_id FROM inv_types WHERE group_id = $1`, graphSmartbombGroup)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	loaded := make(map[int32]bool)
	for rows.Next() {
		var id int32
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		loaded[id] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	graphSmartbombTypes.Lock()
	if graphSmartbombTypes.ids == nil {
		graphSmartbombTypes.ids = loaded
	}
	cached = graphSmartbombTypes.ids
	graphSmartbombTypes.Unlock()
	return cached, nil
}

// cronGraphPurge prunes relationships that have aged out.
func (d *Deps) cronGraphPurge(ctx context.Context) (string, error) {
	if d.Graph == nil {
		return "", nil
	}

	// Batched so one run cannot build a transaction large enough to exhaust
	// Memgraph's memory. A partial prune is fine — the next run continues.
	const batch = 10_000

	res, err := d.Graph.Purge(ctx, batch)
	if err != nil {
		return "", err
	}
	if res.Edges == 0 && res.Orphans == 0 {
		return "", nil
	}
	return fmt.Sprintf("%d edges and %d orphaned characters pruned", res.Edges, res.Orphans), nil
}
