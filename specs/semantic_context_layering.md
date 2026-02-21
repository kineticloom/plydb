# Semantic context layering

Semantic context is intended to provide AI's with context about the semantics of
underlying connected data sources made available by the service so the AI has
context to write SQL queries.

The auto scanning semantic context provider as specified in
specs/auto_scanning_semantic_context_provider.md provides the foundational layer
of data for this.

On top of the automatically scanned semantic context, we should have a means of
layering in additional context on top of what was automatically scanned. This is
to provide users a means to further customize the semantic context that is made
available when calling `semantic-context` in the CLI and `get_semantic_context`
in the MCP server.

Semantic context layering should:

1. Use the auto scanned semantic data as the foundational layer
2. The CLI semantic-context command (see: cmd/semanticcontext.go), via a new
   flag should allow a user to optionally specify one or more files of
   additional semantic context to overlay on top of the auto scanned semantic
   context. These files should use the OSI spec
   (https://github.com/open-semantic-interchange/OSI) format, which the auto
   scanning semantic context provider outputs as well.
3. Additional semantic context being layered on top of the auto scanned context
   should not add catalogs, tables, or keys that do not exist in the auto
   scanned semantic context.

Implementation notes:

- Semantic context is fetched via the CLI (see cmd/semanticcontext.go) and MCP
  server (see cmd/mcp.go). These currently just use auto scanned semantic
  context and should be updated to use the new layering logic.
- Consider consolidating semantic context layering logic. See the
  semanticcontext package.
