package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/eve-kill/shrike/internal/queue"
)

const (
	domainBannerMaximumSize     = 4 << 20
	domainLogoMaximumSize       = 2 << 20
	domainBackgroundMaximumSize = 6 << 20
	domainAssetMultipartSlop    = 64 << 10
	domainMaximumBackgrounds    = 8
)

var domainAllowedImageTypes = map[string]struct{}{
	"image/jpeg": {}, "image/png": {}, "image/webp": {}, "image/gif": {},
}

// DomainAssetStorage is the object-store boundary used by custom-domain
// images. The API owns key generation and validates every key read from the
// database before calling this interface; an implementation should treat a
// nil body and nil error from Get as "not found".
//
// Keeping this boundary small lets production use its S3-compatible B2 bucket
// while tests use an in-memory store without coupling the API to an SDK.
type DomainAssetStorage interface {
	Put(context.Context, string, []byte, string) error
	Get(context.Context, string) ([]byte, error)
	Delete(context.Context, string) error
}

type domainAssetUpload struct {
	Type         string
	Body         []byte
	DeclaredMIME string
	DetectedMIME string
	Hash         string
}

type domainStorageReference struct {
	AssetID  int32
	DomainID int32
	Type     string
	Key      string
}

type domainAssetEventDispatcher interface {
	AssetPending(
		context.Context,
		domainPendingAssetEvent,
	)
}

type domainPendingAssetEvent struct {
	DomainID    int32
	DomainLabel string
	AssetID     int32
	AssetKind   string
	AssetURL    string
	Uploader    Principal
}

type discordDomainAssetArgs struct {
	Type        string `json:"type"`
	DomainID    int32  `json:"domainId"`
	DomainLabel string `json:"domainLabel"`
	AssetID     int32  `json:"assetId"`
	AssetKind   string `json:"assetKind"`
	AssetURL    string `json:"assetUrl"`
	Uploader    struct {
		ID              int32   `json:"id"`
		Name            string  `json:"name"`
		CorporationID   *int32  `json:"corporation_id"`
		CorporationName *string `json:"corporation_name"`
		AllianceID      *int32  `json:"alliance_id"`
		AllianceName    *string `json:"alliance_name"`
	} `json:"uploader"`
}

func (discordDomainAssetArgs) Kind() string { return "discord_events" }

type riverDomainAssetEventDispatcher struct {
	client *queue.Client
}

func (d *riverDomainAssetEventDispatcher) AssetPending(
	ctx context.Context,
	event domainPendingAssetEvent,
) {
	if d == nil || d.client == nil {
		return
	}
	args := discordDomainAssetArgs{
		Type: "upload.custom_asset", DomainID: event.DomainID,
		DomainLabel: event.DomainLabel, AssetID: event.AssetID,
		AssetKind: event.AssetKind, AssetURL: event.AssetURL,
	}
	args.Uploader.ID = event.Uploader.CharacterID
	args.Uploader.Name = event.Uploader.CharacterName
	args.Uploader.CorporationID = event.Uploader.CorporationID
	args.Uploader.CorporationName = event.Uploader.CorporationName
	args.Uploader.AllianceID = event.Uploader.AllianceID
	args.Uploader.AllianceName = event.Uploader.AllianceName
	_, _ = queue.Dispatch(
		context.WithoutCancel(ctx), d.client, args, queue.Live,
	)
}

