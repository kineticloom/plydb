// Copyright 2026 Paul Tzen
// SPDX-License-Identifier: Apache-2.0

package queryresult

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

const (
	MaxRows  = 2048
	MaxChars = 50000
)

// QueryResult is the structured result returned by the query tool.
type QueryResult struct {
	Success     bool     `json:"success"`
	Columns     []string `json:"columns,omitempty"`
	ColumnTypes []string `json:"column_types,omitempty"`
	Rows        [][]any  `json:"rows,omitempty"`
	RowCount    int      `json:"row_count"`
	Truncated   bool     `json:"truncated"`
	Message     string   `json:"message,omitempty"`
}

// BuildQueryResult scans sql.Rows into a QueryResult.
// It stops after MaxRows and enforces MaxChars on the JSON representation.
func BuildQueryResult(rows *sql.Rows) (*QueryResult, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("reading columns: %w", err)
	}

	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, fmt.Errorf("reading column types: %w", err)
	}
	typeNames := make([]string, len(colTypes))
	for i, ct := range colTypes {
		typeNames[i] = ct.DatabaseTypeName()
	}

	var allRows [][]any
	truncated := false

	scanPtrs := make([]any, len(cols))
	scanVals := make([]any, len(cols))
	for i := range scanVals {
		scanPtrs[i] = &scanVals[i]
	}

	for rows.Next() {
		if len(allRows) >= MaxRows {
			truncated = true
			break
		}
		if err := rows.Scan(scanPtrs...); err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		row := make([]any, len(cols))
		for i, v := range scanVals {
			row[i] = normalizeValue(v)
		}
		allRows = append(allRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating rows: %w", err)
	}

	result := &QueryResult{
		Success:     true,
		Columns:     cols,
		ColumnTypes: typeNames,
		Rows:        allRows,
		RowCount:    len(allRows),
		Truncated:   truncated,
	}
	if truncated {
		result.Message = fmt.Sprintf("Results truncated to %d rows", MaxRows)
	}

	// Enforce character limit via binary search.
	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshaling result: %w", err)
	}
	if len(data) > MaxChars {
		result = truncateToFit(result)
	}

	return result, nil
}

// truncateToFit uses binary search to find the maximum number of rows
// that fit within MaxChars when marshaled to JSON.
func truncateToFit(result *QueryResult) *QueryResult {
	lo, hi := 0, len(result.Rows)
	for lo < hi {
		mid := (lo + hi + 1) / 2
		candidate := *result
		candidate.Rows = result.Rows[:mid]
		candidate.RowCount = mid
		candidate.Truncated = true
		candidate.Message = fmt.Sprintf("Results truncated to %d rows to fit within character limit", mid)
		data, err := json.Marshal(&candidate)
		if err != nil || len(data) > MaxChars {
			hi = mid - 1
		} else {
			lo = mid
		}
	}

	result.Rows = result.Rows[:lo]
	result.RowCount = lo
	result.Truncated = true
	result.Message = fmt.Sprintf("Results truncated to %d rows to fit within character limit", lo)
	return result
}

// normalizeValue converts types that don't serialize cleanly to JSON.
func normalizeValue(v any) any {
	switch val := v.(type) {
	case []byte:
		return string(val)
	default:
		return v
	}
}

// MarshalResult marshals a QueryResult to a JSON string.
func MarshalResult(result *QueryResult) (string, error) {
	data, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("marshaling result: %w", err)
	}
	return string(data), nil
}

// ErrorResult creates a QueryResult representing an error.
func ErrorResult(msg string) *QueryResult {
	return &QueryResult{
		Success: false,
		Message: msg,
	}
}
