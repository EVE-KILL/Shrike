package api

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
)

const (
	fittingIDLength       = 14
	fittingNameMax        = 100
	fittingDescriptionMax = 2000
	fittingItemsMax       = 200
	fittingBodyLimit      = 64 << 10
)

const fittingIDAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

var (
	requiredFittingSession = []map[string][]string{{"eveSession": {}}}
	optionalFittingSession = []map[string][]string{{}, {"eveSession": {}}}
)

// registerFittingRoutes installs both user-owned fitting CRUD and the public
// derived fitting catalogue. Canonical routes use one plural /fittings domain;
// the /fit, /fits, and /item aliases exist only while the Nuxt caller migrates.
func registerFittingRoutes(a huma.API, opts Options) {
	auth := newAuthService(opts)

	registerFittingCRUDRoutes(a, opts, auth)
	registerFittingCatalogueRoutes(a, opts)
}

func registerFittingCRUDRoutes(a huma.API, opts Options, auth *authService) {
	create := createFittingHandler(opts, auth)
	for i, path := range []string{"/fittings", "/fit"} {
		registerLegacy(a, documentJSONBody[fittingCreateDocument](a, huma.Operation{
			OperationID: fittingOperationID("fitting-create", i),
			Method:      http.MethodPost,
			Path:        path,
			Summary:     "Save a ship fitting",
			Tags:        []string{"fittings", "account"},
			Security:    requiredFittingSession,
		}), create)
	}

	for _, route := range []struct {
		Name, Method, Canonical, Alias, Summary string
		Security                                []map[string][]string
		Build                                   func(string) legacyHandler
		// Document attaches the request schema for routes that take a body.
		// Reads leave it nil.
		Document func(huma.Operation) huma.Operation
	}{
		{
			Name: "fitting-detail", Method: http.MethodGet,
			Canonical: "/fittings/{id}", Alias: "/fit/{fit_id}",
			Summary: "Get a saved ship fitting", Security: optionalFittingSession,
			Build: func(param string) legacyHandler {
				return getFittingHandler(opts, auth, param)
			},
		},
		{
			Name: "fitting-update", Method: http.MethodPatch,
			Canonical: "/fittings/{id}", Alias: "/fit/{fit_id}",
			Summary: "Update a saved ship fitting", Security: requiredFittingSession,
			Build: func(param string) legacyHandler {
				return updateFittingHandler(opts, auth, param)
			},
			Document: func(op huma.Operation) huma.Operation {
				return documentJSONBody[fittingUpdateDocument](a, op)
			},
		},
		{
			Name: "fitting-delete", Method: http.MethodDelete,
			Canonical: "/fittings/{id}", Alias: "/fit/{fit_id}",
			Summary: "Delete a saved ship fitting", Security: requiredFittingSession,
			Build: func(param string) legacyHandler {
				return deleteFittingHandler(opts, auth, param)
			},
		},
		{
			Name: "fitting-rating-put", Method: http.MethodPut,
			Canonical: "/fittings/{id}/rating", Alias: "/fit/{fit_id}/rating",
			Summary: "Rate a saved ship fitting", Security: requiredFittingSession,
			Build: func(param string) legacyHandler {
				return putFittingRatingHandler(opts, auth, param)
			},
			Document: func(op huma.Operation) huma.Operation {
				return documentJSONBody[fittingRatingDocument](a, op)
			},
		},
		{
			Name: "fitting-rating-delete", Method: http.MethodDelete,
			Canonical: "/fittings/{id}/rating", Alias: "/fit/{fit_id}/rating",
			Summary: "Remove a ship fitting rating", Security: requiredFittingSession,
			Build: func(param string) legacyHandler {
				return deleteFittingRatingHandler(opts, auth, param)
			},
		},
	} {
		document := route.Document
		if document == nil {
			document = func(op huma.Operation) huma.Operation { return op }
		}
		registerLegacy(a, document(huma.Operation{
			OperationID: route.Name,
			Method:      route.Method,
			Path:        route.Canonical,
			Summary:     route.Summary,
			Tags:        []string{"fittings", "account"},
			Security:    route.Security,
		}), route.Build("id"))
		registerLegacy(a, document(huma.Operation{
			OperationID: route.Name + "-legacy",
			Method:      route.Method,
			Path:        route.Alias,
			Summary:     route.Summary,
			Tags:        []string{"fittings", "account"},
			Security:    route.Security,
		}), route.Build("fit_id"))
	}
}

