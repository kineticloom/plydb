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
// Each overlay file must contain exactly one semantic_model entry.
func (p *OverlayProvider) Provide(ctx context.Context, existing *SemanticModelFile) (*SemanticModelFile, error) {
	result := existing
	if result == nil {
		result = &SemanticModelFile{SemanticModel: []SemanticModel{{}}}
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

		if len(overlay.SemanticModel) != 1 {
			return nil, fmt.Errorf("overlay file %q: semantic_model must contain exactly one entry, got %d", path, len(overlay.SemanticModel))
		}

		result = applyOverlay(result, &overlay)
	}

	return result, nil
}

// applyOverlay merges overlay.SemanticModel[0] into base.SemanticModel[0] and
// returns the enriched model. It does not modify base or overlay in place.
// The caller must ensure overlay.SemanticModel has exactly one entry.
func applyOverlay(base, overlay *SemanticModelFile) *SemanticModelFile {
	result := *base

	// Copy the models slice so we can mutate [0] safely.
	models := make([]SemanticModel, len(base.SemanticModel))
	copy(models, base.SemanticModel)

	// Ensure base has at least one model to apply onto.
	if len(models) == 0 {
		models = []SemanticModel{{}}
	}

	model := models[0]
	overlayModel := overlay.SemanticModel[0]

	// 1. Model-level name, description and ai_context.
	if overlayModel.Name != "" {
		model.Name = overlayModel.Name
	}
	if overlayModel.Description != "" {
		model.Description = overlayModel.Description
	}
	if !overlayModel.AIContext.IsZero() {
		model.AIContext = overlayModel.AIContext
	}

	// Build an index of base datasets by name for O(1) lookup.
	datasetIdx := make(map[string]int, len(model.Datasets))
	for i, ds := range model.Datasets {
		datasetIdx[ds.Name] = i
	}

	// Copy the datasets slice so we can mutate individual entries safely.
	datasets := make([]Dataset, len(model.Datasets))
	copy(datasets, model.Datasets)

	// 2. Enrich datasets that exist in base.
	for _, overlayDS := range overlayModel.Datasets {
		dsIdx, ok := datasetIdx[overlayDS.Name]
		if !ok {
			// Overlay dataset not in base — ignore.
			continue
		}

		ds := datasets[dsIdx]

		if overlayDS.Description != "" {
			ds.Description = overlayDS.Description
		}
		if !overlayDS.AIContext.IsZero() {
			ds.AIContext = overlayDS.AIContext
		}
		if len(overlayDS.PrimaryKey) > 0 {
			ds.PrimaryKey = overlayDS.PrimaryKey
		}
		if len(overlayDS.UniqueKeys) > 0 {
			ds.UniqueKeys = overlayDS.UniqueKeys
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
			if !overlayField.AIContext.IsZero() {
				fields[fi].AIContext = overlayField.AIContext
			}
			if overlayField.Dimension != nil {
				fields[fi].Dimension = overlayField.Dimension
			}
			if len(overlayField.Expression.Dialects) > 0 {
				fields[fi].Expression = overlayField.Expression
			}
		}
		ds.Fields = fields

		datasets[dsIdx] = ds
	}

	model.Datasets = datasets

	// 3. Relationships: add only when both sides exist in base.
	relationships := make([]Relationship, len(model.Relationships))
	copy(relationships, model.Relationships)

	for _, rel := range overlayModel.Relationships {
		_, fromOK := datasetIdx[rel.From]
		_, toOK := datasetIdx[rel.To]
		if fromOK && toOK {
			relationships = append(relationships, rel)
		}
	}
	model.Relationships = relationships

	// 4. Metrics: add new; update by Name if already present.
	metrics := make([]Metric, len(model.Metrics))
	copy(metrics, model.Metrics)
	metricIdx := make(map[string]int, len(metrics))
	for i, m := range metrics {
		metricIdx[m.Name] = i
	}

	for _, overlayMetric := range overlayModel.Metrics {
		if mi, exists := metricIdx[overlayMetric.Name]; exists {
			metrics[mi] = overlayMetric
		} else {
			metricIdx[overlayMetric.Name] = len(metrics)
			metrics = append(metrics, overlayMetric)
		}
	}
	model.Metrics = metrics

	models[0] = model
	result.SemanticModel = models
	return &result
}
