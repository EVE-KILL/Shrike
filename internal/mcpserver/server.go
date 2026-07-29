package mcpserver

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/jsonschema-go/jsonschema"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

type ToolHandler[In, Out any] func(context.Context, In) (Out, error)

type Registry struct {
	deps   Dependencies
	mcp    *sdkmcp.Server
	huma   huma.API
	prefix string
}

type ToolDefinition struct {
	Name        string
	Title       string
	Description string
}

type humaToolInput[T any] struct {
	Body T
}

type humaToolOutput[T any] struct {
	Body T
}

func NewServer(
	deps Dependencies,
	version string,
	api huma.API,
) (*sdkmcp.Server, error) {
	if strings.TrimSpace(version) == "" {
		version = "dev"
	}
	server := sdkmcp.NewServer(&sdkmcp.Implementation{
		Name:    "evekill-mcp",
		Title:   "EVE-KILL",
		Version: version,
	}, &sdkmcp.ServerOptions{
		Instructions: "Use these read-only tools to search and analyze EVE Online combat data.",
	})
	registry := &Registry{
		deps:   deps,
		mcp:    server,
		huma:   api,
		prefix: "/mcp/tools/",
	}
	if err := registerTools(registry); err != nil {
		return nil, err
	}
	return server, nil
}

func addTool[In, Out any](
	registry *Registry,
	definition ToolDefinition,
	handler ToolHandler[In, Out],
) error {
	inputSchema, err := jsonschema.For[In](mcpSchemaOptions())
	if err != nil {
		return fmt.Errorf("%s input schema: %w", definition.Name, err)
	}
	outputSchema, err := jsonschema.For[Out](mcpSchemaOptions())
	if err != nil {
		return fmt.Errorf("%s output schema: %w", definition.Name, err)
	}

	tool := &sdkmcp.Tool{
		Name:         definition.Name,
		Title:        definition.Title,
		Description:  definition.Description,
		InputSchema:  inputSchema,
		OutputSchema: outputSchema,
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:   true,
			IdempotentHint: true,
		},
	}
	sdkmcp.AddTool(registry.mcp, tool, func(
		ctx context.Context,
		_ *sdkmcp.CallToolRequest,
		input In,
	) (*sdkmcp.CallToolResult, Out, error) {
		output, err := handler(ctx, input)
		return nil, output, err
	})

	if registry.huma != nil {
		path := registry.prefix + definition.Name
		huma.Register(registry.huma, huma.Operation{
			OperationID: "mcp-" + strings.ReplaceAll(definition.Name, "_", "-"),
			Method:      http.MethodPost,
			Path:        path,
			Summary:     definition.Title,
			Description: definition.Description,
			Tags:        []string{"mcp"},
		}, func(
			ctx context.Context,
			input *humaToolInput[In],
		) (*humaToolOutput[Out], error) {
			output, err := handler(ctx, input.Body)
			if err != nil {
				return nil, huma.Error400BadRequest(err.Error())
			}
			return &humaToolOutput[Out]{Body: output}, nil
		})
	}
	return nil
}

func Handler(
	deps Dependencies,
	version string,
	logger *slog.Logger,
) (http.Handler, error) {
	if logger == nil {
		logger = slog.Default()
	}
	server, err := NewServer(deps, version, nil)
	if err != nil {
		return nil, err
	}
	streamable := sdkmcp.NewStreamableHTTPHandler(
		func(*http.Request) *sdkmcp.Server { return server },
		&sdkmcp.StreamableHTTPOptions{
			Stateless:                    true,
			JSONResponse:                 true,
			Logger:                       logger,
			MaxRequestBodyBytes:          1 << 20,
			PropagateRequestCancellation: true,
		},
	)
	return withCORS(streamable), nil
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header.Set("Access-Control-Allow-Origin", "*")
		header.Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		header.Set(
			"Access-Control-Allow-Headers",
			"Accept, Authorization, Content-Type, Last-Event-ID, "+
				"MCP-Protocol-Version, MCP-Session-Id, Mcp-Method, Mcp-Name",
		)
		header.Set(
			"Access-Control-Expose-Headers",
			"MCP-Protocol-Version, MCP-Session-Id",
		)
		if r.Method == http.MethodOptions {
			header.Set("Access-Control-Max-Age", "86400")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		// Browser documentation and older clients predate the Streamable HTTP
		// Accept requirement. Supply both protocol media types when omitted.
		if strings.TrimSpace(r.Header.Get("Accept")) == "" {
			r.Header.Set("Accept", "application/json, text/event-stream")
		}
		next.ServeHTTP(w, r)
	})
}
