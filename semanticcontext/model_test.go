package semanticcontext

import (
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestSemanticModelFile_YAMLRoundTrip(t *testing.T) {
	boolPtr := func(b bool) *bool { return &b }

	model := SemanticModelFile{
		SemanticModel: SemanticModel{
			Name:        "Test Model",
			Description: "A test semantic model",
			Datasets: []Dataset{
				{
					Name:        "catalog.schema.users",
					Description: "User accounts",
					Source:      "catalog.schema.users",
					Fields: []Field{
						{Name: "id", DataType: "INTEGER", Nullable: boolPtr(false)},
						{Name: "name", DataType: "VARCHAR"},
						{Name: "created_at", DataType: "TIMESTAMP"},
					},
					Dimensions: []Dimension{
						{Name: "created_at", IsTime: true},
					},
				},
			},
			Relationships: []Relationship{
				{
					Name:  "user_orders",
					Left:  JoinSide{Dataset: "catalog.schema.users", Field: "id"},
					Right: JoinSide{Dataset: "catalog.schema.orders", Field: "user_id"},
				},
			},
			Metrics: []Metric{
				{
					Name:        "total_revenue",
					Description: "Sum of all order amounts",
					Expression:  Expression{SQL: "SUM(orders.amount)"},
				},
			},
		},
	}

	data, err := yaml.Marshal(&model)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	output := string(data)

	// Verify key fields are present in the YAML output.
	for _, want := range []string{
		"semantic_model:",
		"name: Test Model",
		"description: A test semantic model",
		"datasets:",
		"source: catalog.schema.users",
		"data_type: INTEGER",
		"nullable: false",
		"is_time: true",
		"relationships:",
		"dataset: catalog.schema.users",
		"metrics:",
		"total_revenue",
		"SUM(orders.amount)",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("YAML output missing %q\nGot:\n%s", want, output)
		}
	}

	// Verify round-trip: unmarshal back and check key fields.
	var decoded SemanticModelFile
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.SemanticModel.Name != "Test Model" {
		t.Errorf("round-trip name = %q, want %q", decoded.SemanticModel.Name, "Test Model")
	}
	if len(decoded.SemanticModel.Datasets) != 1 {
		t.Fatalf("round-trip datasets count = %d, want 1", len(decoded.SemanticModel.Datasets))
	}
	ds := decoded.SemanticModel.Datasets[0]
	if len(ds.Fields) != 3 {
		t.Errorf("round-trip fields count = %d, want 3", len(ds.Fields))
	}
	if ds.Fields[0].Nullable == nil || *ds.Fields[0].Nullable != false {
		t.Errorf("round-trip nullable not preserved")
	}
}
