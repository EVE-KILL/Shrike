package retry

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"testing"
)

func TestTransientUpstreamClassificationPreservesIndependentFailures(t *testing.T) {
	cause := errors.New("ESI response")
	httpError := Upstream(502, cause)
	timeout := &url.Error{Op: "Get", URL: "https://esi.evetech.net/", Err: context.DeadlineExceeded}
	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{"server error", httpError, true},
		{"wrapped server error", fmt.Errorf("war: %w", httpError), true},
		{"HTTP client timeout", timeout, true},
		{"joined upstream failures", errors.Join(httpError, timeout), true},
		{"independent database failure", errors.Join(httpError, errors.New("database failed")), false},
		{"client error", Upstream(400, cause), false},
		{"unexpected decode failure", errors.New("invalid JSON"), false},
		{"parent deadline alone", context.DeadlineExceeded, false},
		{"nil", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Transient(tc.err); got != tc.want {
				t.Fatalf("Transient=%v, want %v", got, tc.want)
			}
		})
	}
	if !errors.Is(httpError, cause) || httpError.Error() != cause.Error() {
		t.Fatal("classification changed the returned error contract")
	}
}
