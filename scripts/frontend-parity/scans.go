package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

type remoteScans struct{ origin string }

func (r remoteScans) SavedScan(ctx context.Context, kind, hash string) (json.RawMessage, bool, error) {
	data, err := fixtureRead[json.RawMessage](ctx, r.origin, "/api/tools/"+kind+"/"+hash)
	var failure *fixtureHTTPError
	if errors.As(err, &failure) && failure.status == http.StatusNotFound {
		return nil, false, nil
	}
	return data, err == nil, err
}
