package cli

import (
	"errors"
	"fmt"
	"time"

	"github.com/eve-kill/shrike/internal/telemetry"
	"github.com/eve-kill/shrike/internal/ui"
	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var sentryCmd = &cobra.Command{Use: "sentry", Short: "Verify Sentry telemetry delivery"}
var sentryTestCmd = &cobra.Command{
	Use: "test", Short: "Send a tagged test error, log, metric, and trace",
	Long: "Verifies the configured SENTRY_DSN without opening database connections. Sends an intentional error tagged verification=true. Use SENTRY_ENVIRONMENT=verification to keep it separate from production errors.",
	RunE: func(cmd *cobra.Command, _ []string) error {
		if !telemetry.Enabled() {
			return errors.New("configure SENTRY_DSN before running sentry:test")
		}
		hub := sentry.CurrentHub().Clone()
		hub.Scope().SetTag("verification", "true")
		ctx := sentry.SetHubOnContext(cmd.Context(), hub)
		span := sentry.StartTransaction(ctx, "Sentry integration verification", sentry.WithOpName("telemetry.verify"), sentry.WithSpanSampled(sentry.SampledTrue))
		ctx = span.Context()
		id := telemetry.Capture(ctx, errors.New("shrike Sentry integration verification"))
		eventID := ""
		if id != nil {
			eventID = string(*id)
		}
		log.Info().Ctx(ctx).Bool("verification", true).Msg("Shrike Sentry log verification")
		telemetry.Count(ctx, "telemetry.verification", 1, telemetry.Attributes("verification", "true"))
		telemetry.Gauge(ctx, "telemetry.verification.gauge", 1)
		telemetry.Duration(ctx, "telemetry.verification.duration", time.Since(span.StartTime))
		telemetry.FinishSpan(span, nil)
		if !telemetry.Flush() {
			return fmt.Errorf("sentry delivery did not finish within %s", telemetry.FlushTimeout)
		}
		if ui.JSONMode {
			return ui.JSON(map[string]any{"delivered": true, "event_id": eventID, "trace_id": span.TraceID.String()})
		}
		ui.Success("Sentry verification delivered; event %s, trace %s", eventID, span.TraceID.String())
		return nil
	},
}

func init() { sentryCmd.AddCommand(sentryTestCmd) }