func fittingOperationID(base string, aliasIndex int) string {
	if aliasIndex == 0 {
		return base
	}
	return base + "-legacy"
}

// Wire types for the fitting write routes.
//
// Integers are plain int64 rather than jsInt: these routes read through
// exactJSONInteger, which never accepted numeric strings, so widening them
// here would document a leniency the handler does not have.
//
// Pointers mark a field the caller may omit. The validators below reproduce
// the same messages and the same 422 status the map-based versions returned,
// because those are the responses the fitting editor already handles.
type fittingItemBody struct {
	SlotGroup    *int64 `json:"slot_group" minimum:"1" maximum:"7" doc:"Slot family: 1-5 are module slots, 6 drones, 7 cargo."`
	Ordinal      *int64 `json:"ordinal" minimum:"0" maximum:"15" doc:"Position within the slot family."`
	TypeID       *int64 `json:"type_id" minimum:"1" doc:"Inventory type fitted in this position."`
	State        *int64 `json:"state" minimum:"0" maximum:"3" doc:"Module state: offline, online, active, overloaded."`
	ChargeTypeID *int64 `json:"charge_type_id,omitempty" minimum:"1" doc:"Charge loaded into this module, when it takes one."`
	Quantity     *int64 `json:"quantity,omitempty" minimum:"1" maximum:"30000" doc:"Stack size. Must be 1 for module slots. Defaults to 1."`
}

type fittingCreateBody struct {
	ShipTypeID  *int64            `json:"ship_type_id" minimum:"1" doc:"Hull the fitting is for."`
	Name        *string           `json:"name" doc:"Display name for the fitting."`
	Description optional[string]  `json:"description,omitempty" doc:"Free text shown with the fitting. Null clears it."`
	Visibility  *int64            `json:"visibility" minimum:"0" maximum:"3" doc:"Who may see the fitting."`
	Items       []fittingItemBody `json:"items" maxItems:"200" doc:"Fitted modules, charges, drones and cargo."`
}

// fittingUpdateBody patches an existing fitting. Every field is optional, and
// an absent field is left alone rather than cleared.
type fittingUpdateBody struct {
	Name        optional[string]            `json:"name,omitempty" doc:"New display name."`
	Description optional[string]            `json:"description,omitempty" doc:"New description. Null clears it."`
	Visibility  optional[int64]             `json:"visibility,omitempty" minimum:"0" maximum:"3" doc:"New visibility."`
	Items       optional[[]fittingItemBody] `json:"items,omitempty" doc:"Replacement item list. Absent leaves the stored items alone."`
}

type fittingRatingBody struct {
	Rating *int64 `json:"rating" minimum:"1" maximum:"5" doc:"Rating from 1 to 5."`
}

type validatedFittingItem struct {
	SlotGroup  int16
	Ordinal    int16
	TypeID     int32
	State      int16
	ChargeType *int32
	Quantity   int16
}

func (item validatedFittingItem) JSON() map[string]any {
	return map[string]any{
		"slot_group": item.SlotGroup, "ordinal": item.Ordinal,
		"type_id": item.TypeID, "state": item.State,
		"charge_type_id": item.ChargeType, "quantity": item.Quantity,
	}
}

type validatedFittingBody struct {
	ShipTypeID     *int32
	Name           *string
	Description    any
	HasDescription bool
	Visibility     *int16
	Items          []validatedFittingItem
	HasItems       bool
}

func validateCreateFitting(body *fittingCreateBody) (validatedFittingBody, error) {
	ship, err := requiredFittingInt(body.ShipTypeID, "ship_type_id", 1, math.MaxInt32)
	if err != nil {
		return validatedFittingBody{}, err
	}
	if body.Name == nil {
		return validatedFittingBody{}, apiError(
			http.StatusUnprocessableEntity, "name must be a string")
	}
	name, err := fittingName(*body.Name)
	if err != nil {
		return validatedFittingBody{}, err
	}
	description, err := fittingDescription(body.Description)
	if err != nil {
		return validatedFittingBody{}, err
	}
	visibility, err := requiredFittingInt(body.Visibility, "visibility", 0, 3)
	if err != nil {
		return validatedFittingBody{}, err
	}
	items, err := validateFittingItems(body.Items)
	if err != nil {
		return validatedFittingBody{}, err
	}
	ship32, visibility16 := int32(ship), int16(visibility)
	return validatedFittingBody{
		ShipTypeID: &ship32, Name: &name,
		Description: description, HasDescription: true,
		Visibility: &visibility16, Items: items, HasItems: true,
	}, nil
}

