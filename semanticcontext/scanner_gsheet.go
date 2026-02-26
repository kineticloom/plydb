// Copyright 2026 Paul Tzen
// SPDX-License-Identifier: Apache-2.0

package semanticcontext

import (
	"context"
	"fmt"

	"github.com/kineticloom/plydb/queryengine"
)

// scanGSheet uses DESCRIBE SELECT to discover columns from a Google Sheet
// and returns an OSI dataset.
func scanGSheet(ctx context.Context, q MetadataQuerier, catalog string, dbCfg queryengine.DatabaseConfig) ([]Dataset, error) {
	sheetName := dbCfg.SheetName
	if sheetName == "" {
		sheetName = catalog // mirror preprocess.go's ref.Name fallback
	}

	var describeSQL string
	if dbCfg.HeaderRow != nil && !*dbCfg.HeaderRow {
		describeSQL = fmt.Sprintf(
			`DESCRIBE SELECT * FROM read_gsheet('%s', sheet='%s', headers=false)`,
			dbCfg.SpreadsheetID, sheetName,
		)
	} else {
		describeSQL = fmt.Sprintf(
			`DESCRIBE SELECT * FROM read_gsheet('%s', sheet='%s')`,
			dbCfg.SpreadsheetID, sheetName,
		)
	}

	rows, err := q.Query(ctx, describeSQL)
	if err != nil {
		return nil, fmt.Errorf("describing GSheet %q: %w", dbCfg.SpreadsheetID, err)
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
