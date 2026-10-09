package main

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/eve-kill/shrike/internal/api"
)

type remoteRoamReports struct{ origin string }

func (r remoteRoamReports) RoamReport(ctx context.Context, id string) (api.RoamReport, bool, error) {
	data, err := fixtureRead[api.RoamReport](ctx, r.origin, "/api/tools/roam-report/"+url.PathEscape(id))
	var failure *fixtureHTTPError
	if errors.As(err, &failure) && failure.status == http.StatusNotFound {
		return data, false, nil
	}
	return data, err == nil, err
}

func (r remoteRoamReports) RoamReportKilllist(ctx context.Context, report api.RoamReport, query url.Values) (api.KilllistPage, error) {
	return fixtureRead[api.KilllistPage](ctx, r.origin, "/api/tools/roam-report/"+url.PathEscape(report.ID)+"/killlist?"+query.Encode())
}
