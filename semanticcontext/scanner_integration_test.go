// Copyright 2026 Paul Tzen
// SPDX-License-Identifier: Apache-2.0

//go:build integration

package semanticcontext

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	_ "github.com/go-sql-driver/mysql"
	"github.com/kineticloom/plydb/queryengine"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// ---------------------------------------------------------------------------
// Container helpers
// ---------------------------------------------------------------------------

func pgContainer(t *testing.T) (host string, port int, cleanup func()) {
	t.Helper()
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:17-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "testuser",
			"POSTGRES_PASSWORD": "testpass",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForSQL("5432/tcp", "postgres",
			func(host string, port nat.Port) string {
				return fmt.Sprintf("postgres://testuser:testpass@%s:%s/testdb?sslmode=disable", host, port.Port())
			}).WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("starting postgres container: %v", err)
	}

	mappedHost, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("getting postgres host: %v", err)
	}

	mappedPort, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		t.Fatalf("getting postgres port: %v", err)
	}

	return mappedHost, mappedPort.Int(), func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("warning: terminating postgres container: %v", err)
		}
	}
}

func mysqlContainer(t *testing.T) (host string, port int, cleanup func()) {
	t.Helper()
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "mysql:8.0",
		ExposedPorts: []string{"3306/tcp"},
		Env: map[string]string{
			"MYSQL_ROOT_PASSWORD": "testpass",
			"MYSQL_DATABASE":      "testdb",
			"MYSQL_USER":          "testuser",
			"MYSQL_PASSWORD":      "testpass",
		},
		WaitingFor: wait.ForSQL("3306/tcp", "mysql",
			func(host string, port nat.Port) string {
				return fmt.Sprintf("testuser:testpass@tcp(%s:%s)/testdb", host, port.Port())
			}).WithStartupTimeout(120 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("starting mysql container: %v", err)
	}

	mappedHost, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("getting mysql host: %v", err)
	}

	mappedPort, err := container.MappedPort(ctx, "3306/tcp")
	if err != nil {
		t.Fatalf("getting mysql port: %v", err)
	}

	return mappedHost, mappedPort.Int(), func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("warning: terminating mysql container: %v", err)
		}
	}
}

// ---------------------------------------------------------------------------
// Seed helpers
// ---------------------------------------------------------------------------

func seedPostgres(t *testing.T, host string, port int) {
	t.Helper()

	dsn := fmt.Sprintf("postgres://testuser:testpass@%s:%d/testdb?sslmode=disable", host, port)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("connecting to postgres for seeding: %v", err)
	}
	defer db.Close()

	stmts := []string{
		`CREATE TABLE IF NOT EXISTS departments (
			id   INTEGER PRIMARY KEY,
			name TEXT NOT NULL
		)`,
		`COMMENT ON TABLE departments IS 'Company departments'`,
		`COMMENT ON COLUMN departments.id IS 'Department identifier'`,
		`COMMENT ON COLUMN departments.name IS 'Department display name'`,

		`INSERT INTO departments (id, name) VALUES (1, 'Engineering'), (2, 'Sales'), (3, 'Marketing')`,

		`CREATE TABLE IF NOT EXISTS employees (
			id            INTEGER PRIMARY KEY,
			name          TEXT NOT NULL,
			department_id INTEGER NOT NULL,
			salary        NUMERIC(10,2) NOT NULL,
			hired_at      TIMESTAMP NOT NULL DEFAULT NOW()
		)`,
		`COMMENT ON TABLE employees IS 'Employee records'`,
		`COMMENT ON COLUMN employees.id IS 'Employee identifier'`,
		`COMMENT ON COLUMN employees.name IS 'Full name'`,
		`COMMENT ON COLUMN employees.department_id IS 'FK to departments'`,
		`COMMENT ON COLUMN employees.salary IS 'Annual salary in USD'`,
		`COMMENT ON COLUMN employees.hired_at IS 'Hire timestamp'`,

		`INSERT INTO employees (id, name, department_id, salary, hired_at) VALUES
			(1, 'Alice',   1, 95000.00, '2024-01-15 09:00:00'),
			(2, 'Bob',     1, 88000.00, '2024-03-01 09:00:00'),
			(3, 'Charlie', 2, 72000.00, '2024-06-10 09:00:00'),
			(4, 'Diana',   3, 68000.00, '2024-08-20 09:00:00'),
			(5, 'Eve',     2, 75000.00, '2024-11-05 09:00:00')`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("seeding postgres: %v\nstatement: %s", err, stmt)
		}
	}
}

