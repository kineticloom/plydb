// Copyright 2026 Paul Tzen
// SPDX-License-Identifier: Apache-2.0

package queryengine

import (
	"fmt"
	"strings"
	"testing"

	pg_query "github.com/pganalyze/pg_query_go/v6"
)

func testConfig() *Config {
	return &Config{
		Databases: map[string]DatabaseConfig{
			"my_pg": {
				Type: PostgreSQL,
				Host: "localhost", Port: 5432, DatabaseName: "mydb",
				Username: "user", PasswordEnvVar: "PG_PASS",
			},
			"my_mysql": {
				Type: MySQL,
				Host: "localhost", Port: 3306, DatabaseName: "mydb",
				Username: "user", PasswordEnvVar: "MYSQL_PASS",
			},
			"my_csv": {
				Type: File,
				Path: "/data/sales.csv",
			},
			"my_parquet": {
				Type: File,
				Path: "/data/events.parquet",
			},
			"my_xlsx": {
				Type: File,
				Path: "/data/report.xlsx",
			},
			"my_s3": {
				Type:   S3,
				URI:    "s3://bucket/path/data.parquet",
				Region: "us-east-1",
			},
		},
	}
}

func TestPreprocessQuery_NoTableRefs(t *testing.T) {
	cfg := testConfig()
	result, err := mustPreprocess(t, "SELECT 1", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "SELECT 1" {
		t.Errorf("expected %q, got %q", "SELECT 1", result)
	}
}

func TestPreprocessQuery_PostgresPassthrough(t *testing.T) {
	cfg := testConfig()
	query := "SELECT id, name FROM my_pg.public.users WHERE active = true"
	result, err := mustPreprocess(t, query, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "my_pg.public.users") {
		t.Errorf("expected postgres ref to be preserved, got %q", result)
	}
}

func TestPreprocessQuery_MySQLPassthrough(t *testing.T) {
	cfg := testConfig()
	query := "SELECT id FROM my_mysql.myschema.orders"
	result, err := mustPreprocess(t, query, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "my_mysql.myschema.orders") {
		t.Errorf("expected mysql ref to be preserved, got %q", result)
	}
}

func TestPreprocessQuery_CSVFileRewrite(t *testing.T) {
	cfg := testConfig()
	query := "SELECT * FROM my_csv.default.sales"
	result, err := mustPreprocess(t, query, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, `"/data/sales.csv"`) {
		t.Errorf("expected CSV path rewrite, got %q", result)
	}
	if strings.Contains(result, "my_csv") {
		t.Errorf("expected catalog to be removed, got %q", result)
	}
}

func TestPreprocessQuery_ParquetFileRewrite(t *testing.T) {
	cfg := testConfig()
	query := "SELECT col1 FROM my_parquet.default.events"
	result, err := mustPreprocess(t, query, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, `"/data/events.parquet"`) {
		t.Errorf("expected parquet path rewrite, got %q", result)
	}
}

func TestPreprocessQuery_XLSXFileRewrite(t *testing.T) {
	cfg := testConfig()
	query := "SELECT col1 FROM my_xlsx.default.sheet1"
	result, err := mustPreprocess(t, query, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "read_xlsx('/data/report.xlsx', sheet := 'sheet1')"
	if !strings.Contains(result, want) {
		t.Errorf("expected %q in result, got %q", want, result)
	}
}

func TestPreprocessQuery_XLSXFileRewriteWithAlias(t *testing.T) {
	cfg := testConfig()
	query := "SELECT r.col1 FROM my_xlsx.default.sheet1 AS r"
	result, err := mustPreprocess(t, query, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "read_xlsx('/data/report.xlsx', sheet := 'sheet1') r"
	if !strings.Contains(result, want) {
		t.Errorf("expected %q in result, got %q", want, result)
	}
}

func TestPreprocessQuery_S3Rewrite(t *testing.T) {
	cfg := testConfig()
	query := "SELECT * FROM my_s3.default.data"
	result, err := mustPreprocess(t, query, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, `"s3://bucket/path/data.parquet"`) {
		t.Errorf("expected S3 URI rewrite, got %q", result)
	}
}

func TestPreprocessQuery_MixedJoin(t *testing.T) {
	cfg := testConfig()
	query := `SELECT u.id, s.amount
		FROM my_pg.public.users AS u
		JOIN my_csv.default.sales AS s ON u.id = s.user_id`
	result, err := mustPreprocess(t, query, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "my_pg.public.users") {
		t.Errorf("expected postgres ref preserved, got %q", result)
	}
	if !strings.Contains(result, `"/data/sales.csv"`) {
		t.Errorf("expected CSV path rewrite, got %q", result)
	}
}

func TestPreprocessQuery_ErrorNotFullyQualified(t *testing.T) {
	cfg := testConfig()
	tests := []struct {
		name  string
		query string
	}{
		{"unqualified", "SELECT * FROM users"},
		{"schema only", "SELECT * FROM public.users"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := PreprocessQuery(tt.query, cfg)
			if err == nil {
				t.Fatal("expected error for non-fully-qualified ref")
			}
			if !strings.Contains(err.Error(), "not fully qualified") {
				t.Errorf("expected 'not fully qualified' error, got: %v", err)
			}
		})
	}
}

func TestPreprocessQuery_ErrorUnknownCatalog(t *testing.T) {
	cfg := testConfig()
	_, err := PreprocessQuery("SELECT * FROM unknown_db.public.users", cfg)
	if err == nil {
		t.Fatal("expected error for unknown catalog")
	}
	if !strings.Contains(err.Error(), "unknown catalog") {
		t.Errorf("expected 'unknown catalog' error, got: %v", err)
	}
}

func TestPreprocessQuery_ErrorInvalidSQL(t *testing.T) {
	cfg := testConfig()
	_, err := PreprocessQuery("NOT VALID SQL !!!", cfg)
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "parse error") {
		t.Errorf("expected 'parse error', got: %v", err)
	}
}

func mustPreprocess(t *testing.T, query string, cfg *Config, validators ...ValidateFunc) (string, error) {
	t.Helper()
	return PreprocessQuery(query, cfg, validators...)
}

func TestPreprocessQuery_WithValidatorPass(t *testing.T) {
	cfg := testConfig()
	validator := func(parsed *pg_query.ParseResult) error {
		return nil
	}
	result, err := PreprocessQuery("SELECT 1", cfg, validator)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "SELECT 1" {
		t.Errorf("expected %q, got %q", "SELECT 1", result)
	}
}

func TestPreprocessQuery_WithValidatorFail(t *testing.T) {
	cfg := testConfig()
	validator := func(parsed *pg_query.ParseResult) error {
		return fmt.Errorf("query not allowed")
	}
	_, err := PreprocessQuery("SELECT 1", cfg, validator)
	if err == nil {
		t.Fatal("expected error from validator")
	}
	if !strings.Contains(err.Error(), "validation error") {
		t.Errorf("expected 'validation error', got: %v", err)
	}
	if !strings.Contains(err.Error(), "query not allowed") {
		t.Errorf("expected wrapped error message, got: %v", err)
	}
}

func TestPreprocessQuery_NilValidator(t *testing.T) {
	cfg := testConfig()
	result, err := PreprocessQuery("SELECT 1", cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "SELECT 1" {
		t.Errorf("expected %q, got %q", "SELECT 1", result)
	}
}

func TestPreprocessQuery_MultipleValidators(t *testing.T) {
	cfg := testConfig()
	var called []int
	v1 := func(_ *pg_query.ParseResult) error { called = append(called, 1); return nil }
	v2 := func(_ *pg_query.ParseResult) error { called = append(called, 2); return nil }
	v3 := func(_ *pg_query.ParseResult) error { called = append(called, 3); return nil }

	_, err := PreprocessQuery("SELECT 1", cfg, v1, v2, v3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(called) != 3 || called[0] != 1 || called[1] != 2 || called[2] != 3 {
		t.Errorf("expected all validators called in order [1 2 3], got %v", called)
	}
}

func TestPreprocessQuery_MultipleValidatorsStopsOnError(t *testing.T) {
	cfg := testConfig()
	var called []int
	v1 := func(_ *pg_query.ParseResult) error { called = append(called, 1); return nil }
	v2 := func(_ *pg_query.ParseResult) error { called = append(called, 2); return fmt.Errorf("v2 failed") }
	v3 := func(_ *pg_query.ParseResult) error { called = append(called, 3); return nil }

	_, err := PreprocessQuery("SELECT 1", cfg, v1, v2, v3)
	if err == nil {
		t.Fatal("expected error from second validator")
	}
	if !strings.Contains(err.Error(), "v2 failed") {
		t.Errorf("expected 'v2 failed' in error, got: %v", err)
	}
	if len(called) != 2 || called[0] != 1 || called[1] != 2 {
		t.Errorf("expected validators [1 2] called before failure, got %v", called)
	}
}
