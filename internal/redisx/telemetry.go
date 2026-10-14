package redisx

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/eve-kill/shrike/internal/telemetry"
	"github.com/redis/go-redis/v9"
)

type telemetryHook struct{}

func (telemetryHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		ctx, span := telemetry.StartSpan(ctx, "db.redis.connect", "Valkey connection")
		started := time.Now()
		conn, err := next(ctx, network, addr)
		telemetry.Observe(ctx, "redis.connect", time.Since(started), err)
		telemetry.FinishSpan(span, err)
		return conn, err
	}
}

func (telemetryHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		ctx, span := telemetry.StartSpan(ctx, "db.redis", cmd.Name())
		started := time.Now()
		err := next(ctx, cmd)
		observedErr := err
		if errors.Is(err, redis.Nil) {
			observedErr = nil
		}
		telemetry.Observe(ctx, "redis.command", time.Since(started), observedErr, "command", cmd.Name())
		telemetry.FinishSpan(span, observedErr)
		return err
	}
}

func (telemetryHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, commands []redis.Cmder) error {
		ctx, span := telemetry.StartSpan(ctx, "db.redis", "Valkey pipeline")
		started := time.Now()
		err := next(ctx, commands)
		observedErr := err
		if errors.Is(err, redis.Nil) {
			observedErr = nil
		}
		if span != nil {
			span.SetData("db.redis.commands", len(commands))
		}
		telemetry.Count(ctx, "redis.pipeline.commands", int64(len(commands)))
		telemetry.Observe(ctx, "redis.pipeline", time.Since(started), observedErr)
		telemetry.FinishSpan(span, observedErr)
		return err
	}
}

var _ redis.Hook = telemetryHook{}