func seedMySQL(t *testing.T, host string, port int) {
	t.Helper()

	dsn := fmt.Sprintf("testuser:testpass@tcp(%s:%d)/testdb", host, port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("connecting to mysql for seeding: %v", err)
	}
	defer db.Close()

	stmts := []string{
		`CREATE TABLE IF NOT EXISTS departments (
			id   INT PRIMARY KEY COMMENT 'Department identifier',
			name VARCHAR(100) NOT NULL COMMENT 'Department display name'
		) COMMENT='Company departments'`,

		`INSERT INTO departments (id, name) VALUES (1, 'Engineering'), (2, 'Sales'), (3, 'Marketing')`,

		`CREATE TABLE IF NOT EXISTS employees (
			id            INT PRIMARY KEY COMMENT 'Employee identifier',
			name          VARCHAR(100) NOT NULL COMMENT 'Full name',
			department_id INT NOT NULL COMMENT 'FK to departments',
			salary        DECIMAL(10,2) NOT NULL COMMENT 'Annual salary in USD',
			hired_at      DATETIME NOT NULL COMMENT 'Hire timestamp'
		) COMMENT='Employee records'`,

		`INSERT INTO employees (id, name, department_id, salary, hired_at) VALUES
			(1, 'Alice',   1, 95000.00, '2024-01-15 09:00:00'),
			(2, 'Bob',     1, 88000.00, '2024-03-01 09:00:00'),
			(3, 'Charlie', 2, 72000.00, '2024-06-10 09:00:00'),
			(4, 'Diana',   3, 68000.00, '2024-08-20 09:00:00'),
			(5, 'Eve',     2, 75000.00, '2024-11-05 09:00:00')`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("seeding mysql: %v\nstatement: %s", err, stmt)
		}
	}
}

// ---------------------------------------------------------------------------
// Engine + querier helpers
// ---------------------------------------------------------------------------

const testPasswordEnvVar = "INTTEST_SC_PASS"

