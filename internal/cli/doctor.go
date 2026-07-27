package cli

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/eve-kill/shrike/internal/config"
	"github.com/eve-kill/shrike/internal/ui"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
)

// doctor verifies that everything this binary depends on is actually reachable
// with the configuration it just loaded. It is the first command implemented
// because it exercises config loading, the UI layer, and error formatting end
// to end — so the skeleton is proven rather than assumed.

// checkTimeout bounds each individual probe. Short enough that a wedged
// dependency does not hang a readiness probe, long enough to survive a
// cold Tailscale route to the production database.
const checkTimeout = 5 * time.Second

// checkResult reports one dependency.
//
// Latency is deliberately a single round trip on an already-established
// connection, not the cost of getting there. Connection setup takes a wildly
// different number of round trips per protocol — one TCP handshake for a bare
// dial, ~5 for Redis (TCP + RESP3 HELLO + two CLIENT SETINFO + PING), ~8 for
// Postgres (TCP + startup + SCRAM's three-message exchange + ReadyForQuery).
// Reporting setup cost as "latency" makes Postgres look 8x slower than a port
// dial into the same datacentre, which is how this table first read.
//
// Status is the verdict of the deeper validation — auth, a real query, a
// version read. That work happens regardless; it just does not set the number.
type checkResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // ok | fail
	Target  string `json:"target"` // redacted
	Detail  string `json:"detail"`
	Latency string `json:"latency"` // one round trip, established connection
	// Connect is kept for JSON consumers only. A cold connect that dwarfs RTT is
	// a real signal — it is why a connection pool needs warm minimum
	// connections — but it does not belong in the latency column.
	Connect string `json:"connect"`

	// order preserves a stable display sequence independent of completion time.
	order int `json:"-"`
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Verify connectivity to every configured dependency",
	Long: `Probes Postgres, Valkey, and Memgraph using the currently loaded
configuration.

Each dependency is fully validated — connect, authenticate, run a real query —
and that outcome decides the status. The reported latency is a single round
trip on the established connection, so the numbers are comparable across
protocols and reflect the network rather than handshake chattiness.

Exits non-zero if any check fails, so it works as a readiness gate:

    shrike doctor --json`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := requireConfig(); err != nil {
			return err
		}

		// Probes are independent and mostly latency-bound, so run them
		// concurrently — a serial run costs up to 4x checkTimeout when
		// several dependencies are down at once.
		checks := []func(context.Context, *config.Config) checkResult{
			checkPostgres,
			checkValkey,
			checkMemgraph,
		}

		results := make([]checkResult, len(checks))
		var wg sync.WaitGroup
		for i, check := range checks {
			wg.Add(1)
			go func(i int, check func(context.Context, *config.Config) checkResult) {
				defer wg.Done()
				r := check(cmd.Context(), cfg)
				r.order = i
				results[i] = r
			}(i, check)
		}
		wg.Wait()

		sort.Slice(results, func(i, j int) bool { return results[i].order < results[j].order })

		failed := 0
		for _, r := range results {
			if r.Status == "fail" {
				failed++
			}
		}

		if ui.JSONMode {
			if err := ui.JSON(map[string]any{
				"healthy": failed == 0,
				"checks":  results,
			}); err != nil {
				return err
			}
		} else {
			ui.Section("Dependencies")
			table := ui.NewTable("COMPONENT", "STATUS", "TARGET", "LATENCY", "DETAIL")
			for _, r := range results {
				table.Row(r.Name, ui.StatusBadge(r.Status), r.Target, r.Latency, r.Detail)
			}
			fmt.Println(table.Render())
			ui.Newline()
		}

		if failed > 0 {
			return fmt.Errorf("%d of %d checks failed", failed, len(results))
		}
		if !ui.JSONMode {
			ui.Success("All %d checks passed.", len(results))
			ui.Newline()
		}
		return nil
	},
}

// checkPostgres connects, authenticates, and reads the server version — the
// deep validation — then times a trivial query for the latency figure.
func checkPostgres(ctx context.Context, c *config.Config) checkResult {
	r := checkResult{Name: "postgres", Target: config.RedactURL(c.DatabaseURL)}
	ctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()

	start := time.Now()
	conn, err := pgx.Connect(ctx, c.DatabaseURL)
	if err != nil {
		return fail(r, err)
	}
	defer conn.Close(ctx)
	r.Connect = dur(time.Since(start))

	var version string
	if err := conn.QueryRow(ctx, "SELECT version()").Scan(&version); err != nil {
		return fail(r, err)
	}

	// Now that the connection is warm and authenticated, measure the round trip.
	ping := time.Now()
	if err := conn.Ping(ctx); err != nil {
		return fail(r, err)
	}
	r.Latency = dur(time.Since(ping))

	r.Status = "ok"
	r.Detail = shortVersion(version)
	return r
}

