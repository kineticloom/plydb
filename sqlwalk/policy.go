// Copyright 2026 Paul Tzen
// SPDX-License-Identifier: Apache-2.0

package sqlwalk

import (
	"encoding/json"
	"fmt"
)

// AccessLevel represents the DQL/DML level of access granted to a table.
// Levels are ordered so that higher levels include all lower privileges.
type AccessLevel int

const (
	AccessNone      AccessLevel = iota
	AccessRead                  // SELECT
	AccessAppend                // SELECT, INSERT
	AccessReadWrite             // SELECT, INSERT, UPDATE, DELETE
	AccessFullDML               // SELECT, INSERT, UPDATE, DELETE, TRUNCATE
)

func parseAccessLevel(s string) (AccessLevel, error) {
	switch s {
	case "none":
		return AccessNone, nil
	case "read":
		return AccessRead, nil
	case "append":
		return AccessAppend, nil
	case "read_write":
		return AccessReadWrite, nil
	case "full_dml":
		return AccessFullDML, nil
	default:
		return AccessNone, fmt.Errorf("unknown access level: %q", s)
	}
}

// Policy is the top-level access policy parsed from JSON.
type Policy struct {
	PolicyID    string              `json:"policy_id"`
	Version     string              `json:"version"`
	Permissions []CatalogPermission `json:"permissions"`
}

// CatalogPermission describes access within a single database catalog.
type CatalogPermission struct {
	Catalog    string             `json:"catalog"`
	BaseAccess string             `json:"base_access"`
	Management *SchemaManagement  `json:"management,omitempty"`
	Schemas    []SchemaPermission `json:"schemas"`
}

// SchemaPermission describes access within a single schema.
type SchemaPermission struct {
	SchemaName string            `json:"schema_name"`
	BaseAccess string            `json:"base_access"`
	AllTables  bool              `json:"all_tables"`
	Management *SchemaManagement `json:"management,omitempty"`
	Overrides  SchemaOverrides   `json:"overrides"`
}

// SchemaManagement holds DDL-level permission flags for a schema.
type SchemaManagement struct {
	AllowDDL   bool `json:"allow_ddl"`
	AllowIndex bool `json:"allow_index"`
	CanDrop    bool `json:"can_drop"`
}

// SchemaOverrides lists tables that deviate from the schema's base access.
type SchemaOverrides struct {
	ReadOnly  []string           `json:"read_only"`
	ReadWrite []string           `json:"read_write"`
	Append    []string           `json:"append"`
	Denied    []string           `json:"denied"`
	Granular  []GranularOverride `json:"granular"`
}

// GranularOverride defines per-table action-level access.
type GranularOverride struct {
	Tables  []string `json:"tables"`
	Actions []string `json:"actions"` // e.g. "SELECT", "INSERT", "UPDATE", "DELETE", "TRUNCATE"
}

// ParsePolicy parses a JSON-encoded access policy.
func ParsePolicy(data []byte) (*Policy, error) {
	var p Policy
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parsing policy JSON: %w", err)
	}
	seenCatalogs := make(map[string]int)
	for i, cp := range p.Permissions {
		if prev, ok := seenCatalogs[cp.Catalog]; ok {
			return nil, fmt.Errorf("duplicate catalog %q in permissions[%d] and permissions[%d]", cp.Catalog, prev, i)
		}
		seenCatalogs[cp.Catalog] = i

		if _, err := parseAccessLevel(cp.BaseAccess); err != nil {
			return nil, fmt.Errorf("permissions[%d].base_access: %w", i, err)
		}

		seenSchemas := make(map[string]int)
		for j, sp := range cp.Schemas {
			if prev, ok := seenSchemas[sp.SchemaName]; ok {
				return nil, fmt.Errorf("duplicate schema_name %q in permissions[%d].schemas[%d] and permissions[%d].schemas[%d]",
					sp.SchemaName, i, prev, i, j)
			}
			seenSchemas[sp.SchemaName] = j

			if sp.BaseAccess != "" {
				if _, err := parseAccessLevel(sp.BaseAccess); err != nil {
					return nil, fmt.Errorf("permissions[%d].schemas[%d].base_access: %w", i, j, err)
				}
			}
		}
	}
	return &p, nil
}

// resolvedPolicy is an internal index built from a Policy for fast lookups.
type resolvedPolicy struct {
	catalogs map[string]*resolvedCatalog
}

type resolvedCatalog struct {
	baseAccess AccessLevel
	management SchemaManagement
	schemas    map[string]*resolvedSchema
}

type resolvedSchema struct {
	allTables  bool
	baseAccess AccessLevel
	management SchemaManagement
	tables     map[string]*resolvedTable // per-table overrides
}

// resolvedTable holds the effective permissions for a single table.
type resolvedTable struct {
	// For shorthand overrides (read_only, read_write, append, denied):
	dmlLevel AccessLevel

	// For granular overrides, actions is non-nil and dmlLevel is ignored.
	actions map[string]struct{} // "SELECT", "INSERT", etc.

	// denied means REVOKE ALL — no DML and no DDL.
	denied bool
}

