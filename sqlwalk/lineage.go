// Copyright 2026 Paul Tzen
// SPDX-License-Identifier: Apache-2.0

package sqlwalk

import pg_query "github.com/pganalyze/pg_query_go/v6"

// TableRef represents a table referenced in a SQL statement.
type TableRef struct {
	Catalog string // e.g. "db" from db.schema.table
	Schema  string // e.g. "schema"
	Name    string // e.g. "table"
	Alias   string // table alias if present
}

// Result holds all table references found in a parse result.
type Result struct {
	Tables []TableRef
}

// Extract walks the pg_query AST and returns every real table reference.
// CTE names are excluded. Every occurrence is reported (no deduplication).
func Extract(parsed *pg_query.ParseResult) Result {
	c := &collector{cteNames: make(map[string]struct{})}
	for _, stmt := range parsed.GetStmts() {
		if stmt.GetStmt() != nil {
			walkNode(stmt.GetStmt(), c)
		}
	}

	var r Result
	for _, t := range c.tables {
		// A CTE reference is an unqualified name matching a CTE definition.
		if t.Catalog == "" && t.Schema == "" {
			if _, isCTE := c.cteNames[t.Name]; isCTE {
				continue
			}
		}
		r.Tables = append(r.Tables, t)
	}
	return r
}
