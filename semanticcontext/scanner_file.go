package semanticcontext

import (
	"context"
	"fmt"
	"strings"

	"github.com/kineticloom/plydb/queryengine"
)

// scanFile uses DESCRIBE SELECT to discover columns from a local file
// (CSV, Parquet, JSON, XLSX) and returns an OSI dataset.
func scanFile(ctx context.Context, q MetadataQuerier, catalog string, dbCfg queryengine.DatabaseConfig) ([]Dataset, error) {
	var describeSQL string
	if strings.HasSuffix(strings.ToLower(dbCfg.Path), ".xlsx") {
		sheet := dbCfg.SheetName
		if sheet == "" {
			sheet = "Sheet1"
		}
		describeSQL = fmt.Sprintf(`DESCRIBE SELECT * FROM read_xlsx('%s', sheet='%s')`, dbCfg.Path, sheet)
	} else {
		describeSQL = fmt.Sprintf(`DESCRIBE SELECT * FROM '%s'`, dbCfg.Path)
	}

	rows, err := q.Query(ctx, describeSQL)
	if err != nil {
		return nil, fmt.Errorf("describing file %q: %w", dbCfg.Path, err)
	}
	defer rows.Close()

	cols, err := scanDescribeRows(rows)
	if err != nil {
		return nil, err
	}

	description := dbCfg.Metadata.Description
	ds := columnsToDataset(catalog, "default", catalog, cols, description)
	return []Dataset{ds}, nil
}

// scanDescribeRows reads the output of a DESCRIBE SELECT statement and
// returns columnInfo entries. DuckDB DESCRIBE returns columns:
// column_name, column_type, null, key, default, extra.
func scanDescribeRows(rows interface {
	Next() bool
	Scan(dest ...any) error
	Columns() ([]string, error)
	Err() error
}) ([]columnInfo, error) {
	colNames, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("reading describe columns: %w", err)
	}

	var result []columnInfo
	for rows.Next() {
		// Make a slice of interface{} pointers for all columns.
		vals := make([]any, len(colNames))
		ptrs := make([]any, len(colNames))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("scanning describe row: %w", err)
		}

		// First two columns are always column_name and column_type.
		ci := columnInfo{
			Column:   fmt.Sprintf("%v", vals[0]),
			DataType: fmt.Sprintf("%v", vals[1]),
		}
		result = append(result, ci)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating describe rows: %w", err)
	}
	return result, nil
}
