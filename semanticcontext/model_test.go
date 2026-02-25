// Copyright 2026 Paul Tzen
// SPDX-License-Identifier: Apache-2.0

package semanticcontext

import (
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestAIContext_UnmarshalString(t *testing.T) {
	input := `ai_context: "some text"`
	var dst struct {
		AIContext AIContext `yaml:"ai_context"`
	}
	if err := yaml.Unmarshal([]byte(input), &dst); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if dst.AIContext.String != "some text" {
		t.Errorf("String = %q, want %q", dst.AIContext.String, "some text")
	}
	if dst.AIContext.Object != nil {
		t.Error("Object should be nil for string-form ai_context")
	}
}

func TestAIContext_UnmarshalObject(t *testing.T) {
	input := `ai_context:
  instructions: Use for sales analysis
  synonyms:
    - orders
    - purchases
  examples:
    - How many orders were placed?
`
	var dst struct {
		AIContext AIContext `yaml:"ai_context"`
	}
	if err := yaml.Unmarshal([]byte(input), &dst); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if dst.AIContext.Object == nil {
		t.Fatal("Object should be non-nil for object-form ai_context")
	}
	obj := dst.AIContext.Object
	if obj.Instructions != "Use for sales analysis" {
		t.Errorf("Instructions = %q, want %q", obj.Instructions, "Use for sales analysis")
	}
	if len(obj.Synonyms) != 2 || obj.Synonyms[0] != "orders" || obj.Synonyms[1] != "purchases" {
		t.Errorf("Synonyms = %v, want [orders purchases]", obj.Synonyms)
	}
	if len(obj.Examples) != 1 || obj.Examples[0] != "How many orders were placed?" {
		t.Errorf("Examples = %v, want [How many orders were placed?]", obj.Examples)
	}
	if dst.AIContext.String != "" {
		t.Error("String should be empty for object-form ai_context")
	}
}

func TestAIContext_MarshalObject(t *testing.T) {
	model := SemanticModelFile{
		SemanticModel: SemanticModel{
			Name: "m",
			AIContext: AIContext{
				Object: &AIContextObject{
					Instructions: "Use for sales",
					Synonyms:     []string{"orders"},
				},
			},
		},
	}
	data, err := yaml.Marshal(&model)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	out := string(data)
	for _, want := range []string{"instructions: Use for sales", "synonyms:", "- orders"} {
		if !strings.Contains(out, want) {
			t.Errorf("YAML output missing %q\nGot:\n%s", want, out)
		}
	}
}

func TestAIContext_ZeroOmittedFromYAML(t *testing.T) {
	model := SemanticModelFile{
		SemanticModel: SemanticModel{
			Name: "m",
		},
	}
	data, err := yaml.Marshal(&model)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if strings.Contains(string(data), "ai_context") {
		t.Errorf("zero AIContext should be omitted from YAML\nGot:\n%s", string(data))
	}
}

func TestSemanticModelFile_YAMLRoundTrip(t *testing.T) {
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
			},
			Relationships: []Relationship{
				{
					Name:        "user_orders",
					From:        "catalog.schema.users",
					To:          "catalog.schema.orders",
					FromColumns: []string{"id"},
					ToColumns:   []string{"user_id"},
				},
			},
			Metrics: []Metric{
				{
					Name:        "total_revenue",
					Description: "Sum of all order amounts",
					Expression: Expression{
						Dialects: []DialectExpression{{Dialect: "ANSI_SQL", Expression: "SUM(orders.amount)"}},
					},
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
		"is_time: true",
		"relationships:",
		"from: catalog.schema.users",
		"to: catalog.schema.orders",
		"from_columns:",
		"to_columns:",
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
	// created_at should have dimension preserved.
	if ds.Fields[2].Dimension == nil || !ds.Fields[2].Dimension.IsTime {
		t.Errorf("round-trip dimension not preserved on created_at")
	}
}
