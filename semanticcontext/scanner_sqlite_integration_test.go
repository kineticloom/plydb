// Copyright 2026 Paul Tzen
// SPDX-License-Identifier: Apache-2.0

//go:build integration

package semanticcontext

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/kineticloom/plydb/queryengine"
)

// ---------------------------------------------------------------------------
// SQLite helpers
// ---------------------------------------------------------------------------

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
			salary        REAL NOT NULL,
			hired_at      TIMESTAMP NOT NULL
		)`,
		`INSERT INTO tmp.employees VALUES
			(1, 'Alice',   1, 95000.00, '2024-01-15 09:00:00'),
			(2, 'Bob',     1, 88000.00, '2024-03-01 09:00:00'),
			(3, 'Charlie', 2, 72000.00, '2024-06-10 09:00:00'),
			(4, 'Diana',   3, 68000.00, '2024-08-20 09:00:00'),
			(5, 'Eve',     2, 75000.00, '2024-11-05 09:00:00')`,
		"DETACH tmp;",
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("creating sqlite test db: %v\nstatement: %s", err, stmt)
		}
	}

	return path
}

func sqliteEngine(t *testing.T, key, path string) (*queryengine.QueryEngine, *queryengine.Config) {
	t.Helper()

	cfg := &queryengine.Config{
		Credentials: map[string]queryengine.Credential{},
		Databases: map[string]queryengine.DatabaseConfig{
			key: {
				Metadata: queryengine.Metadata{Name: "Integration SQLite", Description: "test"},
				Type:     queryengine.SQLite,
				Path:     path,
			},
		},
	}

	engine, err := queryengine.New(cfg)
	if err != nil {
		t.Fatalf("creating engine: %v", err)
	}
	t.Cleanup(func() { engine.Close() })
	return engine, cfg
}

// ---------------------------------------------------------------------------
// SQLite integration tests
// ---------------------------------------------------------------------------

func TestIntegrationSQLite(t *testing.T) {
	path := createSQLiteTestDB(t)

	t.Run("ScanTables", func(t *testing.T) {
		engine, cfg := sqliteEngine(t, "sq", path)

		provider := NewAutoScanProvider(cfg, engine)
		result, err := provider.Provide(context.Background(), nil)
		if err != nil {
			t.Fatalf("Provide: %v", err)
		}

		// Should discover both departments and employees.
		if len(result.SemanticModel.Datasets) != 2 {
			names := make([]string, len(result.SemanticModel.Datasets))
			for i, ds := range result.SemanticModel.Datasets {
				names[i] = ds.Name
			}
			t.Fatalf("expected 2 datasets, got %d: %v", len(result.SemanticModel.Datasets), names)
		}

		deptDS := findDataset(t, result.SemanticModel.Datasets, "sq.main.departments")
		empDS := findDataset(t, result.SemanticModel.Datasets, "sq.main.employees")

		// Departments: id, name
		if len(deptDS.Fields) != 2 {
			t.Fatalf("departments: expected 2 fields, got %d", len(deptDS.Fields))
		}
		if deptDS.Source != "sq.main.departments" {
			t.Errorf("departments source = %q", deptDS.Source)
		}

		// Employees: id, name, department_id, salary, hired_at
		if len(empDS.Fields) != 5 {
			t.Fatalf("employees: expected 5 fields, got %d", len(empDS.Fields))
		}
	})

	t.Run("DataTypes", func(t *testing.T) {
		engine, cfg := sqliteEngine(t, "sq", path)

		provider := NewAutoScanProvider(cfg, engine)
		result, err := provider.Provide(context.Background(), nil)
		if err != nil {
			t.Fatalf("Provide: %v", err)
		}

		empDS := findDataset(t, result.SemanticModel.Datasets, "sq.main.employees")

		// Verify each field has an expression.
		for _, f := range empDS.Fields {
			if f.Expression == nil || len(f.Expression.Dialects) == 0 {
				t.Errorf("field %q has empty Expression", f.Name)
			}
		}
	})

	t.Run("TimeDimensions", func(t *testing.T) {
		engine, cfg := sqliteEngine(t, "sq", path)

		provider := NewAutoScanProvider(cfg, engine)
		result, err := provider.Provide(context.Background(), nil)
		if err != nil {
			t.Fatalf("Provide: %v", err)
		}

		// SQLite has no native date/time types — DuckDB maps TIMESTAMP to
		// VARCHAR via type affinity, so time dimensions are not detected.
		// Verify no fields have time dimensions in either table.
		for _, dsName := range []string{"sq.main.departments", "sq.main.employees"} {
			ds := findDataset(t, result.SemanticModel.Datasets, dsName)
			for _, f := range ds.Fields {
				if f.Dimension != nil {
					t.Errorf("dataset %q field %q: expected no dimension (SQLite type affinity), got %+v",
						dsName, f.Name, f.Dimension)
				}
			}
		}
	})
}
