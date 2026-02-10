package queryengine

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNewEngineFileOnly(t *testing.T) {
	cfg := &Config{
		Credentials: map[string]Credential{},
		Databases: map[string]DatabaseConfig{
			"local-csv": {
				Metadata: Metadata{Name: "Test CSV", Description: "test"},
				Type:     File,
				Path:     "/tmp/test.csv",
				Format:   "csv",
			},
		},
	}

	engine, err := New(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer engine.Close()

	rows, err := engine.Query(context.Background(), "SELECT 1 AS n")
	if err != nil {
		t.Fatalf("query error: %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Fatal("expected one row")
	}
	var n int
	if err := rows.Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("got %d, want 1", n)
	}
}

func TestNewEngineEmpty(t *testing.T) {
	cfg := &Config{
		Credentials: map[string]Credential{},
		Databases:   map[string]DatabaseConfig{},
	}

	engine, err := New(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer engine.Close()

	rows, err := engine.Query(context.Background(), "SELECT 42 AS answer")
	if err != nil {
		t.Fatalf("query error: %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Fatal("expected one row")
	}
	var answer int
	if err := rows.Scan(&answer); err != nil {
		t.Fatal(err)
	}
	if answer != 42 {
		t.Fatalf("got %d, want 42", answer)
	}
}

func testdataPath(name string) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata", name)
}

func TestEngineReadCSV(t *testing.T) {
	csvPath := testdataPath("products.csv")

	cfg := &Config{
		Credentials: map[string]Credential{},
		Databases: map[string]DatabaseConfig{
			"products": {
				Metadata: Metadata{Name: "Products", Description: "test product catalog"},
				Type:     File,
				Path:     csvPath,
				Format:   "csv",
			},
		},
	}

	engine, err := New(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer engine.Close()

	// Query all rows ordered by id.
	query := fmt.Sprintf(`SELECT id, name, price, in_stock FROM read_csv('%s', header=true, auto_detect=true) ORDER BY id`, csvPath)
	rows, err := engine.Query(context.Background(), query)
	if err != nil {
		t.Fatalf("query error: %v", err)
	}
	defer rows.Close()

	type product struct {
		id      int
		name    string
		price   float64
		inStock bool
	}

	want := []product{
		{1, "Widget", 9.99, true},
		{2, "Gadget", 24.50, true},
		{3, "Doohickey", 3.75, false},
		{4, "Thingamajig", 15.00, true},
		{5, "Whatchamacallit", 7.25, false},
	}

	var got []product
	for rows.Next() {
		var p product
		if err := rows.Scan(&p.id, &p.name, &p.price, &p.inStock); err != nil {
			t.Fatalf("scan error: %v", err)
		}
		got = append(got, p)
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
}

func TestEngineReadCSVWithFilter(t *testing.T) {
	csvPath := testdataPath("products.csv")

	cfg := &Config{
		Credentials: map[string]Credential{},
		Databases:   map[string]DatabaseConfig{},
	}

	engine, err := New(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer engine.Close()

	query := fmt.Sprintf(`SELECT name, price FROM read_csv('%s', header=true, auto_detect=true) WHERE in_stock = true ORDER BY price DESC`, csvPath)
	rows, err := engine.Query(context.Background(), query)
	if err != nil {
		t.Fatalf("query error: %v", err)
	}
	defer rows.Close()

	type result struct {
		name  string
		price float64
	}

	want := []result{
		{"Gadget", 24.50},
		{"Thingamajig", 15.00},
		{"Widget", 9.99},
	}

	var got []result
	for rows.Next() {
		var r result
		if err := rows.Scan(&r.name, &r.price); err != nil {
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
}

func TestNewEngineMissingEnvVar(t *testing.T) {
	cfg := &Config{
		Credentials: map[string]Credential{
			"aws1": {
				AccessKeyEnv: "MISSING_AWS_KEY_XYZ_999",
				SecretKeyEnv: "MISSING_AWS_SECRET_XYZ_999",
			},
		},
		Databases: map[string]DatabaseConfig{
			"s3-data": {
				Metadata:          Metadata{Name: "S3", Description: "test"},
				Type:              S3,
				URI:               "s3://bucket/path/",
				CredentialProfile: "aws1",
				Region:            "us-east-1",
				Format:            "parquet",
			},
		},
	}

	_, err := New(cfg)
	if err == nil {
		t.Fatal("expected error for missing env var")
	}
	if !strings.Contains(err.Error(), "MISSING_AWS_KEY_XYZ_999") {
		t.Fatalf("expected error mentioning env var, got: %v", err)
	}
}
