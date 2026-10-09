package api

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// SessionPageReader resolves the existing browser session for SSR. It neither
// creates a session nor refreshes its activity; /auth/me owns that lifecycle.
type SessionPageReader struct{ auth *authService }

func (s *Service) Session() *SessionPageReader { return s.session }

func (r *SessionPageReader) BrowserSession(ctx context.Context, cookie string) (*Principal, error) {
	if cookie == "" {
		return nil, nil
	}
	if _, err := uuid.Parse(cookie); err != nil {
		return nil, nil
	}
	if r == nil || r.auth == nil {
		return nil, errors.New("session reader is not configured")
	}
	if r.auth.storeErr != nil {
		return nil, r.auth.storeErr
	}
	principal, _, err := r.auth.store.ResolveSession(ctx, cookie, r.auth.now().UTC())
	if errors.Is(err, errNoAuthSession) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &principal, nil
}
