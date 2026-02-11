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
	Datasets      []Dataset      `yaml:"datasets,omitempty"`
	Relationships []Relationship `yaml:"relationships,omitempty"`
	Metrics       []Metric       `yaml:"metrics,omitempty"`
}

// Dataset represents a single table or file in the semantic model.
type Dataset struct {
	Name        string      `yaml:"name"`
	Description string      `yaml:"description,omitempty"`
	Source      string      `yaml:"source"`
	Fields      []Field     `yaml:"fields,omitempty"`
	Dimensions  []Dimension `yaml:"dimensions,omitempty"`
}

// Field describes a single column.
type Field struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	DataType    string `yaml:"data_type"`
	Nullable    *bool  `yaml:"nullable,omitempty"`
}

// Dimension describes a dimension for analytics.
type Dimension struct {
	Name   string `yaml:"name"`
	IsTime bool   `yaml:"is_time,omitempty"`
}

// Expression holds a dialect-specific SQL expression.
type Expression struct {
	Dialect Dialect `yaml:"dialect,omitempty"`
	SQL     string  `yaml:"sql"`
}

// Dialect identifies the SQL dialect for an expression.
type Dialect struct {
	Name string `yaml:"name"`
}

// Relationship describes a join between two datasets.
type Relationship struct {
	Name       string     `yaml:"name"`
	Left       JoinSide   `yaml:"left"`
	Right      JoinSide   `yaml:"right"`
	Expression Expression `yaml:"expression,omitempty"`
}

// JoinSide identifies one side of a relationship.
type JoinSide struct {
	Dataset string `yaml:"dataset"`
	Field   string `yaml:"field"`
}

// Metric describes a derived measure.
type Metric struct {
	Name        string     `yaml:"name"`
	Description string     `yaml:"description,omitempty"`
	Expression  Expression `yaml:"expression"`
}
