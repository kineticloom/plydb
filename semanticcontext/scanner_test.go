// Copyright 2026 Paul Tzen
// SPDX-License-Identifier: Apache-2.0

package semanticcontext

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
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
		name       string
		expression string
	}{
		{"id", "id"},
		{"name", "name"},
		{"price", "price"},
		{"in_stock", "in_stock"},
	}
	for i, want := range expectedFields {
		got := ds.Fields[i]
		if got.Name != want.name {
			t.Errorf("field[%d].Name = %q, want %q", i, got.Name, want.name)
		}
		if len(got.Expression.Dialects) == 0 {
			t.Errorf("field[%d].Expression is empty", i)
		} else if got.Expression.Dialects[0].Expression != want.expression {
			t.Errorf("field[%d].Expression = %q, want %q", i, got.Expression.Dialects[0].Expression, want.expression)
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
	if len(result.SemanticModel[0].Datasets) != 2 {
		t.Fatalf("expected 2 datasets, got %d", len(result.SemanticModel[0].Datasets))
	}

	if result.SemanticModel[0].Datasets[0].Name != "customers.default.customers" {
		t.Errorf("first dataset = %q, want customers", result.SemanticModel[0].Datasets[0].Name)
	}
	if result.SemanticModel[0].Datasets[1].Name != "products.default.products" {
		t.Errorf("second dataset = %q, want products", result.SemanticModel[0].Datasets[1].Name)
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

	// created_at should have a time dimension on the field.
	if ds.Fields[2].Dimension == nil || !ds.Fields[2].Dimension.IsTime {
		t.Errorf("fields[2].Dimension should have IsTime=true, got %+v", ds.Fields[2].Dimension)
	}

	// id should not have a dimension.
	if ds.Fields[0].Dimension != nil {
		t.Errorf("fields[0].Dimension should be nil, got %+v", ds.Fields[0].Dimension)
	}

	// All fields should have expressions.
	for i, f := range ds.Fields {
		if len(f.Expression.Dialects) == 0 {
			t.Errorf("fields[%d].Expression is empty", i)
		}
	}
}

// mockQuerier implements MetadataQuerier for testing.
// It ignores the actual SQL and returns pre-canned DESCRIBE-shaped rows
// using an in-memory DuckDB connection, while capturing the query string.
type mockQuerier struct {
	db           *sql.DB
	capturedSQL  string
	describeRows []struct{ name, dtype string }
}

func (m *mockQuerier) Query(ctx context.Context, sqlQuery string) (*sql.Rows, error) {
	m.capturedSQL = sqlQuery
	if len(m.describeRows) == 0 {
		return m.db.QueryContext(ctx,
			"SELECT NULL::VARCHAR AS column_name, NULL::VARCHAR AS column_type WHERE false")
	}
	parts := make([]string, len(m.describeRows))
	for i, r := range m.describeRows {
		parts[i] = fmt.Sprintf(
			"SELECT '%s' AS column_name, '%s' AS column_type",
			r.name, r.dtype,
		)
	}
	return m.db.QueryContext(ctx, strings.Join(parts, " UNION ALL "))
}

func newMockQuerier(t *testing.T, rows []struct{ name, dtype string }) (*mockQuerier, func()) {
	t.Helper()
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("opening duckdb for mock: %v", err)
	}
	return &mockQuerier{db: db, describeRows: rows}, func() { db.Close() }
}

func TestScanGSheet(t *testing.T) {
	descRows := []struct{ name, dtype string }{
		{"order_id", "INTEGER"},
		{"amount", "DOUBLE"},
	}
	mq, cleanup := newMockQuerier(t, descRows)
	defer cleanup()

	dbCfg := queryengine.DatabaseConfig{
		Type:          queryengine.GSheet,
		SpreadsheetID: "sheet123",
		SheetName:     "Orders",
		Metadata: queryengine.Metadata{
			Description: "Order data",
		},
	}

	datasets, err := scanGSheet(context.Background(), mq, "orders", dbCfg)
	if err != nil {
		t.Fatalf("scanGSheet error: %v", err)
	}

	if len(datasets) != 1 {
		t.Fatalf("expected 1 dataset, got %d", len(datasets))
	}
	ds := datasets[0]
	if ds.Name != "orders.default.orders" {
		t.Errorf("dataset name = %q, want %q", ds.Name, "orders.default.orders")
	}
	if ds.Source != "orders.default.orders" {
		t.Errorf("dataset source = %q, want %q", ds.Source, "orders.default.orders")
	}
	if ds.Description != "Order data" {
		t.Errorf("dataset description = %q, want %q", ds.Description, "Order data")
	}
	if len(ds.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(ds.Fields))
	}
	if ds.Fields[0].Name != "order_id" {
		t.Errorf("fields[0].Name = %q, want %q", ds.Fields[0].Name, "order_id")
	}
	if ds.Fields[1].Name != "amount" {
		t.Errorf("fields[1].Name = %q, want %q", ds.Fields[1].Name, "amount")
	}

	// SQL should reference the configured sheet name and spreadsheet ID.
	if !strings.Contains(mq.capturedSQL, "sheet123") {
		t.Errorf("SQL missing spreadsheet ID: %s", mq.capturedSQL)
	}
	if !strings.Contains(mq.capturedSQL, "Orders") {
		t.Errorf("SQL missing sheet name: %s", mq.capturedSQL)
	}
	// Default (headers present) should not include headers=false.
	if strings.Contains(mq.capturedSQL, "headers=false") {
		t.Errorf("SQL should not contain headers=false: %s", mq.capturedSQL)
	}
}