func validateUpdateFitting(body *fittingUpdateBody) (validatedFittingBody, error) {
	var out validatedFittingBody
	if body.Name.present() {
		if body.Name.Value == nil {
			return out, apiError(
				http.StatusUnprocessableEntity, "name must be a string")
		}
		name, err := fittingName(*body.Name.Value)
		if err != nil {
			return out, err
		}
		out.Name = &name
	}
	if body.Description.present() {
		description, err := fittingDescription(body.Description)
		if err != nil {
			return out, err
		}
		out.Description, out.HasDescription = description, true
	}
	if body.Visibility.present() {
		visibility, err := requiredFittingInt(body.Visibility.Value, "visibility", 0, 3)
		if err != nil {
			return out, err
		}
		visibility16 := int16(visibility)
		out.Visibility = &visibility16
	}
	if body.Items.present() {
		items, err := validateFittingItems(body.Items.valueOr(nil))
		if err != nil {
			return out, err
		}
		out.Items, out.HasItems = items, true
	}
	if out.Name == nil && !out.HasDescription &&
		out.Visibility == nil && !out.HasItems {
		return out, apiError(http.StatusBadRequest, "No updatable fields supplied")
	}
	return out, nil
}

// requiredFittingInt reproduces the message and the 422 the map-based
// validators returned for a missing or out-of-range integer.
func requiredFittingInt(value *int64, field string, min, max int64) (int64, error) {
	if value == nil || *value < min || *value > max {
		return 0, apiError(http.StatusUnprocessableEntity, fmt.Sprintf(
			"%s must be an integer in [%d, %d]", field, min, max))
	}
	return *value, nil
}

func exactJSONInteger(value any) (int64, bool) {
	switch typed := value.(type) {
	case json.Number:
		number, err := strconv.ParseInt(string(typed), 10, 64)
		return number, err == nil
	case int:
		return int64(typed), true
	case int8:
		return int64(typed), true
	case int16:
		return int64(typed), true
	case int32:
		return int64(typed), true
	case int64:
		return typed, true
	case float64:
		if math.IsNaN(typed) || math.IsInf(typed, 0) || math.Trunc(typed) != typed {
			return 0, false
		}
		return int64(typed), typed >= math.MinInt64 && typed <= math.MaxInt64
	default:
		return 0, false
	}
}

func fittingName(value any) (string, error) {
	raw, ok := value.(string)
	if !ok {
		return "", apiError(http.StatusUnprocessableEntity, "name must be a string")
	}
	name := strings.TrimSpace(raw)
	length := utf8.RuneCountInString(name)
	switch {
	case length < 1:
		return "", apiError(
			http.StatusUnprocessableEntity,
			"name must be at least 1 characters",
		)
	case length > fittingNameMax:
		return "", apiError(
			http.StatusUnprocessableEntity,
			"name must be at most 100 characters",
		)
	default:
		return name, nil
	}
}

// fittingDescription normalizes the description. An empty string and an
// explicit null both mean "no description", which is what the editor sends
// when the field is cleared.
func fittingDescription(value optional[string]) (any, error) {
	if value.Value == nil {
		return nil, nil
	}
	text := *value.Value
	if text == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(text) > fittingDescriptionMax {
		return nil, apiError(
			http.StatusUnprocessableEntity,
			"description must be at most 2000 characters",
		)
	}
	return text, nil
}

