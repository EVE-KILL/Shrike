// Package config loads runtime configuration from the environment, using the
// exact variable names the existing Bun services already read. The Go binary
// has to be a drop-in for those pods, so no new names are invented here and
// no defaults diverge from backend/src/database/redis.ts or api/src/db.ts.
package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Source records where a value came from, so `config show` can explain itself.
// Debugging "why is it connecting to localhost" is otherwise guesswork.
type Source string

const (
	SourceEnv     Source = "env"
	SourceDotenv  Source = ".env"
	SourceDefault Source = "default"
)

type Config struct {
	// Postgres. api/src/db.ts falls back to a local socket URL; we keep the
	// same default so `doctor` behaves identically to the TS services.
	DatabaseURL string

	// Queue Redis — REDIS_* is the canonical set (backend/src/database/redis.ts).
	RedisHost      string
	RedisPort      int
	RedisPassword  string
	RedisDB        int
	ValkeyQueueURL string

	// Cache Redis. Separate host/port with fallback to the queue instance,
	// mirroring getCacheRedis(). Password and DB are deliberately shared:
	// the TS implementation does not override them either.
	RedisCacheHost string
	RedisCachePort int
	ValkeyCacheURL string

	// Memgraph, over Bolt.
	MemgraphURL string

	// EVE SSO / ESI. The client credentials are the production application's
	// even locally: refresh tokens in user_esi_tokens are bound to the issuing
	// client_id and cannot be exchanged by a different application.
	EVEClientID     string
	EVEClientSecret string
	EVECallbackURL  string

	// ESIUserAgent identifies us to CCP. Required by their acceptable-use
	// policy, and the static-data endpoints may throttle requests without it.
	ESIUserAgent string

	// S3-compatible Backblaze B2 storage used for custom-domain images.
	// Names match frontend/server/utils/storage.ts so the Go process can take
	// over without changing deployment secrets.
	B2Endpoint     string
	B2MediaBucket  string
	B2ImagesBucket string
	B2KeyID        string
	B2AppKey       string

	// ImageCacheBytes bounds the encoded-response LRU owned by `shrike serve`.
	// It is a ceiling, not a reservation, and is deliberately not used by
	// workers or import commands.
	ImageCacheBytes int64

	// HTTP listener for whatever `serve` subcommand is running.
	Port int

	// NuxtEntrypoint is the built Nitro server module supervised by `serve`.
	// Empty auto-discovers web/.output/server/index.mjs in a source checkout.
	NuxtEntrypoint string

	// NuxtSocket is where the supervised Nitro renderer listens. Every request
	// matching no Go-owned surface is proxied there.
	NuxtSocket string

	// APISocket is the private HTTP listener used by Nuxt SSR. Browser requests
	// still use relative same-origin HTTP through Caddy.
	APISocket string

	// DataDir is where the ingress keeps its own state — certificates, once
	// Shrike terminates TLS.
	DataDir string

	// Observability / environment.
	NodeEnv  string
	GitSHA   string
	LogLevel string
	LogJSON  bool

	// sources maps field name -> where its value came from.
	sources map[string]Source
}

