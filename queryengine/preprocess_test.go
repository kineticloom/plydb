package queryengine

import (
	"strings"
	"testing"
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
				Type: S3,
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

func mustPreprocess(t *testing.T, query string, cfg *Config) (string, error) {
	t.Helper()
	return PreprocessQuery(query, cfg)
}
