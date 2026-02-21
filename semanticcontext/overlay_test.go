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
						{Name: "id", DataType: "integer"},
						{Name: "name", DataType: "varchar"},
						{Name: "created_at", DataType: "timestamp"},
					},
					Dimensions: []Dimension{
						{Name: "created_at", IsTime: true},
					},
				},
				{
					Name:   "catalog.public.orders",
					Source: "catalog.public.orders",
					Fields: []Field{
						{Name: "order_id", DataType: "integer"},
						{Name: "user_id", DataType: "integer"},
					},
				},
			},
			Metrics: []Metric{
				{
					Name:        "existing_metric",
					Description: "An existing metric",
					Expression:  Expression{SQL: "COUNT(*)"},
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
					Name:  "users_orders",
					Left:  JoinSide{Dataset: "catalog.public.users", Field: "id"},
					Right: JoinSide{Dataset: "catalog.public.orders", Field: "user_id"},
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
					Name:  "bad_relationship",
					Left:  JoinSide{Dataset: "catalog.public.users", Field: "id"},
					Right: JoinSide{Dataset: "catalog.public.nonexistent", Field: "user_id"},
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
					Expression:  Expression{SQL: "SUM(amount)"},
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
					Expression:  Expression{SQL: "COUNT(DISTINCT id)"},
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
	if result.SemanticModel.Metrics[0].Expression.SQL != "COUNT(DISTINCT id)" {
		t.Errorf("metric expression = %q, want %q", result.SemanticModel.Metrics[0].Expression.SQL, "COUNT(DISTINCT id)")
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
	overlay := &SemanticModelFile{
		SemanticModel: SemanticModel{
			Datasets: []Dataset{
				{
					Name: "catalog.public.orders",
					Dimensions: []Dimension{
						// "user_id" is an existing field — should be added as dimension.
						{Name: "user_id", IsTime: false},
						// "nonexistent" is not a field — should be ignored.
						{Name: "nonexistent", IsTime: false},
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

	// Should have exactly 1 dimension (user_id added; nonexistent ignored).
	if len(orderDS.Dimensions) != 1 {
		t.Errorf("dimension count = %d, want 1", len(orderDS.Dimensions))
	}
	if len(orderDS.Dimensions) > 0 && orderDS.Dimensions[0].Name != "user_id" {
		t.Errorf("dimension name = %q, want %q", orderDS.Dimensions[0].Name, "user_id")
	}
}
