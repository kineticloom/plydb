// Copyright 2026 Paul Tzen
// SPDX-License-Identifier: Apache-2.0

package semanticcontext

import (
	"context"
	"fmt"

	"github.com/kineticloom/plydb/queryengine"
)

// scanDuckDB queries duckdb_columns() for all user tables in the attached
// DuckDB catalog and returns OSI datasets. Like SQLite attachments, DuckDB
// attachments do not expose information_schema, so we use DuckDB's built-in
// metadata function instead.
func scanDuckDB(ctx context.Context, q MetadataQuerier, catalog string, _ queryengine.DatabaseConfig) ([]Dataset, error) {
	query := fmt.Sprintf(`
		SELECT schema_name, table_name, column_name, data_type
		FROM duckdb_columns()
		WHERE database_name = '%s'
		ORDER BY schema_name, table_name, column_index
	`, catalog)

	rows, err := q.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying columns: %w", err)
	}
	defer rows.Close()

	type tableKey struct{ schema, table string }
	ordered := make([]tableKey, 0)
	grouped := make(map[tableKey][]columnInfo)

	for rows.Next() {
		var ci columnInfo
		if err := rows.Scan(&ci.Schema, &ci.Table, &ci.Column, &ci.DataType); err != nil {
			return nil, fmt.Errorf("scanning column row: %w", err)
		}
		tk := tableKey{ci.Schema, ci.Table}
		if _, exists := grouped[tk]; !exists {
			ordered = append(ordered, tk)
		}
		grouped[tk] = append(grouped[tk], ci)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating columns: %w", err)
	}

	datasets := make([]Dataset, 0, len(ordered))
	for _, tk := range ordered {
		ds := columnsToDataset(catalog, tk.schema, tk.table, grouped[tk], "")
		datasets = append(datasets, ds)
	}
	return datasets, nil
}
