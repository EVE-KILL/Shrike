package config

import (
	"os"
	"path/filepath"
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
		"REDIS_CACHE_HOST", "REDIS_CACHE_PORT", "VALKEY_QUEUE", "VALKEY_CACHE",
		"MEMGRAPH_URL", "PORT",
		"EVE_CLIENT_ID", "EVE_CLIENT_SECRET", "EVE_CALLBACK_URL", "ESI_USER_AGENT",
		"B2_ENDPOINT", "B2_MEDIA_BUCKET", "B2_IMAGES_BUCKET",
		"B2_KEY_ID", "B2_APP_KEY", "IMAGE_CACHE_BYTES",
		"NODE_ENV", "GIT_SHA", "LOG_LEVEL", "LOG_FORMAT",
		"NUXT_SOCKET", "DATA_DIR",
	} {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
	// Load walks upward from cwd looking for .env; run from a scratch directory
	// so the repository's own files are never picked up.
	t.Chdir(t.TempDir())
}

func TestB2BucketsShareCredentialsButConfigureIndependently(t *testing.T) {
	clearEnv(t)
	c, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if c.B2Configured() || c.B2PartiallyConfigured() {
		t.Fatalf("empty B2 config = complete %t, partial %t",
			c.B2Configured(), c.B2PartiallyConfigured())
	}

	t.Setenv("B2_ENDPOINT", "https://s3.example.test")
	c, err = Load("")
	if err != nil {
		t.Fatal(err)
	}
	if c.B2Configured() || !c.B2PartiallyConfigured() {
		t.Fatalf("partial B2 config = complete %t, partial %t",
			c.B2Configured(), c.B2PartiallyConfigured())
	}

	t.Setenv("B2_MEDIA_BUCKET", "evekill-media")
	t.Setenv("B2_IMAGES_BUCKET", "evekill-images")
	t.Setenv("B2_KEY_ID", "key-id")
	t.Setenv("B2_APP_KEY", "application-key")
	c, err = Load("")
	if err != nil {
		t.Fatal(err)
	}
	if !c.B2Configured() || c.B2PartiallyConfigured() {
		t.Fatalf("complete B2 config = complete %t, partial %t",
			c.B2Configured(), c.B2PartiallyConfigured())
	}
	if !c.B2ImagesConfigured() || c.B2ImagesPartiallyConfigured() {
		t.Fatalf("image B2 config = complete %t, partial %t",
			c.B2ImagesConfigured(), c.B2ImagesPartiallyConfigured())
	}

	t.Setenv("B2_MEDIA_BUCKET", "")
	c, err = Load("")
	if err != nil {
		t.Fatal(err)
	}
	if c.B2Configured() || c.B2PartiallyConfigured() {
		t.Fatalf("omitted media bucket = complete %t, partial %t",
			c.B2Configured(), c.B2PartiallyConfigured())
	}
	if !c.B2ImagesConfigured() {
		t.Fatal("omitting the media bucket disabled the image bucket")
	}
}

func TestImageCacheBytesIsBoundedConfiguration(t *testing.T) {
	clearEnv(t)
	t.Setenv("IMAGE_CACHE_BYTES", "268435456")
	c, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if c.ImageCacheBytes != 256<<20 {
		t.Fatalf("ImageCacheBytes = %d, want %d", c.ImageCacheBytes, 256<<20)
	}

	t.Setenv("IMAGE_CACHE_BYTES", "-1")
	c, err = Load("")
	if err != nil {
		t.Fatal(err)
	}
	if c.ImageCacheBytes != 1<<30 {
		t.Fatalf("negative ImageCacheBytes = %d, want default", c.ImageCacheBytes)
	}
}

func writeDotenv(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	return path
}
