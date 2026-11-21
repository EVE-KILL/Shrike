module github.com/eve-kill/shrike

go 1.27.4

replace github.com/eve-kill/shrike/frontend => ./frontend

require (
	charm.land/lipgloss/v2 v2.1.1
	github.com/a-h/templ v0.3.1118
	github.com/aws/aws-sdk-go-v2 v1.48.0
	github.com/coder/websocket v1.8.15
	github.com/coreos/go-oidc/v3 v3.21.0
	github.com/danielgtaylor/huma/v2 v2.41.0
	github.com/deepteams/webp v1.2.9
	github.com/eve-kill/shrike/frontend v0.0.0
	github.com/felixge/httpsnoop v1.1.0
	github.com/getsentry/sentry-go v0.50.0
	github.com/go-chi/chi/v5 v5.5.0
	github.com/google/flatbuffers v25.12.19+incompatible
	github.com/google/jsonschema-go v0.4.3
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.13.0
	github.com/modelcontextprotocol/go-sdk v1.8.0
	github.com/neo4j/neo4j-go-driver/v5 v5.28.6
	github.com/pressly/goose/v3 v3.30.1
	github.com/redis/go-redis/v9 v9.23.0
	github.com/riverqueue/river v0.51.2
	github.com/riverqueue/river/riverdriver/riverpgxv5 v0.51.2
	github.com/riverqueue/river/rivertype v0.51.2
	github.com/rs/zerolog v1.35.1
	github.com/russross/blackfriday/v2 v2.1.0
	github.com/spf13/cobra v1.10.2
	github.com/spf13/pflag v1.1.0
	golang.org/x/image v0.48.0
	golang.org/x/net v0.63.0
	golang.org/x/oauth2 v0.38.0
	golang.org/x/sync v0.25.0
	golang.org/x/term v0.48.0
	golang.org/x/text v0.44.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/a-h/parse v0.0.42 // indirect
	github.com/andybalholm/brotli v1.3.0 // indirect
	github.com/aws/smithy-go v1.28.9 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/charmbracelet/colorprofile v0.4.3 // indirect
	github.com/charmbracelet/ultraviolet v0.0.0-20261119151026-8ecbcf0e188c // indirect
	github.com/charmbracelet/x/ansi v0.11.10 // indirect
	github.com/charmbracelet/x/term v0.2.2 // indirect
	github.com/charmbracelet/x/termios v0.1.1 // indirect
	github.com/charmbracelet/x/windows v0.2.2 // indirect
	github.com/cli/browser v1.3.0 // indirect
	github.com/clipperhouse/displaywidth v0.11.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/fatih/color v1.19.0 // indirect
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/go-jose/go-jose/v4 v4.1.9 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.4 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.4.1 // indirect
	github.com/mattn/go-colorable v0.1.16 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/mattn/go-runewidth v0.0.31 // indirect
	github.com/mfridman/interpolate v0.2.0 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/natefinch/atomic v1.0.1 // indirect
	github.com/riverqueue/river/riverdriver v0.51.2 // indirect
	github.com/riverqueue/river/rivershared v0.51.2 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/rogpeppe/go-internal v1.16.0 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/sethvargo/go-retry v0.5.0 // indirect
	github.com/tidwall/gjson v1.20.5 // indirect
	github.com/tidwall/match v1.2.0 // indirect
	github.com/tidwall/pretty v1.3.0 // indirect
	github.com/tidwall/sjson v1.2.11 // indirect
	github.com/xo/terminfo v1.3.1 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	go.uber.org/atomic v1.12.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/mod v0.43.0 // indirect
	golang.org/x/sys v0.50.0 // indirect
	golang.org/x/time v0.16.0 // indirect
	golang.org/x/tools v0.53.0 // indirect
)

tool github.com/a-h/templ/cmd/templ
