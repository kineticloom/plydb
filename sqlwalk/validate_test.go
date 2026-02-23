// Copyright 2026 Paul Tzen
// SPDX-License-Identifier: Apache-2.0

package sqlwalk

import (
	"strings"
	"testing"
)

// specPolicy returns a policy matching the v2 spec's comprehensive example.
func specPolicy() *Policy {
	return &Policy{
		PolicyID: "global_enterprise_policy_2026",
		Version:  "1.3",
		Permissions: []CatalogPermission{
			{
				Catalog:    "production_db",
				BaseAccess: "read",
				Management: &SchemaManagement{AllowDDL: false, AllowIndex: false},
				Schemas: []SchemaPermission{
					{
						SchemaName: "app_data",
						BaseAccess: "read_write",
						AllTables:  true,
						Management: &SchemaManagement{AllowIndex: true},
						Overrides: SchemaOverrides{
							Denied:   []string{"user_passwords", "credit_card_numbers"},
							ReadOnly: []string{"app_config_immutable"},
						},
					},
					{
						SchemaName: "reporting_sandbox",
						BaseAccess: "full_dml",
						AllTables:  true,
						Management: &SchemaManagement{AllowDDL: true, CanDrop: true},
					},
				},
			},
			{
				Catalog:    "staging_db",
				BaseAccess: "read_write",
				Management: &SchemaManagement{AllowDDL: true, AllowIndex: true},
				Schemas: []SchemaPermission{
					{
						SchemaName: "finance_audit",
						BaseAccess: "read",
						AllTables:  true,
						Management: &SchemaManagement{AllowDDL: false},
						Overrides: SchemaOverrides{
							Granular: []GranularOverride{
								{
									Tables:  []string{"audit_trail"},
									Actions: []string{"SELECT", "INSERT"},
								},
							},
						},
					},
				},
			},
		},
	}
}