func (s *domainService) uploadHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		principal, err := s.requireAccount(ctx, req, true)
		if err != nil {
			return legacyPayload{}, err
		}
		if s.assets == nil {
			return legacyPayload{}, apiError(
				http.StatusServiceUnavailable,
				"Domain asset storage is not configured",
			)
		}
		id, err := domainID(req.Param("id"), "Invalid domain ID")
		if err != nil {
			return legacyPayload{}, err
		}
		upload, err := parseDomainAssetUpload(req)
		if err != nil {
			return legacyPayload{}, err
		}
		asset, label, err := s.createPendingDomainAsset(
			ctx, id, *principal, upload,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		assetID := int32From(asset["id"])
		key, ok := domainAssetStorageKey(id, upload.Type, assetID)
		if !ok {
			_ = s.removePendingDomainAsset(
				context.WithoutCancel(ctx), assetID,
			)
			return legacyPayload{}, errors.New(
				"could not generate domain asset storage key",
			)
		}
		if err := s.assets.Put(
			ctx, key, upload.Body, upload.DetectedMIME,
		); err != nil {
			_ = s.removePendingDomainAsset(
				context.WithoutCancel(ctx), assetID,
			)
			return legacyPayload{}, err
		}
		if err := s.setDomainAssetStorageKey(ctx, assetID, key); err != nil {
			_ = s.assets.Delete(context.WithoutCancel(ctx), key)
			_ = s.removePendingDomainAsset(
				context.WithoutCancel(ctx), assetID,
			)
			return legacyPayload{}, err
		}

		token := s.domainPreviewToken(assetID, upload.Hash)
		previewURL := fmt.Sprintf(
			"https://eve-kill.com/api/domains/preview/%d",
			assetID,
		)
		if token != "" {
			previewURL += "?token=" + token
		}
		if s.dispatcher != nil {
			s.dispatcher.AssetPending(ctx, domainPendingAssetEvent{
				DomainID: id, DomainLabel: label, AssetID: assetID,
				AssetKind: upload.Type, AssetURL: previewURL,
				Uploader: *principal,
			})
		}
		return jsonPayload(map[string]any{
			"assetId": assetID,
			"status":  "pending",
			"message": "Image uploaded and pending admin approval",
		}), nil
	}
}

// Wire types for the domain asset routes.
type domainAssetTypeBody struct {
	Type string `json:"type,omitempty" doc:"Asset slot to clear: background, preview or icon."`
}

type domainAssetReviewBody struct {
	Action string `json:"action,omitempty" doc:"Review outcome for the uploaded asset."`
	Reason string `json:"reason,omitempty" doc:"Operator note recorded with the decision."`
}

func (s *domainService) deleteAssetTypeHandler(
	legacyBody bool,
) legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		principal, err := s.requireAccount(ctx, req, true)
		if err != nil {
			return legacyPayload{}, err
		}
		id, err := domainID(req.Param("id"), "Invalid domain ID")
		if err != nil {
			return legacyPayload{}, err
		}
		assetType := strings.TrimSpace(req.Query.Get("type"))
		if legacyBody {
			body, bodyErr := decodeJSONBody[domainAssetTypeBody](req, domainBodyLimit)
			if bodyErr != nil {
				return legacyPayload{}, bodyErr
			}
			assetType = body.Type
		}
		if assetType != "banner" && assetType != "logo" {
			return legacyPayload{}, apiError(
				http.StatusBadRequest,
				`Missing or invalid type (must be "banner" or "logo")`,
			)
		}
		refs, err := s.deleteDomainAssetType(
			ctx, id, principal.CharacterID, assetType,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		s.deleteStorageKeys(ctx, refs)
		return jsonPayload(map[string]any{"success": true}), nil
	}
}

func (s *domainService) deleteAssetHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		principal, err := s.requireAccount(ctx, req, true)
		if err != nil {
			return legacyPayload{}, err
		}
		id, err := domainID(req.Param("id"), "Invalid IDs")
		if err != nil {
			return legacyPayload{}, err
		}
		assetID, err := domainID(req.Param("assetId"), "Invalid IDs")
		if err != nil {
			return legacyPayload{}, err
		}
		ref, err := s.deleteDomainAsset(
			ctx, id, assetID, principal.CharacterID,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		s.deleteStorageKeys(ctx, []domainStorageReference{ref})
		return jsonPayload(map[string]any{"success": true}), nil
	}
}

