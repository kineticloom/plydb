The MCP server is intended to enable AI's to query the underlying connected data sources via SQL. So that AI's understand the semantic context of the underlying data, the MCP server should provide a `get_semantic_context` tool that returns data serialized as yaml following the OSI structure constructed via (semanticcontext.AutoScanProvider).Provide in the semanticcontext/scanner.go file.

Implementation notes

- The mcp server implementation is at cmd/mcp.go
- The cli in main.go has a scan-context command that can provide an example implementation of parts of this
- For now keep things simple by just scanning and loading the semantic context data into memory just once, during mcp server startup.
- Later on, we may introduce other means of gathering additional semantic context and layering that in, but that can be out of scope for now.
