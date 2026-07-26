package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestLoadDefaultsMatchTypeScriptServices(t *testing.T) {
	// The Go binary has to be a drop-in for the Bun pods, so these defaults
	// must not drift from backend/src/database/redis.ts and api/src/db.ts.
	clearEnv(t)

	c, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if c.RedisHost != "localhost" {
		t.Errorf("RedisHost = %q, want localhost", c.RedisHost)
	}
	if c.RedisPort != 6379 {
		t.Errorf("RedisPort = %d, want 6379", c.RedisPort)
	}
	if c.RedisDB != 0 {
		t.Errorf("RedisDB = %d, want 0", c.RedisDB)
	}
	if c.MemgraphURL != "bolt://memgraph:7687" {
		t.Errorf("MemgraphURL = %q, want bolt://memgraph:7687", c.MemgraphURL)
	}
	if c.Port != 4000 {
		t.Errorf("Port = %d, want 4000", c.Port)
	}
}

func TestCacheRedisFallsBackToQueueInstance(t *testing.T) {
	// getCacheRedis() falls back host-then-port to the queue instance; losing
	// that would silently split the cache onto the wrong Redis.
	clearEnv(t)
	t.Setenv("REDIS_HOST", "queue.internal")
	t.Setenv("REDIS_PORT", "6380")

	c, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.RedisCacheHost != "queue.internal" {
		t.Errorf("RedisCacheHost = %q, want the queue host", c.RedisCacheHost)
	}
	if c.RedisCachePort != 6380 {
		t.Errorf("RedisCachePort = %d, want the queue port", c.RedisCachePort)
	}

	// An explicit cache host must win, while the port still falls back.
	t.Setenv("REDIS_CACHE_HOST", "cache.internal")
	c, err = Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.RedisCacheHost != "cache.internal" {
		t.Errorf("RedisCacheHost = %q, want cache.internal", c.RedisCacheHost)
	}
	if c.RedisCachePort != 6380 {
		t.Errorf("RedisCachePort = %d, want the queue port fallback", c.RedisCachePort)
	}
}

func TestEnvBeatsDotenv(t *testing.T) {
	clearEnv(t)
	path := writeDotenv(t, "DATABASE_URL=postgresql://from-dotenv/db\nPORT=1111\n")

	t.Setenv("PORT", "2222")

	c, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.Port != 2222 {
		t.Errorf("Port = %d, want the environment to win", c.Port)
	}
	if c.SourceOf("Port") != SourceEnv {
		t.Errorf("Port source = %q, want env", c.SourceOf("Port"))
	}
	if c.DatabaseURL != "postgresql://from-dotenv/db" {
		t.Errorf("DatabaseURL = %q, want the .env value", c.DatabaseURL)
	}
	if c.SourceOf("DatabaseURL") != SourceDotenv {
		t.Errorf("DatabaseURL source = %q, want .env", c.SourceOf("DatabaseURL"))
	}
	if c.SourceOf("MemgraphURL") != SourceDefault {
		t.Errorf("MemgraphURL source = %q, want default", c.SourceOf("MemgraphURL"))
	}
}

func TestDotenvParsing(t *testing.T) {
	clearEnv(t)
	path := writeDotenv(t, strings.Join([]string{
		"# a comment",
		"",
		"DATABASE_URL=postgresql://user:pw@host:5432/db",
		`REDIS_PASSWORD="quoted secret"`,
		"export PORT=9999",
		"MALFORMED_LINE_WITHOUT_EQUALS",
		"REDIS_HOST = spaced.host ",
	}, "\n"))

	c, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.RedisPassword != "quoted secret" {
		t.Errorf("RedisPassword = %q, want the unquoted value", c.RedisPassword)
	}
	if c.Port != 9999 {
		t.Errorf("Port = %d, want the export-prefixed value", c.Port)
	}
	if c.RedisHost != "spaced.host" {
		t.Errorf("RedisHost = %q, want whitespace trimmed", c.RedisHost)
	}
}

func TestExplicitMissingConfigIsAnError(t *testing.T) {
	// Falling back silently would leave the user connected somewhere they did
	// not ask for, which is the worst outcome for a --config typo.
	clearEnv(t)
	if _, err := Load(filepath.Join(t.TempDir(), "absent.env")); err == nil {
		t.Fatal("Load with a missing explicit path should error")
	}
}

