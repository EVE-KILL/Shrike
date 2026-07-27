package images

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/rs/zerolog/log"
)

// Register adds the canonical /images namespace to the shared Huma document.
func Register(a huma.API, service *Service) {
	registerOverview(a)
	for _, kind := range []EntityKind{Character, Corporation, Alliance} {
		registerEntityRoute(a, service, kind, false)
		registerEntityRoute(a, service, kind, true)
	}
	registerTypeRoute(a, service)
	for _, category := range []string{
		"regions", "systems", "constellations", "ui",
	} {
		registerStaticRoute(a, service, category)
	}
	registerOldCharacterRoute(a, service)
	registerKillmailSocialRoute(a, service)
}

func registerOverview(a huma.API) {
	op := huma.Operation{
		OperationID: "images-overview",
		Method:      http.MethodGet,
		Path:        "/images",
		Summary:     "Image API overview",
		Tags:        []string{"images"},
		Servers:     imageServers(),
		Extensions:  map[string]any{"x-audience": "public"},
	}
	huma.Register(a, op, func(_ context.Context, _ *struct{}) (*struct {
		Body struct {
			Service string   `json:"service"`
			Routes  []string `json:"routes"`
		}
	}, error) {
		output := &struct {
			Body struct {
				Service string   `json:"service"`
				Routes  []string `json:"routes"`
			}
		}{}
		output.Body.Service = "EVE-KILL Images"
		output.Body.Routes = []string{
			"/images/characters/{id}/portrait",
			"/images/corporations/{id}/logo",
			"/images/alliances/{id}/logo",
			"/images/types/{id}/{variant}",
			"/images/regions/{id}",
			"/images/systems/{id}",
			"/images/constellations/{id}",
			"/images/ui/{name}",
			"/images/oldcharacters/{id}",
			"/images/killmail/{id}/social.png",
		}
		return output, nil
	})
}

func registerKillmailSocialRoute(a huma.API, service *Service) {
	registerBinary(a, huma.Operation{
		OperationID: "image-killmail-social",
		Method:      http.MethodGet,
		Path:        "/images/killmail/{id}/social.png",
		Summary:     "Killmail social card",
		Tags:        []string{"images"},
		Parameters: []*huma.Param{
			{Name: "id", In: "path", Required: true,
				Schema: &huma.Schema{Type: huma.TypeInteger, Format: "int64"}},
		},
	}, func(ctx huma.Context) (Result, error) {
		id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
		if err != nil {
			return Result{}, statusError(http.StatusBadRequest, "Invalid killmail ID", nil)
		}
		return service.KillmailSocial(ctx.Context(), id)
	})
}

func registerEntityRoute(
	a huma.API,
	service *Service,
	kind EntityKind,
	withVariant bool,
) {
	path := "/images/" + string(kind) + "/{id}"
	operationID := "image-" + strings.TrimSuffix(string(kind), "s")
	if withVariant {
		path += "/{variant}"
		operationID += "-variant"
	}
	registerBinary(a, huma.Operation{
		OperationID: operationID,
		Method:      http.MethodGet,
		Path:        path,
		Summary:     "EVE " + strings.TrimSuffix(string(kind), "s") + " image",
		Tags:        []string{"images"},
		Parameters: []*huma.Param{
			{Name: "id", In: "path", Required: true,
				Schema: &huma.Schema{Type: huma.TypeInteger, Format: "int64"}},
			imageSizeParam(false),
			imageFormatParam(),
			legacyImageTypeParam(),
		},
	}, func(ctx huma.Context) (Result, error) {
		id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
		if err != nil {
			return Result{}, statusError(http.StatusBadRequest, "Invalid image ID", nil)
		}
		format, err := requestedFormat(ctx, "")
		if err != nil {
			return Result{}, err
		}
		return service.Entity(
			ctx.Context(),
			kind,
			id,
			parseSize(ctx.Query("size")),
			format,
		)
	})
}