func TestScanGSheet_SheetNameFallback(t *testing.T) {
	descRows := []struct{ name, dtype string }{
		{"col1", "VARCHAR"},
	}
	mq, cleanup := newMockQuerier(t, descRows)
	defer cleanup()

	// SheetName is empty — should fall back to the catalog key.
	dbCfg := queryengine.DatabaseConfig{
		Type:          queryengine.GSheet,
		SpreadsheetID: "spreadsheet456",
	}

	_, err := scanGSheet(context.Background(), mq, "mysheet", dbCfg)
	if err != nil {
		t.Fatalf("scanGSheet error: %v", err)
	}

	if !strings.Contains(mq.capturedSQL, "mysheet") {
		t.Errorf("SQL should use catalog key as sheet name fallback: %s", mq.capturedSQL)
	}
}

func TestScanGSheet_HeaderRowFalse(t *testing.T) {
	descRows := []struct{ name, dtype string }{
		{"column0", "VARCHAR"},
	}
	mq, cleanup := newMockQuerier(t, descRows)
	defer cleanup()

	headerRow := false
	dbCfg := queryengine.DatabaseConfig{
		Type:          queryengine.GSheet,
		SpreadsheetID: "nohdr789",
		SheetName:     "Raw",
		HeaderRow:     &headerRow,
	}

	_, err := scanGSheet(context.Background(), mq, "rawdata", dbCfg)
	if err != nil {
		t.Fatalf("scanGSheet error: %v", err)
	}

	if !strings.Contains(mq.capturedSQL, "headers=false") {
		t.Errorf("SQL should contain headers=false when HeaderRow=false: %s", mq.capturedSQL)
	}
}

func TestProvide_GSheet(t *testing.T) {
	descRows := []struct{ name, dtype string }{
		{"id", "INTEGER"},
		{"value", "VARCHAR"},
	}
	mq, cleanup := newMockQuerier(t, descRows)
	defer cleanup()

	cfg := &queryengine.Config{
		Databases: map[string]queryengine.DatabaseConfig{
			"sales": {
				Type:          queryengine.GSheet,
				SpreadsheetID: "salessheet",
				SheetName:     "Sales",
				Metadata: queryengine.Metadata{
					Description: "Sales data",
				},
			},
		},
	}

	provider := NewAutoScanProvider(cfg, mq)
	result, err := provider.Provide(context.Background(), nil)
	if err != nil {
		t.Fatalf("Provide error: %v", err)
	}

	if len(result.SemanticModel[0].Datasets) != 1 {
		t.Fatalf("expected 1 dataset, got %d", len(result.SemanticModel[0].Datasets))
	}
	if result.SemanticModel[0].Datasets[0].Name != "sales.default.sales" {
		t.Errorf("dataset name = %q, want %q",
			result.SemanticModel[0].Datasets[0].Name, "sales.default.sales")
	}
}

