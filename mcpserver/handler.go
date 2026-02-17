package mcpserver

import (
	"context"
	"fmt"
	"time"

	"github.com/kineticloom/plydb/queryengine"
	"github.com/kineticloom/plydb/semanticcontext"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.yaml.in/yaml/v4"
)

const queryTimeout = 55 * time.Second

// QueryInput is the input schema for the query tool.
type QueryInput struct {
	SQL string `json:"sql" jsonschema:"The SQL query to execute against the configured data sources"`
}

// NewServer creates an MCP server with the query and semantic context tools registered.
func NewServer(cfg *queryengine.Config, engine *queryengine.QueryEngine, model *semanticcontext.SemanticModelFile) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Title:      "PlyDB",
		Name:       "PlyDB",
		Version:    "0.1.0",
		WebsiteURL: "https://github.com/kineticloom/plydb",
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "query",
		Description: "Execute a SQL query against the configured data sources. Tables must be referenced as fully-qualified 3-part names: catalog.schema.table.",
	}, makeQueryHandler(cfg, engine))

	// Pre-serialize semantic context YAML once at init time.
	contextYAML, err := yaml.Marshal(model)
	if err != nil {
		contextYAML = []byte(fmt.Sprintf("error serializing semantic context: %v", err))
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_semantic_context",
		Description: "Returns the semantic context of the configured data sources, including dataset schemas, field types, descriptions, and dimensions, in Open Semantic Interchange (OSI) YAML format. For more on the OSI spec, see: https://github.com/open-semantic-interchange/OSI",
	}, makeSemanticContextHandler(string(contextYAML)))

	return server
}

// makeSemanticContextHandler returns a handler that returns the pre-serialized semantic context YAML.
func makeSemanticContextHandler(yamlStr string) mcp.ToolHandlerFor[any, any] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input any) (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: yamlStr},
			},
		}, nil, nil
	}
}

// makeQueryHandler returns a typed tool handler for the query tool.
func makeQueryHandler(cfg *queryengine.Config, engine *queryengine.QueryEngine) mcp.ToolHandlerFor[QueryInput, any] {
	policy := queryengine.ReadOnlyPolicy(cfg)
	validator := queryengine.NewPolicyValidator(policy)

	return func(ctx context.Context, req *mcp.CallToolRequest, input QueryInput) (*mcp.CallToolResult, any, error) {
		if input.SQL == "" {
			return nil, nil, fmt.Errorf("sql is required")
		}

		preprocessed, err := queryengine.PreprocessQuery(input.SQL, cfg, validator)
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
			StructuredContent: result,
		}, nil, nil
	}
}