func pgEngine(t *testing.T, key, host string, port int) (*queryengine.QueryEngine, *queryengine.Config) {
	t.Helper()
	t.Setenv(testPasswordEnvVar, "testpass")

	cfg := &queryengine.Config{
		Credentials: map[string]queryengine.Credential{},
		Databases: map[string]queryengine.DatabaseConfig{
			key: {
				Metadata:       queryengine.Metadata{Name: "Integration Postgres", Description: "test"},
				Type:           queryengine.PostgreSQL,
				Host:           host,
				Port:           port,
				DatabaseName:   "testdb",
				Username:       "testuser",
				PasswordEnvVar: testPasswordEnvVar,
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

func mysqlEngine(t *testing.T, key, host string, port int) (*queryengine.QueryEngine, *queryengine.Config) {
	t.Helper()
	t.Setenv(testPasswordEnvVar, "testpass")

	cfg := &queryengine.Config{
		Credentials: map[string]queryengine.Credential{},
		Databases: map[string]queryengine.DatabaseConfig{
			key: {
				Metadata:       queryengine.Metadata{Name: "Integration MySQL", Description: "test"},
				Type:           queryengine.MySQL,
				Host:           host,
				Port:           port,
				DatabaseName:   "testdb",
				Username:       "testuser",
				PasswordEnvVar: testPasswordEnvVar,
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
// Helpers for assertions
// ---------------------------------------------------------------------------

func findDataset(t *testing.T, datasets []Dataset, name string) Dataset {
	t.Helper()
	for _, ds := range datasets {
		if ds.Name == name {
			return ds
		}
	}
	names := make([]string, len(datasets))
	for i, ds := range datasets {
		names[i] = ds.Name
	}
	t.Fatalf("dataset %q not found; available: %v", name, names)
	return Dataset{}
}

func findField(t *testing.T, fields []Field, name string) Field {
	t.Helper()
	for _, f := range fields {
		if f.Name == name {
			return f
		}
	}
	fnames := make([]string, len(fields))
	for i, f := range fields {
		fnames[i] = f.Name
	}
	t.Fatalf("field %q not found; available: %v", name, fnames)
	return Field{}
}

// ---------------------------------------------------------------------------
// Postgres integration tests
// ---------------------------------------------------------------------------

func TestIntegrationPostgres(t *testing.T) {
	host, port, cleanup := pgContainer(t)
	defer cleanup()

	seedPostgres(t, host, port)

	t.Run("ScanTables", func(t *testing.T) {
		engine, cfg := pgEngine(t, "pg", host, port)

		provider := NewAutoScanProvider(cfg, engine)
		result, err := provider.Provide(context.Background(), nil)
		if err != nil {
			t.Fatalf("Provide: %v", err)
		}

		// Should discover both departments and employees.
		if len(result.SemanticModel.Datasets) != 2 {
			t.Fatalf("expected 2 datasets, got %d", len(result.SemanticModel.Datasets))
		}

		deptDS := findDataset(t, result.SemanticModel.Datasets, "pg.public.departments")
		empDS := findDataset(t, result.SemanticModel.Datasets, "pg.public.employees")

		// Departments: id, name
		if len(deptDS.Fields) != 2 {
			t.Fatalf("departments: expected 2 fields, got %d", len(deptDS.Fields))
		}
		if deptDS.Source != "pg.public.departments" {
			t.Errorf("departments source = %q", deptDS.Source)
		}

		// Employees: id, name, department_id, salary, hired_at
		if len(empDS.Fields) != 5 {
			t.Fatalf("employees: expected 5 fields, got %d", len(empDS.Fields))
		}
	})

	t.Run("ColumnComments", func(t *testing.T) {
		engine, cfg := pgEngine(t, "pg", host, port)

		provider := NewAutoScanProvider(cfg, engine)
		result, err := provider.Provide(context.Background(), nil)
		if err != nil {
			t.Fatalf("Provide: %v", err)
		}

		empDS := findDataset(t, result.SemanticModel.Datasets, "pg.public.employees")

		// Verify column comments are propagated as field descriptions.
		tests := []struct {
			fieldName string
			wantDesc  string
		}{
			{"id", "Employee identifier"},
			{"name", "Full name"},
			{"department_id", "FK to departments"},
			{"salary", "Annual salary in USD"},
			{"hired_at", "Hire timestamp"},
		}
		for _, tt := range tests {
			f := findField(t, empDS.Fields, tt.fieldName)
			if f.Description != tt.wantDesc {
				t.Errorf("field %q description = %q, want %q", tt.fieldName, f.Description, tt.wantDesc)
			}
		}

		// Also check department column comments.
		deptDS := findDataset(t, result.SemanticModel.Datasets, "pg.public.departments")
		idField := findField(t, deptDS.Fields, "id")
		if idField.Description != "Department identifier" {
			t.Errorf("departments.id description = %q, want %q", idField.Description, "Department identifier")
		}
	})

	t.Run("TableComments", func(t *testing.T) {
		engine, cfg := pgEngine(t, "pg", host, port)

		provider := NewAutoScanProvider(cfg, engine)
		result, err := provider.Provide(context.Background(), nil)
		if err != nil {
			t.Fatalf("Provide: %v", err)
		}

		// Verify table-level COMMENT ON TABLE is propagated as Dataset.Description.
		tests := []struct {
			datasetName string
			wantDesc    string
		}{
			{"pg.public.departments", "Company departments"},
			{"pg.public.employees", "Employee records"},
		}
		for _, tt := range tests {
			ds := findDataset(t, result.SemanticModel.Datasets, tt.datasetName)
			if ds.Description != tt.wantDesc {
				t.Errorf("dataset %q description = %q, want %q", tt.datasetName, ds.Description, tt.wantDesc)
			}
		}
	})

	t.Run("DataTypes", func(t *testing.T) {
		engine, cfg := pgEngine(t, "pg", host, port)

		provider := NewAutoScanProvider(cfg, engine)
		result, err := provider.Provide(context.Background(), nil)
		if err != nil {
			t.Fatalf("Provide: %v", err)
		}

		empDS := findDataset(t, result.SemanticModel.Datasets, "pg.public.employees")

		// Verify each field has an expression with the column name.
		for _, f := range empDS.Fields {
			if f.Expression == nil || len(f.Expression.Dialects) == 0 {
				t.Errorf("field %q has empty Expression", f.Name)
			} else if f.Expression.Dialects[0].Expression != f.Name {
				t.Errorf("field %q expression = %q, want %q", f.Name, f.Expression.Dialects[0].Expression, f.Name)
			}
		}
	})

	t.Run("TimeDimensions", func(t *testing.T) {
		engine, cfg := pgEngine(t, "pg", host, port)

		provider := NewAutoScanProvider(cfg, engine)
		result, err := provider.Provide(context.Background(), nil)
		if err != nil {
			t.Fatalf("Provide: %v", err)
		}

		empDS := findDataset(t, result.SemanticModel.Datasets, "pg.public.employees")

		// hired_at is a timestamp → should have a time dimension on the field.
		hiredAt := findField(t, empDS.Fields, "hired_at")
		if hiredAt.Dimension == nil || !hiredAt.Dimension.IsTime {
			t.Errorf("expected hired_at to have time dimension, got %+v", hiredAt.Dimension)
		}

		// departments has no time columns → no fields with dimensions.
		deptDS := findDataset(t, result.SemanticModel.Datasets, "pg.public.departments")
		for _, f := range deptDS.Fields {
			if f.Dimension != nil {
				t.Errorf("expected no dimensions for departments, but field %q has %+v", f.Name, f.Dimension)
			}
		}
	})

	t.Run("ExistingModel", func(t *testing.T) {
		engine, cfg := pgEngine(t, "pg", host, port)

		existing := &SemanticModelFile{
			SemanticModel: SemanticModel{
				Name:        "My Model",
				Description: "Pre-existing model",
			},
		}

		provider := NewAutoScanProvider(cfg, engine)
		result, err := provider.Provide(context.Background(), existing)
		if err != nil {
			t.Fatalf("Provide: %v", err)
		}

		// Should preserve existing model metadata.
		if result.SemanticModel.Name != "My Model" {
			t.Errorf("name = %q, want %q", result.SemanticModel.Name, "My Model")
		}
		if result.SemanticModel.Description != "Pre-existing model" {
			t.Errorf("description = %q, want %q", result.SemanticModel.Description, "Pre-existing model")
		}

		// Should still have scanned datasets appended.
		if len(result.SemanticModel.Datasets) != 2 {
			t.Fatalf("expected 2 datasets, got %d", len(result.SemanticModel.Datasets))
		}
	})
}

// ---------------------------------------------------------------------------
// MySQL integration tests
// ---------------------------------------------------------------------------

func TestIntegrationMySQL(t *testing.T) {
	host, port, cleanup := mysqlContainer(t)
	defer cleanup()

	seedMySQL(t, host, port)

	t.Run("ScanTables", func(t *testing.T) {
		engine, cfg := mysqlEngine(t, "my", host, port)

		provider := NewAutoScanProvider(cfg, engine)
		result, err := provider.Provide(context.Background(), nil)
		if err != nil {
			t.Fatalf("Provide: %v", err)
		}

		// Should discover both departments and employees in testdb schema.
		if len(result.SemanticModel.Datasets) != 2 {
			t.Fatalf("expected 2 datasets, got %d", len(result.SemanticModel.Datasets))
		}

		deptDS := findDataset(t, result.SemanticModel.Datasets, "my.testdb.departments")
		empDS := findDataset(t, result.SemanticModel.Datasets, "my.testdb.employees")

		// Departments: id, name
		if len(deptDS.Fields) != 2 {
			t.Fatalf("departments: expected 2 fields, got %d", len(deptDS.Fields))
		}

		// Employees: id, name, department_id, salary, hired_at
		if len(empDS.Fields) != 5 {
			t.Fatalf("employees: expected 5 fields, got %d", len(empDS.Fields))
		}
	})

	t.Run("ColumnComments", func(t *testing.T) {
		engine, cfg := mysqlEngine(t, "my", host, port)

		provider := NewAutoScanProvider(cfg, engine)
		result, err := provider.Provide(context.Background(), nil)
		if err != nil {
			t.Fatalf("Provide: %v", err)
		}

		empDS := findDataset(t, result.SemanticModel.Datasets, "my.testdb.employees")

		// Verify column comments from MySQL's column_comment are propagated.
		tests := []struct {
			fieldName string
			wantDesc  string
		}{
			{"id", "Employee identifier"},
			{"name", "Full name"},
			{"department_id", "FK to departments"},
			{"salary", "Annual salary in USD"},
			{"hired_at", "Hire timestamp"},
		}
		for _, tt := range tests {
			f := findField(t, empDS.Fields, tt.fieldName)
			if f.Description != tt.wantDesc {
				t.Errorf("field %q description = %q, want %q", tt.fieldName, f.Description, tt.wantDesc)
			}
		}
	})

	t.Run("TableComments", func(t *testing.T) {
		engine, cfg := mysqlEngine(t, "my", host, port)

		provider := NewAutoScanProvider(cfg, engine)
		result, err := provider.Provide(context.Background(), nil)
		if err != nil {
			t.Fatalf("Provide: %v", err)
		}

		// Verify table-level COMMENT= is propagated as Dataset.Description.
		tests := []struct {
			datasetName string
			wantDesc    string
		}{
			{"my.testdb.departments", "Company departments"},
			{"my.testdb.employees", "Employee records"},
		}
		for _, tt := range tests {
			ds := findDataset(t, result.SemanticModel.Datasets, tt.datasetName)
			if ds.Description != tt.wantDesc {
				t.Errorf("dataset %q description = %q, want %q", tt.datasetName, ds.Description, tt.wantDesc)
			}
		}
	})

	t.Run("DataTypes", func(t *testing.T) {
		engine, cfg := mysqlEngine(t, "my", host, port)

		provider := NewAutoScanProvider(cfg, engine)
		result, err := provider.Provide(context.Background(), nil)
		if err != nil {
			t.Fatalf("Provide: %v", err)
		}

		empDS := findDataset(t, result.SemanticModel.Datasets, "my.testdb.employees")

		// Verify each field has an expression with the column name.
		for _, f := range empDS.Fields {
			if f.Expression == nil || len(f.Expression.Dialects) == 0 {
				t.Errorf("field %q has empty Expression", f.Name)
			}
		}
	})

	t.Run("TimeDimensions", func(t *testing.T) {
		engine, cfg := mysqlEngine(t, "my", host, port)

		provider := NewAutoScanProvider(cfg, engine)
		result, err := provider.Provide(context.Background(), nil)
		if err != nil {
			t.Fatalf("Provide: %v", err)
		}

		empDS := findDataset(t, result.SemanticModel.Datasets, "my.testdb.employees")

		hiredAt := findField(t, empDS.Fields, "hired_at")
		if hiredAt.Dimension == nil || !hiredAt.Dimension.IsTime {
			t.Errorf("expected hired_at to have time dimension, got %+v", hiredAt.Dimension)
		}
	})

	t.Run("MixedWithFile", func(t *testing.T) {
		t.Setenv(testPasswordEnvVar, "testpass")

		cfg := &queryengine.Config{
			Credentials: map[string]queryengine.Credential{},
			Databases: map[string]queryengine.DatabaseConfig{
				"my": {
					Metadata:       queryengine.Metadata{Name: "MySQL", Description: "test"},
					Type:           queryengine.MySQL,
					Host:           host,
					Port:           port,
					DatabaseName:   "testdb",
					Username:       "testuser",
					PasswordEnvVar: testPasswordEnvVar,
				},
				"products_csv": {
					Type:   queryengine.File,
					Path:   "../queryengine/testdata/products.csv",
					Format: "csv",
					Metadata: queryengine.Metadata{
						Name:        "Products",
						Description: "Product catalog",
					},
				},
			},
		}

		engine, err := queryengine.New(cfg)
		if err != nil {
			t.Fatalf("creating engine: %v", err)
		}
		defer engine.Close()

		provider := NewAutoScanProvider(cfg, engine)
		result, err := provider.Provide(context.Background(), nil)
		if err != nil {
			t.Fatalf("Provide: %v", err)
		}

		// 2 MySQL tables + 1 CSV = 3 datasets.
		if len(result.SemanticModel.Datasets) != 3 {
			names := make([]string, len(result.SemanticModel.Datasets))
			for i, ds := range result.SemanticModel.Datasets {
				names[i] = ds.Name
			}
			t.Fatalf("expected 3 datasets, got %d: %v", len(result.SemanticModel.Datasets), names)
		}

		// CSV dataset should have its description from Metadata.
		csvDS := findDataset(t, result.SemanticModel.Datasets, "products_csv.default.products_csv")
		if csvDS.Description != "Product catalog" {
			t.Errorf("csv description = %q, want %q", csvDS.Description, "Product catalog")
		}
	})
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestIntegrationPostgresEmptyDatabase(t *testing.T) {
	host, port, cleanup := pgContainer(t)
	defer cleanup()

	// Don't seed — database has no user tables.
	engine, cfg := pgEngine(t, "pg", host, port)

	provider := NewAutoScanProvider(cfg, engine)
	result, err := provider.Provide(context.Background(), nil)
	if err != nil {
		t.Fatalf("Provide: %v", err)
	}

	if len(result.SemanticModel.Datasets) != 0 {
		t.Fatalf("expected 0 datasets for empty database, got %d", len(result.SemanticModel.Datasets))
	}
}

func TestIntegrationMySQLEmptyDatabase(t *testing.T) {
	host, port, cleanup := mysqlContainer(t)
	defer cleanup()

	// Don't seed.
	engine, cfg := mysqlEngine(t, "my", host, port)

	provider := NewAutoScanProvider(cfg, engine)
	result, err := provider.Provide(context.Background(), nil)
	if err != nil {
		t.Fatalf("Provide: %v", err)
	}

	if len(result.SemanticModel.Datasets) != 0 {
		t.Fatalf("expected 0 datasets for empty database, got %d", len(result.SemanticModel.Datasets))
	}
}