func TestValidate(t *testing.T) {
	policy := specPolicy()

	tests := []struct {
		name       string
		sql        string
		wantCount  int
		wantSubstr string // substring in first violation's Error()
	}{
		// ---- app_data schema: base_access=read_write, management: allow_index=true ----
		{
			name:      "app_data: SELECT allowed by base read_write",
			sql:       "SELECT * FROM production_db.app_data.users",
			wantCount: 0,
		},
		{
			name:      "app_data: INSERT allowed by base read_write",
			sql:       "INSERT INTO production_db.app_data.users (id) VALUES (1)",
			wantCount: 0,
		},
		{
			name:      "app_data: UPDATE allowed by base read_write",
			sql:       "UPDATE production_db.app_data.users SET name = 'x'",
			wantCount: 0,
		},
		{
			name:       "app_data: SELECT denied on user_passwords (denied)",
			sql:        "SELECT * FROM production_db.app_data.user_passwords",
			wantCount:  1,
			wantSubstr: "SELECT on production_db.app_data.user_passwords",
		},
		{
			name:       "app_data: ALTER TABLE denied on user_passwords (denied blocks DDL)",
			sql:        "ALTER TABLE production_db.app_data.user_passwords ADD COLUMN x TEXT",
			wantCount:  1,
			wantSubstr: "ALTER TABLE on production_db.app_data.user_passwords",
		},
		{
			name:       "app_data: CREATE INDEX denied on credit_card_numbers (denied blocks DDL)",
			sql:        "CREATE INDEX idx_cc ON production_db.app_data.credit_card_numbers (num)",
			wantCount:  1,
			wantSubstr: "CREATE INDEX on production_db.app_data.credit_card_numbers",
		},
		{
			name:      "app_data: read_only override allows SELECT",
			sql:       "SELECT * FROM production_db.app_data.app_config_immutable",
			wantCount: 0,
		},
		{
			name:       "app_data: read_only override blocks INSERT",
			sql:        "INSERT INTO production_db.app_data.app_config_immutable (k) VALUES ('v')",
			wantCount:  1,
			wantSubstr: "INSERT on production_db.app_data.app_config_immutable",
		},
		{
			name:      "app_data: CREATE INDEX allowed by schema management",
			sql:       "CREATE INDEX idx_users_name ON production_db.app_data.users (name)",
			wantCount: 0,
		},
		{
			name:      "app_data: DROP INDEX allowed by schema management (allow_index)",
			sql:       "DROP INDEX production_db.app_data.idx_users_name",
			wantCount: 0,
		},
		{
			name:       "app_data: ALTER TABLE denied (schema management allow_ddl=false)",
			sql:        "ALTER TABLE production_db.app_data.users ADD COLUMN email TEXT",
			wantCount:  1,
			wantSubstr: "ALTER TABLE on production_db.app_data.users",
		},
		// ---- reporting_sandbox schema: base_access=full_dml, management: allow_ddl, can_drop ----
		{
			name:      "reporting_sandbox: SELECT allowed",
			sql:       "SELECT * FROM production_db.reporting_sandbox.temp",
			wantCount: 0,
		},
		{
			name:      "reporting_sandbox: INSERT allowed",
			sql:       "INSERT INTO production_db.reporting_sandbox.temp (id) VALUES (1)",
			wantCount: 0,
		},
		{
			name:      "reporting_sandbox: TRUNCATE allowed (full_dml)",
			sql:       "TRUNCATE production_db.reporting_sandbox.temp",
			wantCount: 0,
		},
		{
			name:      "reporting_sandbox: ALTER TABLE allowed (allow_ddl=true)",
			sql:       "ALTER TABLE production_db.reporting_sandbox.temp ADD COLUMN y INT",
			wantCount: 0,
		},
		{
			name:      "reporting_sandbox: DROP TABLE allowed (can_drop=true)",
			sql:       "DROP TABLE production_db.reporting_sandbox.temp",
			wantCount: 0,
		},
		// ---- catalog-level defaults for unlisted schemas ----
		{
			name:      "unlisted schema: SELECT allowed by catalog base_access=read",
			sql:       "SELECT * FROM production_db.unknown_schema.some_table",
			wantCount: 0,
		},
		{
			name:       "unlisted schema: INSERT denied (catalog base_access=read)",
			sql:        "INSERT INTO production_db.unknown_schema.some_table (id) VALUES (1)",
			wantCount:  1,
			wantSubstr: "INSERT on production_db.unknown_schema.some_table",
		},
		{
			name:       "unlisted schema: ALTER TABLE denied (catalog management allow_ddl=false)",
			sql:        "ALTER TABLE production_db.unknown_schema.some_table ADD COLUMN x INT",
			wantCount:  1,
			wantSubstr: "ALTER TABLE on production_db.unknown_schema.some_table",
		},
		// ---- staging_db: catalog management inheritance ----
		{
			name:      "staging_db unlisted schema: SELECT allowed (catalog base_access=read_write)",
			sql:       "SELECT * FROM staging_db.dev_sandbox.t1",
			wantCount: 0,
		},
		{
			name:      "staging_db unlisted schema: INSERT allowed (catalog base_access=read_write)",
			sql:       "INSERT INTO staging_db.dev_sandbox.t1 (id) VALUES (1)",
			wantCount: 0,
		},
		{
			name:      "staging_db unlisted schema: ALTER TABLE allowed (catalog management allow_ddl=true)",
			sql:       "ALTER TABLE staging_db.dev_sandbox.t1 ADD COLUMN y INT",
			wantCount: 0,
		},
		{
			name:      "staging_db unlisted schema: CREATE INDEX allowed (catalog management allow_index=true)",
			sql:       "CREATE INDEX idx_t1 ON staging_db.dev_sandbox.t1 (id)",
			wantCount: 0,
		},
		// ---- staging_db.finance_audit: schema management override ----
		{
			name:      "finance_audit: SELECT allowed by base read",
			sql:       "SELECT * FROM staging_db.finance_audit.some_table",
			wantCount: 0,
		},
		{
			name:       "finance_audit: INSERT denied (base_access=read)",
			sql:        "INSERT INTO staging_db.finance_audit.some_table (id) VALUES (1)",
			wantCount:  1,
			wantSubstr: "INSERT on staging_db.finance_audit.some_table",
		},
		{
			name:       "finance_audit: ALTER TABLE denied (schema management overrides catalog: allow_ddl=false)",
			sql:        "ALTER TABLE staging_db.finance_audit.some_table ADD COLUMN x INT",
			wantCount:  1,
			wantSubstr: "ALTER TABLE on staging_db.finance_audit.some_table",
		},
		{
			name:       "finance_audit: CREATE INDEX denied (schema management full replacement, allow_index not set)",
			sql:        "CREATE INDEX idx_fa ON staging_db.finance_audit.some_table (id)",
			wantCount:  1,
			wantSubstr: "CREATE INDEX on staging_db.finance_audit.some_table",
		},
		// ---- staging_db.finance_audit: granular override ----
		{
			name:      "finance_audit: granular SELECT allowed on audit_trail",
			sql:       "SELECT * FROM staging_db.finance_audit.audit_trail",
			wantCount: 0,
		},
		{
			name:      "finance_audit: granular INSERT allowed on audit_trail",
			sql:       "INSERT INTO staging_db.finance_audit.audit_trail (id) VALUES (1)",
			wantCount: 0,
		},
		{
			name:       "finance_audit: granular UPDATE denied on audit_trail",
			sql:        "UPDATE staging_db.finance_audit.audit_trail SET id = 2",
			wantCount:  1,
			wantSubstr: "UPDATE on staging_db.finance_audit.audit_trail",
		},
		{
			name:       "finance_audit: granular ALTER TABLE denied on audit_trail (schema management allow_ddl=false)",
			sql:        "ALTER TABLE staging_db.finance_audit.audit_trail ADD COLUMN y INT",
			wantCount:  1,
			wantSubstr: "ALTER TABLE on staging_db.finance_audit.audit_trail",
		},
		// ---- access level: append (via separate policy) ----
		{
			name:      "append: SELECT and INSERT allowed",
			sql:       "INSERT INTO mydb.logs.events (id) SELECT id FROM mydb.logs.events",
			wantCount: 0,
		},
		{
			name:       "append: UPDATE denied",
			sql:        "UPDATE mydb.logs.events SET id = 1",
			wantCount:  1,
			wantSubstr: "UPDATE on mydb.logs.events",
		},
		// ---- access level: full_dml (via separate policy) ----
		{
			name:      "full_dml: TRUNCATE allowed",
			sql:       "TRUNCATE mydb.scratch.temp",
			wantCount: 0,
		},
		{
			name:       "full_dml: DROP TABLE denied (no can_drop)",
			sql:        "DROP TABLE mydb.scratch.temp",
			wantCount:  1,
			wantSubstr: "DROP TABLE on mydb.scratch.temp",
		},
		// ---- append override shorthand ----
		{
			name:      "append override: SELECT allowed",
			sql:       "SELECT * FROM mydb.mixed.event_log",
			wantCount: 0,
		},
		{
			name:      "append override: INSERT allowed",
			sql:       "INSERT INTO mydb.mixed.event_log (id) VALUES (1)",
			wantCount: 0,
		},
		{
			name:       "append override: UPDATE denied",
			sql:        "UPDATE mydb.mixed.event_log SET id = 1",
			wantCount:  1,
			wantSubstr: "UPDATE on mydb.mixed.event_log",
		},
		// ---- compound statements ----
		{
			name:      "JOIN both allowed",
			sql:       "SELECT * FROM production_db.app_data.users u JOIN production_db.app_data.app_config_immutable c ON u.id = c.id",
			wantCount: 0,
		},
		{
			name:       "JOIN one denied",
			sql:        "SELECT * FROM production_db.app_data.users u JOIN production_db.app_data.user_passwords p ON u.id = p.user_id",
			wantCount:  1,
			wantSubstr: "user_passwords",
		},
		{
			name:       "INSERT SELECT from denied table",
			sql:        "INSERT INTO production_db.app_data.users (id) SELECT id FROM production_db.app_data.user_passwords",
			wantCount:  1,
			wantSubstr: "user_passwords",
		},
		{
			name:      "CTE reading allowed tables",
			sql:       "WITH cte AS (SELECT * FROM production_db.app_data.users) SELECT * FROM cte",
			wantCount: 0,
		},
		{
			name:       "subquery in WHERE accessing denied table",
			sql:        "SELECT * FROM production_db.app_data.users WHERE id IN (SELECT user_id FROM production_db.app_data.user_passwords)",
			wantCount:  1,
			wantSubstr: "user_passwords",
		},
		{
			name:       "unknown catalog denied",
			sql:        "SELECT * FROM unknown_db.public.t1",
			wantCount:  1,
			wantSubstr: "SELECT on unknown_db.public.t1",
		},
		// ---- MERGE ----
		{
			name: "MERGE with INSERT+UPDATE on reporting_sandbox allowed",
			sql: `MERGE INTO production_db.reporting_sandbox.temp t
				  USING production_db.reporting_sandbox.src s ON t.id = s.id
				  WHEN MATCHED THEN UPDATE SET val = s.val
				  WHEN NOT MATCHED THEN INSERT (id, val) VALUES (s.id, s.val)`,
			wantCount: 0,
		},
		{
			name: "MERGE target on read-only table denied",
			sql: `MERGE INTO production_db.app_data.app_config_immutable t
				  USING production_db.app_data.users s ON t.id = s.id
				  WHEN MATCHED THEN UPDATE SET name = s.name`,
			wantCount:  1,
			wantSubstr: "UPDATE on production_db.app_data.app_config_immutable",
		},
		// ---- multiple violations ----
		{
			name:      "multiple DDL+DML violations",
			sql:       "ALTER TABLE production_db.app_data.users ADD COLUMN x INT; DROP TABLE production_db.app_data.users",
			wantCount: 2,
		},
		// ---- DROP INDEX ----
		{
			name:      "DROP INDEX allowed when allow_index=true",
			sql:       "DROP INDEX production_db.app_data.idx_test",
			wantCount: 0,
		},
		{
			name:       "DROP INDEX denied when allow_index=false (catalog default)",
			sql:        "DROP INDEX production_db.unknown_schema.idx_test",
			wantCount:  1,
			wantSubstr: "DROP INDEX on production_db.unknown_schema.idx_test",
		},
	}

	// Add the append/full_dml policy for those tests.
	appendFullDMLPolicy := &Policy{
		Permissions: []CatalogPermission{
			{
				Catalog:    "mydb",
				BaseAccess: "none",
				Schemas: []SchemaPermission{
					{
						SchemaName: "logs",
						BaseAccess: "append",
						AllTables:  true,
					},
					{
						SchemaName: "scratch",
						BaseAccess: "full_dml",
						AllTables:  true,
					},
					{
						SchemaName: "mixed",
						BaseAccess: "read",
						AllTables:  true,
						Overrides: SchemaOverrides{
							Append: []string{"event_log"},
						},
					},
				},
			},
		},
	}

	// Merge both policies for test lookup.
	mergedPolicy := &Policy{
		Permissions: append(policy.Permissions, appendFullDMLPolicy.Permissions...),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed := parse(t, tt.sql)
			violations, err := Validate(parsed, mergedPolicy)
			if err != nil {
				t.Fatalf("Validate: %v", err)
			}

			if len(violations) != tt.wantCount {
				for _, v := range violations {
					t.Logf("  violation: %s", v.Error())
				}
				t.Fatalf("got %d violations, want %d", len(violations), tt.wantCount)
			}

			if tt.wantSubstr != "" && tt.wantCount > 0 {
				errStr := violations[0].Error()
				if !strings.Contains(errStr, tt.wantSubstr) {
					t.Errorf("violation %q does not contain %q", errStr, tt.wantSubstr)
				}
			}
		})
	}
}

