package sde

import (
	"context"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Derivation is a post-import fill for a column the archive does not carry
// directly but that can be computed from data already loaded.
//
// These run after every table is in place, because they read across tables.
type Derivation struct {
	Name        string
	Description string
	SQL         string
}

// DeriveResult reports one derivation.
type DeriveResult struct {
	Name    string `json:"name"`
	Rows    int64  `json:"rows"`
	Elapsed string `json:"elapsed"`
}

// Derivations are ordered: solar_systems must be loaded before stations can
// inherit region and constellation from it.
var Derivations = []Derivation{
	{
		Name:        "inv_types.category_id",
		Description: "Denormalise category from the type's group",
		// inv_types has a category_id column but the SDE type record only carries
		// groupID. Storing it denormalised avoids a join on the hottest lookup in
		// the codebase — every killmail resolves a ship's category.
		//
		// Written as a correlated subquery rather than UPDATE ... FROM
		// inv_groups. A join only touches rows that match, so it can set a value
		// but never clear a stale one: a type whose group has disappeared would
		// keep the category it was given by an earlier import. Three types
		// (48464, 48465, 49782) reference group_id 0, which does not exist, and
		// must end up NULL.
		SQL: `
            UPDATE inv_types t
            SET category_id = (
                SELECT g.category_id FROM inv_groups g WHERE g.group_id = t.group_id
            )
            WHERE t.category_id IS DISTINCT FROM (
                SELECT g.category_id FROM inv_groups g WHERE g.group_id = t.group_id
            )
        `,
	},
	{
		Name:        "stations.region_id / constellation_id",
		Description: "Inherit region and constellation from the station's system",
		// npcStations.jsonl carries only solarSystemID; region and constellation
		// are looked up rather than stored by CCP. Subqueries for the same reason
		// as above — a station whose system vanished must lose its region, not
		// keep a stale one.
		SQL: `
            UPDATE stations s
            SET region_id = (
                    SELECT sys.region_id FROM solar_systems sys
                    WHERE sys.solar_system_id = s.solar_system_id
                ),
                constellation_id = (
                    SELECT sys.constellation_id FROM solar_systems sys
                    WHERE sys.solar_system_id = s.solar_system_id
                )
            WHERE s.region_id IS DISTINCT FROM (
                    SELECT sys.region_id FROM solar_systems sys
                    WHERE sys.solar_system_id = s.solar_system_id
                )
               OR s.constellation_id IS DISTINCT FROM (
                    SELECT sys.constellation_id FROM solar_systems sys
                    WHERE sys.solar_system_id = s.solar_system_id
                )
        `,
	},
}

// StationNameDerivation is separate from Derivations because it depends on
// celestials, which are imported after the declarative tables. Verified against
// production: stations.station_name is identical to the station's celestial name
// for all 5,210 rows.
var StationNameDerivation = Derivation{
	Name:        "stations.station_name",
	Description: "Copy the composed name from the station's celestial row",
	SQL: `
        UPDATE stations s
        SET station_name = (
            SELECT c.item_name FROM celestials c WHERE c.item_id = s.station_id
        )
        WHERE s.station_name IS DISTINCT FROM (
            SELECT c.item_name FROM celestials c WHERE c.item_id = s.station_id
        )
    `,
}

// DeriveOne runs a single derivation.
func DeriveOne(ctx context.Context, pool *pgxpool.Pool, d Derivation) (DeriveResult, error) {
	start := time.Now()
	tag, err := pool.Exec(ctx, d.SQL)
	return DeriveResult{
		Name:    d.Name,
		Rows:    tag.RowsAffected(),
		Elapsed: time.Since(start).Round(time.Millisecond).String(),
	}, err
}

// Derive runs every derivation in order.
func Derive(ctx context.Context, pool *pgxpool.Pool) ([]DeriveResult, error) {
	out := make([]DeriveResult, 0, len(Derivations))
	for _, d := range Derivations {
		start := time.Now()
		tag, err := pool.Exec(ctx, d.SQL)
		if err != nil {
			return out, err
		}
		out = append(out, DeriveResult{
			Name:    d.Name,
			Rows:    tag.RowsAffected(),
			Elapsed: time.Since(start).Round(time.Millisecond).String(),
		})
	}
	return out, nil
}

// RecordBuild stores the imported build number in the config table so
// sde:status can report what is loaded and the cron can skip an unchanged build.
func RecordBuild(ctx context.Context, pool *pgxpool.Pool, build int64, release string) error {
	// config is (key text, value text) with no timestamp column, so the build
	// number is stored as text. The key matches what the TypeScript importer
	// already writes in production ('sde:buildNumber'), so both implementations
	// read and write the same row rather than tracking the build separately.
	_, err := pool.Exec(ctx, `
        INSERT INTO config (key, value) VALUES
            ('sde:buildNumber', $1),
            ('sde:releaseDate', $2)
        ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
    `, strconv.FormatInt(build, 10), release)
	return err
}

// LoadedBuild reads back what RecordBuild stored. Returns 0 when nothing has
// been imported yet.
func LoadedBuild(ctx context.Context, pool *pgxpool.Pool) (int64, string, error) {
	var buildText, release string
	err := pool.QueryRow(ctx, `
        SELECT
            COALESCE(MAX(value) FILTER (WHERE key = 'sde:buildNumber'), '0'),
            COALESCE(MAX(value) FILTER (WHERE key = 'sde:releaseDate'), '')
        FROM config
        WHERE key IN ('sde:buildNumber', 'sde:releaseDate')
    `).Scan(&buildText, &release)
	if err != nil {
		return 0, "", err
	}
	// Stored as text, and a non-numeric value should read as "nothing imported"
	// rather than failing the whole status command.
	build, convErr := strconv.ParseInt(buildText, 10, 64)
	if convErr != nil {
		return 0, release, nil
	}
	return build, release, nil
}
