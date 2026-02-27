// Copyright 2026 Paul Tzen
// SPDX-License-Identifier: Apache-2.0

//go:build integration

package queryengine

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// ---------------------------------------------------------------------------
// DuckDB helpers
// ---------------------------------------------------------------------------

// createDuckDBTestDB creates a temporary DuckDB file with departments and
// employees tables matching the existing test patterns.
func createDuckDBTestDB(t *testing.T) string {
	t.Helper()

	f, err := os.CreateTemp("", "plydb_test_*.duckdb")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	path := f.Name()
	f.Close()
	os.Remove(path) // Remove so DuckDB can create it fresh.
	t.Cleanup(func() { os.Remove(path) })

	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("opening duckdb: %v", err)
	}
	defer db.Close()

	stmts := []string{
		`CREATE TABLE departments (
			id   INTEGER PRIMARY KEY,
			name TEXT NOT NULL
		)`,
		`INSERT INTO departments VALUES (1, 'Engineering'), (2, 'Sales'), (3, 'Marketing')`,
		`CREATE TABLE employees (
			id            INTEGER PRIMARY KEY,
			name          TEXT NOT NULL,
			department_id INTEGER NOT NULL,
			salary        DOUBLE NOT NULL
		)`,
		`INSERT INTO employees VALUES
			(1, 'Alice',   1, 95000.00),
			(2, 'Bob',     1, 88000.00),
			(3, 'Charlie', 2, 72000.00),
			(4, 'Diana',   3, 68000.00),
			(5, 'Eve',     2, 75000.00)`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("creating duckdb test db: %v\nstatement: %s", err, stmt)
		}
	}

	return path
}

func duckdbConfig(t *testing.T, key, path string) *Config {
	t.Helper()
	return &Config{
		Credentials: map[string]Credential{},
		Databases: map[string]DatabaseConfig{
			key: {
				Metadata: Metadata{Name: "Integration DuckDB", Description: "test"},
				Type:     DuckDB,
				Path:     path,
			},
		},
	}
}

// ---------------------------------------------------------------------------
// DuckDB integration tests
// ---------------------------------------------------------------------------

func TestIntegrationDuckDB(t *testing.T) {
	path := createDuckDBTestDB(t)

	t.Run("BasicSelect", func(t *testing.T) {
		cfg := duckdbConfig(t, "dk", path)
		engine, err := New(cfg)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		defer engine.Close()

		rows, err := engine.Query(context.Background(),
			`SELECT id, name, department_id, salary FROM "dk".main.employees ORDER BY id`)
		if err != nil {
			t.Fatalf("query error: %v", err)
		}
		defer rows.Close()

		type employee struct {
			id     int
			name   string
			deptID int
			salary float64
		}

		want := []employee{
			{1, "Alice", 1, 95000.00},
			{2, "Bob", 1, 88000.00},
			{3, "Charlie", 2, 72000.00},
			{4, "Diana", 3, 68000.00},
			{5, "Eve", 2, 75000.00},
		}

		var got []employee
		for rows.Next() {
			var e employee
			if err := rows.Scan(&e.id, &e.name, &e.deptID, &e.salary); err != nil {
				t.Fatalf("scan error: %v", err)
			}
			got = append(got, e)
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("rows iteration error: %v", err)
		}

		if len(got) != len(want) {
			t.Fatalf("got %d rows, want %d", len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("row %d: got %+v, want %+v", i, got[i], want[i])
			}
		}
	})

	t.Run("Filter", func(t *testing.T) {
		cfg := duckdbConfig(t, "dk", path)
		engine, err := New(cfg)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		defer engine.Close()

		rows, err := engine.Query(context.Background(),
			`SELECT name, salary FROM "dk".main.employees WHERE salary > 80000 ORDER BY salary DESC`)
		if err != nil {
			t.Fatalf("query error: %v", err)
		}
		defer rows.Close()

		type result struct {
			name   string
			salary float64
		}

		want := []result{
			{"Alice", 95000.00},
			{"Bob", 88000.00},
		}

		var got []result
		for rows.Next() {
			var r result
			if err := rows.Scan(&r.name, &r.salary); err != nil {
				t.Fatalf("scan error: %v", err)
			}
			got = append(got, r)
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("rows iteration error: %v", err)
		}

		if len(got) != len(want) {
			t.Fatalf("got %d rows, want %d", len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("row %d: got %+v, want %+v", i, got[i], want[i])
			}
		}
	})

	t.Run("JoinWithinDB", func(t *testing.T) {
		cfg := duckdbConfig(t, "dk", path)
		engine, err := New(cfg)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		defer engine.Close()

		rows, err := engine.Query(context.Background(),
			`SELECT e.name, d.name AS dept
			 FROM "dk".main.employees e
			 JOIN "dk".main.departments d ON e.department_id = d.id
			 ORDER BY e.id`)
		if err != nil {
			t.Fatalf("query error: %v", err)
		}
		defer rows.Close()

		type result struct {
			name string
			dept string
		}

		want := []result{
			{"Alice", "Engineering"},
			{"Bob", "Engineering"},
			{"Charlie", "Sales"},
			{"Diana", "Marketing"},
			{"Eve", "Sales"},
		}

		var got []result
		for rows.Next() {
			var r result
			if err := rows.Scan(&r.name, &r.dept); err != nil {
				t.Fatalf("scan error: %v", err)
			}
			got = append(got, r)
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("rows iteration error: %v", err)
		}

		if len(got) != len(want) {
			t.Fatalf("got %d rows, want %d", len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("row %d: got %+v, want %+v", i, got[i], want[i])
			}
		}
	})

	t.Run("PreprocessThenQuery", func(t *testing.T) {
		cfg := duckdbConfig(t, "dk", path)
		engine, err := New(cfg)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		defer engine.Close()

		original := `SELECT name, salary FROM dk.main.employees WHERE salary >= 75000 ORDER BY name`
		rewritten, err := PreprocessQuery(original, cfg)
		if err != nil {
			t.Fatalf("PreprocessQuery: %v", err)
		}

		// DuckDB is passthrough — query should be preserved.
		if !strings.Contains(rewritten, "dk.main.employees") {
			t.Fatalf("expected duckdb ref preserved, got: %s", rewritten)
		}

		rows, err := engine.Query(context.Background(), rewritten)
		if err != nil {
			t.Fatalf("query error: %v", err)
		}
		defer rows.Close()

		var names []string
		for rows.Next() {
			var name string
			var salary float64
			if err := rows.Scan(&name, &salary); err != nil {
				t.Fatalf("scan error: %v", err)
			}
			names = append(names, name)
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("rows iteration error: %v", err)
		}

		wantNames := []string{"Alice", "Bob", "Eve"}
		if len(names) != len(wantNames) {
			t.Fatalf("got %d rows, want %d", len(names), len(wantNames))
		}
		for i := range wantNames {
			if names[i] != wantNames[i] {
				t.Errorf("row %d: got %q, want %q", i, names[i], wantNames[i])
			}
		}
	})
}