func (s *domainService) reviewAssetHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		admin, err := s.requireAdmin(ctx, req, true)
		if err != nil {
			return legacyPayload{}, err
		}
		domainIDValue, err := domainID(req.Param("id"), "Invalid IDs")
		if err != nil {
			return legacyPayload{}, err
		}
		assetID, err := domainID(req.Param("assetId"), "Invalid IDs")
		if err != nil {
			return legacyPayload{}, err
		}
		body, err := decodeJSONBody[domainAssetReviewBody](req, domainBodyLimit)
		if err != nil {
			return legacyPayload{}, err
		}
		action := body.Action
		if action != "approve" && action != "reject" {
			return legacyPayload{}, apiError(
				http.StatusBadRequest,
				`Action must be "approve" or "reject"`,
			)
		}
		var reason *string
		if action == "reject" {
			if raw := body.Reason; raw != "" {
				raw = strings.TrimSpace(raw)
				runes := []rune(raw)
				if len(runes) > 500 {
					raw = string(runes[:500])
				}
				if raw != "" {
					reason = &raw
				}
			}
		}
		status, err := s.reviewDomainAsset(
			ctx, domainIDValue, assetID, admin.CharacterID,
			action, reason,
		)
		if err != nil {
			return legacyPayload{}, err
		}
		return jsonPayload(map[string]any{
			"success": true, "status": status,
		}), nil
	}
}

func (s *domainService) publicSlotAssetHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		id, err := domainID(req.Param("id"), "Invalid asset path")
		if err != nil {
			return legacyPayload{}, err
		}
		assetType := req.Param("type")
		if assetType != "banner" && assetType != "logo" {
			return legacyPayload{}, apiError(
				http.StatusBadRequest, "Invalid asset path",
			)
		}
		asset, err := domainQueryMap(ctx, s.db, `
			SELECT id, domain_id, type, status, storage_key,
			       content_type, file_size, file_hash
			FROM domain_assets
			WHERE domain_id = $1 AND type = $2
			  AND status = 'approved'
			ORDER BY created_at DESC, id DESC
			LIMIT 1`, id, assetType)
		if err != nil {
			return legacyPayload{}, err
		}
		if asset == nil {
			return legacyPayload{}, apiError(
				http.StatusNotFound, "Asset not found",
			)
		}
		return s.domainAssetPayload(
			ctx, asset,
			"public, max-age=86400, s-maxage=604800",
			"Asset not found in storage",
		)
	}
}

func (s *domainService) publicBackgroundHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		assetID, err := domainID(
			req.Param("assetId"), "Invalid asset ID",
		)
		if err != nil {
			return legacyPayload{}, err
		}
		asset, err := domainQueryMap(ctx, s.db, `
			SELECT id, domain_id, type, status, storage_key,
			       content_type, file_size, file_hash
			FROM domain_assets
			WHERE id = $1 AND type = 'background'
			  AND status = 'approved'
			LIMIT 1`, assetID)
		if err != nil {
			return legacyPayload{}, err
		}
		if asset == nil {
			return legacyPayload{}, apiError(
				http.StatusNotFound, "Background not found",
			)
		}
		return s.domainAssetPayload(
			ctx, asset,
			"public, max-age=86400, s-maxage=604800",
			"Background not found in storage",
		)
	}
}

func (s *domainService) publicPreviewHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		assetID, err := domainID(
			req.Param("assetId"), "Invalid asset ID",
		)
		if err != nil {
			return legacyPayload{}, err
		}
		asset, err := domainQueryMap(ctx, s.db, `
			SELECT asset.id, asset.domain_id, asset.type, asset.status,
			       asset.storage_key, asset.content_type, asset.file_size,
			       asset.file_hash, domain.user_id
			FROM domain_assets asset
			JOIN custom_domains domain ON domain.id = asset.domain_id
			WHERE asset.id = $1
			LIMIT 1`, assetID)
		if err != nil {
			return legacyPayload{}, err
		}
		if asset == nil {
			return legacyPayload{}, apiError(
				http.StatusNotFound, "Asset not found",
			)
		}
		status := fmt.Sprint(asset["status"])
		authorized := status == "approved"
		if !authorized && status == "pending" {
			authorized = s.validDomainPreviewToken(
				assetID, fmt.Sprint(asset["file_hash"]),
				req.Query.Get("token"),
			)
		}
		if !authorized {
			principal, authErr := s.auth.resolvePrincipal(ctx, req)
			if authErr != nil {
				return legacyPayload{}, authErr
			}
			authorized = principal != nil &&
				(principal.IsAdmin ||
					principal.CharacterID == int32From(asset["user_id"]))
		}
		if !authorized {
			// Do not disclose the existence or review status of an upload.
			return legacyPayload{}, apiError(
				http.StatusNotFound, "Asset not found",
			)
		}
		cacheControl := "private, no-store"
		if status == "approved" {
			cacheControl = "public, max-age=3600"
		}
		return s.domainAssetPayload(
			ctx, asset, cacheControl, "Asset not found in storage",
		)
	}
}

