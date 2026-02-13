package semanticcontext

import (
	"context"
	"fmt"
	"log"

	"github.com/kineticloom/plydb/queryengine"
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

	// Best-effort: fetch table-level comments.
	tableComments := fetchMySQLTableComments(ctx, q, catalog)

	datasets := make([]Dataset, 0, len(ordered))
	for _, tk := range ordered {
		tableDesc := tableComments[fmt.Sprintf("%s.%s", tk.schema, tk.table)]
		ds := columnsToDataset(catalog, tk.schema, tk.table, grouped[tk], tableDesc)
		datasets = append(datasets, ds)
	}
	return datasets, nil
}

// fetchMySQLTableComments queries information_schema.tables for table-level
// comments. Returns a map of "schema.table" -> comment.
func fetchMySQLTableComments(ctx context.Context, q MetadataQuerier, catalog string) map[string]string {
	query := fmt.Sprintf(`
		SELECT table_schema, table_name, table_comment
		FROM "%s".information_schema.tables
		WHERE table_schema NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys')
		  AND table_comment != ''
	`, catalog)

	rows, err := q.Query(ctx, query)
	if err != nil {
		log.Printf("warning: could not fetch table comments for %q: %v", catalog, err)
		return make(map[string]string)
	}
	defer rows.Close()

	comments := make(map[string]string)
	for rows.Next() {
		var schema, table, comment string
		if err := rows.Scan(&schema, &table, &comment); err != nil {
			log.Printf("warning: error scanning table comment row for %q: %v", catalog, err)
			return comments
		}
		comments[fmt.Sprintf("%s.%s", schema, table)] = comment
	}
	return comments
}
