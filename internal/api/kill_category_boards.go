package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/eve-kill/shrike/internal/killtype"
	"github.com/redis/go-redis/v9"
)

const (
	killCategoryBoardPrefix = "shrike:kill-category-top:v1:"
	killCategoryBoardCursor = killCategoryBoardPrefix + "cursor"
	killCategoryBoardTTL    = 2 * time.Hour
)

var killCategoryTopTypes = [...]string{"characters", "corporations", "alliances"}

func (r *HomepageReader) preparedKillCategoryTop(ctx context.Context, kind string) map[string][]TopEntry {
	if r.redis == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 150*time.Millisecond)
	defer cancel()
	body, err := r.redis.Get(ctx, killCategoryBoardPrefix+kind).Bytes()
	if err != nil {
		return nil
	}
	var boards map[string][]TopEntry
	if json.Unmarshal(body, &boards) != nil {
		return nil
	}
	return boards
}

// PrepareNextKillCategoryTop refreshes one category per run. All three boards
// share a cutoff and are published together only after every read succeeds.
func PrepareNextKillCategoryTop(ctx context.Context, db Database, client *redis.Client) (string, error) {
	sequence, err := client.Incr(ctx, killCategoryBoardCursor).Uint64()
	if err != nil {
		return "", fmt.Errorf("advance kill-category leaderboard cursor: %w", err)
	}
	kind := killtype.Types[(sequence-1)%uint64(len(killtype.Types))]
	since := time.Now().UTC().AddDate(0, 0, -KillCategoryTopDays)
	boards := make(map[string][]TopEntry, len(killCategoryTopTypes))
	for _, dataType := range killCategoryTopTypes {
		boards[dataType], err = loadTopKillEntries(ctx, db, kind, dataType, KillCategoryTopLimit, since)
		if err != nil {
			return kind, fmt.Errorf("prepare %s %s leaderboard: %w", kind, dataType, err)
		}
	}
	body, err := json.Marshal(boards)
	if err != nil {
		return kind, err
	}
	return kind, client.Set(ctx, killCategoryBoardPrefix+kind, body, killCategoryBoardTTL).Err()
}