func (s *domainService) adminPreviewHandler() legacyHandler {
	return func(ctx context.Context, req *legacyRequest) (legacyPayload, error) {
		if _, err := s.requireAdmin(ctx, req, false); err != nil {
			return legacyPayload{}, err
		}
		domainIDValue, err := domainID(req.Param("id"), "Invalid IDs")
		if err != nil {
			return legacyPayload{}, err
		}
		assetID, err := domainID(req.Param("assetId"), "Invalid IDs")
		if err != nil {
			return legacyPayload{}, err
		}
		asset, err := domainQueryMap(ctx, s.db, `
			SELECT id, domain_id, type, status, storage_key,
			       content_type, file_size, file_hash
			FROM domain_assets
			WHERE id = $1 AND domain_id = $2
			LIMIT 1`, assetID, domainIDValue)
		if err != nil {
			return legacyPayload{}, err
		}
		if asset == nil {
			return legacyPayload{}, apiError(
				http.StatusNotFound, "Asset not found",
			)
		}
		return s.domainAssetPayload(
			ctx, asset, "private, no-store",
			"Asset not found in storage",
		)
	}
}

func parseDomainAssetUpload(
	req *legacyRequest,
) (domainAssetUpload, error) {
	contentType := req.Huma.Header("Content-Type")
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType != "multipart/form-data" ||
		params["boundary"] == "" {
		return domainAssetUpload{}, apiError(
			http.StatusBadRequest, "Expected multipart form data",
		)
	}
	limited := io.LimitReader(
		req.Body,
		domainBackgroundMaximumSize+domainAssetMultipartSlop+1,
	)
	reader := multipart.NewReader(limited, params["boundary"])
	var upload domainAssetUpload
	parts := 0
	for {
		part, nextErr := reader.NextPart()
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			return domainAssetUpload{}, apiError(
				http.StatusBadRequest, "Invalid multipart upload",
			)
		}
		parts++
		if parts > 8 {
			_ = part.Close()
			return domainAssetUpload{}, apiError(
				http.StatusBadRequest, "Too many multipart fields",
			)
		}
		switch part.FormName() {
		case "type":
			value, readErr := io.ReadAll(io.LimitReader(part, 33))
			if readErr != nil || len(value) > 32 {
				_ = part.Close()
				return domainAssetUpload{}, apiError(
					http.StatusBadRequest, "Invalid asset type",
				)
			}
			upload.Type = string(value)
		case "file":
			if part.FileName() == "" || upload.Body != nil {
				_ = part.Close()
				return domainAssetUpload{}, apiError(
					http.StatusBadRequest,
					"Exactly one image file is required",
				)
			}
			upload.DeclaredMIME = strings.ToLower(
				strings.TrimSpace(part.Header.Get("Content-Type")),
			)
			body, readErr := io.ReadAll(io.LimitReader(
				part, domainBackgroundMaximumSize+1,
			))
			if readErr != nil {
				_ = part.Close()
				return domainAssetUpload{}, apiError(
					http.StatusBadRequest, "Could not read image file",
				)
			}
			upload.Body = body
		}
		_ = part.Close()
	}
	if upload.Type != "banner" && upload.Type != "logo" &&
		upload.Type != "background" {
		return domainAssetUpload{}, apiError(
			http.StatusBadRequest,
			`Missing or invalid type (must be "banner", "logo", or "background")`,
		)
	}
	if upload.Body == nil {
		return domainAssetUpload{}, apiError(
			http.StatusBadRequest, "No image file provided",
		)
	}
	if _, ok := domainAllowedImageTypes[upload.DeclaredMIME]; !ok {
		return domainAssetUpload{}, apiError(
			http.StatusBadRequest,
			"Invalid image type. Allowed: JPEG, PNG, WebP, GIF",
		)
	}
	maximum := domainAssetMaximumSize(upload.Type)
	if len(upload.Body) > maximum {
		return domainAssetUpload{}, apiError(
			http.StatusBadRequest,
			fmt.Sprintf(
				"File too large. Maximum %d MB", maximum/(1<<20),
			),
		)
	}
	upload.DetectedMIME = detectDomainImageMIME(upload.Body)
	if upload.DetectedMIME == "" {
		return domainAssetUpload{}, apiError(
			http.StatusBadRequest,
			"File does not appear to be a valid image",
		)
	}
	hash := sha256.Sum256(upload.Body)
	upload.Hash = hex.EncodeToString(hash[:])
	return upload, nil
}

