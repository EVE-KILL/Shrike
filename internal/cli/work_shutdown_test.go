package cli

import (
	"context"
	"errors"
	"testing"
)

type fakeRiverLifecycle struct {
	startCtx context.Context
	stop     func(context.Context) error
	hardStop func(context.Context) error
}

func (f *fakeRiverLifecycle) Start(ctx context.Context) error {
	f.startCtx = ctx
	return nil
}

func (f *fakeRiverLifecycle) Stop(ctx context.Context) error {
	return f.stop(ctx)
}

func (f *fakeRiverLifecycle) StopAndCancel(ctx context.Context) error {
	return f.hardStop(ctx)
}

func TestRunRiverLifecycleDrainsWithoutCancellingStartContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	hardCalled := false
	client := &fakeRiverLifecycle{
		stop: func(context.Context) error { return nil },
		hardStop: func(context.Context) error {
			hardCalled = true
			return nil
		},
	}

	if err := runRiverLifecycle(ctx, client); err != nil {
		t.Fatalf("runRiverLifecycle: %v", err)
	}
	if err := client.startCtx.Err(); err != nil {
		t.Fatalf("River Start context was cancelled during graceful drain: %v", err)
	}
	if hardCalled {
		t.Fatal("StopAndCancel called after a successful soft drain")
	}
}

func TestRunRiverLifecycleEscalatesAfterSoftDeadline(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	hardCalled := false
	client := &fakeRiverLifecycle{
		stop: func(context.Context) error { return context.DeadlineExceeded },
		hardStop: func(context.Context) error {
			hardCalled = true
			return nil
		},
	}

	if err := runRiverLifecycle(ctx, client); err != nil {
		t.Fatalf("runRiverLifecycle: %v", err)
	}
	if !hardCalled {
		t.Fatal("StopAndCancel was not called after the soft drain deadline")
	}
}

func TestRunRiverLifecycleReturnsNonTimeoutStopError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	want := errors.New("stop failed")
	client := &fakeRiverLifecycle{
		stop:     func(context.Context) error { return want },
		hardStop: func(context.Context) error { return nil },
	}

	if err := runRiverLifecycle(ctx, client); !errors.Is(err, want) {
		t.Fatalf("runRiverLifecycle error = %v, want %v", err, want)
	}
}
