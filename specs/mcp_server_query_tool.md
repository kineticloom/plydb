The MCP server is intended to enable AI's to query the underlying connected data sources via SQL. The MCP server should provide a `query` tool that:

1. The `query` tool takes a SQL string as input
2. Pre-processes the SQL query as per specs/query_pre_processing.md
3. Query the underlying data sources, and
4. Return a response.

Successful responses should look like

```
{
  "success": true,
  "columns": ["customer_name", "order_count"],
  "columnTypes": ["VARCHAR", "BIGINT"],
  "rows": [
    ["Bob Smith", 20],
    ["Jane Kim", 56],
    ["Tom Anderson", 83]
  ],
  "rowCount": 3
}
```

Limits:

- Result limit: Maximum 2,048 rows and 50,000 characters. Results exceeding these limits should be truncated with a truncation message.
- Query timeout: 55 seconds, to stay within common client timeouts. Queries exceeding this limit should be cancelled and the response should include an error message.

## Implementation notes

- Connection configuration is spec'd in specs/database_connections_schema.md, and there is already parsing and database connection logic in the queryengine package
- A query engine has already been implemented in the queryengine package
- Queries should go through pre-processing as specified in specs/query_pre_processing.md. The foundational functionality is already implemented in the queryengine package and used in cmd/query.go
- The cli should be updated with a new command to start the mcp server.
- We require support for both STDIO and Streamable HTTP transport for the MCP server. Let this be configurable at the cli's entrypoint. e.g. `plydb mcp --transport stdio --config /path/to/config.json`

## Out of scope

- This document only covers the `query` MCP tool. Other tools will be defined elsewhere.