func detectDomainImageMIME(body []byte) string {
	if len(body) < 12 {
		return ""
	}
	switch {
	case body[0] == 0xff && body[1] == 0xd8 && body[2] == 0xff:
		return "image/jpeg"
	case body[0] == 0x89 && body[1] == 0x50 &&
		body[2] == 0x4e && body[3] == 0x47:
		return "image/png"
	case body[0] == 0x47 && body[1] == 0x49 &&
		body[2] == 0x46 && body[3] == 0x38:
		return "image/gif"
	case body[0] == 0x52 && body[1] == 0x49 &&
		body[2] == 0x46 && body[3] == 0x46 &&
		body[8] == 0x57 && body[9] == 0x45 &&
		body[10] == 0x42 && body[11] == 0x50:
		return "image/webp"
	default:
		return ""
	}
}

func domainAssetMaximumSize(assetType string) int {
	switch assetType {
	case "banner":
		return domainBannerMaximumSize
	case "logo":
		return domainLogoMaximumSize
	default:
		return domainBackgroundMaximumSize
	}
}

func domainAssetStorageKey(
	domainID int32,
	assetType string,
	assetID int32,
) (string, bool) {
	if domainID <= 0 || assetID <= 0 ||
		(assetType != "banner" && assetType != "logo" &&
			assetType != "background") {
		return "", false
	}
	return fmt.Sprintf(
		"domains/%d/%s_%d", domainID, assetType, assetID,
	), true
}

func validDomainStorageReference(ref domainStorageReference) bool {
	expected, ok := domainAssetStorageKey(
		ref.DomainID, ref.Type, ref.AssetID,
	)
	return ok && hmac.Equal([]byte(expected), []byte(ref.Key))
}

func (s *domainService) domainAssetPayload(
	ctx context.Context,
	asset map[string]any,
	cacheControl string,
	notFoundMessage string,
) (legacyPayload, error) {
	if s.assets == nil {
		return legacyPayload{}, apiError(
			http.StatusServiceUnavailable,
			"Domain asset storage is not configured",
		)
	}
	ref := domainStorageReference{
		AssetID:  int32From(asset["id"]),
		DomainID: int32From(asset["domain_id"]),
		Type:     fmt.Sprint(asset["type"]),
		Key:      fmt.Sprint(asset["storage_key"]),
	}
	if !validDomainStorageReference(ref) {
		return legacyPayload{}, errors.New(
			"domain asset has an invalid storage key",
		)
	}
	body, err := s.assets.Get(ctx, ref.Key)
	if err != nil {
		return legacyPayload{}, err
	}
	if body == nil {
		return legacyPayload{}, apiError(
			http.StatusNotFound, notFoundMessage,
		)
	}
	maximum := domainAssetMaximumSize(ref.Type)
	if len(body) > maximum {
		return legacyPayload{}, errors.New(
			"domain asset exceeds its configured size limit",
		)
	}
	if expectedSize, ok := int64Value(asset["file_size"]); ok &&
		expectedSize >= 0 && int64(len(body)) != expectedSize {
		return legacyPayload{}, errors.New(
			"domain asset size does not match database metadata",
		)
	}
	detected := detectDomainImageMIME(body)
	storedType := fmt.Sprint(asset["content_type"])
	if detected == "" || detected != storedType {
		return legacyPayload{}, errors.New(
			"domain asset type does not match database metadata",
		)
	}
	if expectedHash := fmt.Sprint(asset["file_hash"]); expectedHash != "" {
		digest := sha256.Sum256(body)
		actual := hex.EncodeToString(digest[:])
		if !hmac.Equal([]byte(actual), []byte(expectedHash)) {
			return legacyPayload{}, errors.New(
				"domain asset hash does not match database metadata",
			)
		}
	}
	headers := make(http.Header)
	headers.Set("Cache-Control", cacheControl)
	headers.Set("X-Content-Type-Options", "nosniff")
	headers.Set(
		"Content-Disposition",
		fmt.Sprintf(`inline; filename="%s-%d"`, ref.Type, ref.AssetID),
	)
	return legacyPayload{
		ContentType: detected, RawBody: body, Headers: headers,
	}, nil
}

