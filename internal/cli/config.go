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
			{"RedisHost", "REDIS_HOST", cfg.RedisHost},
			{"RedisPort", "REDIS_PORT", strconv.Itoa(cfg.RedisPort)},
			{"RedisPassword", "REDIS_PASSWORD", config.RedactSecret(cfg.RedisPassword)},
			{"RedisDB", "REDIS_DB", strconv.Itoa(cfg.RedisDB)},
			{"ValkeyQueueURL", "VALKEY_QUEUE", config.RedactURL(cfg.ValkeyQueueURL)},
			{"RedisCacheHost", "REDIS_CACHE_HOST", cfg.RedisCacheHost},
			{"RedisCachePort", "REDIS_CACHE_PORT", strconv.Itoa(cfg.RedisCachePort)},
			{"ValkeyCacheURL", "VALKEY_CACHE", config.RedactURL(cfg.ValkeyCacheURL)},
			{"MemgraphURL", "MEMGRAPH_URL", cfg.MemgraphURL},
			{"B2Endpoint", "B2_ENDPOINT", cfg.B2Endpoint},
			{"B2MediaBucket", "B2_MEDIA_BUCKET", cfg.B2MediaBucket},
			{"B2ImagesBucket", "B2_IMAGES_BUCKET", cfg.B2ImagesBucket},
			{"B2KeyID", "B2_KEY_ID", config.RedactSecret(cfg.B2KeyID)},
			{"B2AppKey", "B2_APP_KEY", config.RedactSecret(cfg.B2AppKey)},
			{"ImageCacheBytes", "IMAGE_CACHE_BYTES", strconv.FormatInt(cfg.ImageCacheBytes, 10)},
			{"Port", "PORT", strconv.Itoa(cfg.Port)},
			{"NuxtEntrypoint", "NUXT_ENTRYPOINT", cfg.NuxtEntrypoint},
			{"NuxtSocket", "NUXT_SOCKET", cfg.NuxtSocket},
			{"APISocket", "SHRIKE_API_SOCKET", cfg.APISocket},
			{"NodeEnv", "NODE_ENV", cfg.NodeEnv},
			{"GitSHA", "GIT_SHA", cfg.GitSHA},
			{"LogLevel", "LOG_LEVEL", cfg.LogLevel},
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
