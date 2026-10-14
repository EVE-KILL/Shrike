package websocket

import (
	"context"
	"sync"
	"testing"

	"github.com/eve-kill/shrike/internal/config"
	"github.com/eve-kill/shrike/internal/telemetry"
	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
)

func TestConnectionGaugeAndSuccessfulDeliveries(t *testing.T) {
	old := sentry.CurrentHub().Clone()
	transport := &sentry.MockTransport{}
	runtime, err := telemetry.Init(telemetry.Options{Config: &config.Config{
		SentryDSN: "https://test@o1.ingest.sentry.io/42", SentryEnvironment: "test", SentryLogLevel: "info", SentryMetrics: true,
	}, Transport: transport})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		runtime.Close()
		sentry.CurrentHub().BindClient(old.Client())
		sentry.CurrentHub().Scope().Clear()
	})
	server := New(nil, zerolog.Nop())
	t.Cleanup(server.cancel)
	endpoint := endpointByPath["/ws/status"]
	var wg sync.WaitGroup
	for range 1000 {
		wg.Go(func() {
			c := newClient(context.Background(), nil, endpoint)
			server.register(c)
			server.unregister(c)
		})
	}
	wg.Wait()
	telemetry.Flush()
	for _, event := range transport.Events() {
		for _, metric := range event.Metrics {
			if value, _ := metric.Value.Float64(); metric.Name == "websocket.connections" && value != 0 {
				t.Fatalf("empty server's connection gauge=%v", value)
			}
		}
	}
	fast := newClient(server.ctx, nil, endpoint)
	slow := newClient(server.ctx, nil, endpoint)
	// These clients test queues without socket I/O. Suppress the slow client's
	// asynchronous socket close; its backpressure and delivery counters remain real.
	slow.closeOnce.Do(func() {})
	for range clientQueueSize {
		slow.enqueue([]byte("queued"))
	}
	server.register(fast)
	server.register(slow)
	server.broadcast(endpoint, []string{"all"}, []byte("event"))
	server.unregister(fast)
	server.unregister(slow)
	runtime.Close()
	var deliveries, backpressure int64
	for _, event := range transport.Events() {
		for _, metric := range event.Metrics {
			value, _ := metric.Value.Int64()
			switch metric.Name {
			case "websocket.deliveries":
				deliveries += value
			case "websocket.backpressure":
				backpressure += value
			}
		}
	}
	if deliveries != 1 || backpressure != 1 || len(fast.out) != 1 {
		t.Fatalf("deliveries=%v backpressure=%v queued=%d", deliveries, backpressure, len(fast.out))
	}
}