func (s *domainService) domainPreviewToken(
	assetID int32,
	fileHash string,
) string {
	if len(s.auth.stateSecret) == 0 || assetID <= 0 || fileHash == "" {
		return ""
	}
	mac := hmac.New(sha256.New, s.auth.stateSecret)
	_, _ = fmt.Fprintf(
		mac, "domain-preview:v1:%d:%s", assetID, fileHash,
	)
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (s *domainService) validDomainPreviewToken(
	assetID int32,
	fileHash string,
	token string,
) bool {
	if token == "" || len(token) > 128 {
		return false
	}
	expected := s.domainPreviewToken(assetID, fileHash)
	return expected != "" &&
		hmac.Equal([]byte(expected), []byte(token))
}

func (s *domainService) createPendingDomainAsset(
	ctx context.Context,
	domainIDValue int32,
	principal Principal,
	upload domainAssetUpload,
) (map[string]any, string, error) {
	tx, err := s.mutations.Begin(ctx)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	domain, err := s.loadOwnedDomain(
		ctx, tx, domainIDValue, principal.CharacterID, true,
	)
	if err != nil {
		return nil, "", err
	}
	if domain == nil {
		return nil, "", apiError(
			http.StatusNotFound, "Domain not found",
		)
	}
	if upload.Type == "background" {
		count, err := domainQueryMap(ctx, tx, `
			SELECT COUNT(*)::integer AS count
			FROM domain_assets
			WHERE domain_id = $1 AND type = 'background'
			  AND status <> 'rejected'`, domainIDValue)
		if err != nil {
			return nil, "", err
		}
		if int32From(count["count"]) >= domainMaximumBackgrounds {
			return nil, "", apiError(
				http.StatusBadRequest,
				fmt.Sprintf(
					"Maximum %d backgrounds per domain",
					domainMaximumBackgrounds,
				),
			)
		}
	}
	rejected, err := domainQueryMap(ctx, tx, `
		SELECT id
		FROM domain_assets
		WHERE file_hash = $1 AND status = 'rejected'
		LIMIT 1`, upload.Hash)
	if err != nil {
		return nil, "", err
	}
	if rejected != nil {
		return nil, "", apiError(
			http.StatusBadRequest,
			"This image was previously rejected and cannot be uploaded again",
		)
	}
	asset, err := domainQueryMap(ctx, tx, `
		INSERT INTO domain_assets (
		  domain_id, type, status, storage_key, content_type,
		  file_size, file_hash, uploaded_by
		)
		VALUES ($1, $2, 'pending', '', $3, $4, $5, $6)
		RETURNING id, domain_id, type, status, storage_key,
		          content_type, file_size, file_hash, uploaded_by,
		          reviewed_by, reviewed_at, reject_reason, created_at`,
		domainIDValue, upload.Type, upload.DetectedMIME,
		len(upload.Body), upload.Hash, principal.CharacterID,
	)
	if err != nil {
		return nil, "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, "", err
	}
	label := fmt.Sprint(domain["subdomain"])
	if siteName, ok := domain["site_name"].(string); ok && siteName != "" {
		label = siteName
	}
	return asset, label, nil
}

func (s *domainService) setDomainAssetStorageKey(
	ctx context.Context,
	assetID int32,
	key string,
) error {
	tag, err := s.mutations.Exec(ctx, `
		UPDATE domain_assets
		SET storage_key = $2
		WHERE id = $1 AND status = 'pending'
		  AND storage_key = ''`, assetID, key)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errors.New(
			"pending domain asset disappeared before storage was linked",
		)
	}
	return nil
}