func registerTypeRoute(a huma.API, service *Service) {
	registerBinary(a, huma.Operation{
		OperationID: "image-type",
		Method:      http.MethodGet,
		Path:        "/images/types/{id}/{variant}",
		Summary:     "EVE inventory type image",
		Tags:        []string{"images"},
		Parameters: []*huma.Param{
			{Name: "id", In: "path", Required: true,
				Schema: &huma.Schema{Type: huma.TypeInteger, Format: "int64"}},
			{Name: "variant", In: "path", Required: true,
				Schema: &huma.Schema{Type: huma.TypeString}},
			imageSizeParam(false),
			imageFormatParam(),
			legacyImageTypeParam(),
		},
	}, func(ctx huma.Context) (Result, error) {
		id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
		if err != nil {
			return Result{}, statusError(http.StatusBadRequest, "Invalid image ID", nil)
		}
		format, err := requestedFormat(ctx, "")
		if err != nil {
			return Result{}, err
		}
		return service.Type(
			ctx.Context(),
			id,
			ctx.Param("variant"),
			parseSize(ctx.Query("size")),
			format,
		)
	})
}

func registerStaticRoute(a huma.API, service *Service, category string) {
	parameter := "id"
	if category == "ui" {
		parameter = "name"
	}
	registerBinary(a, huma.Operation{
		OperationID: "image-" + strings.TrimSuffix(category, "s"),
		Method:      http.MethodGet,
		Path:        "/images/" + category + "/{" + parameter + "}",
		Summary:     "EVE " + strings.TrimSuffix(category, "s") + " image",
		Tags:        []string{"images"},
		Parameters: []*huma.Param{
			{Name: parameter, In: "path", Required: true,
				Schema: &huma.Schema{Type: huma.TypeString}},
			imageSizeParam(true),
			imageFormatParam(),
			legacyImageTypeParam(),
		},
	}, func(ctx huma.Context) (Result, error) {
		format, err := requestedFormat(ctx, "png")
		if err != nil {
			return Result{}, err
		}
		return service.Static(
			ctx.Context(),
			category,
			ctx.Param(parameter),
			parseSize(ctx.Query("size")),
			format,
		)
	})
}

func registerOldCharacterRoute(a huma.API, service *Service) {
	registerBinary(a, huma.Operation{
		OperationID: "image-old-character",
		Method:      http.MethodGet,
		Path:        "/images/oldcharacters/{id}",
		Summary:     "Legacy EVE character portrait",
		Tags:        []string{"images"},
		Parameters: []*huma.Param{
			{Name: "id", In: "path", Required: true,
				Schema: &huma.Schema{Type: huma.TypeInteger, Format: "int64"}},
			imageFormatParam(),
			legacyImageTypeParam(),
		},
	}, func(ctx huma.Context) (Result, error) {
		id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
		if err != nil {
			return Result{}, statusError(http.StatusBadRequest, "Invalid image ID", nil)
		}
		format, err := requestedFormat(ctx, "jpeg")
		if err != nil {
			return Result{}, err
		}
		return service.OldCharacter(
			ctx.Context(),
			id,
			format,
		)
	})
}

func registerBinary(
	a huma.API,
	op huma.Operation,
	handler func(huma.Context) (Result, error),
) {
	op.Extensions = map[string]any{"x-audience": "public"}
	op.Servers = imageServers()
	op.Responses = binaryResponses()
	a.OpenAPI().AddOperation(&op)
	a.Adapter().Handle(&op, func(ctx huma.Context) {
		result, err := handler(ctx)
		if err != nil {
			writeError(ctx, err)
			return
		}
		writeResult(ctx, result)
	})
}

func imageServers() []*huma.Server {
	return []*huma.Server{{
		URL:         "/",
		Description: "EVE-KILL images",
	}}
}

func binaryResponses() map[string]*huma.Response {
	content := make(map[string]*huma.MediaType)
	for _, contentType := range []string{"image/jpeg", "image/png", "image/webp"} {
		content[contentType] = &huma.MediaType{
			Schema: &huma.Schema{Type: huma.TypeString, Format: "binary"},
		}
	}
	return map[string]*huma.Response{
		"200": {Description: "Image", Content: content},
		"304": {Description: "Not modified"},
		"400": {Description: "Invalid request"},
		"404": {Description: "Image not found"},
		"502": {Description: "Image origin unavailable"},
		"503": {Description: "Image storage unavailable"},
	}
}