// Load reads .env files (nearest first, walking up from cwd) and then the
// process environment, which always wins. Missing values fall back to the same
// defaults the TS services use.
func Load(explicitPath string) (*Config, error) {
	dotenv, dotenvPath, err := loadDotenv(explicitPath)
	if err != nil {
		return nil, err
	}

	c := &Config{sources: map[string]Source{}}
	get := func(field, key, def string) string {
		if v, ok := os.LookupEnv(key); ok && v != "" {
			c.sources[field] = SourceEnv
			return v
		}
		if v, ok := dotenv[key]; ok && v != "" {
			c.sources[field] = SourceDotenv
			return v
		}
		c.sources[field] = SourceDefault
		return def
	}
	getInt := func(field, key string, def int) int {
		raw := get(field, key, strconv.Itoa(def))
		n, convErr := strconv.Atoi(raw)
		if convErr != nil {
			// A malformed port is a config bug worth surfacing, but not worth
			// aborting startup over — fall back and mark it as a default so
			// `config show` reveals the value was not honoured.
			c.sources[field] = SourceDefault
			return def
		}
		return n
	}
	getInt64 := func(field, key string, def int64) int64 {
		raw := get(field, key, strconv.FormatInt(def, 10))
		n, convErr := strconv.ParseInt(raw, 10, 64)
		if convErr != nil || n < 0 {
			c.sources[field] = SourceDefault
			return def
		}
		return n
	}

	c.DatabaseURL = get("DatabaseURL", "DATABASE_URL", "postgresql://localhost:5432/evekill")
	c.RedisHost = get("RedisHost", "REDIS_HOST", "localhost")
	c.RedisPort = getInt("RedisPort", "REDIS_PORT", 6379)
	c.RedisPassword = get("RedisPassword", "REDIS_PASSWORD", "")
	c.RedisDB = getInt("RedisDB", "REDIS_DB", 0)
	c.ValkeyQueueURL = get("ValkeyQueueURL", "VALKEY_QUEUE", "")

	// Fall back to the queue instance when the cache-specific vars are unset,
	// exactly as getCacheRedis() does.
	c.RedisCacheHost = get("RedisCacheHost", "REDIS_CACHE_HOST", c.RedisHost)
	c.RedisCachePort = getInt("RedisCachePort", "REDIS_CACHE_PORT", c.RedisPort)
	c.ValkeyCacheURL = get("ValkeyCacheURL", "VALKEY_CACHE", "")

	c.MemgraphURL = get("MemgraphURL", "MEMGRAPH_URL", "bolt://memgraph:7687")

	c.EVEClientID = get("EVEClientID", "EVE_CLIENT_ID", "")
	c.EVEClientSecret = get("EVEClientSecret", "EVE_CLIENT_SECRET", "")
	c.EVECallbackURL = get("EVECallbackURL", "EVE_CALLBACK_URL", "")
	c.ESIUserAgent = get("ESIUserAgent", "ESI_USER_AGENT", "")
	c.B2Endpoint = get("B2Endpoint", "B2_ENDPOINT", "")
	c.B2MediaBucket = get("B2MediaBucket", "B2_MEDIA_BUCKET", "")
	c.B2ImagesBucket = get("B2ImagesBucket", "B2_IMAGES_BUCKET", "")
	c.B2KeyID = get("B2KeyID", "B2_KEY_ID", "")
	c.B2AppKey = get("B2AppKey", "B2_APP_KEY", "")
	c.ImageCacheBytes = getInt64(
		"ImageCacheBytes",
		"IMAGE_CACHE_BYTES",
		1<<30,
	)
	c.Port = getInt("Port", "PORT", 4000)

	c.NuxtEntrypoint = get("NuxtEntrypoint", "NUXT_ENTRYPOINT", "")
	c.NuxtSocket = get("NuxtSocket", "NUXT_SOCKET", "")
	c.APISocket = get("APISocket", "SHRIKE_API_SOCKET", "")
	c.DataDir = get("DataDir", "DATA_DIR", "./data")
	c.NodeEnv = get("NodeEnv", "NODE_ENV", "development")
	c.GitSHA = get("GitSHA", "GIT_SHA", "unknown")
	c.LogLevel = get("LogLevel", "LOG_LEVEL", "info")
	c.LogJSON = strings.EqualFold(get("LogFormat", "LOG_FORMAT", "console"), "json")

	if dotenvPath != "" {
		c.sources["_dotenvPath"] = Source(dotenvPath)
	}
	return c, nil
}

// SourceOf reports where a field's value came from.
func (c *Config) SourceOf(field string) Source {
	if s, ok := c.sources[field]; ok {
		return s
	}
	return SourceDefault
}

