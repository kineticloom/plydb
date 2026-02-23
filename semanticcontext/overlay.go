// Copyright 2026 Paul Tzen
// SPDX-License-Identifier: Apache-2.0

package semanticcontext

import (
	"context"
	"fmt"
	"os"

	"go.yaml.in/yaml/v4"
)

// OverlayProvider applies one or more OSI YAML overlay files on top of an
// existing semantic model. Overlays can enrich descriptions, add relationships,
// and add/update metrics, but cannot add new datasets or fields.
type OverlayProvider struct {
	filePaths []string
}

// NewOverlayProvider creates an OverlayProvider that applies the given YAML
// overlay files in order.
func NewOverlayProvider(filePaths []string) *OverlayProvider {
	return &OverlayProvider{filePaths: filePaths}
}

// Provide applies each overlay file in order onto existing and returns the
// enriched model. If existing is nil, a new empty model is used as the base.
func (p *OverlayProvider) Provide(ctx context.Context, existing *SemanticModelFile) (*SemanticModelFile, error) {
	result := existing
	if result == nil {
		result = &SemanticModelFile{}
	}

	for _, path := range p.filePaths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading overlay file %q: %w", path, err)
		}

		var overlay SemanticModelFile
		if err := yaml.Unmarshal(data, &overlay); err != nil {
			return nil, fmt.Errorf("parsing overlay file %q: %w", path, err)
		}

		result = applyOverlay(result, &overlay)
	}

	return result, nil
}

// applyOverlay merges overlay into base and returns the enriched model.
// It does not modify base or overlay in place.
func applyOverlay(base, overlay *SemanticModelFile) *SemanticModelFile {
	result := *base

	// 1. Model-level description.
	if overlay.SemanticModel.Description != "" {
		result.SemanticModel.Description = overlay.SemanticModel.Description
	}

	// Build an index of base datasets by name for O(1) lookup.
	datasetIdx := make(map[string]int, len(base.SemanticModel.Datasets))
	for i, ds := range base.SemanticModel.Datasets {
		datasetIdx[ds.Name] = i
	}

	// Copy the datasets slice so we can mutate individual entries safely.
	datasets := make([]Dataset, len(base.SemanticModel.Datasets))
	copy(datasets, base.SemanticModel.Datasets)

	// 2. Enrich datasets that exist in base.
	for _, overlayDS := range overlay.SemanticModel.Datasets {
		idx, ok := datasetIdx[overlayDS.Name]
		if !ok {
			// Overlay dataset not in base — ignore.
			continue
		}

		ds := datasets[idx]

		if overlayDS.Description != "" {
			ds.Description = overlayDS.Description
		}

		// Index base fields by name.
		fieldIdx := make(map[string]int, len(ds.Fields))
		for i, f := range ds.Fields {
			fieldIdx[f.Name] = i
		}

		// Copy fields so we can mutate.
		fields := make([]Field, len(ds.Fields))
		copy(fields, ds.Fields)

		for _, overlayField := range overlayDS.Fields {
			fi, ok := fieldIdx[overlayField.Name]
			if !ok {
				// Overlay field not in base — ignore.
				continue
			}
			if overlayField.Description != "" {
				fields[fi].Description = overlayField.Description
			}
			if overlayField.Dimension != nil {
				fields[fi].Dimension = overlayField.Dimension
			}
			if overlayField.Expression != nil {
				fields[fi].Expression = overlayField.Expression
			}
		}
		ds.Fields = fields

		datasets[idx] = ds
	}

	result.SemanticModel.Datasets = datasets

	// 3. Relationships: add only when both sides exist in base.
	relationships := make([]Relationship, len(base.SemanticModel.Relationships))
	copy(relationships, base.SemanticModel.Relationships)

	for _, rel := range overlay.SemanticModel.Relationships {
		_, fromOK := datasetIdx[rel.From]
		_, toOK := datasetIdx[rel.To]
		if fromOK && toOK {
			relationships = append(relationships, rel)
		}
	}
	result.SemanticModel.Relationships = relationships

	// 4. Metrics: add new; update by Name if already present.
	metrics := make([]Metric, len(base.SemanticModel.Metrics))
	copy(metrics, base.SemanticModel.Metrics)
	metricIdx := make(map[string]int, len(metrics))
	for i, m := range metrics {
		metricIdx[m.Name] = i
	}

	for _, overlayMetric := range overlay.SemanticModel.Metrics {
		if mi, exists := metricIdx[overlayMetric.Name]; exists {
			metrics[mi] = overlayMetric
		} else {
			metricIdx[overlayMetric.Name] = len(metrics)
			metrics = append(metrics, overlayMetric)
		}
	}
	result.SemanticModel.Metrics = metrics

	return &result
}