func (s *domainService) removePendingDomainAsset(
	ctx context.Context,
	assetID int32,
) error {
	_, err := s.mutations.Exec(ctx, `
		DELETE FROM domain_assets
		WHERE id = $1 AND status = 'pending'`, assetID)
	return err
}

func (s *domainService) deleteDomainAssetType(
	ctx context.Context,
	domainIDValue int32,
	userID int32,
	assetType string,
) ([]domainStorageReference, error) {
	tx, err := s.mutations.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	domain, err := s.loadOwnedDomain(
		ctx, tx, domainIDValue, userID, true,
	)
	if err != nil {
		return nil, err
	}
	if domain == nil {
		return nil, apiError(http.StatusNotFound, "Domain not found")
	}
	rows, err := domainQueryMaps(ctx, tx, `
		SELECT id, domain_id, type, storage_key
		FROM domain_assets
		WHERE domain_id = $1 AND type = $2
		FOR UPDATE`, domainIDValue, assetType)
	if err != nil {
		return nil, err
	}
	refs := domainStorageReferences(rows)
	if _, err := tx.Exec(ctx, `
		DELETE FROM domain_assets
		WHERE domain_id = $1 AND type = $2`,
		domainIDValue, assetType,
	); err != nil {
		return nil, err
	}
	themeKey := "bannerUrl"
	if assetType == "logo" {
		themeKey = "logoUrl"
	}
	if _, err := tx.Exec(ctx, `
		UPDATE custom_domains
		SET theme = COALESCE(theme, '{}'::jsonb) - $2,
		    updated_at = $3
		WHERE id = $1`, domainIDValue, themeKey, s.now().UTC()); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return refs, nil
}

