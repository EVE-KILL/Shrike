package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgconn"
)

const maxSavedSearches = 50

// Version 1 stores rolling presets, not their resolved dates. Display names
// are hints only; the existing advanced-query builder uses canonical IDs.
type savedSearchDocument struct {
	Version    int             `json:"version" enum:"1"`
	Filters    advancedFilters `json:"filters"`
	View       string          `json:"view" enum:"kills,fits"`
	Dedup      string          `json:"dedup" enum:"none,exact,family"`
	FitHash    string          `json:"fitHash,omitempty"`
	FamilyHash string          `json:"familyHash,omitempty"`
}

type savedSearchBody struct {
	Name     string              `json:"name" minLength:"1" maxLength:"100"`
	Document savedSearchDocument `json:"document"`
	Revision int                 `json:"revision,omitempty" minimum:"0" doc:"Required current revision when replacing a saved search."`
}

type savedSearchRecord struct {
	ID        int64               `json:"id"`
	Name      string              `json:"name"`
	Document  savedSearchDocument `json:"document"`
	Revision  int                 `json:"revision"`
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`
}

type savedSearchService struct {
	auth *authService
	db   MutationDatabase
	err  error
}

func (s *savedSearchService) principal(ctx context.Context, req *legacyRequest) (*Principal, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.auth.requirePrincipal(ctx, req)
}

func registerSavedSearchRoutes(a huma.API, opts Options, security []map[string][]string) {
	db, err := mutationDatabase(opts)
	service := &savedSearchService{auth: newAuthService(opts), db: db, err: err}
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete} {
		path := "/me/saved-searches"
		if method == http.MethodPut || method == http.MethodDelete {
			path += "/{id}"
		}
		op := huma.Operation{OperationID: "account-saved-searches-" + strings.ToLower(method), Method: method, Path: path,
			Summary: map[string]string{"GET": "List account saved searches", "POST": "Create a saved search", "PUT": "Replace a saved search", "DELETE": "Delete a saved search"}[method],
			Tags:    []string{"account", "search"}, Security: security}
		if method == http.MethodPost || method == http.MethodPut {
			op = documentJSONBody[savedSearchBody](a, op)
		}
		schema := huma.SchemaFromType(a.OpenAPI().Components.Schemas, reflect.TypeOf(savedSearchRecord{}))
		if method == http.MethodGet {
			schema = responseSchema(map[string]*huma.Schema{"searches": arraySchema(schema), "limit": intSchema()}, "searches", "limit")
		}
		if method == http.MethodDelete {
			schema = responseSchema(map[string]*huma.Schema{"deleted": boolSchema()}, "deleted")
		}
		op.Responses = map[string]*huma.Response{"200": {Description: "Success", Content: map[string]*huma.MediaType{"application/json": {Schema: schema}}}}
		registerLegacy(a, op, service.savedSearchHandler(method))
	}
}

func validateSavedSearch(body *savedSearchBody) error {
	invalid := func(message string) error { return apiError(http.StatusBadRequest, message) }
	body.Name = strings.TrimSpace(body.Name)
	if n := utf8.RuneCountInString(body.Name); n < 1 || n > 100 || strings.ContainsRune(body.Name, 0) {
		return invalid("Search name must contain 1–100 characters")
	}
	doc := &body.Document
	if doc.Version != 1 {
		return invalid("Unsupported saved-search version")
	}
	if !slices.Contains([]string{"kills", "fits"}, doc.View) || !slices.Contains([]string{"none", "exact", "family"}, doc.Dedup) {
		return invalid("Invalid search view")
	}
	if doc.FitHash != "" && doc.FamilyHash != "" {
		return invalid("Use either an exact fit or a fitting family")
	}
	f := &doc.Filters
	if f.TimeRange == nil {
		f.TimeRange = &advancedTimeRange{Preset: "30d"}
	}
	if p := f.TimeRange.Preset; p != "" {
		if !slices.Contains([]string{"today", "yesterday", "24h", "7d", "30d", "90d", "thisWeek", "thisMonth"}, p) || f.TimeRange.From != "" || f.TimeRange.To != "" {
			return invalid("Invalid time range")
		}
	}
	if f.Sort != nil && (!slices.Contains([]string{"killmail_time", "total_value", "attacker_count"}, f.Sort.Field) || !slices.Contains([]string{"asc", "desc"}, f.Sort.Direction)) {
		return invalid("Invalid search sort")
	}
	if !slices.Contains([]string{"", "npc", "ganked"}, f.AttackerType) {
		return invalid("Invalid attacker type")
	}
	if f.AttackerCount != "" && f.AttackerCount != "solo" {
		n, err := strconv.ParseInt(strings.TrimSuffix(f.AttackerCount, "+"), 10, 32)
		if err != nil || n < 1 || !strings.HasSuffix(f.AttackerCount, "+") {
			return invalid("Invalid attacker count")
		}
	}
	if !slices.Contains([]string{"", "1b", "5b", "10b", "50b", "100b"}, f.ISKValue) || !slices.Contains([]string{"", "t1", "t2", "t3", "faction"}, f.TechLevel) || !slices.Contains([]string{"", "frigates", "destroyers", "cruisers", "battlecruisers", "battleships", "capitals", "supercarriers", "titans", "supercapitals", "freighters", "citadels"}, f.ShipCategory) {
		return invalid("Invalid search filter")
	}
	if (f.ISKMin != nil && *f.ISKMin < 0) || (f.ISKMax != nil && *f.ISKMax < 0) || (f.ISKMin != nil && f.ISKMax != nil && *f.ISKMin > *f.ISKMax) {
		return invalid("Invalid ISK range")
	}
	if f.Location != nil {
		for _, value := range f.Location.SecurityTypes {
			if !slices.Contains([]string{"highsec", "lowsec", "nullsec", "wspace", "abyssal", "pochven"}, value) {
				return invalid("Invalid security type")
			}
		}
		for _, id := range []int64{f.Location.SystemID, f.Location.RegionID, f.Location.ConstellationID} {
			if id < 0 || id > 2147483647 {
				return invalid("Invalid location ID")
			}
		}
	}

	if f.Entities != nil {
		for _, list := range [][]advancedEntity{f.Entities.Victim, f.Entities.Attacker, f.Entities.Both} {
			for _, entity := range list {
				if entity.ID <= 0 || entity.ID > 2147483647 || !slices.Contains([]string{"character", "corporation", "alliance", "faction", "ship", "shipgroup", "weapon"}, entity.Type) {
					return invalid("Invalid search entity")
				}
			}
		}
	}
	for _, item := range f.Items {
		if (item.TypeID == nil) == (item.GroupID == nil) {
			return invalid("Specify one item type or group")
		}
		for _, id := range []*int64{item.TypeID, item.GroupID} {
			if id != nil && (*id <= 0 || *id > 2147483647) {
				return invalid("Invalid item ID")
			}
		}
		if !slices.Contains([]string{"", "fitted", "cargo", "any"}, item.Slot) || !slices.Contains([]string{"", "victim", "attacker", "either"}, item.Side) {
			return invalid("Invalid item filter")
		}
	}
	raw, err := json.Marshal(f)
	if err != nil {
		return invalid("Invalid search filters")
	}
	_, err = parseAdvancedKilllistQuery(&legacyRequest{Query: url.Values{"filters": {string(raw)}, "view": {doc.View}, "dedup": {doc.Dedup}, "fitHash": {doc.FitHash}, "familyHash": {doc.FamilyHash}}}, time.Now())
	if err != nil {
		return err
	}
	raw, err = json.Marshal(doc)
	if err != nil || len(raw) > 12000 || strings.Contains(string(raw), `\u0000`) {
		return invalid("Saved search is too large")
	}
	return nil
}

const savedSearchColumns = `search_id AS id, name, document, revision, created_at, updated_at`

func savedSearchWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return apiError(http.StatusConflict, "A search with this name already exists. Choose another name.")
	}
	return err
}

func (s *savedSearchService) savedSearchHandler(method string) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		setAccountNoStore(req.Huma)
		if method != http.MethodGet {
			if err := requireSameOriginMutation(req.Huma); err != nil {
				return legacyPayload{}, err
			}
		}
		principal, err := s.principal(ctx, req)
		if err != nil {
			return legacyPayload{}, err
		}
		owner := principal.CharacterID
		if method == http.MethodGet {
			rows, err := queryMaps(ctx, s.db, `SELECT `+savedSearchColumns+` FROM account_saved_searches WHERE character_id=$1 ORDER BY updated_at DESC, search_id DESC LIMIT 50`, owner)
			return accountNoStorePayload(map[string]any{"searches": rows, "limit": maxSavedSearches}), err
		}
		var id int64
		if method != http.MethodPost {
			id, err = strconv.ParseInt(req.Param("id"), 10, 64)
			if err != nil || id <= 0 {
				return legacyPayload{}, apiError(http.StatusBadRequest, "Invalid saved-search ID")
			}
			if err != nil {
				return legacyPayload{}, err
			}
		}
		if method == http.MethodDelete {
			revision, err := strconv.Atoi(req.Query.Get("revision"))
			if err != nil || revision < 1 {
				return legacyPayload{}, apiError(http.StatusBadRequest, "Current revision is required")
			}
			result, err := s.db.Exec(ctx, `DELETE FROM account_saved_searches WHERE search_id=$1 AND character_id=$2 AND revision=$3`, id, owner, revision)
			if err != nil {
				return legacyPayload{}, err
			}
			if result.RowsAffected() == 0 {
				return legacyPayload{}, apiError(http.StatusConflict, "Search changed or is unavailable. Refresh and try again.")
			}
			return accountNoStorePayload(map[string]bool{"deleted": true}), nil
		}
		body, err := decodeJSONBody[savedSearchBody](req, 16384)
		if err != nil {
			return legacyPayload{}, err
		}
		if err := validateSavedSearch(body); err != nil {
			return legacyPayload{}, err
		}
		raw, _ := json.Marshal(body.Document)
		if method == http.MethodPut {
			if body.Revision < 1 {
				return legacyPayload{}, apiError(http.StatusBadRequest, "Current revision is required")
			}
			row, err := queryMap(ctx, s.db, `UPDATE account_saved_searches SET name=$3, document=$4::jsonb, revision=revision+1, updated_at=now() WHERE search_id=$1 AND character_id=$2 AND revision=$5 RETURNING `+savedSearchColumns, id, owner, body.Name, raw, body.Revision)
			if err != nil {
				return legacyPayload{}, savedSearchWriteError(err)
			}
			if row == nil {
				return legacyPayload{}, apiError(http.StatusConflict, "Search changed or is unavailable. Refresh and try again.")
			}
			return accountNoStorePayload(row), nil
		}
		tx, err := s.db.Begin(ctx)
		if err != nil {
			return legacyPayload{}, err
		}
		defer tx.Rollback(ctx) //nolint:errcheck
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1,$2)`, 20260907, owner); err != nil {
			return legacyPayload{}, err
		}
		var count int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM account_saved_searches WHERE character_id=$1`, owner).Scan(&count); err != nil {
			return legacyPayload{}, err
		}
		if count >= maxSavedSearches {
			return legacyPayload{}, apiError(http.StatusConflict, "Saved-search limit reached (50)")
		}
		row, err := queryMap(ctx, txDatabase{tx}, `INSERT INTO account_saved_searches (character_id,name,document) VALUES ($1,$2,$3::jsonb) RETURNING `+savedSearchColumns, owner, body.Name, raw)
		if err != nil {
			return legacyPayload{}, savedSearchWriteError(err)
		}
		if err := tx.Commit(ctx); err != nil {
			return legacyPayload{}, err
		}
		return accountNoStorePayload(row), nil
	}
}
