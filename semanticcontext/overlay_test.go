package semanticcontext

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// writeOverlay writes YAML content to a temp file and returns the path.
func writeOverlay(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "overlay.yaml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("writing overlay: %v", err)
	}
	return path
}

// baseModel returns a SemanticModelFile for use as a base in tests.
func baseModel() *SemanticModelFile {
	return &SemanticModelFile{
		SemanticModel: SemanticModel{
			Name:        "Test Model",
			Description: "Original description",
			Datasets: []Dataset{
				{
					Name:        "catalog.public.users",
					Description: "User table",
					Source:      "catalog.public.users",
					Fields: []Field{
						{
							Name: "id",
							Expression: &Expression{
								Dialects: []DialectExpression{{Dialect: "ANSI_SQL", Expression: "id"}},
							},
						},
						{
							Name: "name",
							Expression: &Expression{
								Dialects: []DialectExpression{{Dialect: "ANSI_SQL", Expression: "name"}},
							},
						},
						{
							Name: "created_at",
							Expression: &Expression{
								Dialects: []DialectExpression{{Dialect: "ANSI_SQL", Expression: "created_at"}},
							},
							Dimension: &Dimension{IsTime: true},
						},
					},
				},
				{
					Name:   "catalog.public.orders",
					Source: "catalog.public.orders",
					Fields: []Field{
						{
							Name: "order_id",
							Expression: &Expression{
								Dialects: []DialectExpression{{Dialect: "ANSI_SQL", Expression: "order_id"}},
							},
						},
						{
							Name: "user_id",
							Expression: &Expression{
								Dialects: []DialectExpression{{Dialect: "ANSI_SQL", Expression: "user_id"}},
							},
						},
					},
				},
			},
			Metrics: []Metric{
				{
					Name:        "existing_metric",
					Description: "An existing metric",
					Expression: Expression{
						Dialects: []DialectExpression{{Dialect: "ANSI_SQL", Expression: "COUNT(*)"}},
					},
				},
			},
		},
	}
}

func TestApplyOverlay_ModelDescription(t *testing.T) {
	base := baseModel()
	overlay := &SemanticModelFile{
		SemanticModel: SemanticModel{
			Description: "Updated model description",
		},
	}
	result := applyOverlay(base, overlay)
	if result.SemanticModel.Description != "Updated model description" {
		t.Errorf("model description = %q, want %q", result.SemanticModel.Description, "Updated model description")
	}
}

func TestApplyOverlay_DatasetDescription(t *testing.T) {
	base := baseModel()
	overlay := &SemanticModelFile{
		SemanticModel: SemanticModel{
			Datasets: []Dataset{
				{
					Name:        "catalog.public.users",
					Description: "Updated user description",
				},
			},
		},
	}
	result := applyOverlay(base, overlay)
	got := result.SemanticModel.Datasets[0].Description
	if got != "Updated user description" {
		t.Errorf("dataset description = %q, want %q", got, "Updated user description")
	}
}

func TestApplyOverlay_FieldDescription(t *testing.T) {
	base := baseModel()
	overlay := &SemanticModelFile{
		SemanticModel: SemanticModel{
			Datasets: []Dataset{
				{
					Name: "catalog.public.users",
					Fields: []Field{
						{Name: "name", Description: "The user's full name"},
					},
				},
			},
		},
	}
	result := applyOverlay(base, overlay)
	ds := result.SemanticModel.Datasets[0]
	var nameDesc string
	for _, f := range ds.Fields {
		if f.Name == "name" {
			nameDesc = f.Description
			break
		}
	}
	if nameDesc != "The user's full name" {
		t.Errorf("field description = %q, want %q", nameDesc, "The user's full name")
	}
}

func TestApplyOverlay_NewDatasetIgnored(t *testing.T) {
	base := baseModel()
	overlay := &SemanticModelFile{
		SemanticModel: SemanticModel{
			Datasets: []Dataset{
				{Name: "catalog.public.new_table", Description: "Should be ignored"},
			},
		},
	}
	result := applyOverlay(base, overlay)
	if len(result.SemanticModel.Datasets) != 2 {
		t.Errorf("dataset count = %d, want 2 (new dataset should be ignored)", len(result.SemanticModel.Datasets))
	}
	for _, ds := range result.SemanticModel.Datasets {
		if ds.Name == "catalog.public.new_table" {
			t.Error("new dataset should not appear in result")
		}
	}
}

func TestApplyOverlay_NewFieldIgnored(t *testing.T) {
	base := baseModel()
	overlay := &SemanticModelFile{
		SemanticModel: SemanticModel{
			Datasets: []Dataset{
				{
					Name: "catalog.public.users",
					Fields: []Field{
						{Name: "nonexistent_field", Description: "Should be ignored"},
					},
				},
			},
		},
	}
	result := applyOverlay(base, overlay)
	ds := result.SemanticModel.Datasets[0]
	if len(ds.Fields) != 3 {
		t.Errorf("field count = %d, want 3 (new field should be ignored)", len(ds.Fields))
	}
	for _, f := range ds.Fields {
		if f.Name == "nonexistent_field" {
			t.Error("nonexistent field should not appear in result")
		}
	}
}