func TestMalformedIntFallsBackToDefault(t *testing.T) {
	clearEnv(t)
	t.Setenv("REDIS_PORT", "not-a-number")

	c, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.RedisPort != 6379 {
		t.Errorf("RedisPort = %d, want the 6379 fallback", c.RedisPort)
	}
	// Reported as a default so `config show` reveals the value was ignored.
	if c.SourceOf("RedisPort") != SourceDefault {
		t.Errorf("RedisPort source = %q, want default", c.SourceOf("RedisPort"))
	}
}

func TestRedactURL(t *testing.T) {
	tests := []struct{ in, want string }{
		{
			// The mask must not be percent-encoded into unreadability.
			in:   "postgresql://evekill:s3cr3t@10.0.0.1:5432/evekill",
			want: "postgresql://evekill:REDACTED@10.0.0.1:5432/evekill",
		},
		{
			in:   "postgresql://localhost:5432/evekill",
			want: "postgresql://localhost:5432/evekill",
		},
		{
			in:   "bolt://memgraph:7687",
			want: "bolt://memgraph:7687",
		},
		{in: "", want: ""},
	}
	for _, tc := range tests {
		if got := RedactURL(tc.in); got != tc.want {
			t.Errorf("RedactURL(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestRedactURLNeverLeaksOnParseFailure(t *testing.T) {
	// An unparseable DSN is exactly where a password is most likely to sit in
	// an unexpected position, so nothing of the input may be echoed back.
	const raw = "postgres://user:hunter2@%%%bad-host/db"
	got := RedactURL(raw)
	if strings.Contains(got, "hunter2") {
		t.Fatalf("RedactURL leaked the password: %q", got)
	}
}

func TestRedactSecret(t *testing.T) {
	// Short secrets are masked whole — revealing a prefix of a 6-character
	// password gives away most of it.
	if got := RedactSecret("short"); got != "*****" {
		t.Errorf("RedactSecret(short) = %q, want full masking", got)
	}
	got := RedactSecret("averylongsecretvalue")
	if strings.Contains(got, "verylongsecretvalu") {
		t.Errorf("RedactSecret leaked too much: %q", got)
	}
	if len(got) != len("averylongsecretvalue") {
		t.Errorf("RedactSecret(%q) changed length: %q", "averylongsecretvalue", got)
	}
	if got := RedactSecret(""); got != "" {
		t.Errorf("RedactSecret(empty) = %q, want empty", got)
	}
}

// clearEnv removes every variable Load reads so a developer's real environment
// cannot influence the result. t.Setenv restores originals on cleanup.
func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"DATABASE_URL", "REDIS_HOST", "REDIS_PORT", "REDIS_PASSWORD", "REDIS_DB",
		"REDIS_CACHE_HOST", "REDIS_CACHE_PORT", "MEMGRAPH_URL", "PORT",
		"NODE_ENV", "GIT_SHA", "LOG_LEVEL", "LOG_FORMAT",
		"PUBLIC_API_HOST", "IMAGES_HOST", "NUXT_SOCKET", "DATA_DIR",
	} {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
	// Load walks upward from cwd looking for .env; run from a scratch directory
	// so the repository's own files are never picked up.
	t.Chdir(t.TempDir())
}

func writeDotenv(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	return path
}

// A list variable and a scalar variable must disagree about what an empty value
// means. For a scalar, empty is a mistake and the default is the safer answer;
// for a list, the empty list is a real value — PUBLIC_API_HOST= is how an
// operator says "do not serve that surface", and quietly restoring the default
// would have the process claim a hostname that was just taken away from it.
func TestEmptyListVariableMeansNoEntries(t *testing.T) {
	clearEnv(t)
	t.Setenv("PUBLIC_API_HOST", "")
	t.Setenv("IMAGES_HOST", "images.example.com, ,images.localhost")

	c, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(c.PublicAPIHosts) != 0 {
		t.Errorf("PublicAPIHosts = %q, want none", c.PublicAPIHosts)
	}
	if got, want := c.ImagesHosts, []string{"images.example.com", "images.localhost"}; !slices.Equal(got, want) {
		t.Errorf("ImagesHosts = %q, want %q", got, want)
	}
}

// The production defaults must not carry development aliases: a pod would then
// answer to hostnames nobody deployed it under.
func TestDefaultHostsAreProductionOnly(t *testing.T) {
	clearEnv(t)

	c, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, hosts := range [][]string{c.PublicAPIHosts, c.ImagesHosts} {
		for _, h := range hosts {
			if strings.HasSuffix(h, ".localhost") {
				t.Errorf("default hosts include the development alias %q", h)
			}
		}
	}
}
