package mcpserver

import (
	"context"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Catalog asks the same registered MCP server that serves /api/mcp for its
// tools, so public documentation follows tool registrations automatically.
// It uses an in-memory transport; no HTTP call or backend connection is made.
func Catalog(ctx context.Context) ([]*sdkmcp.Tool, error) {
	server, err := NewServer(Dependencies{}, "catalog", nil)
	if err != nil {
		return nil, err
	}
	serverTransport, clientTransport := sdkmcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = serverSession.Close() }()
	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "shrike-catalog", Version: "1"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = clientSession.Close() }()
	result, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		return nil, err
	}
	return result.Tools, nil
}