func validateFittingItems(raw []fittingItemBody) ([]validatedFittingItem, error) {
	if len(raw) > fittingItemsMax {
		return nil, apiError(
			http.StatusUnprocessableEntity,
			"items must not exceed 200 entries",
		)
	}
	items := make([]validatedFittingItem, 0, len(raw))
	seen := make(map[[2]int64]bool, len(raw))
	for index, row := range raw {
		prefix := fmt.Sprintf("items[%d].", index)
		slot, err := requiredFittingInt(row.SlotGroup, prefix+"slot_group", 1, 7)
		if err != nil {
			return nil, err
		}
		ordinal, err := requiredFittingInt(row.Ordinal, prefix+"ordinal", 0, 15)
		if err != nil {
			return nil, err
		}
		typeID, err := requiredFittingInt(
			row.TypeID, prefix+"type_id", 1, math.MaxInt32)
		if err != nil {
			return nil, err
		}
		state, err := requiredFittingInt(row.State, prefix+"state", 0, 3)
		if err != nil {
			return nil, err
		}

		var charge *int32
		if row.ChargeTypeID != nil {
			chargeID, err := requiredFittingInt(
				row.ChargeTypeID, prefix+"charge_type_id", 1, math.MaxInt32)
			if err != nil {
				return nil, err
			}
			charge32 := int32(chargeID)
			charge = &charge32
		}

		quantity := int64(1)
		if row.Quantity != nil {
			quantity, err = requiredFittingInt(
				row.Quantity, prefix+"quantity", 1, 30000)
			if err != nil {
				return nil, err
			}
		}
		if slot <= 5 && quantity != 1 {
			return nil, apiError(
				http.StatusUnprocessableEntity,
				fmt.Sprintf(
					"items[%d].quantity must be 1 for module slots (slot_group %d)",
					index, slot,
				),
			)
		}

		key := [2]int64{slot, ordinal}
		if seen[key] {
			return nil, apiError(
				http.StatusUnprocessableEntity,
				fmt.Sprintf(
					"Duplicate slot_group=%d ordinal=%d at items[%d]",
					slot, ordinal, index,
				),
			)
		}
		seen[key] = true
		items = append(items, validatedFittingItem{
			SlotGroup: int16(slot), Ordinal: int16(ordinal),
			TypeID: int32(typeID), State: int16(state),
			ChargeType: charge, Quantity: int16(quantity),
		})
	}
	return items, nil
}

func fittingID(req *legacyRequest, parameter string) (string, error) {
	id := req.Param(parameter)
	if id == "" {
		return "", apiError(http.StatusBadRequest, "Invalid fit_id")
	}
	return id, nil
}

func generateFittingID() (string, error) {
	random := make([]byte, fittingIDLength)
	if _, err := io.ReadFull(rand.Reader, random); err != nil {
		return "", err
	}
	result := make([]byte, fittingIDLength)
	for index, value := range random {
		result[index] = fittingIDAlphabet[int(value)%len(fittingIDAlphabet)]
	}
	return string(result), nil
}

func fittingMutationPrincipal(
	ctx context.Context,
	req *legacyRequest,
	auth *authService,
) (*Principal, error) {
	setAccountNoStore(req.Huma)
	if err := requireSameOriginMutation(req.Huma); err != nil {
		return nil, err
	}
	return auth.requirePrincipal(ctx, req)
}