// DotenvPath returns the .env file that was loaded, or "" if none was found.
func (c *Config) DotenvPath() string {
	return string(c.sources["_dotenvPath"])
}

// RedisAddr and RedisCacheAddr build host:port pairs for the Redis clients.
func (c *Config) RedisAddr() string {
	return fmt.Sprintf("%s:%d", c.RedisHost, c.RedisPort)
}

func (c *Config) RedisCacheAddr() string {
	return fmt.Sprintf("%s:%d", c.RedisCacheHost, c.RedisCachePort)
}

// IsProduction reports whether we are running with production semantics.
// NODE_ENV is checked rather than a Go-specific variable because the whole
// deployment already keys off it.
func (c *Config) IsProduction() bool {
	return c.NodeEnv == "production"
}

// B2Configured reports whether the complete existing frontend storage
// contract is available. B2PartiallyConfigured distinguishes a deliberately
// disabled local setup from a deployment typo that should fail loudly.
func (c *Config) B2Configured() bool {
	return c.b2CredentialsConfigured() && c.B2MediaBucket != ""
}

func (c *Config) B2PartiallyConfigured() bool {
	return c.B2MediaPartiallyConfigured()
}

// B2ImagesConfigured reports whether the dedicated image bucket and shared B2
// credentials are complete.
func (c *Config) B2ImagesConfigured() bool {
	return c.b2CredentialsConfigured() && c.B2ImagesBucket != ""
}

// B2MediaPartiallyConfigured and B2ImagesPartiallyConfigured distinguish a
// deliberately omitted bucket from incomplete shared credentials. Once the
// endpoint and credentials are complete, either bucket may independently be
// disabled by leaving its name empty.
func (c *Config) B2MediaPartiallyConfigured() bool {
	return c.b2CredentialsPartiallyConfigured() ||
		(c.B2MediaBucket != "" && !c.b2CredentialsConfigured())
}

func (c *Config) B2ImagesPartiallyConfigured() bool {
	return c.b2CredentialsPartiallyConfigured() ||
		(c.B2ImagesBucket != "" && !c.b2CredentialsConfigured())
}

func (c *Config) b2CredentialsConfigured() bool {
	return c.B2Endpoint != "" && c.B2KeyID != "" && c.B2AppKey != ""
}

func (c *Config) b2CredentialsPartiallyConfigured() bool {
	configured := 0
	for _, value := range []string{
		c.B2Endpoint, c.B2KeyID, c.B2AppKey,
	} {
		if value != "" {
			configured++
		}
	}
	return configured > 0 && configured < 3
}

// loadDotenv finds and parses a .env file. With an explicit path, a missing
// file is an error — the user asked for that file specifically. Without one we
// walk up from cwd and silently accept finding nothing, since production pods
// inject real environment variables and have no .env at all.
func loadDotenv(explicitPath string) (map[string]string, string, error) {
	if explicitPath != "" {
		vals, err := parseDotenv(explicitPath)
		if err != nil {
			return nil, "", fmt.Errorf("read config %s: %w", explicitPath, err)
		}
		return vals, explicitPath, nil
	}

	dir, err := os.Getwd()
	if err != nil {
		return map[string]string{}, "", nil
	}
	for {
		candidate := filepath.Join(dir, ".env")
		if _, statErr := os.Stat(candidate); statErr == nil {
			vals, parseErr := parseDotenv(candidate)
			if parseErr != nil {
				return nil, "", fmt.Errorf("read config %s: %w", candidate, parseErr)
			}
			return vals, candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return map[string]string{}, "", nil
		}
		dir = parent
	}
}

// parseDotenv handles the subset of dotenv syntax the repo actually uses:
// KEY=value, # comments, optional `export` prefix, and quoted values.
func parseDotenv(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	vals := map[string]string{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}
		vals[key] = value
	}
	return vals, scanner.Err()
}
