// Copyright 2026 Paul Tzen
// SPDX-License-Identifier: Apache-2.0

//go:build integration

package queryengine

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// ---------------------------------------------------------------------------
// SQLite helpers
// ---------------------------------------------------------------------------

// createSQLiteTestDB uses DuckDB to create a temporary SQLite file with
// departments and employees tables matching the existing test patterns.
func createSQLiteTestDB(t *testing.T) string {
	t.Helper()

	f, err := os.CreateTemp("", "plydb_test_*.sqlite")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	path := f.Name()
	f.Close()
	t.Cleanup(func() { os.Remove(path) })

	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("opening duckdb: %v", err)
	}
	defer db.Close()

	stmts := []string{
		"INSTALL sqlite;",
		"LOAD sqlite;",
		fmt.Sprintf(`ATTACH '%s' AS tmp (TYPE SQLITE);`, path),
		`CREATE TABLE tmp.departments (
			id   INTEGER PRIMARY KEY,
			name TEXT NOT NULL
		)`,
		`INSERT INTO tmp.departments VALUES (1, 'Engineering'), (2, 'Sales'), (3, 'Marketing')`,
		`CREATE TABLE tmp.employees (
			id            INTEGER PRIMARY KEY,
			name          TEXT NOT NULL,
			department_id INTEGER NOT NULL,
			salary        REAL NOT NULL
		)`,
		`INSERT INTO tmp.employees VALUES
			(1, 'Alice',   1, 95000.00),
			(2, 'Bob',     1, 88000.00),
			(3, 'Charlie', 2, 72000.00),
			(4, 'Diana',   3, 68000.00),
			(5, 'Eve',     2, 75000.00)`,
		"DETACH tmp;",
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("creating sqlite test db: %v\nstatement: %s", err, stmt)
		}
	}

	return path
}

func sqliteConfig(t *testing.T, key, path string) *Config {
	t.Helper()
	return &Config{
		Credentials: map[string]Credential{},
		Databases: map[string]DatabaseConfig{
			key: {
				Metadata: Metadata{Name: "Integration SQLite", Description: "test"},
				Type:     SQLite,
				Path:     path,
			},
		},
	}
}

// ---------------------------------------------------------------------------
// SQLite integration tests
// ---------------------------------------------------------------------------

func TestIntegrationSQLite(t *testing.T) {
	path := createSQLiteTestDB(t)

	t.Run("BasicSelect", func(t *testing.T) {
		cfg := sqliteConfig(t, "sq", path)
		engine, err := New(cfg)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		defer engine.Close()

		rows, err := engine.Query(context.Background(),
			`SELECT id, name, department_id, salary FROM "sq".main.employees ORDER BY id`)
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
		cfg := sqliteConfig(t, "sq", path)
		engine, err := New(cfg)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		defer engine.Close()

		rows, err := engine.Query(context.Background(),
			`SELECT name, salary FROM "sq".main.employees WHERE salary > 80000 ORDER BY salary DESC`)
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
		cfg := sqliteConfig(t, "sq", path)
		engine, err := New(cfg)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		defer engine.Close()

		rows, err := engine.Query(context.Background(),
			`SELECT e.name, d.name AS dept
			 FROM "sq".main.employees e
			 JOIN "sq".main.departments d ON e.department_id = d.id
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
		cfg := sqliteConfig(t, "sq", path)
		engine, err := New(cfg)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		defer engine.Close()

		original := `SELECT name, salary FROM sq.main.employees WHERE salary >= 75000 ORDER BY name`
		rewritten, err := PreprocessQuery(original, cfg)
		if err != nil {
			t.Fatalf("PreprocessQuery: %v", err)
		}

		// SQLite is passthrough — query should be preserved.
		if !strings.Contains(rewritten, "sq.main.employees") {
			t.Fatalf("expected sqlite ref preserved, got: %s", rewritten)
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
