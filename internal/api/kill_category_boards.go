package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/eve-kill/shrike/internal/killtype"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

const (
	killCategoryBoardPrefix  = "shrike:kill-category-top:v2:"
	killCategoryBoardTTL     = 10 * time.Minute
	killCategoryCacheTimeout = 150 * time.Millisecond
)

var killCategoryTopEntities = [...]string{"character", "corporation", "alliance"}

func usesPreparedKillCategoryTop(kind, dataType string, days, limit int) bool {
	if days != KillCategoryTopDays || limit != KillCategoryTopLimit {
		return false
	}
	_, known := killtype.Predicates()[kind]
	return known && (dataType == "characters" || dataType == "corporations" || dataType == "alliances")
}

func (r *HomepageReader) preparedKillCategoryTop(ctx context.Context, kind string) (map[string][]TopEntry, error) {
	key := killCategoryBoardPrefix + kind
	var cacheMiss bool
	if r.redis != nil {
		cacheCtx, cancel := context.WithTimeout(ctx, killCategoryCacheTimeout)
		body, err := r.redis.Get(cacheCtx, key).Bytes()
		cancel()
		cacheMiss = errors.Is(err, redis.Nil)
		if err == nil {
			var boards map[string][]TopEntry
			if json.Unmarshal(body, &boards) == nil && len(boards) == len(killCategoryTopEntities) {
				return boards, nil
			}
		}
	}
	if r.db == nil {
		return nil, nil
	}
	var body []byte
	err := r.db.QueryRow(ctx, `SELECT boards FROM kill_category_top_snapshots WHERE kind = $1`, kind).Scan(&body)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var boards map[string][]TopEntry
	if err := json.Unmarshal(body, &boards); err != nil {
		return nil, err
	}
	if cacheMiss {
		cacheCtx, cancel := context.WithTimeout(ctx, killCategoryCacheTimeout)
		defer cancel()
		// A refresh may have published a newer snapshot during the read.
		_ = r.redis.SetNX(cacheCtx, key, body, killCategoryBoardTTL).Err()
	}
	return boards, nil
}

// PrepareKillCategoryTop replaces the last successful snapshot only after all
// three boards have been built. Valkey is a cache for the saved result.
func PrepareKillCategoryTop(ctx context.Context, db MutationDatabase, client *redis.Client, kind string) error {
	if _, known := killtype.Predicates()[kind]; !known {
		return fmt.Errorf("unknown kill category %q", kind)
	}
	preparedAt := time.Now().UTC()
	boards, err := loadKillCategoryTopBoards(ctx, db, kind, preparedAt.AddDate(0, 0, -KillCategoryTopDays))
	if err != nil {
		return err
	}
	body, err := json.Marshal(boards)
	if err != nil {
		return err
	}
	tag, err := db.Exec(ctx, `
		INSERT INTO kill_category_top_snapshots (kind, prepared_at, boards) VALUES ($1, $2, $3)
		ON CONFLICT (kind) DO UPDATE SET prepared_at = EXCLUDED.prepared_at, boards = EXCLUDED.boards
		WHERE kill_category_top_snapshots.prepared_at < EXCLUDED.prepared_at`, kind, preparedAt, body)
	if err != nil {
		return err
	}
	if tag.RowsAffected() > 0 && client != nil {
		cacheCtx, cancel := context.WithTimeout(ctx, killCategoryCacheTimeout)
		defer cancel()
		_ = client.Set(cacheCtx, killCategoryBoardPrefix+kind, body, killCategoryBoardTTL).Err()
	}
	return nil
}

func loadKillCategoryTopBoards(ctx context.Context, db Database, kind string, since time.Time) (map[string][]TopEntry, error) {
	boards := make(map[string][]TopEntry, len(killCategoryTopEntities))
	switch kind {
	case "latest", "solo", "npc", "pvp":
		// Use the existing stats and victim queries for special boards. The
		// independent plans also beat materialization for the broad PvP set.
		for _, entity := range killCategoryTopEntities {
			key := entity + "s"
			entries, err := loadTopKillEntries(ctx, db, kind, key, KillCategoryTopLimit, since)
			if err != nil {
				return nil, fmt.Errorf("load %s leaderboard: %w", key, err)
			}
			boards[key] = entries
		}
		return boards, nil
	}
	// Scan the qualifying kills and attackers once. Keep the original distinct
	// kill count per entity and require the entity to exist in the same registry.
	parts := make([]string, 0, len(killCategoryTopEntities))
	for _, entity := range killCategoryTopEntities {
		column := entity + "_id"
		key := entity + "s"
		boards[key] = []TopEntry{}
		parts = append(parts, fmt.Sprintf(`(
			SELECT '%[1]s' AS type, a.%[2]s AS id, e.name, COUNT(DISTINCT a.killmail_id)::bigint AS count
			FROM eligible_attackers a JOIN %[3]s e ON e.%[2]s = a.%[2]s
			GROUP BY a.%[2]s, e.name ORDER BY count DESC, id LIMIT $2
		)`, entity, column, key))
	}
	rows, err := queryMaps(ctx, db, `
		WITH eligible_attackers AS MATERIALIZED (
			SELECT a.killmail_id, a.character_id, a.corporation_id, a.alliance_id
			FROM killmails k JOIN killmail_attackers a ON a.killmail_id = k.killmail_id
			WHERE k.killmail_time >= $1 AND a.killmail_time >= $1 AND (`+killtype.Predicates()[kind]+`)
		)
		`+strings.Join(parts, ` UNION ALL `)+` ORDER BY type, count DESC, id`, since, KillCategoryTopLimit)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if stringOrEmpty(row["name"]) == "" {
			row["name"] = "Unknown"
		}
	}
	rows, err = attachGlobalStatsPalettes(ctx, db, rows)
	if err != nil {
		return nil, err
	}
	entries, err := decodeHomepageRows[TopEntry](nonNilUniverseRows(rows))
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		key := entry.Type + "s"
		boards[key] = append(boards[key], entry)
	}
	return boards, nil
}
