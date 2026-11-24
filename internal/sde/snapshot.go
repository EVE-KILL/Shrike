package sde

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Snapshot struct {
	BuildNumber   int64
	ReleaseDate   string
	Release       string
	TableVersions map[string]string
}

// Current checks both release identity and the set of tables this importer owns.
func (s Snapshot) Current(manifest *Manifest) bool {
	return s.Release == manifest.Release &&
		s.hasTables(snapshotFor(manifest, importMembers(AllTables(), true)).TableVersions)
}

func (s Snapshot) hasTables(versions map[string]string) bool {
	return len(s.TableVersions) != 0 && maps.Equal(s.TableVersions, versions)
}

func snapshotFor(manifest *Manifest, members []string) Snapshot {
	versions := make(map[string]string, len(members))
	for _, name := range members {
		versions[name] = manifest.Tables[name].Version
	}
	return Snapshot{
		BuildNumber: manifest.BuildNumber, ReleaseDate: manifest.ReleaseDate.Format(time.RFC3339Nano),
		Release: manifest.Release, TableVersions: versions,
	}
}

// Record all fields together. Keep the existing build and date keys for
// diagnostics and compatibility with databases populated by the ZIP importer.
func recordSnapshot(ctx context.Context, pool *pgxpool.Pool, snapshot Snapshot) error {
	versions, err := json.Marshal(snapshot.TableVersions)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO config (key, value) VALUES
			('sde:buildNumber', $1),
			('sde:releaseDate', $2),
			('sde:release', $3),
			('sde:tableVersions', $4)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
	`, strconv.FormatInt(snapshot.BuildNumber, 10), snapshot.ReleaseDate, snapshot.Release, string(versions))
	return err
}

func clearTableVersions(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `DELETE FROM config WHERE key = 'sde:tableVersions'`)
	return err
}

func LoadedSnapshot(ctx context.Context, pool *pgxpool.Pool) (Snapshot, error) {
	var snapshot Snapshot
	var buildText, versions string
	err := pool.QueryRow(ctx, `
		SELECT
			COALESCE(MAX(value) FILTER (WHERE key = 'sde:buildNumber'), '0'),
			COALESCE(MAX(value) FILTER (WHERE key = 'sde:releaseDate'), ''),
			COALESCE(MAX(value) FILTER (WHERE key = 'sde:release'), ''),
			COALESCE(MAX(value) FILTER (WHERE key = 'sde:tableVersions'), '')
		FROM config
		WHERE key IN ('sde:buildNumber', 'sde:releaseDate', 'sde:release', 'sde:tableVersions')
	`).Scan(&buildText, &snapshot.ReleaseDate, &snapshot.Release, &versions)
	if err != nil {
		return Snapshot{}, err
	}
	snapshot.BuildNumber, _ = strconv.ParseInt(buildText, 10, 64)
	if versions != "" {
		if err := json.Unmarshal([]byte(versions), &snapshot.TableVersions); err != nil {
			return Snapshot{}, fmt.Errorf("parse loaded SDE table versions: %w", err)
		}
	}
	return snapshot, nil
}
