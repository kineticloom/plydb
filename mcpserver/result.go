package mcpserver

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

const (
	maxRows  = 2048
	maxChars = 50000
)

// QueryResult is the structured result returned by the query tool.
type QueryResult struct {
	Success     bool              `json:"success"`
	Columns     []string          `json:"columns,omitempty"`
	ColumnTypes []string          `json:"column_types,omitempty"`
	Rows        [][]any           `json:"rows,omitempty"`
	RowCount    int               `json:"row_count"`
	Truncated   bool              `json:"truncated"`
	Message     string            `json:"message,omitempty"`
}

// buildQueryResult scans sql.Rows into a QueryResult.
// It stops after maxRows and enforces maxChars on the JSON representation.
func buildQueryResult(rows *sql.Rows) (*QueryResult, error) {
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
		if len(allRows) >= maxRows {
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
		result.Message = fmt.Sprintf("Results truncated to %d rows", maxRows)
	}

	// Enforce character limit via binary search.
	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshaling result: %w", err)
	}
	if len(data) > maxChars {
		result = truncateToFit(result)
	}

	return result, nil
}

// truncateToFit uses binary search to find the maximum number of rows
// that fit within maxChars when marshaled to JSON.
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
		if err != nil || len(data) > maxChars {
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

// marshalResult marshals a QueryResult to a JSON string, sorted keys.
func marshalResult(result *QueryResult) (string, error) {
	data, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("marshaling result: %w", err)
	}
	return string(data), nil
}

// errorResult creates a QueryResult representing an error.
func errorResult(msg string) *QueryResult {
	return &QueryResult{
		Success: false,
		Message: msg,
	}
}