func (s *domainService) deleteDomainAsset(
	ctx context.Context,
	domainIDValue int32,
	assetID int32,
	userID int32,
) (domainStorageReference, error) {
	tx, err := s.mutations.Begin(ctx)
	if err != nil {
		return domainStorageReference{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	domain, err := s.loadOwnedDomain(
		ctx, tx, domainIDValue, userID, true,
	)
	if err != nil {
		return domainStorageReference{}, err
	}
	if domain == nil {
		return domainStorageReference{}, apiError(
			http.StatusNotFound, "Domain not found",
		)
	}
	asset, err := domainQueryMap(ctx, tx, `
		SELECT id, domain_id, type, status, storage_key
		FROM domain_assets
		WHERE id = $1 AND domain_id = $2
		LIMIT 1
		FOR UPDATE`, assetID, domainIDValue)
	if err != nil {
		return domainStorageReference{}, err
	}
	if asset == nil {
		return domainStorageReference{}, apiError(
			http.StatusNotFound, "Asset not found",
		)
	}
	ref := domainStorageReference{
		AssetID: assetID, DomainID: domainIDValue,
		Type: fmt.Sprint(asset["type"]), Key: fmt.Sprint(asset["storage_key"]),
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM domain_assets WHERE id = $1`, assetID); err != nil {
		return domainStorageReference{}, err
	}
	if (ref.Type == "banner" || ref.Type == "logo") &&
		fmt.Sprint(asset["status"]) == "approved" {
		remaining, err := domainQueryMap(ctx, tx, `
			SELECT id
			FROM domain_assets
			WHERE domain_id = $1 AND type = $2
			  AND status = 'approved'
			LIMIT 1`, domainIDValue, ref.Type)
		if err != nil {
			return domainStorageReference{}, err
		}
		if remaining == nil {
			themeKey := "bannerUrl"
			if ref.Type == "logo" {
				themeKey = "logoUrl"
			}
			expectedURL := fmt.Sprintf(
				"/api/domains/asset/%d/%s",
				domainIDValue, ref.Type,
			)
			if _, err := tx.Exec(ctx, `
				UPDATE custom_domains
				SET theme = COALESCE(theme, '{}'::jsonb) - $2,
				    updated_at = $4
				WHERE id = $1 AND theme->>$2 = $3`,
				domainIDValue, themeKey, expectedURL, s.now().UTC(),
			); err != nil {
				return domainStorageReference{}, err
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return domainStorageReference{}, err
	}
	return ref, nil
}

func domainStorageReferences(
	rows []map[string]any,
) []domainStorageReference {
	result := make([]domainStorageReference, 0, len(rows))
	for _, row := range rows {
		result = append(result, domainStorageReference{
			AssetID:  int32From(row["id"]),
			DomainID: int32From(row["domain_id"]),
			Type:     fmt.Sprint(row["type"]),
			Key:      fmt.Sprint(row["storage_key"]),
		})
	}
	return result
}

func (s *domainService) deleteStorageKeys(
	ctx context.Context,
	refs []domainStorageReference,
) {
	if s.assets == nil {
		return
	}
	for _, ref := range refs {
		if ref.Key == "" || !validDomainStorageReference(ref) {
			continue
		}
		_ = s.assets.Delete(context.WithoutCancel(ctx), ref.Key)
	}
}

func (s *domainService) reviewDomainAsset(
	ctx context.Context,
	domainIDValue int32,
	assetID int32,
	reviewerID int32,
	action string,
	rejectReason *string,
) (string, error) {
	tx, err := s.mutations.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	asset, err := domainQueryMap(ctx, tx, `
		SELECT id, domain_id, type, status, storage_key
		FROM domain_assets
		WHERE id = $1 AND domain_id = $2
		LIMIT 1
		FOR UPDATE`, assetID, domainIDValue)
	if err != nil {
		return "", err
	}
	if asset == nil {
		return "", apiError(http.StatusNotFound, "Asset not found")
	}
	if fmt.Sprint(asset["status"]) != "pending" {
		return "", apiError(
			http.StatusConflict, "Asset already reviewed",
		)
	}
	if fmt.Sprint(asset["storage_key"]) == "" {
		return "", apiError(
			http.StatusConflict, "Asset upload is not complete",
		)
	}
	status := "rejected"
	if action == "approve" {
		status = "approved"
		rejectReason = nil
	}
	tag, err := tx.Exec(ctx, `
		UPDATE domain_assets
		SET status = $3, reviewed_by = $4, reviewed_at = $5,
		    reject_reason = $6
		WHERE id = $1 AND domain_id = $2 AND status = 'pending'`,
		assetID, domainIDValue, status, reviewerID, s.now().UTC(),
		rejectReason,
	)
	if err != nil {
		return "", err
	}
	if tag.RowsAffected() != 1 {
		return "", apiError(
			http.StatusConflict, "Asset already reviewed",
		)
	}
	assetType := fmt.Sprint(asset["type"])
	if action == "approve" &&
		(assetType == "banner" || assetType == "logo") {
		themeKey := "bannerUrl"
		if assetType == "logo" {
			themeKey = "logoUrl"
		}
		assetURL := fmt.Sprintf(
			"/api/domains/asset/%d/%s", domainIDValue, assetType,
		)
		if _, err := tx.Exec(ctx, `
			UPDATE custom_domains
			SET theme = jsonb_set(
			      COALESCE(theme, '{}'::jsonb),
			      ARRAY[$2]::text[], to_jsonb($3::text), true
			    ),
			    updated_at = $4
			WHERE id = $1`,
			domainIDValue, themeKey, assetURL, s.now().UTC(),
		); err != nil {
			return "", err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return status, nil
}
