package semanticcontext

import (
	"context"
	"fmt"

	"github.com/ypt/experiment-nexus/queryengine"
)

// scanS3 uses DESCRIBE SELECT to discover columns from an S3 URI
// and returns an OSI dataset.
func scanS3(ctx context.Context, q MetadataQuerier, catalog string, dbCfg queryengine.DatabaseConfig) ([]Dataset, error) {
	describeSQL := fmt.Sprintf(`DESCRIBE SELECT * FROM '%s'`, dbCfg.URI)

	rows, err := q.Query(ctx, describeSQL)
	if err != nil {
		return nil, fmt.Errorf("describing S3 URI %q: %w", dbCfg.URI, err)
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