func TestValidateFailFast(t *testing.T) {
	policy := specPolicy()
	// Query that produces 2 violations without FailFast.
	sql := "SELECT * FROM production_db.app_data.user_passwords p JOIN production_db.app_data.credit_card_numbers c ON p.id = c.id"

	parsed := parse(t, sql)

	all, err := Validate(parsed, policy)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 violations without FailFast, got %d", len(all))
	}

	fast, err := Validate(parsed, policy, FailFast())
	if err != nil {
		t.Fatalf("Validate(FailFast): %v", err)
	}
	if len(fast) != 1 {
		t.Fatalf("expected 1 violation with FailFast, got %d", len(fast))
	}
}

func TestParsePolicy(t *testing.T) {
	data := []byte(`{
		"policy_id": "pol_test",
		"version": "1.3",
		"permissions": [
			{
				"catalog": "mydb",
				"base_access": "read",
				"management": { "allow_ddl": false },
				"schemas": [
					{
						"schema_name": "public",
						"base_access": "read_write",
						"all_tables": true,
						"management": {
							"allow_ddl": true,
							"allow_index": true
						},
						"overrides": {
							"denied": ["secret"],
							"append": ["event_log"],
							"granular": [
								{
									"tables": ["audit"],
									"actions": ["SELECT", "INSERT"]
								}
							]
						}
					}
				]
			}
		]
	}`)

	p, err := ParsePolicy(data)
	if err != nil {
		t.Fatalf("ParsePolicy: %v", err)
	}

	if p.PolicyID != "pol_test" {
		t.Errorf("PolicyID = %q, want %q", p.PolicyID, "pol_test")
	}
	if p.Version != "1.3" {
		t.Errorf("Version = %q, want %q", p.Version, "1.3")
	}
	if len(p.Permissions) != 1 {
		t.Fatalf("len(Permissions) = %d, want 1", len(p.Permissions))
	}
	cp := p.Permissions[0]
	if cp.BaseAccess != "read" {
		t.Errorf("catalog BaseAccess = %q, want %q", cp.BaseAccess, "read")
	}
	if cp.Management == nil {
		t.Fatal("expected non-nil catalog Management")
	}
	schema := cp.Schemas[0]
	if schema.Management == nil {
		t.Fatal("expected non-nil schema Management")
	}
	if !schema.Management.AllowDDL {
		t.Error("expected AllowDDL=true")
	}
	if len(schema.Overrides.Append) != 1 {
		t.Errorf("expected 1 append override, got %d", len(schema.Overrides.Append))
	}
	if len(schema.Overrides.Granular) != 1 {
		t.Fatalf("expected 1 granular override, got %d", len(schema.Overrides.Granular))
	}
	if len(schema.Overrides.Granular[0].Actions) != 2 {
		t.Errorf("expected 2 granular actions, got %d", len(schema.Overrides.Granular[0].Actions))
	}
}

