package db

import (
	"context"
	"time"

	"github.com/eve-kill/shrike/internal/requeststats"
	"github.com/eve-kill/shrike/internal/telemetry"
	"github.com/getsentry/sentry-go"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// queryTracer supplies both development request statistics and optional
// production telemetry. Without either collector it remains a no-op.
type queryTracer struct{ role string }
type querySpanKey struct{}
type batchSpanKey struct{}
type copySpanKey struct{}
type acquireSpanKey struct{}
type dbSpan struct {
	started         time.Time
	stats           *requeststats.Stats
	span            *sentry.Span
	role, operation string
}

func startSpan(ctx context.Context, key any, operation, statement, role string) context.Context {
	stats := requeststats.From(ctx)
	if stats == nil && !telemetry.Enabled() {
		return ctx
	}
	ctx, span := telemetry.StartSpan(ctx, "db."+operation, statement)
	if span != nil {
		span.SetData("db.system", "postgresql")
		span.SetData("db.pool", role)
	}
	return context.WithValue(ctx, key, dbSpan{started: time.Now(), stats: stats, span: span, role: role, operation: operation})
}
func endSpan(ctx context.Context, key any, err error) {
	if span, ok := ctx.Value(key).(dbSpan); ok {
		elapsed := time.Since(span.started)
		span.stats.AddDatabase(elapsed, err != nil)
		telemetry.Observe(ctx, "db."+span.operation, elapsed, err, "pool", span.role)
		telemetry.FinishSpan(span.span, err)
	}
}
func (t queryTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	requeststats.From(ctx).AddQuery()
	statement := ""
	if telemetry.Recording(ctx) {
		statement = telemetry.NormalizeSQL(data.SQL)
	}
	return startSpan(ctx, querySpanKey{}, "query", statement, t.role)
}
func (queryTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	endSpan(ctx, querySpanKey{}, data.Err)
}
func (t queryTracer) TraceBatchStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceBatchStartData) context.Context {
	requeststats.From(ctx).AddBatch()
	return startSpan(ctx, batchSpanKey{}, "batch", "PostgreSQL batch", t.role)
}
func (t queryTracer) TraceBatchQuery(ctx context.Context, _ *pgx.Conn, data pgx.TraceBatchQueryData) {
	requeststats.From(ctx).AddQuery()
	telemetry.Count(ctx, "db.batch.statements", 1, telemetry.Attributes("pool", t.role))
	if span := sentry.SpanFromContext(ctx); telemetry.Enabled() && span != nil && data.Err != nil {
		span.SetData("db.failed_statement", telemetry.NormalizeSQL(data.SQL))
	}
}
func (queryTracer) TraceBatchEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceBatchEndData) {
	endSpan(ctx, batchSpanKey{}, data.Err)
}
func (t queryTracer) TraceCopyFromStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceCopyFromStartData) context.Context {
	requeststats.From(ctx).AddCopy()
	return startSpan(ctx, copySpanKey{}, "copy", "PostgreSQL COPY", t.role)
}
func (queryTracer) TraceCopyFromEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceCopyFromEndData) {
	telemetry.Count(ctx, "db.copy.rows", data.CommandTag.RowsAffected())
	endSpan(ctx, copySpanKey{}, data.Err)
}
func (t queryTracer) TraceAcquireStart(ctx context.Context, _ *pgxpool.Pool, _ pgxpool.TraceAcquireStartData) context.Context {
	return startSpan(ctx, acquireSpanKey{}, "acquire", "PostgreSQL pool wait", t.role)
}
func (queryTracer) TraceAcquireEnd(ctx context.Context, _ *pgxpool.Pool, data pgxpool.TraceAcquireEndData) {
	if span, ok := ctx.Value(acquireSpanKey{}).(dbSpan); ok {
		elapsed := time.Since(span.started)
		span.stats.AddPoolWait(elapsed)
		telemetry.Observe(ctx, "db.pool.wait", elapsed, data.Err, "pool", span.role)
		telemetry.FinishSpan(span.span, data.Err)
	}
}

var (
	_ pgx.QueryTracer       = queryTracer{}
	_ pgx.BatchTracer       = queryTracer{}
	_ pgx.CopyFromTracer    = queryTracer{}
	_ pgxpool.AcquireTracer = queryTracer{}
)
