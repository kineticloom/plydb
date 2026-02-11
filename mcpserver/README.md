# MCP Server

An MCP (Model Context Protocol) server that exposes a `query` tool, allowing AI assistants to execute SQL queries against configured data sources.

## Usage

```bash
nexus mcp --config <path> [--transport stdio|http] [--addr host:port]
```

### Flags

- `--config` (required): Path to the connection config JSON file
- `--transport`: Transport type — `stdio` (default) or `http`
- `--addr`: Listen address for HTTP transport (default `localhost:8080`)

## STDIO Transport

The default transport. The server communicates via JSON-RPC over stdin/stdout.

```bash
nexus mcp --config config.json
```

### Manual Verification

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"capabilities":{},"clientInfo":{"name":"test"},"protocolVersion":"2025-03-26"}}' \
  | ./experiment-nexus mcp --transport stdio --config config.json
```

## HTTP Transport

Starts a Streamable HTTP server.

```bash
nexus mcp --config config.json --transport http --addr localhost:9090
```

### Manual Verification

Initialize:

```bash
curl -X POST http://localhost:9090 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"capabilities":{},"clientInfo":{"name":"test"},"protocolVersion":"2025-03-26"}}'
```

Call the query tool (use the session ID from the initialize response):

```bash
curl -X POST http://localhost:9090 \
  -H "Content-Type: application/json" \
  -H "Mcp-Session-Id: <session-id>" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"query","arguments":{"sql":"SELECT 1 AS n"}}}'
```

## Tool: `query`

Executes a SQL query against the configured data sources.

### Input

| Field | Type   | Description                        |
|-------|--------|------------------------------------|
| `sql` | string | The SQL query to execute           |

Tables must be referenced as fully-qualified 3-part names: `catalog.schema.table`.

### Output

JSON object with fields:

| Field          | Type     | Description                              |
|----------------|----------|------------------------------------------|
| `success`      | boolean  | Whether the query succeeded              |
| `columns`      | string[] | Column names                             |
| `column_types` | string[] | Column type names                        |
| `rows`         | any[][]  | Row data                                 |
| `row_count`    | integer  | Number of rows returned                  |
| `truncated`    | boolean  | Whether results were truncated           |
| `message`      | string   | Human-readable message (errors, limits)  |

Results are limited to 2,048 rows and 50,000 characters of JSON output.
