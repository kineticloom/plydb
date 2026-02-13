package semanticcontext

import (
	"context"
	"fmt"
	"log"

	"github.com/kineticloom/plydb/queryengine"
)

// scanPostgres queries information_schema.columns for all user tables in the
// attached PostgreSQL catalog and returns OSI datasets.
func scanPostgres(ctx context.Context, q MetadataQuerier, catalog string, _ queryengine.DatabaseConfig) ([]Dataset, error) {
	query := fmt.Sprintf(`
		SELECT table_schema, table_name, column_name, data_type
		FROM "%s".information_schema.columns
		WHERE table_schema NOT IN ('information_schema', 'pg_catalog')
		ORDER BY table_schema, table_name, ordinal_position
	`, catalog)

	rows, err := q.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying columns: %w", err)
	}
	defer rows.Close()

	// Collect columns grouped by schema.table.
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

	// Best-effort: fetch table and column comments from pg_catalog.
	comments := fetchPgColumnComments(ctx, q, catalog)
	tableComments := fetchPgTableComments(ctx, q, catalog)

	// Apply comments to column info.
	for tk, cols := range grouped {
		for i := range cols {
			key := fmt.Sprintf("%s.%s.%s", tk.schema, tk.table, cols[i].Column)
			if c, ok := comments[key]; ok {
				cols[i].Comment = c
			}
		}
	}

	datasets := make([]Dataset, 0, len(ordered))
	for _, tk := range ordered {
		tableDesc := tableComments[fmt.Sprintf("%s.%s", tk.schema, tk.table)]
		ds := columnsToDataset(catalog, tk.schema, tk.table, grouped[tk], tableDesc)
		datasets = append(datasets, ds)
	}
	return datasets, nil
}

// fetchPgTableComments attempts to read table-level comments from pg_catalog.
// Returns a map of "schema.table" -> comment. Logs a warning and returns an
// empty map if the query fails.
func fetchPgTableComments(ctx context.Context, q MetadataQuerier, catalog string) map[string]string {
	query := fmt.Sprintf(`
		SELECT
			n.nspname AS schema_name,
			c.relname AS table_name,
			d.description
		FROM "%s".pg_catalog.pg_description d
		JOIN "%s".pg_catalog.pg_class c ON d.objoid = c.oid
		JOIN "%s".pg_catalog.pg_namespace n ON c.relnamespace = n.oid
		WHERE d.objsubid = 0
		  AND c.relkind IN ('r', 'v', 'm', 'f', 'p')
	`, catalog, catalog, catalog)

	rows, err := q.Query(ctx, query)
	if err != nil {
		log.Printf("warning: could not fetch table comments for %q: %v", catalog, err)
		return make(map[string]string)
	}
	defer rows.Close()

	comments := make(map[string]string)
	for rows.Next() {
		var schema, table, desc string
		if err := rows.Scan(&schema, &table, &desc); err != nil {
			log.Printf("warning: error scanning table comment row for %q: %v", catalog, err)
			return comments
		}
		comments[fmt.Sprintf("%s.%s", schema, table)] = desc
	}
	return comments
}

// fetchPgColumnComments attempts to read table/column comments from pg_catalog.
// Returns a map of "schema.table.column" -> comment. Logs a warning and
// returns an empty map if the query fails (e.g. DuckDB Postgres scanner
// doesn't support pg_description).
func fetchPgColumnComments(ctx context.Context, q MetadataQuerier, catalog string) map[string]string {
	query := fmt.Sprintf(`
		SELECT
			n.nspname AS schema_name,
			c.relname AS table_name,
			a.attname AS column_name,
			d.description
		FROM "%s".pg_catalog.pg_description d
		JOIN "%s".pg_catalog.pg_class c ON d.objoid = c.oid
		JOIN "%s".pg_catalog.pg_namespace n ON c.relnamespace = n.oid
		JOIN "%s".pg_catalog.pg_attribute a ON a.attrelid = c.oid AND a.attnum = d.objsubid
		WHERE d.objsubid > 0
	`, catalog, catalog, catalog, catalog)

	rows, err := q.Query(ctx, query)
	if err != nil {
		log.Printf("warning: could not fetch pg_description for %q (may not be supported): %v", catalog, err)
		return make(map[string]string)
	}
	defer rows.Close()

	comments := make(map[string]string)
	for rows.Next() {
		var schema, table, column, desc string
		if err := rows.Scan(&schema, &table, &column, &desc); err != nil {
			log.Printf("warning: error scanning pg_description row for %q: %v", catalog, err)
			return comments
		}
		comments[fmt.Sprintf("%s.%s.%s", schema, table, column)] = desc
	}
	return comments
}
