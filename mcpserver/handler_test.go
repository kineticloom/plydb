// Copyright 2026 Paul Tzen
// SPDX-License-Identifier: Apache-2.0

package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/kineticloom/plydb/queryengine"
	"github.com/kineticloom/plydb/queryresult"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// startTestServer creates an MCP server, connects a client, and returns the client session.
func startTestServer(t *testing.T, cfg *queryengine.Config, engine *queryengine.QueryEngine) *mcp.ClientSession {
	t.Helper()
	server := NewServer(cfg, engine, nil)
	ctx := context.Background()

	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	go server.Run(ctx, serverTransport)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.1.0"}, nil)
	cs, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect error: %v", err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs
}

func newTestEngine(t *testing.T) (*queryengine.Config, *queryengine.QueryEngine) {
	t.Helper()
	cfg := &queryengine.Config{
		Credentials: map[string]queryengine.Credential{},
		Databases:   map[string]queryengine.DatabaseConfig{},
	}
	engine, err := queryengine.New(cfg)
	if err != nil {
		t.Fatalf("creating engine: %v", err)
	}
	t.Cleanup(func() { engine.Close() })
	return cfg, engine
}

func TestBuildQueryResult(t *testing.T) {
	_, engine := newTestEngine(t)

	rows, err := engine.Query(context.Background(), "SELECT 1 AS id, 'hello' AS name")
	if err != nil {
		t.Fatalf("query error: %v", err)
	}
	defer rows.Close()

	result, err := queryresult.BuildQueryResult(rows)
	if err != nil {
		t.Fatalf("buildQueryResult error: %v", err)
	}

	if !result.Success {
		t.Fatal("expected success")
	}
	if len(result.Columns) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(result.Columns))
	}
	if result.Columns[0] != "id" || result.Columns[1] != "name" {
		t.Fatalf("unexpected columns: %v", result.Columns)
	}
	if result.RowCount != 1 {
		t.Fatalf("expected 1 row, got %d", result.RowCount)
	}
	if result.Truncated {
		t.Fatal("expected not truncated")
	}
}

func TestBuildQueryResultMultipleRows(t *testing.T) {
	_, engine := newTestEngine(t)

	rows, err := engine.Query(context.Background(), "SELECT * FROM generate_series(1, 10) AS t(n)")
	if err != nil {
		t.Fatalf("query error: %v", err)
	}
	defer rows.Close()

	result, err := queryresult.BuildQueryResult(rows)
	if err != nil {
		t.Fatalf("buildQueryResult error: %v", err)
	}

	if result.RowCount != 10 {
		t.Fatalf("expected 10 rows, got %d", result.RowCount)
	}
	if result.Truncated {
		t.Fatal("expected not truncated")
	}
}

func TestBuildQueryResultRowLimitTruncation(t *testing.T) {
	_, engine := newTestEngine(t)

	query := fmt.Sprintf("SELECT * FROM generate_series(1, %d) AS t(n)", queryresult.MaxRows+100)
	rows, err := engine.Query(context.Background(), query)
	if err != nil {
		t.Fatalf("query error: %v", err)
	}
	defer rows.Close()

	result, err := queryresult.BuildQueryResult(rows)
	if err != nil {
		t.Fatalf("buildQueryResult error: %v", err)
	}

	if result.RowCount != queryresult.MaxRows {
		t.Fatalf("expected %d rows, got %d", queryresult.MaxRows, result.RowCount)
	}
	if !result.Truncated {
		t.Fatal("expected truncated")
	}
}

func TestBuildQueryResultCharLimitTruncation(t *testing.T) {
	_, engine := newTestEngine(t)

	// Generate rows with large string values to exceed character limit.
	query := "SELECT generate_series AS id, repeat('x', 200) AS big_value FROM generate_series(1, 2000)"
	rows, err := engine.Query(context.Background(), query)
	if err != nil {
		t.Fatalf("query error: %v", err)
	}
	defer rows.Close()

	result, err := queryresult.BuildQueryResult(rows)
	if err != nil {
		t.Fatalf("buildQueryResult error: %v", err)
	}

	if !result.Truncated {
		t.Fatal("expected truncated due to character limit")
	}

	// Verify the JSON output fits within the limit.
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if len(data) > queryresult.MaxChars {
		t.Fatalf("JSON output %d chars exceeds limit %d", len(data), queryresult.MaxChars)
	}
	if result.RowCount < 1 {
		t.Fatal("expected at least 1 row after char truncation")
	}
}

func TestBuildQueryResultColumnTypes(t *testing.T) {
	_, engine := newTestEngine(t)

	rows, err := engine.Query(context.Background(), "SELECT 1::INTEGER AS int_col, 'text'::VARCHAR AS str_col, 3.14::DOUBLE AS float_col")
	if err != nil {
		t.Fatalf("query error: %v", err)
	}
	defer rows.Close()

	result, err := queryresult.BuildQueryResult(rows)
	if err != nil {
		t.Fatalf("buildQueryResult error: %v", err)
	}

	if len(result.ColumnTypes) != 3 {
		t.Fatalf("expected 3 column types, got %d", len(result.ColumnTypes))
	}
	// DuckDB should return type names.
	for i, ct := range result.ColumnTypes {
		if ct == "" {
			t.Errorf("column type %d is empty", i)
		}
	}
}

func TestQueryHandlerSuccess(t *testing.T) {
	cfg, engine := newTestEngine(t)
	cs := startTestServer(t, cfg, engine)

	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "query",
		Arguments: map[string]any{"sql": "SELECT 42 AS answer"},
	})
	if err != nil {
		t.Fatalf("CallTool error: %v", err)
	}

	if result.IsError {
		t.Fatal("expected no error in result")
	}
	if len(result.Content) == 0 {
		t.Fatal("expected content in result")
	}

	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}

	var qr queryresult.QueryResult
	if err := json.Unmarshal([]byte(textContent.Text), &qr); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if !qr.Success {
		t.Fatal("expected success in query result")
	}
	if qr.RowCount != 1 {
		t.Fatalf("expected 1 row, got %d", qr.RowCount)
	}
}

func TestQueryHandlerEmptySQL(t *testing.T) {
	cfg, engine := newTestEngine(t)
	cs := startTestServer(t, cfg, engine)

	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "query",
		Arguments: map[string]any{"sql": ""},
	})
	if err != nil {
		t.Fatalf("CallTool error: %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error for empty SQL")
	}
}

func TestQueryHandlerBadSQL(t *testing.T) {
	cfg, engine := newTestEngine(t)
	cs := startTestServer(t, cfg, engine)

	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "query",
		Arguments: map[string]any{"sql": "SELECT * FROM nonexistent_table"},
	})
	if err != nil {
		t.Fatalf("CallTool error: %v", err)
	}

	if !result.IsError {
		t.Fatal("expected error for bad SQL")
	}
	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	if !strings.Contains(textContent.Text, "nonexistent_table") {
		t.Fatalf("expected error to mention table name, got: %s", textContent.Text)
	}
}

func TestQueryHandlerPreprocessingError(t *testing.T) {
	cfg, engine := newTestEngine(t)
	cs := startTestServer(t, cfg, engine)

	// An unqualified table reference should fail preprocessing.
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "query",
		Arguments: map[string]any{"sql": "SELECT * FROM some_table"},
	})
	if err != nil {
		t.Fatalf("CallTool error: %v", err)
	}

	if !result.IsError {
		t.Fatal("expected preprocessing error")
	}
	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	if !strings.Contains(textContent.Text, "not fully qualified") {
		t.Fatalf("expected preprocessing error message, got: %s", textContent.Text)
	}
}
