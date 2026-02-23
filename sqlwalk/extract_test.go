// Copyright 2026 Paul Tzen
// SPDX-License-Identifier: Apache-2.0

package sqlwalk

import (
	"slices"
	"testing"

	pg_query "github.com/pganalyze/pg_query_go/v6"
)

func parse(t *testing.T, sql string) *pg_query.ParseResult {
	t.Helper()
	result, err := pg_query.Parse(sql)
	if err != nil {
		t.Fatalf("Parse(%q): %v", sql, err)
	}
	return result
}

func TestExtract(t *testing.T) {
	tests := []struct {
		name   string
		sql    string
		tables []TableRef
	}{
		{
			name: "simple SELECT",
			sql:  "SELECT a, b FROM t1",
			tables: []TableRef{
				{Name: "t1"},
			},
		},
		{
			name: "star select",
			sql:  "SELECT * FROM t1",
			tables: []TableRef{
				{Name: "t1"},
			},
		},
		{
			name: "qualified star",
			sql:  "SELECT t.* FROM t1 AS t",
			tables: []TableRef{
				{Name: "t1", Alias: "t"},
			},
		},
		{
			name: "three-part table name",
			sql:  "SELECT x FROM db.schema.tbl",
			tables: []TableRef{
				{Catalog: "db", Schema: "schema", Name: "tbl"},
			},
		},
		{
			name: "JOIN with ON clause",
			sql:  "SELECT a.id, b.name FROM alpha a JOIN beta b ON a.id = b.alpha_id",
			tables: []TableRef{
				{Name: "alpha", Alias: "a"},
				{Name: "beta", Alias: "b"},
			},
		},
		{
			name: "CTE",
			sql:  "WITH cte AS (SELECT id FROM src) SELECT id FROM cte",
			tables: []TableRef{
				{Name: "src"},
			},
		},
		{
			name: "subquery in FROM",
			sql:  "SELECT x FROM (SELECT y FROM inner_t) sub",
			tables: []TableRef{
				{Name: "inner_t"},
			},
		},
		{
			name: "subquery in WHERE (SubLink)",
			sql:  "SELECT a FROM t1 WHERE a IN (SELECT b FROM t2)",
			tables: []TableRef{
				{Name: "t1"},
				{Name: "t2"},
			},
		},
		{
			name: "INSERT with SELECT source",
			sql:  "INSERT INTO dst (col1) SELECT col1 FROM src",
			tables: []TableRef{
				{Name: "dst"},
				{Name: "src"},
			},
		},
		{
			name: "UPDATE with FROM",
			sql:  "UPDATE t1 SET x = t2.y FROM t2 WHERE t1.id = t2.id",
			tables: []TableRef{
				{Name: "t1"},
				{Name: "t2"},
			},
		},
		{
			name: "DELETE with USING",
			sql:  "DELETE FROM t1 USING t2 WHERE t1.id = t2.id",
			tables: []TableRef{
				{Name: "t1"},
				{Name: "t2"},
			},
		},
		{
			name: "UNION",
			sql:  "SELECT a FROM t1 UNION SELECT b FROM t2",
			tables: []TableRef{
				{Name: "t1"},
				{Name: "t2"},
			},
		},
		{
			name: "MERGE",
			sql: `MERGE INTO target t
				  USING source s ON t.id = s.id
				  WHEN MATCHED THEN UPDATE SET val = s.val
				  WHEN NOT MATCHED THEN INSERT (id, val) VALUES (s.id, s.val)`,
			tables: []TableRef{
				{Name: "target", Alias: "t"},
				{Name: "source", Alias: "s"},
			},
		},
		{
			name: "multiple statements",
			sql:  "SELECT a FROM t1; SELECT b FROM t2",
			tables: []TableRef{
				{Name: "t1"},
				{Name: "t2"},
			},
		},
		{
			name: "RETURNING clause",
			sql:  "DELETE FROM t1 WHERE id = 1 RETURNING id, name",
			tables: []TableRef{
				{Name: "t1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed := parse(t, tt.sql)
			result := Extract(parsed)

			got := result.Tables
			want := tt.tables

			if len(got) != len(want) {
				t.Fatalf("tables mismatch: got %v, want %v", got, want)
			}

			// Order-independent comparison
			remaining := slices.Clone(want)
			for _, g := range got {
				idx := slices.Index(remaining, g)
				if idx == -1 {
					t.Fatalf("tables mismatch: unexpected %v in got %v, want %v", g, got, want)
				}
				remaining = slices.Delete(remaining, idx, idx+1)
			}
			if len(remaining) > 0 {
				t.Fatalf("tables mismatch: missing %v from got %v, want %v", remaining, got, want)
			}
		})
	}
}
