package frontend

import (
	"context"
	"net/http"

	"github.com/eve-kill/shrike/internal/api"
)

type SessionReader interface {
	BrowserSession(context.Context, string) (*api.Principal, error)
}

func (s *site) browserSession(r *http.Request) (*api.Principal, error) {
	if s.options.Sessions == nil {
		return nil, nil
	}
	cookie, err := r.Cookie("evelogin")
	if err != nil {
		return nil, nil
	}
	return s.options.Sessions.BrowserSession(r.Context(), cookie.Value)
}
