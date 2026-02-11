package mcpserver

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ypt/experiment-nexus/queryengine"
)

const queryTimeout = 55 * time.Second

// QueryInput is the input schema for the query tool.
type QueryInput struct {
	SQL string `json:"sql" jsonschema:"The SQL query to execute against the configured data sources"`
}

// NewServer creates an MCP server with the query tool registered.
func NewServer(cfg *queryengine.Config, engine *queryengine.QueryEngine) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "nexus",
		Version: "0.1.0",
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "query",
		Description: "Execute a SQL query against the configured data sources. Tables must be referenced as fully-qualified 3-part names: catalog.schema.table.",
	}, makeQueryHandler(cfg, engine))

	return server
}

// makeQueryHandler returns a typed tool handler for the query tool.
func makeQueryHandler(cfg *queryengine.Config, engine *queryengine.QueryEngine) mcp.ToolHandlerFor[QueryInput, any] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input QueryInput) (*mcp.CallToolResult, any, error) {
		if input.SQL == "" {
			return nil, nil, fmt.Errorf("sql is required")
		}

		preprocessed, err := queryengine.PreprocessQuery(input.SQL, cfg)
		if err != nil {
			return nil, nil, fmt.Errorf("preprocessing query: %w", err)
		}

		queryCtx, cancel := context.WithTimeout(ctx, queryTimeout)
		defer cancel()

		rows, err := engine.Query(queryCtx, preprocessed)
		if err != nil {
			return nil, nil, fmt.Errorf("executing query: %w", err)
		}
		defer rows.Close()

		result, err := buildQueryResult(rows)
		if err != nil {
			return nil, nil, fmt.Errorf("building result: %w", err)
		}

		text, err := marshalResult(result)
		if err != nil {
			return nil, nil, fmt.Errorf("marshaling result: %w", err)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: text},
			},
		}, nil, nil
	}
}
