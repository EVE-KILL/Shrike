package api

import (
	"context"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

func registerLocationRoute(a huma.API, opts Options) {
	registerLegacy(a, huma.Operation{
		OperationID: "location",
		Method:      http.MethodGet,
		Path:        "/location",
		Summary:     "Resolve coordinates to the nearest celestial",
		Tags:        []string{"universe"},
	}, func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		x, xOK := finiteQueryNumber(req, "x")
		y, yOK := finiteQueryNumber(req, "y")
		z, zOK := finiteQueryNumber(req, "z")
		if !xOK || !yOK || !zOK {
			return legacyPayload{}, apiError(http.StatusBadRequest, "x, y, z must all be finite numbers")
		}
		systemID, ok := finiteQueryNumber(req, "system_id")
		if !ok || systemID <= 0 {
			return legacyPayload{}, apiError(http.StatusBadRequest, "system_id must be a positive integer")
		}

		nearest, err := queryMap(ctx, opts.DB, `
			SELECT item_id, item_name, type_id, group_id,
			       sqrt(power(x - $1, 2) + power(y - $2, 2) + power(z - $3, 2)) AS distance
			FROM celestials
			WHERE solar_system_id = $4
			  AND x IS NOT NULL AND y IS NOT NULL AND z IS NOT NULL
			ORDER BY distance ASC
			LIMIT 1`,
			x, y, z, systemID,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{
			"system_id": systemID,
			"x":         x,
			"y":         y,
			"z":         z,
			"nearest":   nearest,
		}), nil
	})
}

func finiteQueryNumber(req *legacyRequest, name string) (float64, bool) {
	values, exists := req.Query[name]
	if !exists {
		return 0, false
	}
	raw := ""
	if len(values) > 0 {
		raw = strings.TrimSpace(values[len(values)-1])
	}
	// JavaScript Number("") is zero.
	if raw == "" {
		return 0, true
	}
	value, err := strconv.ParseFloat(raw, 64)
	return value, err == nil && !math.IsNaN(value) && !math.IsInf(value, 0)
}