func writeResult(ctx huma.Context, result Result) {
	ctx.SetHeader("Content-Type", result.ContentType)
	ctx.SetHeader("Cache-Control", result.CacheControl)
	ctx.SetHeader("ETag", result.ETag)
	ctx.SetHeader("Vary", "Accept")
	if !result.LastModified.IsZero() {
		ctx.SetHeader("Last-Modified", result.LastModified.UTC().Format(http.TimeFormat))
	}
	if etagMatches(ctx.Header("If-None-Match"), result.ETag) {
		ctx.SetStatus(http.StatusNotModified)
		return
	}
	ctx.SetStatus(http.StatusOK)
	_, _ = ctx.BodyWriter().Write(result.Body)
}

func writeError(ctx huma.Context, err error) {
	status, message := asStatus(err)
	if status >= 500 {
		log.Error().Err(err).Msg("image request failed")
	}
	body, _ := json.Marshal(map[string]string{"error": message})
	ctx.SetHeader("Content-Type", "application/json")
	ctx.SetHeader("Cache-Control", "no-store")
	ctx.SetStatus(status)
	_, _ = ctx.BodyWriter().Write(body)
}

func imageSizeParam(mapAsset bool) *huma.Param {
	values := []any{8, 16, 32, 64, 128, 256, 512, 1024}
	if mapAsset {
		values = []any{32, 64, 128}
	}
	return &huma.Param{
		Name:        "size",
		In:          "query",
		Description: "Maximum width and height in pixels. Images are never upscaled.",
		Schema: &huma.Schema{
			Type: huma.TypeInteger,
			Enum: values,
		},
	}
}

func imageFormatParam() *huma.Param {
	return &huma.Param{
		Name:        "format",
		In:          "query",
		Description: "Output format. Auto uses WebP when the request Accept header supports it.",
		Schema: &huma.Schema{
			Type:    huma.TypeString,
			Enum:    []any{"auto", "source", "webp"},
			Default: "auto",
		},
	}
}

func legacyImageTypeParam() *huma.Param {
	return &huma.Param{
		Name:        "imagetype",
		In:          "query",
		Description: "Deprecated alias for format=webp.",
		Deprecated:  true,
		Schema: &huma.Schema{
			Type: huma.TypeString,
			Enum: []any{"webp"},
		},
	}
}

func requestedFormat(ctx huma.Context, fallback string) (string, error) {
	switch strings.ToLower(ctx.Query("format")) {
	case "webp":
		return "webp", nil
	case "source":
		return fallback, nil
	case "", "auto":
		// Negotiate below.
	default:
		return "", statusError(
			http.StatusBadRequest,
			"Image format must be auto, source, or webp",
			nil,
		)
	}
	if raw := strings.ToLower(ctx.Query("imagetype")); raw != "" {
		if raw == "webp" {
			return "webp", nil
		}
		return "", statusError(
			http.StatusBadRequest,
			"Image type must be webp",
			nil,
		)
	}
	if acceptsWebP(ctx.Header("Accept")) {
		return "webp", nil
	}
	return fallback, nil
}

func acceptsWebP(header string) bool {
	for _, mediaRange := range strings.Split(strings.ToLower(header), ",") {
		parts := strings.Split(mediaRange, ";")
		if strings.TrimSpace(parts[0]) != "image/webp" {
			continue
		}
		quality := 1.0
		for _, parameter := range parts[1:] {
			name, value, found := strings.Cut(strings.TrimSpace(parameter), "=")
			if !found || strings.TrimSpace(name) != "q" {
				continue
			}
			parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
			if err != nil {
				quality = 0
			} else {
				quality = parsed
			}
		}
		if quality > 0 {
			return true
		}
	}
	return false
}

func parseSize(raw string) int {
	size, _ := strconv.Atoi(raw)
	return size
}

func etagMatches(header, etag string) bool {
	for _, candidate := range strings.Split(header, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" || candidate == etag ||
			strings.TrimPrefix(candidate, "W/") == etag {
			return true
		}
	}
	return false
}