func resolve(p *Policy) (*resolvedPolicy, error) {
	rp := &resolvedPolicy{catalogs: make(map[string]*resolvedCatalog)}
	for _, cp := range p.Permissions {
		catalogAccess, _ := parseAccessLevel(cp.BaseAccess)
		rc := &resolvedCatalog{
			baseAccess: catalogAccess,
			schemas:    make(map[string]*resolvedSchema),
		}
		if cp.Management != nil {
			rc.management = *cp.Management
		}

		for _, sp := range cp.Schemas {
			rs := &resolvedSchema{
				allTables: sp.AllTables,
				tables:    make(map[string]*resolvedTable),
			}

			// Effective base_access: schema's if set, else catalog's.
			if sp.BaseAccess != "" {
				lvl, _ := parseAccessLevel(sp.BaseAccess)
				rs.baseAccess = lvl
			} else {
				rs.baseAccess = catalogAccess
			}

			// Effective management: schema's if present (full replacement), else catalog's.
			if sp.Management != nil {
				rs.management = *sp.Management
			} else {
				rs.management = rc.management
			}

			for _, t := range sp.Overrides.ReadOnly {
				rs.tables[t] = &resolvedTable{dmlLevel: AccessRead}
			}
			for _, t := range sp.Overrides.ReadWrite {
				rs.tables[t] = &resolvedTable{dmlLevel: AccessReadWrite}
			}
			for _, t := range sp.Overrides.Append {
				rs.tables[t] = &resolvedTable{dmlLevel: AccessAppend}
			}
			for _, t := range sp.Overrides.Denied {
				rs.tables[t] = &resolvedTable{dmlLevel: AccessNone, denied: true}
			}
			for _, g := range sp.Overrides.Granular {
				actions := make(map[string]struct{}, len(g.Actions))
				for _, a := range g.Actions {
					actions[a] = struct{}{}
				}
				for _, t := range g.Tables {
					rs.tables[t] = &resolvedTable{
						actions: actions,
					}
				}
			}
			rc.schemas[sp.SchemaName] = rs
		}
		rp.catalogs[cp.Catalog] = rc
	}
	return rp, nil
}

// lookup returns the effective table permissions for a given catalog.schema.table.
func (rp *resolvedPolicy) lookup(catalog, schema, table string) tablePermission {
	rc, ok := rp.catalogs[catalog]
	if !ok {
		return tablePermission{dmlLevel: AccessNone}
	}

	rs, ok := rc.schemas[schema]
	if !ok {
		// Unlisted schema → catalog base_access + catalog management.
		return tablePermission{
			dmlLevel:   rc.baseAccess,
			allowDDL:   rc.management.AllowDDL,
			allowIndex: rc.management.AllowIndex,
			canDrop:    rc.management.CanDrop,
		}
	}

	// Check table-level overrides.
	if rt, ok := rs.tables[table]; ok {
		// Denied tables → REVOKE ALL (no DML, no DDL).
		if rt.denied {
			return tablePermission{dmlLevel: AccessNone}
		}
		if rt.actions != nil {
			// Granular override: DDL from effective management.
			return tablePermission{
				actions:    rt.actions,
				allowDDL:   rs.management.AllowDDL,
				allowIndex: rs.management.AllowIndex,
				canDrop:    rs.management.CanDrop,
			}
		}
		// Shorthand override: DDL from effective management.
		return tablePermission{
			dmlLevel:   rt.dmlLevel,
			allowDDL:   rs.management.AllowDDL,
			allowIndex: rs.management.AllowIndex,
			canDrop:    rs.management.CanDrop,
		}
	}

	// Fall back to schema base if all_tables is set.
	if rs.allTables {
		return tablePermission{
			dmlLevel:   rs.baseAccess,
			allowDDL:   rs.management.AllowDDL,
			allowIndex: rs.management.AllowIndex,
			canDrop:    rs.management.CanDrop,
		}
	}

	// Listed schema, all_tables=false, table not in overrides → none + no management.
	return tablePermission{dmlLevel: AccessNone}
}

// tablePermission is the resolved set of permissions for a single table access check.
type tablePermission struct {
	dmlLevel   AccessLevel
	actions    map[string]struct{} // non-nil for granular overrides
	allowDDL   bool
	allowIndex bool
	canDrop    bool
}

// allows checks whether the given operation is permitted.
func (tp tablePermission) allows(op OpType) bool {
	switch op {
	case OpAlterTable:
		return tp.allowDDL
	case OpCreateIndex, OpDropIndex:
		return tp.allowIndex
	case OpDropTable:
		return tp.canDrop
	}

	// DML/DQL checks.
	if tp.actions != nil {
		actionStr := op.ActionString()
		_, ok := tp.actions[actionStr]
		if op == OpTruncate {
			return ok || tp.canDrop
		}
		return ok
	}

	switch op {
	case OpSelect:
		return tp.dmlLevel >= AccessRead
	case OpInsert:
		return tp.dmlLevel >= AccessAppend
	case OpUpdate, OpDelete:
		return tp.dmlLevel >= AccessReadWrite
	case OpTruncate:
		return tp.dmlLevel >= AccessFullDML || tp.canDrop
	}
	return false
}
