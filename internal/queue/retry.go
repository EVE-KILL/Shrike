package queue

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/eve-kill/shrike/internal/jobs"
	"github.com/riverqueue/river/rivertype"
)

// MaxRetryDelay caps the backoff.
//
// Without a cap the formula runs away: the ESI queues retry ten times from a
// 30-second base, and 30s × 2⁹ is over four hours. A job that has been waiting
// four hours to retry is not retrying, it is abandoned with extra steps, and it
// holds a slot in the backlog the whole time. An hour is long enough to ride
// out any ESI outage worth riding out.
const MaxRetryDelay = time.Hour

// RetryJitter is how much the delay is randomised, as a fraction.
//
// This is a deliberate departure from BullMQ, whose exponential backoff has no
// jitter. It matters here because failures in this system are correlated rather
// than independent: when Tranquility goes down, every in-flight ESI job fails
// within the same second and, without jitter, every one of them retries in the
// same second too — repeatedly, in lockstep, for as long as the outage lasts.
// Spreading them out costs nothing and removes the synchronised stampede.
const RetryJitter = 0.15

// RegistryRetryPolicy computes retry timing from the queue declarations, so the
// backoff each queue gets is the one written down in internal/jobs rather than
// a single client-wide default.
//
// River's retry policy is per client, not per queue, so the per-queue base
// delay is recovered by looking the job's kind up in the registry.
type RegistryRetryPolicy struct {
	// Rand is the jitter source. Nil uses the global one; tests supply their
	// own so the delay is exact and assertions can be about the formula rather
	// than about a range.
	Rand func() float64

	// Now is the clock, injectable for tests.
	Now func() time.Time
}

// NextRetry returns when the job should next be attempted.
//
// The formula is BullMQ's — base × 2^(attempt-1) — so a queue's declared
// backoff means the same thing after the port as before it.
//
// The delay is measured from now rather than from when the attempt started,
// which is both what BullMQ does and the only version that works. Measuring
// from the attempt means a job that runs for longer than its own backoff gets a
// retry time in the past: the killmails queue has a one-second base delay and a
// killmail that takes a second to fail lands exactly there. River rejects a
// past retry time and silently substitutes its own default policy, so the
// declared per-queue backoff would quietly stop being used at all — which is
// exactly what happened the first time this ran against the live feed.
func (p *RegistryRetryPolicy) NextRetry(job *rivertype.JobRow) time.Time {
	return p.now().Add(p.delay(job.Kind, job.Attempt))
}

func (p *RegistryRetryPolicy) now() time.Time {
	if p.Now != nil {
		return p.Now()
	}
	return time.Now().UTC()
}

func (p *RegistryRetryPolicy) delay(kind string, attempt int) time.Duration {
	base := DefaultBackoff
	if q := jobs.QueueByName(kind); q != nil && q.BackoffDelay > 0 {
		base = time.Duration(q.BackoffDelay) * time.Millisecond
	}

	// Attempt is 1-based on the first run, so the first retry waits exactly the
	// base delay rather than twice it.
	if attempt < 1 {
		attempt = 1
	}
	// Bound the exponent before it is applied: 2^63 overflows a Duration into
	// something negative, which would schedule the retry in the past and spin.
	exponent := min(attempt-1, 32)

	delay := time.Duration(float64(base) * math.Pow(2, float64(exponent)))
	if delay > MaxRetryDelay || delay <= 0 {
		delay = MaxRetryDelay
	}

	jitter := 1 + RetryJitter*(2*p.random()-1)
	delay = max(
		// Jitter must never produce a non-positive delay, which would retry
		// immediately and turn a failing job into a hot loop.
		time.Duration(float64(delay)*jitter), time.Millisecond)
	return delay
}

func (p *RegistryRetryPolicy) random() float64 {
	if p.Rand != nil {
		return p.Rand()
	}
	return rand.Float64() //nolint:gosec // jitter, not cryptography
}

// DefaultBackoff applies to a kind with no registry entry — the cron queue, and
// anything added to River without being declared.
const DefaultBackoff = time.Second
