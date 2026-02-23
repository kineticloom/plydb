// Copyright 2026 Paul Tzen
// SPDX-License-Identifier: Apache-2.0

package semanticcontext

// OSI-compliant semantic model structs.
// See: https://github.com/open-semantic-interchange/OSI

// SemanticModelFile is the top-level container for a semantic model YAML file.
type SemanticModelFile struct {
	SemanticModel SemanticModel `yaml:"semantic_model"`
}

// SemanticModel describes the overall data model.
type SemanticModel struct {
	Name          string         `yaml:"name"`
	Description   string         `yaml:"description,omitempty"`
	AIContext     string         `yaml:"ai_context,omitempty"`
	Datasets      []Dataset      `yaml:"datasets,omitempty"`
	Relationships []Relationship `yaml:"relationships,omitempty"`
	Metrics       []Metric       `yaml:"metrics,omitempty"`
}

// Dataset represents a single table or file in the semantic model.
type Dataset struct {
	Name        string     `yaml:"name"`
	Source      string     `yaml:"source"`
	Description string     `yaml:"description,omitempty"`
	AIContext   string     `yaml:"ai_context,omitempty"`
	PrimaryKey  []string   `yaml:"primary_key,omitempty"`
	UniqueKeys  [][]string `yaml:"unique_keys,omitempty"`
	Fields      []Field    `yaml:"fields,omitempty"`
}

// Field describes a single column.
type Field struct {
	Name        string      `yaml:"name"`
	Description string      `yaml:"description,omitempty"`
	AIContext   string      `yaml:"ai_context,omitempty"`
	Expression  *Expression `yaml:"expression,omitempty"`
	Dimension   *Dimension  `yaml:"dimension,omitempty"`
}

// Dimension describes a dimension for analytics.
type Dimension struct {
	IsTime bool `yaml:"is_time,omitempty"`
}

// Expression holds dialect-specific SQL expressions.
type Expression struct {
	Dialects []DialectExpression `yaml:"dialects"`
}

// DialectExpression holds a single dialect and its expression.
type DialectExpression struct {
	Dialect    string `yaml:"dialect"`
	Expression string `yaml:"expression"`
}

// Relationship describes a join between two datasets.
type Relationship struct {
	Name        string   `yaml:"name"`
	From        string   `yaml:"from"`
	To          string   `yaml:"to"`
	FromColumns []string `yaml:"from_columns"`
	ToColumns   []string `yaml:"to_columns"`
	AIContext   string   `yaml:"ai_context,omitempty"`
}

// Metric describes a derived measure.
type Metric struct {
	Name        string     `yaml:"name"`
	Description string     `yaml:"description,omitempty"`
	AIContext   string     `yaml:"ai_context,omitempty"`
	Expression  Expression `yaml:"expression"`
}