func TestParsePolicyDuplicateCatalog(t *testing.T) {
	data := []byte(`{
		"permissions": [
			{"catalog": "mydb", "base_access": "read", "schemas": [{"schema_name": "public", "base_access": "read", "all_tables": true}]},
			{"catalog": "mydb", "base_access": "read", "schemas": [{"schema_name": "other", "base_access": "read", "all_tables": true}]}
		]
	}`)

	_, err := ParsePolicy(data)
	if err == nil {
		t.Fatal("expected error for duplicate catalog")
	}
	if !strings.Contains(err.Error(), "duplicate catalog") {
		t.Errorf("error %q should mention duplicate catalog", err)
	}
}

func TestParsePolicyDuplicateSchema(t *testing.T) {
	data := []byte(`{
		"permissions": [
			{
				"catalog": "mydb",
				"base_access": "read",
				"schemas": [
					{"schema_name": "public", "base_access": "read", "all_tables": true},
					{"schema_name": "public", "base_access": "read_write", "all_tables": true}
				]
			}
		]
	}`)

	_, err := ParsePolicy(data)
	if err == nil {
		t.Fatal("expected error for duplicate schema_name")
	}
	if !strings.Contains(err.Error(), "duplicate schema_name") {
		t.Errorf("error %q should mention duplicate schema_name", err)
	}
}