// infoSchemaMockQuerier implements MetadataQuerier for testing scanners that
// query information_schema.columns (4-column result: table_schema, table_name,
// column_name, data_type).
type infoSchemaMockQuerier struct {
	db          *sql.DB
	capturedSQL string
	rows        []struct{ schema, table, column, dtype string }
}

func (m *infoSchemaMockQuerier) Query(ctx context.Context, sqlQuery string) (*sql.Rows, error) {
	m.capturedSQL = sqlQuery
	if len(m.rows) == 0 {
		return m.db.QueryContext(ctx,
			"SELECT NULL::VARCHAR, NULL::VARCHAR, NULL::VARCHAR, NULL::VARCHAR WHERE false")
	}
	parts := make([]string, len(m.rows))
	for i, r := range m.rows {
		parts[i] = fmt.Sprintf(
			"SELECT '%s' AS table_schema, '%s' AS table_name, '%s' AS column_name, '%s' AS data_type",
			r.schema, r.table, r.column, r.dtype,
		)
	}
	return m.db.QueryContext(ctx, strings.Join(parts, " UNION ALL "))
}

func newInfoSchemaMockQuerier(t *testing.T, rows []struct{ schema, table, column, dtype string }) (*infoSchemaMockQuerier, func()) {
	t.Helper()
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("opening duckdb for mock: %v", err)
	}
	return &infoSchemaMockQuerier{db: db, rows: rows}, func() { db.Close() }
}

func TestScanSQLite(t *testing.T) {
	infoRows := []struct{ schema, table, column, dtype string }{
		{"main", "customers", "id", "INTEGER"},
		{"main", "customers", "name", "TEXT"},
		{"main", "customers", "created_at", "TIMESTAMP"},
		{"main", "orders", "id", "INTEGER"},
		{"main", "orders", "amount", "DOUBLE"},
	}
	mq, cleanup := newInfoSchemaMockQuerier(t, infoRows)
	defer cleanup()

	dbCfg := queryengine.DatabaseConfig{
		Type: queryengine.SQLite,
		Path: "/data/app.sqlite",
	}

	datasets, err := scanSQLite(context.Background(), mq, "shop", dbCfg)
	if err != nil {
		t.Fatalf("scanSQLite error: %v", err)
	}

	if len(datasets) != 2 {
		t.Fatalf("expected 2 datasets, got %d", len(datasets))
	}

	// First dataset: shop.main.customers
	ds := datasets[0]
	if ds.Name != "shop.main.customers" {
		t.Errorf("dataset[0] name = %q, want %q", ds.Name, "shop.main.customers")
	}
	if len(ds.Fields) != 3 {
		t.Fatalf("dataset[0] fields = %d, want 3", len(ds.Fields))
	}
	if ds.Fields[0].Name != "id" {
		t.Errorf("fields[0].Name = %q, want %q", ds.Fields[0].Name, "id")
	}
	if ds.Fields[1].Name != "name" {
		t.Errorf("fields[1].Name = %q, want %q", ds.Fields[1].Name, "name")
	}
	// created_at should have a time dimension.
	if ds.Fields[2].Dimension == nil || !ds.Fields[2].Dimension.IsTime {
		t.Errorf("expected created_at to have time dimension")
	}

	// Second dataset: shop.main.orders
	ds = datasets[1]
	if ds.Name != "shop.main.orders" {
		t.Errorf("dataset[1] name = %q, want %q", ds.Name, "shop.main.orders")
	}
	if len(ds.Fields) != 2 {
		t.Fatalf("dataset[1] fields = %d, want 2", len(ds.Fields))
	}

	// All fields should have expressions.
	for _, d := range datasets {
		for i, f := range d.Fields {
			if len(f.Expression.Dialects) == 0 {
				t.Errorf("dataset %q field[%d].Expression is empty", d.Name, i)
			}
		}
	}

	// SQL should reference the catalog.
	if !strings.Contains(mq.capturedSQL, "'shop'") {
		t.Errorf("SQL missing catalog reference: %s", mq.capturedSQL)
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

	// Check that order_date field has a time dimension.
	found := false
	for _, f := range ds.Fields {
		if f.Name == "order_date" && f.Dimension != nil && f.Dimension.IsTime {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected order_date field with time dimension, got fields: %+v", ds.Fields)
	}
}
