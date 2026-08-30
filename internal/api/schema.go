package api

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

type responseSchemaRoute struct {
	method      string
	path        string
	schemaPath  string
	staticParts int
}

type responseSchemaResolver struct {
	routes []responseSchemaRoute
}

// newResponseSchemaResolver gives the compatibility handlers the same useful
// response metadata as typed Huma handlers. They intentionally use Huma's
// low-level adapter so legacy validation errors keep their established
// {"error":"..."} shape, which means Huma's struct-only schema transformer
// cannot add this metadata for us.
func newResponseSchemaResolver(document *huma.OpenAPI) *responseSchemaResolver {
	resolver := &responseSchemaResolver{}
	if document == nil {
		return resolver
	}

	for path, item := range document.Paths {
		for _, candidate := range pathItemOperations(item) {
			op := candidate.operation
			if op == nil || op.OperationID == "" {
				continue
			}
			route := responseSchemaRoute{
				method: candidate.method, path: path,
				staticParts: countStaticPathParts(path),
			}
			if !hasJSONResponse(op) {
				resolver.routes = append(resolver.routes, route)
				continue
			}
			name := op.OperationID + "-response"
			schemaPath := "/schemas/" + name + ".json"
			schemaProperty := &huma.Schema{
				Type:        huma.TypeString,
				Format:      "uri",
				Description: "A URL to the JSON Schema for this response.",
				ReadOnly:    true,
			}
			bodySchema := publicOperationResponseSchema(op.OperationID)
			if bodySchema == nil {
				// Keep the route operable, but leave a conspicuously incomplete
				// schema for the catalogue test to reject. A newly added
				// endpoint must make an explicit response-shape decision.
				bodySchema = responseSchema(map[string]*huma.Schema{})
			}
			bodySchema.Properties["$schema"] = schemaProperty
			bodySchema.Required = append(bodySchema.Required, "$schema")
			document.Components.Schemas.Map()[name] = bodySchema
			for _, response := range op.Responses {
				if response == nil {
					continue
				}
				if media := response.Content["application/json"]; media != nil {
					media.Schema = &huma.Schema{
						Ref: "#/components/schemas/" + name,
					}
				}
			}
			route.schemaPath = schemaPath
			resolver.routes = append(resolver.routes, route)
		}
	}

	// A literal route such as /history/latest must win over /history/{date}.
	sort.Slice(resolver.routes, func(i, j int) bool {
		if resolver.routes[i].staticParts != resolver.routes[j].staticParts {
			return resolver.routes[i].staticParts > resolver.routes[j].staticParts
		}
		return resolver.routes[i].path < resolver.routes[j].path
	})
	return resolver
}

type operationCandidate struct {
	method    string
	operation *huma.Operation
}

func pathItemOperations(item *huma.PathItem) []operationCandidate {
	if item == nil {
		return nil
	}
	return []operationCandidate{
		{http.MethodGet, item.Get},
		{http.MethodPost, item.Post},
		{http.MethodPut, item.Put},
		{http.MethodPatch, item.Patch},
		{http.MethodDelete, item.Delete},
	}
}

func hasJSONResponse(op *huma.Operation) bool {
	for _, response := range op.Responses {
		if response != nil && response.Content["application/json"] != nil {
			return true
		}
	}
	return false
}

func countStaticPathParts(path string) int {
	count := 0
	for part := range strings.SplitSeq(strings.Trim(path, "/"), "/") {
		if !isPathParameter(part) {
			count++
		}
	}
	return count
}

func isPathParameter(part string) bool {
	return strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}")
}

func (r *responseSchemaResolver) lookup(method, path string) (string, bool) {
	if r == nil {
		return "", false
	}
	for _, route := range r.routes {
		if route.method == method && routePathMatches(route.path, path) {
			return route.schemaPath, true
		}
	}
	return "", false
}

func routePathMatches(pattern, path string) bool {
	if pattern == path {
		return true
	}
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	if len(patternParts) != len(pathParts) {
		return false
	}
	for i, part := range patternParts {
		if isPathParameter(part) {
			if pathParts[i] == "" {
				return false
			}
			continue
		}
		if part != pathParts[i] {
			return false
		}
	}
	return true
}

func addResponseSchema(
	r *http.Request,
	headers http.Header,
	body []byte,
	schemaPath string,
) []byte {
	if schemaPath == "" || !strings.HasPrefix(
		headers.Get("Content-Type"), "application/json",
	) {
		return body
	}

	start := 0
	for start < len(body) && (body[start] == ' ' || body[start] == '\n' ||
		body[start] == '\r' || body[start] == '\t') {
		start++
	}
	if start >= len(body) || body[start] != '{' {
		return body
	}

	externalPath := schemaPath
	if prefix, _ := r.Context().Value(sameOriginPrefixContextKey{}).(string); prefix != "" {
		externalPath = strings.TrimSuffix(prefix, "/") + schemaPath
	}
	schemaURL := responseSchemaURL(r, externalPath)
	encodedURL, err := json.Marshal(schemaURL)
	if err != nil {
		return body
	}
	headers.Set("Link", "<"+externalPath+">; rel=\"describedBy\"")

	result := make([]byte, 0, len(body)+len(encodedURL)+12)
	result = append(result, body[:start+1]...)
	result = append(result, `"$schema":`...)
	result = append(result, encodedURL...)
	if nextJSONToken(body[start+1:]) != '}' {
		result = append(result, ',')
	}
	result = append(result, body[start+1:]...)
	return result
}

func nextJSONToken(body []byte) byte {
	for _, value := range body {
		switch value {
		case ' ', '\n', '\r', '\t':
			continue
		default:
			return value
		}
	}
	return 0
}

func responseSchemaURL(r *http.Request, path string) string {
	scheme := firstForwarded(r.Header.Get("X-Forwarded-Proto"))
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := firstForwarded(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = r.Host
	}
	return scheme + "://" + host + path
}
