package retry

import (
	"net/url"
)

// Upstream marks a failed HTTP attempt whose server error can recover on retry.
// It preserves the original error and does not change the queue's retry budget.
func Upstream(status int, cause error) error {
	if cause != nil && status >= 500 && status <= 599 {
		return &upstreamError{cause}
	}
	return cause
}

type upstreamError struct{ cause error }

func (e *upstreamError) Error() string { return e.cause.Error() }
func (e *upstreamError) Unwrap() error { return e.cause }

// Transient recognizes only known upstream failures. Every independent joined
// cause must qualify, so a simultaneous database/programming failure stays loud.
// Callers must separately check their parent context and remaining retry budget.
func Transient(err error) bool {
	switch e := err.(type) {
	case *upstreamError:
		return true
	case *url.Error:
		return e.Timeout()
	case interface{ Unwrap() []error }:
		causes := e.Unwrap()
		if len(causes) == 0 {
			return false
		}
		for _, cause := range causes {
			if !Transient(cause) {
				return false
			}
		}
		return true
	case interface{ Unwrap() error }:
		return Transient(e.Unwrap())
	default:
		return false
	}
}