func createFittingHandler(opts Options, auth *authService) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		principal, err := fittingMutationPrincipal(ctx, req, auth)
		if err != nil {
			return legacyPayload{}, err
		}
		raw, err := decodeJSONBody[fittingCreateBody](req, fittingBodyLimit)
		if err != nil {
			return legacyPayload{}, err
		}
		body, err := validateCreateFitting(raw)
		if err != nil {
			return legacyPayload{}, err
		}
		id, err := generateFittingID()
		if err != nil {
			return legacyPayload{}, err
		}
		db, err := mutationDatabase(opts)
		if err != nil {
			return legacyPayload{}, err
		}
		tx, err := db.Begin(ctx)
		if err != nil {
			return legacyPayload{}, err
		}
		defer tx.Rollback(ctx) //nolint:errcheck
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_fittings (
				fit_id, owner_character_id, ship_type_id,
				name, description, visibility
			) VALUES ($1, $2, $3, $4, $5, $6)`,
			id, principal.CharacterID, *body.ShipTypeID,
			*body.Name, body.Description, *body.Visibility,
		); err != nil {
			return legacyPayload{}, err
		}
		if err := copyFittingItems(ctx, tx, id, body.Items); err != nil {
			return legacyPayload{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return legacyPayload{}, err
		}
		items := fittingItemsJSON(body.Items)
		return accountNoStorePayload(map[string]any{
			"fit_id": id, "owner_character_id": principal.CharacterID,
			"ship_type_id": *body.ShipTypeID, "name": *body.Name,
			"description": body.Description, "visibility": *body.Visibility,
			"items": items,
		}), nil
	}
}

func copyFittingItems(
	ctx context.Context,
	tx pgx.Tx,
	id string,
	items []validatedFittingItem,
) error {
	if len(items) == 0 {
		return nil
	}
	rows := make([][]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, []any{
			id, item.SlotGroup, item.Ordinal, item.TypeID,
			item.State, item.ChargeType, item.Quantity,
		})
	}
	_, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"user_fitting_items"},
		[]string{
			"fit_id", "slot_group", "ordinal", "type_id",
			"state", "charge_type_id", "quantity",
		},
		pgx.CopyFromRows(rows),
	)
	return err
}

func fittingItemsJSON(items []validatedFittingItem) []map[string]any {
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, item.JSON())
	}
	return result
}

func getFittingHandler(
	opts Options,
	auth *authService,
	parameter string,
) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		id, err := fittingID(req, parameter)
		if err != nil {
			return legacyPayload{}, err
		}
		principal, err := auth.resolvePrincipal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		fit, err := queryMap(ctx, opts.DB, `
			SELECT f.fit_id, f.owner_character_id, f.ship_type_id,
			       f.name, f.description, f.visibility,
			       f.rating_avg, f.rating_count,
			       f.created_at, f.updated_at,
			       owner.corporation_id AS owner_corp_id,
			       owner.alliance_id AS owner_alliance_id
			FROM user_fittings f
			LEFT JOIN characters owner
			  ON owner.character_id = f.owner_character_id
			WHERE f.fit_id = $1
			LIMIT 1`, id)
		if err != nil {
			return legacyPayload{}, err
		}
		if fit == nil {
			return legacyPayload{}, apiError(http.StatusNotFound, "Fit not found")
		}
		if !canViewFitting(principal, fit) {
			return legacyPayload{}, apiError(
				http.StatusForbidden, "Fit is not visible to you",
			)
		}
		queries := []databaseQuery{{
			SQL: `
				SELECT slot_group, ordinal, type_id, state,
				       charge_type_id, quantity
				FROM user_fitting_items
				WHERE fit_id = $1
				ORDER BY slot_group, ordinal`,
			Args: []any{id},
		}}
		if principal != nil {
			queries = append(queries, databaseQuery{
				SQL: `
					SELECT rating
					FROM user_fitting_ratings
					WHERE fit_id = $1 AND rater_character_id = $2
					LIMIT 1`,
				Args: []any{id, principal.CharacterID},
			})
		}
		results, err := queryMapsConcurrent(ctx, opts.DB, queries...)
		if err != nil {
			return legacyPayload{}, err
		}
		viewerRating := any(nil)
		if len(results) > 1 && len(results[1]) > 0 {
			viewerRating = results[1][0]["rating"]
		}
		for _, key := range []string{"owner_corp_id", "owner_alliance_id"} {
			delete(fit, key)
		}
		return accountNoStorePayload(map[string]any{
			"fit": fit, "items": nonNilFittingRows(results[0]),
			"viewer_rating": viewerRating,
		}), nil
	}
}

func canViewFitting(principal *Principal, fit map[string]any) bool {
	visibility := int64OrZero(fit["visibility"])
	if visibility == 3 {
		return true
	}
	if principal == nil {
		return false
	}
	owner := int64OrZero(fit["owner_character_id"])
	if owner == int64(principal.CharacterID) {
		return true
	}
	switch visibility {
	case 1:
		return principal.CorporationID != nil &&
			int64(*principal.CorporationID) == int64OrZero(fit["owner_corp_id"])
	case 2:
		return principal.AllianceID != nil &&
			int64(*principal.AllianceID) == int64OrZero(fit["owner_alliance_id"])
	default:
		return false
	}
}

func updateFittingHandler(
	opts Options,
	auth *authService,
	parameter string,
) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		principal, err := fittingMutationPrincipal(ctx, req, auth)
		if err != nil {
			return legacyPayload{}, err
		}
		id, err := fittingID(req, parameter)
		if err != nil {
			return legacyPayload{}, err
		}
		raw, err := decodeJSONBody[fittingUpdateBody](req, fittingBodyLimit)
		if err != nil {
			return legacyPayload{}, err
		}
		patch, err := validateUpdateFitting(raw)
		if err != nil {
			return legacyPayload{}, err
		}
		db, err := mutationDatabase(opts)
		if err != nil {
			return legacyPayload{}, err
		}
		owner, err := queryMap(ctx, opts.DB, `
			SELECT owner_character_id FROM user_fittings
			WHERE fit_id = $1 LIMIT 1`, id)
		if err != nil {
			return legacyPayload{}, err
		}
		if owner == nil {
			return legacyPayload{}, apiError(http.StatusNotFound, "Fit not found")
		}
		if int64OrZero(owner["owner_character_id"]) != int64(principal.CharacterID) {
			return legacyPayload{}, apiError(http.StatusForbidden, "Not your fit")
		}

		tx, err := db.Begin(ctx)
		if err != nil {
			return legacyPayload{}, err
		}
		defer tx.Rollback(ctx) //nolint:errcheck
		if _, err := tx.Exec(ctx, `
			UPDATE user_fittings
			SET name = CASE WHEN $2::boolean THEN $3 ELSE name END,
			    description = CASE WHEN $4::boolean THEN $5 ELSE description END,
			    visibility = CASE WHEN $6::boolean THEN $7 ELSE visibility END,
			    updated_at = NOW()
			WHERE fit_id = $1`,
			id, patch.Name != nil, pointerStringValue(patch.Name),
			patch.HasDescription, patch.Description,
			patch.Visibility != nil, pointerInt16Value(patch.Visibility),
		); err != nil {
			return legacyPayload{}, err
		}
		if patch.HasItems {
			if _, err := tx.Exec(
				ctx, `DELETE FROM user_fitting_items WHERE fit_id = $1`, id,
			); err != nil {
				return legacyPayload{}, err
			}
			if err := copyFittingItems(ctx, tx, id, patch.Items); err != nil {
				return legacyPayload{}, err
			}
		}
		if err := tx.Commit(ctx); err != nil {
			return legacyPayload{}, err
		}
		updated, err := queryMap(ctx, opts.DB,
			`SELECT * FROM user_fittings WHERE fit_id = $1 LIMIT 1`, id)
		if err != nil {
			return legacyPayload{}, err
		}
		items, err := queryMaps(ctx, opts.DB, `
			SELECT slot_group, ordinal, type_id, state,
			       charge_type_id, quantity
			FROM user_fitting_items
			WHERE fit_id = $1
			ORDER BY slot_group, ordinal`, id)
		if err != nil {
			return legacyPayload{}, err
		}
		return accountNoStorePayload(map[string]any{
			"fit": updated, "items": nonNilFittingRows(items),
		}), nil
	}
}

func pointerStringValue(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func pointerInt16Value(value *int16) any {
	if value == nil {
		return nil
	}
	return *value
}

func deleteFittingHandler(
	opts Options,
	auth *authService,
	parameter string,
) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		principal, err := fittingMutationPrincipal(ctx, req, auth)
		if err != nil {
			return legacyPayload{}, err
		}
		id, err := fittingID(req, parameter)
		if err != nil {
			return legacyPayload{}, err
		}
		owner, err := queryMap(ctx, opts.DB, `
			SELECT owner_character_id FROM user_fittings
			WHERE fit_id = $1 LIMIT 1`, id)
		if err != nil {
			return legacyPayload{}, err
		}
		if owner == nil {
			return legacyPayload{}, apiError(http.StatusNotFound, "Fit not found")
		}
		if int64OrZero(owner["owner_character_id"]) != int64(principal.CharacterID) &&
			!principal.IsAdmin {
			return legacyPayload{}, apiError(http.StatusForbidden, "Not your fit")
		}
		db, err := mutationDatabase(opts)
		if err != nil {
			return legacyPayload{}, err
		}
		if _, err := db.Exec(
			ctx, `DELETE FROM user_fittings WHERE fit_id = $1`, id,
		); err != nil {
			return legacyPayload{}, err
		}
		return accountNoStorePayload(map[string]any{"ok": true}), nil
	}
}

func putFittingRatingHandler(
	opts Options,
	auth *authService,
	parameter string,
) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		principal, err := fittingMutationPrincipal(ctx, req, auth)
		if err != nil {
			return legacyPayload{}, err
		}
		id, err := fittingID(req, parameter)
		if err != nil {
			return legacyPayload{}, err
		}
		body, err := decodeJSONBody[fittingRatingBody](req, fittingBodyLimit)
		if err != nil {
			return legacyPayload{}, err
		}
		rating := int64(0)
		ok := body.Rating != nil
		if ok {
			rating = *body.Rating
		}
		if !ok || rating < 1 || rating > 5 {
			return legacyPayload{}, apiError(
				http.StatusUnprocessableEntity,
				"rating must be an integer in [1, 5]",
			)
		}
		fit, err := queryMap(ctx, opts.DB, `
			SELECT f.owner_character_id, f.visibility,
			       owner.corporation_id AS owner_corp_id,
			       owner.alliance_id AS owner_alliance_id
			FROM user_fittings f
			LEFT JOIN characters owner
			  ON owner.character_id = f.owner_character_id
			WHERE f.fit_id = $1 LIMIT 1`, id)
		if err != nil {
			return legacyPayload{}, err
		}
		if fit == nil {
			return legacyPayload{}, apiError(http.StatusNotFound, "Fit not found")
		}
		if !canViewFitting(principal, fit) {
			return legacyPayload{}, apiError(
				http.StatusForbidden, "Fit is not visible to you",
			)
		}
		db, err := mutationDatabase(opts)
		if err != nil {
			return legacyPayload{}, err
		}
		tx, err := db.Begin(ctx)
		if err != nil {
			return legacyPayload{}, err
		}
		defer tx.Rollback(ctx) //nolint:errcheck
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_fitting_ratings (
				fit_id, rater_character_id, rating
			) VALUES ($1, $2, $3)
			ON CONFLICT (fit_id, rater_character_id) DO UPDATE
			SET rating = EXCLUDED.rating, updated_at = NOW()`,
			id, principal.CharacterID, rating,
		); err != nil {
			return legacyPayload{}, err
		}
		aggregate, err := updateFittingRatingAggregate(ctx, tx, id)
		if err != nil {
			return legacyPayload{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return legacyPayload{}, err
		}
		return accountNoStorePayload(map[string]any{
			"rating": rating, "aggregate": aggregate,
		}), nil
	}
}

