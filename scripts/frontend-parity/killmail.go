package main

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/eve-kill/shrike/internal/api"
)

type remoteKillmail struct{ origin string }

func (r remoteKillmail) KillmailPage(ctx context.Context, id int64) (api.KillmailPageData, bool, error) {
	data, err := fixtureRead[api.KillmailPageData](ctx, r.origin, "/api/killmail/"+strconv.FormatInt(id, 10))
	var failure *fixtureHTTPError
	if errors.As(err, &failure) && failure.status == http.StatusNotFound {
		return data, false, nil
	}
	return data, err == nil, err
}
