package workers

import (
	"context"

	"github.com/eve-kill/shrike/internal/queue"
	"github.com/eve-kill/shrike/internal/relay"
	"github.com/riverqueue/river"
)

// Announcement and comment events.
//
// Both are produced by the frontend rather than by anything here — the Nitro
// server enqueues one whenever an admin posts an announcement or a user leaves
// a comment — and both do nothing but forward the payload to the relay.
//
// The payload is therefore carried as raw JSON and republished byte for byte.
// Decoding it into a Go struct would mean this package owning a schema it does
// not define, and every field the frontend adds would be silently dropped until
// somebody remembered to add it here too. Forwarding opaquely means the
// producer and the consumer of the payload are the two ends that already agree
// about it, with nothing in between that can disagree.

// AnnouncementEventWorker forwards announcement lifecycle events.
type AnnouncementEventWorker struct {
	river.WorkerDefaults[queue.AnnouncementEventArgs]
	Deps *Deps
}

func (w *AnnouncementEventWorker) Work(ctx context.Context, job *river.Job[queue.AnnouncementEventArgs]) error {
	w.Deps.Relay.Publish(ctx, relay.ChannelAnnouncements, []string{"all"}, job.Args.Payload)
	return nil
}

// CommentEventWorker forwards comment lifecycle events.
type CommentEventWorker struct {
	river.WorkerDefaults[queue.CommentEventArgs]
	Deps *Deps
}

func (w *CommentEventWorker) Work(ctx context.Context, job *river.Job[queue.CommentEventArgs]) error {
	// Routing is coarse because the relay's /comments endpoint has topic
	// routing disabled and clients filter by target locally. Per-target keys
	// could be emitted here without changing the producer if that ever changes.
	w.Deps.Relay.Publish(ctx, relay.ChannelComments, []string{"all"}, job.Args.Payload)
	return nil
}
