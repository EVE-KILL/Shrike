package cli

import (
	"fmt"
	"strconv"

	"github.com/eve-kill/shrike/internal/config"
	"github.com/eve-kill/shrike/internal/ui"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Inspect configuration",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show resolved configuration and where each value came from",
	Long: `Prints every setting with its effective value and origin — process
environment, a .env file, or the built-in default.

Credentials are redacted. The SOURCE column is the point of this command:
"why is it talking to localhost" is otherwise pure guesswork.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := requireConfig(); err != nil {
			return err
		}

		rows := []struct {
			field string
			key   string
			value string
		}{
			{"DatabaseURL", "DATABASE_URL", config.RedactURL(cfg.DatabaseURL)},
			{"DatabaseMaxConnections", "DB_MAX_CONNS", strconv.Itoa(cfg.DatabaseMaxConnections)},
			{"DatabaseReadURL", "DATABASE_READ_URL", config.RedactURL(cfg.DatabaseReadURL)},
			{"DatabaseReadMaxConnections", "DB_READ_MAX_CONNS", strconv.Itoa(cfg.DatabaseReadMaxConnections)},
			{"RedisHost", "REDIS_HOST", cfg.RedisHost},
			{"RedisPort", "REDIS_PORT", strconv.Itoa(cfg.RedisPort)},
			{"RedisPassword", "REDIS_PASSWORD", config.RedactSecret(cfg.RedisPassword)},
			{"RedisDB", "REDIS_DB", strconv.Itoa(cfg.RedisDB)},
			{"MemgraphURL", "MEMGRAPH_URL", cfg.MemgraphURL},
			{"OpenAIAPIKey", "OPENAI_API_KEY", config.RedactSecret(cfg.OpenAIAPIKey)},
			{"KlipyAPIKey", "KLIPY_API_KEY", config.RedactSecret(cfg.KlipyAPIKey)},
			{"B2Endpoint", "B2_ENDPOINT", cfg.B2Endpoint},
			{"B2MediaBucket", "B2_MEDIA_BUCKET", cfg.B2MediaBucket},
			{"B2ImagesBucket", "B2_IMAGES_BUCKET", cfg.B2ImagesBucket},
			{"B2KeyID", "B2_KEY_ID", config.RedactSecret(cfg.B2KeyID)},
			{"B2AppKey", "B2_APP_KEY", config.RedactSecret(cfg.B2AppKey)},
			{"ImageStoragePath", "IMAGE_STORAGE_PATH", cfg.ImageStoragePath},
			{"ImageCacheBytes", "IMAGE_CACHE_BYTES", strconv.FormatInt(cfg.ImageCacheBytes, 10)},
			{"APICacheBytes", "API_CACHE_BYTES", strconv.FormatInt(cfg.APICacheBytes, 10)},
			{"Port", "PORT", strconv.Itoa(cfg.Port)},
			{"NodeEnv", "NODE_ENV", cfg.NodeEnv},
			{"GitSHA", "GIT_SHA", cfg.GitSHA},
			{"LogLevel", "LOG_LEVEL", cfg.LogLevel},
			{"SentryDSN", "SENTRY_DSN", config.RedactSecret(cfg.SentryDSN)},
			{"SentryEnvironment", "SENTRY_ENVIRONMENT", cfg.SentryEnvironment},
			{"SentryTracesSampleRate", "SENTRY_TRACES_SAMPLE_RATE", strconv.FormatFloat(cfg.SentryTracesSampleRate, 'f', -1, 64)},
			{"SentryLogs", "SENTRY_LOGS", strconv.FormatBool(cfg.SentryLogs)},
			{"SentryMetrics", "SENTRY_METRICS", strconv.FormatBool(cfg.SentryMetrics)},
			{"SentryLogLevel", "SENTRY_LOG_LEVEL", cfg.SentryLogLevel},
			{"SentryLogSampleRate", "SENTRY_LOG_SAMPLE_RATE", strconv.FormatFloat(cfg.SentryLogSampleRate, 'f', -1, 64)},
		}

		if ui.JSONMode {
			out := make(map[string]map[string]string, len(rows))
			for _, r := range rows {
				out[r.key] = map[string]string{
					"value":  r.value,
					"source": string(cfg.SourceOf(r.field)),
				}
			}
			return ui.JSON(map[string]any{
				"dotenv": cfg.DotenvPath(),
				"config": out,
			})
		}

		ui.Section("Configuration")
		table := ui.NewTable("VARIABLE", "VALUE", "SOURCE")
		for _, r := range rows {
			source := string(cfg.SourceOf(r.field))
			// Defaults are the ones worth noticing when something is wrong,
			// so they are the only source dimmed rather than plain.
			if source == string(config.SourceDefault) {
				source = ui.Dim(source)
			}
			value := r.value
			if value == "" {
				value = ui.Dim("(empty)")
			}
			table.Row(r.key, value, source)
		}
		fmt.Println(table.Render())

		if path := cfg.DotenvPath(); path != "" {
			ui.Newline()
			ui.KV("Loaded .env", path)
		} else {
			ui.Newline()
			ui.KV("Loaded .env", ui.Dim("none found"))
		}
		ui.Newline()
		return nil
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
}