func deleteFittingRatingHandler(
	opts Options,
	auth *authService,
	parameter string,
) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		principal, err := fittingMutationPrincipal(ctx, req, auth)
		if err != nil {
			return legacyPayload{}, err
		}
		id, err := fittingID(req, parameter)
		if err != nil {
			return legacyPayload{}, err
		}
		db, err := mutationDatabase(opts)
		if err != nil {
			return legacyPayload{}, err
		}
		tx, err := db.Begin(ctx)
		if err != nil {
			return legacyPayload{}, err
		}
		defer tx.Rollback(ctx) //nolint:errcheck
		if _, err := tx.Exec(ctx, `
			DELETE FROM user_fitting_ratings
			WHERE fit_id = $1 AND rater_character_id = $2`,
			id, principal.CharacterID,
		); err != nil {
			return legacyPayload{}, err
		}
		aggregate, err := updateFittingRatingAggregate(ctx, tx, id)
		if err != nil {
			return legacyPayload{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return legacyPayload{}, err
		}
		return accountNoStorePayload(map[string]any{
			"deleted": true, "aggregate": aggregate,
		}), nil
	}
}

func updateFittingRatingAggregate(
	ctx context.Context,
	tx pgx.Tx,
	id string,
) (map[string]any, error) {
	var average *float64
	var count int64
	if err := tx.QueryRow(ctx, `
		WITH aggregate AS (
			SELECT AVG(rating)::double precision AS rating_avg,
			       COUNT(*)::bigint AS rating_count
			FROM user_fitting_ratings WHERE fit_id = $1
		), updated AS (
			UPDATE user_fittings f
			SET rating_avg = aggregate.rating_avg,
			    rating_count = aggregate.rating_count::int,
			    updated_at = NOW()
			FROM aggregate
			WHERE f.fit_id = $1
			RETURNING aggregate.rating_avg, aggregate.rating_count
		)
		SELECT rating_avg, rating_count FROM updated
		UNION ALL
		SELECT rating_avg, rating_count FROM aggregate
		WHERE NOT EXISTS (SELECT 1 FROM updated)
		LIMIT 1`, id).Scan(&average, &count); err != nil {
		return nil, err
	}
	return map[string]any{
		"rating_avg": average, "rating_count": count,
	}, nil
}

func nonNilFittingRows(rows []map[string]any) []map[string]any {
	if rows == nil {
		return []map[string]any{}
	}
	return rows
}
