package semanticcontext

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/kineticloom/plydb/queryengine"
)

// duckDBQuerier wraps a *sql.DB to satisfy MetadataQuerier.
type duckDBQuerier struct {
	db *sql.DB
}

func (d *duckDBQuerier) Query(ctx context.Context, sqlQuery string) (*sql.Rows, error) {
	return d.db.QueryContext(ctx, sqlQuery)
}

func newTestQuerier(t *testing.T) (*duckDBQuerier, func()) {
	t.Helper()
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("opening duckdb: %v", err)
	}
	return &duckDBQuerier{db: db}, func() { db.Close() }
}

func TestScanFile_CSV(t *testing.T) {
	q, cleanup := newTestQuerier(t)
	defer cleanup()

	dbCfg := queryengine.DatabaseConfig{
		Type:   queryengine.File,
		Path:   "../queryengine/testdata/products.csv",
		Format: "csv",
		Metadata: queryengine.Metadata{
			Name:        "Products",
			Description: "Product catalog",
		},
	}

	datasets, err := scanFile(context.Background(), q, "products", dbCfg)
	if err != nil {
		t.Fatalf("scanFile error: %v", err)
	}

	if len(datasets) != 1 {
		t.Fatalf("expected 1 dataset, got %d", len(datasets))
	}

	ds := datasets[0]
	if ds.Name != "products.default.products" {
		t.Errorf("dataset name = %q, want %q", ds.Name, "products.default.products")
	}
	if ds.Source != "products.default.products" {
		t.Errorf("dataset source = %q, want %q", ds.Source, "products.default.products")
	}
	if ds.Description != "Product catalog" {
		t.Errorf("dataset description = %q, want %q", ds.Description, "Product catalog")
	}

	// products.csv has columns: id, name, price, in_stock
	if len(ds.Fields) != 4 {
		t.Fatalf("expected 4 fields, got %d", len(ds.Fields))
	}

	expectedFields := []struct {
		name     string
		dataType string
	}{
		{"id", "BIGINT"},
		{"name", "VARCHAR"},
		{"price", "DOUBLE"},
		{"in_stock", "BOOLEAN"},
	}
	for i, want := range expectedFields {
		got := ds.Fields[i]
		if got.Name != want.name {
			t.Errorf("field[%d].Name = %q, want %q", i, got.Name, want.name)
		}
		if got.DataType != want.dataType {
			t.Errorf("field[%d].DataType = %q, want %q", i, got.DataType, want.dataType)
		}
	}
}

func TestScanFile_MultipleFiles(t *testing.T) {
	q, cleanup := newTestQuerier(t)
	defer cleanup()

	cfg := &queryengine.Config{
		Databases: map[string]queryengine.DatabaseConfig{
			"products": {
				Type:   queryengine.File,
				Path:   "../queryengine/testdata/products.csv",
				Format: "csv",
				Metadata: queryengine.Metadata{
					Name:        "Products",
					Description: "Product catalog",
				},
			},
			"customers": {
				Type:   queryengine.File,
				Path:   "testdata/customers.csv",
				Format: "csv",
				Metadata: queryengine.Metadata{
					Name:        "Customers",
					Description: "Customer contact information.",
				},
			},
		},
	}

	provider := NewAutoScanProvider(cfg, q)
	result, err := provider.Provide(context.Background(), nil)
	if err != nil {
		t.Fatalf("Provide error: %v", err)
	}

	// Should have 2 datasets, in sorted key order: customers, products.
	if len(result.SemanticModel.Datasets) != 2 {
		t.Fatalf("expected 2 datasets, got %d", len(result.SemanticModel.Datasets))
	}

	if result.SemanticModel.Datasets[0].Name != "customers.default.customers" {
		t.Errorf("first dataset = %q, want customers", result.SemanticModel.Datasets[0].Name)
	}
	if result.SemanticModel.Datasets[1].Name != "products.default.products" {
		t.Errorf("second dataset = %q, want products", result.SemanticModel.Datasets[1].Name)
	}
}

func TestIsTimeLike(t *testing.T) {
	tests := []struct {
		dataType string
		want     bool
	}{
		{"DATE", true},
		{"TIMESTAMP", true},
		{"TIMESTAMP WITH TIME ZONE", true},
		{"DATETIME", true},
		{"TIME", true},
		{"INTERVAL", true},
		{"VARCHAR", false},
		{"INTEGER", false},
		{"BIGINT", false},
		{"BOOLEAN", false},
		{"DOUBLE", false},
	}

	for _, tt := range tests {
		got := isTimeLike(tt.dataType)
		if got != tt.want {
			t.Errorf("isTimeLike(%q) = %v, want %v", tt.dataType, got, tt.want)
		}
	}
}

func TestColumnsToDataset(t *testing.T) {
	cols := []columnInfo{
		{Column: "id", DataType: "INTEGER"},
		{Column: "name", DataType: "VARCHAR", Comment: "User name"},
		{Column: "created_at", DataType: "TIMESTAMP"},
	}

	ds := columnsToDataset("mydb", "public", "users", cols, "User table")

	if ds.Name != "mydb.public.users" {
		t.Errorf("name = %q, want %q", ds.Name, "mydb.public.users")
	}
	if ds.Source != "mydb.public.users" {
		t.Errorf("source = %q, want %q", ds.Source, "mydb.public.users")
	}
	if ds.Description != "User table" {
		t.Errorf("description = %q, want %q", ds.Description, "User table")
	}
	if len(ds.Fields) != 3 {
		t.Fatalf("fields count = %d, want 3", len(ds.Fields))
	}
	if ds.Fields[1].Description != "User name" {
		t.Errorf("fields[1].Description = %q, want %q", ds.Fields[1].Description, "User name")
	}

	// Should have one time dimension for created_at.
	if len(ds.Dimensions) != 1 {
		t.Fatalf("dimensions count = %d, want 1", len(ds.Dimensions))
	}
	if ds.Dimensions[0].Name != "created_at" || !ds.Dimensions[0].IsTime {
		t.Errorf("dimension = %+v, want created_at with IsTime=true", ds.Dimensions[0])
	}
}

func TestScanFile_TimeDimensions(t *testing.T) {
	q, cleanup := newTestQuerier(t)
	defer cleanup()

	// orders.csv has an order_date column that should be detected as time.
	dbCfg := queryengine.DatabaseConfig{
		Type:   queryengine.File,
		Path:   "testdata/orders.csv",
		Format: "csv",
		Metadata: queryengine.Metadata{
			Name:        "Orders",
			Description: "Order history",
		},
	}

	datasets, err := scanFile(context.Background(), q, "orders", dbCfg)
	if err != nil {
		t.Fatalf("scanFile error: %v", err)
	}

	ds := datasets[0]

	// Check that order_date is detected as a time dimension.
	found := false
	for _, dim := range ds.Dimensions {
		if dim.Name == "order_date" && dim.IsTime {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected order_date time dimension, got dimensions: %+v", ds.Dimensions)
	}
}
