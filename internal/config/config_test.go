package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
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
	if c.APICacheBytes != 256<<20 {
		t.Errorf("APICacheBytes = %d, want %d", c.APICacheBytes, 256<<20)
	}
	if c.MemgraphURL != "bolt://memgraph:7687" {
		t.Errorf("MemgraphURL = %q, want bolt://memgraph:7687", c.MemgraphURL)
	}
	if c.Port != 4000 {
		t.Errorf("Port = %d, want 4000", c.Port)
	}
	if c.DatabaseMaxConnections != 10 {
		t.Errorf("DatabaseMaxConnections = %d, want 10", c.DatabaseMaxConnections)
	}
	if c.DatabaseReadURL != c.DatabaseURL {
		t.Errorf("DatabaseReadURL = %q, want DATABASE_URL fallback %q", c.DatabaseReadURL, c.DatabaseURL)
	}
	if c.DatabaseReadMaxConnections != 10 {
		t.Errorf("DatabaseReadMaxConnections = %d, want 10", c.DatabaseReadMaxConnections)
	}
}

func TestReadDatabaseCanBeConfiguredSeparately(t *testing.T) {
	clearEnv(t)
	t.Setenv("DATABASE_URL", "postgresql://writer@primary/evekill")
	t.Setenv("DATABASE_READ_URL", "postgresql://reader@replicas/evekill")
	t.Setenv("DB_MAX_CONNS", "7")
	t.Setenv("DB_READ_MAX_CONNS", "19")

	c, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.DatabaseReadURL != "postgresql://reader@replicas/evekill" {
		t.Errorf("DatabaseReadURL = %q", c.DatabaseReadURL)
	}
	if c.DatabaseMaxConnections != 7 || c.DatabaseReadMaxConnections != 19 {
		t.Errorf("database connection limits = write:%d read:%d, want 7 and 19",
			c.DatabaseMaxConnections, c.DatabaseReadMaxConnections)
	}
}

func TestSharedValkeyUsesOneAddress(t *testing.T) {
	clearEnv(t)
	t.Setenv("REDIS_HOST", "valkey.internal")
	t.Setenv("REDIS_PORT", "6380")
	// These transition-era variables must not silently split cache traffic
	// from coordination traffic again.
	t.Setenv("REDIS_CACHE_HOST", "ignored-cache.internal")
	t.Setenv("REDIS_CACHE_PORT", "6381")
	t.Setenv("VALKEY_QUEUE", "redis://ignored-queue.internal:6382")
	t.Setenv("VALKEY_CACHE", "redis://ignored-cache.internal:6383")

	c, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.RedisHost != "valkey.internal" {
		t.Errorf("RedisHost = %q, want valkey.internal", c.RedisHost)
	}
	if c.RedisPort != 6380 {
		t.Errorf("RedisPort = %d, want 6380", c.RedisPort)
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
		"DATABASE_URL", "DATABASE_READ_URL", "DB_MAX_CONNS", "DB_READ_MAX_CONNS",
		"REDIS_HOST", "REDIS_PORT", "REDIS_PASSWORD", "REDIS_DB",
		"REDIS_CACHE_HOST", "REDIS_CACHE_PORT", "VALKEY_QUEUE", "VALKEY_CACHE",
		"MEMGRAPH_URL", "PORT",
		"EVE_CLIENT_ID", "EVE_CLIENT_SECRET", "EVE_CALLBACK_URL", "ESI_USER_AGENT",
		"OPENAI_API_KEY", "NUXT_OPENAI_API_KEY",
		"KLIPY_API_KEY", "NUXT_KLIPY_API_KEY",
		"B2_ENDPOINT", "B2_MEDIA_BUCKET", "B2_IMAGES_BUCKET",
		"B2_KEY_ID", "B2_APP_KEY", "IMAGE_STORAGE_PATH", "IMAGE_CACHE_BYTES", "API_CACHE_BYTES",
		"NODE_ENV", "GIT_SHA", "LOG_LEVEL", "LOG_FORMAT",
		"NUXT_ENTRYPOINT", "NUXT_SOCKET", "SHRIKE_API_SOCKET", "DATA_DIR",
	} {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
	// Load walks upward from cwd looking for .env; run from a scratch directory
	// so the repository's own files are never picked up.
	t.Chdir(t.TempDir())
}

func TestThirdPartyAPIKeysLoadFromCanonicalAndLegacyNames(t *testing.T) {
	clearEnv(t)
	path := writeDotenv(t, "NUXT_OPENAI_API_KEY=legacy-openai\nNUXT_KLIPY_API_KEY=legacy-klipy\n")

	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.OpenAIAPIKey != "legacy-openai" || c.KlipyAPIKey != "legacy-klipy" {
		t.Fatalf(
			"legacy keys = OpenAI %q, Klipy %q",
			c.OpenAIAPIKey,
			c.KlipyAPIKey,
		)
	}

	t.Setenv("OPENAI_API_KEY", "canonical-openai")
	t.Setenv("KLIPY_API_KEY", "canonical-klipy")
	c, err = Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.OpenAIAPIKey != "canonical-openai" ||
		c.KlipyAPIKey != "canonical-klipy" {
		t.Fatalf(
			"canonical keys = OpenAI %q, Klipy %q",
			c.OpenAIAPIKey,
			c.KlipyAPIKey,
		)
	}
}

func TestRendererSocketConfiguration(t *testing.T) {
	clearEnv(t)
	t.Setenv("NUXT_ENTRYPOINT", "/app/web/server/index.mjs")
	t.Setenv("NUXT_SOCKET", "/tmp/frontend.sock")
	t.Setenv("SHRIKE_API_SOCKET", "/tmp/api.sock")

	c, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if c.NuxtEntrypoint != "/app/web/server/index.mjs" {
		t.Fatalf("NuxtEntrypoint = %q", c.NuxtEntrypoint)
	}
	if c.NuxtSocket != "/tmp/frontend.sock" {
		t.Fatalf("NuxtSocket = %q", c.NuxtSocket)
	}
	if c.APISocket != "/tmp/api.sock" {
		t.Fatalf("APISocket = %q", c.APISocket)
	}
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
