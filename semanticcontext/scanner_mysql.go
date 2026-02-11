package semanticcontext

import (
	"context"
	"fmt"

	"github.com/ypt/experiment-nexus/queryengine"
)

// scanMySQL queries information_schema.columns for all user tables in the
// attached MySQL catalog and returns OSI datasets.
func scanMySQL(ctx context.Context, q MetadataQuerier, catalog string, _ queryengine.DatabaseConfig) ([]Dataset, error) {
	query := fmt.Sprintf(`
		SELECT table_schema, table_name, column_name, data_type, column_comment
		FROM "%s".information_schema.columns
		WHERE table_schema NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys')
		ORDER BY table_schema, table_name, ordinal_position
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
		if err := rows.Scan(&ci.Schema, &ci.Table, &ci.Column, &ci.DataType, &ci.Comment); err != nil {
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
