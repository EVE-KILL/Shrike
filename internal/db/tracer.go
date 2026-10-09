package db

import (
	"context"
	"time"

	"github.com/eve-kill/shrike/internal/requeststats"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// queryTracer is attached only to development pools. Calls without a request
// collector are no-ops, including background workers and startup pings.
type queryTracer struct{}
type querySpanKey struct{}
type batchSpanKey struct{}
type copySpanKey struct{}
type acquireSpanKey struct{}
type dbSpan struct {
	started time.Time
	stats   *requeststats.Stats
}

func startSpan(ctx context.Context, key any) context.Context {
	if stats := requeststats.From(ctx); stats != nil {
		return context.WithValue(ctx, key, dbSpan{started: time.Now(), stats: stats})
	}
	return ctx
}
func endSpan(ctx context.Context, key any, failed bool) {
	if span, ok := ctx.Value(key).(dbSpan); ok {
		span.stats.AddDatabase(time.Since(span.started), failed)
	}
}
func (queryTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	requeststats.From(ctx).AddQuery()
	return startSpan(ctx, querySpanKey{})
}
func (queryTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	endSpan(ctx, querySpanKey{}, data.Err != nil)
}
func (queryTracer) TraceBatchStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceBatchStartData) context.Context {
	requeststats.From(ctx).AddBatch()
	return startSpan(ctx, batchSpanKey{})
}
func (queryTracer) TraceBatchQuery(ctx context.Context, _ *pgx.Conn, _ pgx.TraceBatchQueryData) {
	requeststats.From(ctx).AddQuery()
}
func (queryTracer) TraceBatchEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceBatchEndData) {
	endSpan(ctx, batchSpanKey{}, data.Err != nil)
}
func (queryTracer) TraceCopyFromStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceCopyFromStartData) context.Context {
	requeststats.From(ctx).AddCopy()
	return startSpan(ctx, copySpanKey{})
}
func (queryTracer) TraceCopyFromEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceCopyFromEndData) {
	endSpan(ctx, copySpanKey{}, data.Err != nil)
}
func (queryTracer) TraceAcquireStart(ctx context.Context, _ *pgxpool.Pool, _ pgxpool.TraceAcquireStartData) context.Context {
	return startSpan(ctx, acquireSpanKey{})
}
func (queryTracer) TraceAcquireEnd(ctx context.Context, _ *pgxpool.Pool, _ pgxpool.TraceAcquireEndData) {
	if span, ok := ctx.Value(acquireSpanKey{}).(dbSpan); ok {
		span.stats.AddPoolWait(time.Since(span.started))
	}
}

var (
	_ pgx.QueryTracer       = queryTracer{}
	_ pgx.BatchTracer       = queryTracer{}
	_ pgx.CopyFromTracer    = queryTracer{}
	_ pgxpool.AcquireTracer = queryTracer{}
)
