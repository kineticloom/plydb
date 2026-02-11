package semanticcontext

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/ypt/experiment-nexus/queryengine"
)

// AutoScanProvider scans all configured data sources and produces a semantic model.
type AutoScanProvider struct {
	cfg     *queryengine.Config
	querier MetadataQuerier
}

// NewAutoScanProvider creates a new AutoScanProvider.
func NewAutoScanProvider(cfg *queryengine.Config, querier MetadataQuerier) *AutoScanProvider {
	return &AutoScanProvider{cfg: cfg, querier: querier}
}

// Provide scans all configured databases and returns a SemanticModelFile.
// If existing is non-nil, it is used as the base; otherwise a new one is created.
func (p *AutoScanProvider) Provide(ctx context.Context, existing *SemanticModelFile) (*SemanticModelFile, error) {
	result := &SemanticModelFile{
		SemanticModel: SemanticModel{
			Name: "Auto-scanned Semantic Model",
		},
	}
	if existing != nil {
		result = existing
	}

	// Iterate databases in sorted key order for deterministic output.
	keys := make([]string, 0, len(p.cfg.Databases))
	for k := range p.cfg.Databases {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		dbCfg := p.cfg.Databases[key]
		var datasets []Dataset
		var err error

		switch dbCfg.Type {
		case queryengine.PostgreSQL:
			datasets, err = scanPostgres(ctx, p.querier, key, dbCfg)
		case queryengine.MySQL:
			datasets, err = scanMySQL(ctx, p.querier, key, dbCfg)
		case queryengine.File:
			datasets, err = scanFile(ctx, p.querier, key, dbCfg)
		case queryengine.S3:
			datasets, err = scanS3(ctx, p.querier, key, dbCfg)
		default:
			log.Printf("warning: unsupported database type %q for %q, skipping", dbCfg.Type, key)
			continue
		}

		if err != nil {
			return nil, fmt.Errorf("scanning %q: %w", key, err)
		}
		result.SemanticModel.Datasets = append(result.SemanticModel.Datasets, datasets...)
	}

	return result, nil
}

// columnInfo is an intermediate struct for metadata harvested from any source.
type columnInfo struct {
	Schema   string
	Table    string
	Column   string
	DataType string
	Comment  string
}

// columnsToDataset groups columns into an OSI Dataset.
func columnsToDataset(catalog, schema, table string, cols []columnInfo, description string) Dataset {
	ds := Dataset{
		Name:        fmt.Sprintf("%s.%s.%s", catalog, schema, table),
		Description: description,
		Source:      fmt.Sprintf("%s.%s.%s", catalog, schema, table),
	}

	for _, c := range cols {
		f := Field{
			Name:     c.Column,
			DataType: c.DataType,
		}
		if c.Comment != "" {
			f.Description = c.Comment
		}
		ds.Fields = append(ds.Fields, f)

		if isTimeLike(c.DataType) {
			ds.Dimensions = append(ds.Dimensions, Dimension{
				Name:   c.Column,
				IsTime: true,
			})
		}
	}

	return ds
}

// isTimeLike returns true if the data type represents a date, time, or timestamp.
func isTimeLike(dataType string) bool {
	dt := strings.ToLower(dataType)
	for _, keyword := range []string{"date", "time", "timestamp", "datetime", "interval"} {
		if strings.Contains(dt, keyword) {
			return true
		}
	}
	return false
}