func TestParsePolicyInvalidAccess(t *testing.T) {
	data := []byte(`{
		"permissions": [
			{
				"catalog": "mydb",
				"base_access": "invalid_level",
				"schemas": []
			}
		]
	}`)

	_, err := ParsePolicy(data)
	if err == nil {
		t.Fatal("expected error for invalid access level")
	}
}

func TestParsePolicyAppendLevel(t *testing.T) {
	data := []byte(`{
		"permissions": [
			{
				"catalog": "mydb",
				"base_access": "none",
				"schemas": [
					{
						"schema_name": "logs",
						"base_access": "append",
						"all_tables": true
					}
				]
			}
		]
	}`)

	p, err := ParsePolicy(data)
	if err != nil {
		t.Fatalf("ParsePolicy: %v", err)
	}
	if p.Permissions[0].Schemas[0].BaseAccess != "append" {
		t.Error("expected base_access=append")
	}
}

func TestParsePolicyFullDMLLevel(t *testing.T) {
	data := []byte(`{
		"permissions": [
			{
				"catalog": "mydb",
				"base_access": "none",
				"schemas": [
					{
						"schema_name": "scratch",
						"base_access": "full_dml",
						"all_tables": true
					}
				]
			}
		]
	}`)

	p, err := ParsePolicy(data)
	if err != nil {
		t.Fatalf("ParsePolicy: %v", err)
	}
	if p.Permissions[0].Schemas[0].BaseAccess != "full_dml" {
		t.Error("expected base_access=full_dml")
	}
}