func checkValkey(ctx context.Context, c *config.Config) checkResult {
	return pingRedis(ctx, "valkey", c.RedisAddr(), c.RedisPassword, c.RedisDB)
}

func pingRedis(ctx context.Context, name, addr, password string, db int) checkResult {
	r := checkResult{Name: name, Target: fmt.Sprintf("%s/%d", addr, db)}
	ctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
		// A health probe wants one attempt and a fast verdict, not go-redis's
		// default five retries with backoff — which turns a refused connection
		// into a multi-second wait.
		MaxRetries:  -1,
		DialTimeout: checkTimeout,
	})
	defer client.Close()

	// First Ping pays dial + RESP3 HELLO + CLIENT SETINFO; that is setup, not RTT.
	start := time.Now()
	if err := client.Ping(ctx).Err(); err != nil {
		return fail(r, err)
	}
	r.Connect = dur(time.Since(start))

	// Deep validation: prove we can actually issue a command and read a reply.
	info, err := client.Info(ctx, "server").Result()
	if err != nil {
		return fail(r, err)
	}

	ping := time.Now()
	if err := client.Ping(ctx).Err(); err != nil {
		return fail(r, err)
	}
	r.Latency = dur(time.Since(ping))

	r.Status = "ok"
	r.Detail = redisVersion(info)
	return r
}

// checkMemgraph dials the Bolt port. This is a TCP reachability check only —
// there is no Bolt handshake, so unlike the other checks nothing here validates
// that Memgraph can actually answer a query. The detail column says so rather
// than implying more confidence than we have; it becomes a real check when the
// graph work pulls in a Bolt driver.
//
// A TCP handshake is itself one round trip, so the latency is comparable to the
// other rows even though the validation is not.
func checkMemgraph(ctx context.Context, c *config.Config) checkResult {
	r := checkResult{Name: "memgraph", Target: c.MemgraphURL}

	u, err := url.Parse(c.MemgraphURL)
	if err != nil {
		r.Status = "fail"
		r.Detail = "unparseable MEMGRAPH_URL"
		return r
	}
	host := u.Host
	if host == "" {
		host = u.Path // bare "memgraph:7687" parses as a path, not a host
	}

	// A single dial. A TCP handshake is one round trip, so this is directly
	// comparable to the other rows. It reads high on the first run after an idle
	// period because Tailscale tears down unused peer paths and the dial pays to
	// re-establish one — that is a real cost of talking to this dependency, not a
	// measurement artifact, so it is reported rather than warmed away.
	dialer := net.Dialer{Timeout: checkTimeout}
	start := time.Now()
	conn, err := dialer.DialContext(ctx, "tcp", host)
	if err != nil {
		return fail(r, err)
	}
	_ = conn.Close()
	elapsed := dur(time.Since(start))

	r.Status = "ok"
	r.Latency = elapsed
	r.Connect = elapsed
	r.Detail = "tcp only (no bolt handshake)"
	return r
}

func fail(r checkResult, err error) checkResult {
	r.Status = "fail"
	// Latency is left blank on failure: there is no round trip to report, and a
	// number here would be the timeout, which says nothing about the network.
	r.Detail = truncate(err.Error(), 52)
	return r
}

// dur formats a latency for display. Millisecond rounding is right for a
// network hop but useless against the local docker stack, where every round
// trip is sub-millisecond and would render as a flat "0s".
func dur(d time.Duration) string {
	if d < time.Millisecond {
		return d.Round(10 * time.Microsecond).String()
	}
	return d.Round(time.Millisecond).String()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

// shortVersion reduces Postgres's banner to something that fits a column:
// "PostgreSQL 18.3 (Debian ...) on x86_64..." becomes "PostgreSQL 18.3".
func shortVersion(banner string) string {
	fields := strings.Fields(banner)
	if len(fields) >= 2 {
		return fields[0] + " " + fields[1]
	}
	return truncate(banner, 40)
}

// redisVersion identifies Valkey by its native version when the compatibility
// redis_version field is also present. Valkey 8 reports redis_version 7.2.4,
// which is a protocol compatibility marker rather than the running version.
func redisVersion(info string) string {
	if version := infoValue(info, "valkey_version:"); version != "" {
		return "Valkey " + version
	}
	if version := infoValue(info, "redis_version:"); version != "" {
		return "Redis " + version
	}
	return ""
}

func infoValue(info, key string) string {
	i := strings.Index(info, key)
	if i < 0 {
		return ""
	}
	rest := info[i+len(key):]
	if end := strings.IndexAny(rest, "\r\n"); end >= 0 {
		return rest[:end]
	}
	return rest
}
