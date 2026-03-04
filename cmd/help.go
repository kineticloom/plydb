// Copyright 2026 Paul Tzen
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"io"
)

const usage = `PlyDB: Query across databases and flat files (Postgres, MySQL, CSV, JSON, Excel, SQLite, DuckDB, Parquet, Google Sheets) using standard SQL - locally or in the cloud.

Usage: plydb <command> [arguments] [flags]

Commands:
  - auth               Authenticate an interactive data source (e.g. gsheet OAuth)
  - query              Execute a SQL query
  - semantic-context   Scan all data sources and output semantic context as YAML
  - mcp                Start a Model Context Protocol (MCP) server exposing query and get_semantic_context tools for AI agents
  - version            Print version information
  - license            Print license information
  - help               Print commands and documentation website urls

Run "plydb <command> -h" for command-specific usage.

Resources:
  - Full documentation: https://www.plydb.com/docs/
  - Configuring Data Sources: https://www.plydb.com/docs/data-sources/
  - Semantic Context: https://www.plydb.com/docs/semantic-context/
  - AI Agent Integration: https://www.plydb.com/docs/agent-integration/
  - FAQ: https://www.plydb.com/docs/faq/
`

func RunHelp(w io.Writer) {
	fmt.Fprint(w, usage)
}
