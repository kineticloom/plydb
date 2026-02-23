// Copyright 2026 Paul Tzen
// SPDX-License-Identifier: Apache-2.0

package semanticcontext

import (
	"context"
	"database/sql"
)

// MetadataQuerier abstracts SQL query execution.
// *queryengine.QueryEngine satisfies this via its existing Query method.
type MetadataQuerier interface {
	Query(ctx context.Context, sql string) (*sql.Rows, error)
}

// Provider produces or enriches a semantic model.
// Designed for future composability (user overrides, LLM enrichment, etc.)
type Provider interface {
	Provide(ctx context.Context, existing *SemanticModelFile) (*SemanticModelFile, error)
}
