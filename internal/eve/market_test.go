package eve

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Market paths end up in user-visible URLs on every killlist row, so a
// difference between this and the TypeScript that produces the same links is a
// broken link rather than a cosmetic difference.

func TestSlugifyMatchesTheTypeScript(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Ships", "ships"},
		{"Battleships", "battleships"},
		// The ampersand becomes "and" before punctuation is stripped, so this
		// is not "ammunition-charges".
		{"Ammunition & Charges", "ammunition-and-charges"},
		{"Ship Equipment", "ship-equipment"},
		{"Drones & Fighters", "drones-and-fighters"},
		// Leading and trailing separators are trimmed rather than left as
		// empty path segments.
		{"  Spaced  ", "spaced"},
		{"---Dashes---", "dashes"},
		{"Tech II", "tech-ii"},
		{"", ""},
	}

	for _, c := range cases {
		if got := Slugify(c.in); got != c.want {
			t.Errorf("Slugify(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func marketTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgresql://evekill:" + "evekill@127.0.0.1:5432/evekill"
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Skipf("no test database: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("no test database: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// The real tree, against the real SDE.
func TestMarketPathsResolveAgainstTheSDE(t *testing.T) {
	pool := marketTestPool(t)
	ctx := context.Background()

	paths, err := LoadMarketPaths(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Skip("inv_market_groups is empty — the SDE is not imported")
	}

	cache, err := Load(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	if cache.CountsByName()["inv_types"] == 0 {
		t.Skip("the SDE is not imported")
	}

	// Every path is rooted and has no empty segments — an empty one would
	// render as a double slash and a dead link.
	for id, p := range paths {
		if len(p) < len("/market/") || p[:len("/market/")] != "/market/" {
			t.Fatalf("market group %d resolved to %q, which is not rooted at /market/", id, p)
		}
		if contains(p, "//") {
			t.Fatalf("market group %d resolved to %q, which has an empty segment", id, p)
		}
	}

	// A hull that is sold on the market must resolve to a path.
	for _, typeID := range []int32{587 /* Rifter */, 17738 /* Machariel */, 11567 /* Avatar */} {
		got := paths.PathForType(cache, typeID)
		if got == "" {
			ty, _ := cache.Type(typeID)
			t.Errorf("type %d (%s, market group %d) has no market path — the killlist "+
				"row would render without its ship link", typeID, ty.Name, ty.MarketGroupID)
		}
	}
}

// A pod has no market group, because you cannot buy one. That must be an empty
// path rather than "/market/" or a lookup failure.
func TestTypesWithNoMarketGroupResolveToNothing(t *testing.T) {
	pool := marketTestPool(t)
	ctx := context.Background()

	paths, err := LoadMarketPaths(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	cache, err := Load(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	if cache.CountsByName()["inv_types"] == 0 {
		t.Skip("the SDE is not imported")
	}

	const capsule = 670
	ty, ok := cache.Type(capsule)
	if !ok {
		t.Skip("the Capsule is not in the loaded SDE")
	}
	if ty.MarketGroupID != 0 {
		t.Skipf("the Capsule now has market group %d, so it is no longer the "+
			"no-market-group example this test relies on", ty.MarketGroupID)
	}

	if got := paths.PathForType(cache, capsule); got != "" {
		t.Errorf("the Capsule resolved to %q, want an empty path", got)
	}
	if got := paths.Path(0); got != "" {
		t.Errorf("market group 0 resolved to %q, want an empty path", got)
	}
}

// An unknown type must not resolve to anything, rather than panicking or
// returning a root path.
func TestUnknownTypeHasNoPath(t *testing.T) {
	pool := marketTestPool(t)
	ctx := context.Background()

	paths, err := LoadMarketPaths(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	cache, err := Load(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}

	if got := paths.PathForType(cache, 2_000_000_000); got != "" {
		t.Errorf("an unknown type resolved to %q", got)
	}
	if got := paths.PathForType(nil, 587); got != "" {
		t.Errorf("a nil cache resolved to %q", got)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