func TestApplyOverlay_RelationshipAdded(t *testing.T) {
	base := baseModel()
	overlay := &SemanticModelFile{
		SemanticModel: SemanticModel{
			Relationships: []Relationship{
				{
					Name:        "users_orders",
					From:        "catalog.public.users",
					To:          "catalog.public.orders",
					FromColumns: []string{"id"},
					ToColumns:   []string{"user_id"},
				},
			},
		},
	}
	result := applyOverlay(base, overlay)
	if len(result.SemanticModel.Relationships) != 1 {
		t.Errorf("relationship count = %d, want 1", len(result.SemanticModel.Relationships))
	}
	if result.SemanticModel.Relationships[0].Name != "users_orders" {
		t.Errorf("relationship name = %q, want %q", result.SemanticModel.Relationships[0].Name, "users_orders")
	}
}

func TestApplyOverlay_RelationshipIgnoredWhenSideMissing(t *testing.T) {
	base := baseModel()
	overlay := &SemanticModelFile{
		SemanticModel: SemanticModel{
			Relationships: []Relationship{
				{
					Name:        "bad_relationship",
					From:        "catalog.public.users",
					To:          "catalog.public.nonexistent",
					FromColumns: []string{"id"},
					ToColumns:   []string{"user_id"},
				},
			},
		},
	}
	result := applyOverlay(base, overlay)
	if len(result.SemanticModel.Relationships) != 0 {
		t.Errorf("relationship count = %d, want 0 (bad relationship should be ignored)", len(result.SemanticModel.Relationships))
	}
}

func TestApplyOverlay_MetricAdded(t *testing.T) {
	base := baseModel()
	overlay := &SemanticModelFile{
		SemanticModel: SemanticModel{
			Metrics: []Metric{
				{
					Name:        "new_metric",
					Description: "A new metric",
					Expression: Expression{
						Dialects: []DialectExpression{{Dialect: "ANSI_SQL", Expression: "SUM(amount)"}},
					},
				},
			},
		},
	}
	result := applyOverlay(base, overlay)
	if len(result.SemanticModel.Metrics) != 2 {
		t.Errorf("metric count = %d, want 2", len(result.SemanticModel.Metrics))
	}
}

func TestApplyOverlay_MetricUpdated(t *testing.T) {
	base := baseModel()
	overlay := &SemanticModelFile{
		SemanticModel: SemanticModel{
			Metrics: []Metric{
				{
					Name:        "existing_metric",
					Description: "Updated description",
					Expression: Expression{
						Dialects: []DialectExpression{{Dialect: "ANSI_SQL", Expression: "COUNT(DISTINCT id)"}},
					},
				},
			},
		},
	}
	result := applyOverlay(base, overlay)
	if len(result.SemanticModel.Metrics) != 1 {
		t.Errorf("metric count = %d, want 1", len(result.SemanticModel.Metrics))
	}
	if result.SemanticModel.Metrics[0].Description != "Updated description" {
		t.Errorf("metric description = %q, want %q", result.SemanticModel.Metrics[0].Description, "Updated description")
	}
	got := result.SemanticModel.Metrics[0].Expression.Dialects[0].Expression
	if got != "COUNT(DISTINCT id)" {
		t.Errorf("metric expression = %q, want %q", got, "COUNT(DISTINCT id)")
	}
}

func TestOverlayProvider_MultipleFiles(t *testing.T) {
	overlay1 := writeOverlay(t, `semantic_model:
  name: overlay1
  datasets:
    - name: catalog.public.users
      description: From overlay 1
`)
	overlay2 := writeOverlay(t, `semantic_model:
  name: overlay2
  datasets:
    - name: catalog.public.users
      description: From overlay 2
`)
	base := baseModel()
	provider := NewOverlayProvider([]string{overlay1, overlay2})
	result, err := provider.Provide(context.Background(), base)
	if err != nil {
		t.Fatalf("Provide error: %v", err)
	}
	// overlay2 should win (applied last).
	got := result.SemanticModel.Datasets[0].Description
	if got != "From overlay 2" {
		t.Errorf("dataset description = %q, want %q", got, "From overlay 2")
	}
}

func TestApplyOverlay_DimensionFromOverlay(t *testing.T) {
	base := baseModel()
	// Apply an overlay that sets a dimension on the "user_id" field of orders.
	overlay := &SemanticModelFile{
		SemanticModel: SemanticModel{
			Datasets: []Dataset{
				{
					Name: "catalog.public.orders",
					Fields: []Field{
						{Name: "user_id", Dimension: &Dimension{IsTime: false}},
						// "nonexistent" is not a field — should be ignored.
						{Name: "nonexistent", Dimension: &Dimension{IsTime: false}},
					},
				},
			},
		},
	}
	result := applyOverlay(base, overlay)

	var orderDS Dataset
	for _, ds := range result.SemanticModel.Datasets {
		if ds.Name == "catalog.public.orders" {
			orderDS = ds
			break
		}
	}

	// user_id field should have a dimension; nonexistent field should be ignored.
	var userIDField *Field
	for _, f := range orderDS.Fields {
		if f.Name == "user_id" {
			userIDField = &f
			break
		}
	}
	if userIDField == nil {
		t.Fatal("user_id field not found")
	}
	if userIDField.Dimension == nil {
		t.Error("user_id field should have a dimension set by overlay")
	}

	// No field named "nonexistent" should exist.
	for _, f := range orderDS.Fields {
		if f.Name == "nonexistent" {
			t.Error("nonexistent field should not appear in result")
		}
	}
}
