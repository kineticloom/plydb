package sqlwalk

import (
	"strings"
	"testing"
)

// specPolicy returns a policy matching the spec's comprehensive example.
func specPolicy() *Policy {
	return &Policy{
		PolicyID: "pol_eng_admin_2026",
		Version:  "1.2",
		Permissions: []CatalogPermission{
			{
				Catalog: "production_data",
				Schemas: []SchemaPermission{
					{
						SchemaName: "public",
						BaseAccess: "read",
						AllTables:  true,
						Management: &SchemaManagement{
							AllowDDL:   false,
							AllowIndex: true,
						},
						Overrides: SchemaOverrides{
							ReadWrite: []string{"application_logs", "session_cache"},
							Denied:    []string{"pii_vault_keys"},
						},
					},
					{
						SchemaName: "staging_area",
						BaseAccess: "read_write",
						AllTables:  true,
						Management: &SchemaManagement{
							AllowDDL: true,
							CanDrop:  true,
						},
						Overrides: SchemaOverrides{
							ReadOnly: []string{"reference_master_data"},
						},
					},
					{
						SchemaName: "finance_reports",
						BaseAccess: "none",
						AllTables:  false,
						Overrides: SchemaOverrides{
							ReadOnly: []string{"public_revenue_summary"},
							Granular: []GranularOverride{
								{
									Tables:   []string{"ledger_entries"},
									Actions:  []string{"SELECT", "INSERT"},
									AllowDDL: false,
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
		// ---- public schema: base_access=read, management: allow_index=true ----
		{
			name:      "public: SELECT allowed by base read",
			sql:       "SELECT * FROM production_data.public.users",
			wantCount: 0,
		},
		{
			name:       "public: INSERT denied on base read table",
			sql:        "INSERT INTO production_data.public.users (id) VALUES (1)",
			wantCount:  1,
			wantSubstr: "INSERT on production_data.public.users",
		},
		{
			name:      "public: INSERT allowed on read_write override",
			sql:       "INSERT INTO production_data.public.application_logs (msg) VALUES ('hi')",
			wantCount: 0,
		},
		{
			name:      "public: UPDATE allowed on read_write override",
			sql:       "UPDATE production_data.public.session_cache SET val = 'x'",
			wantCount: 0,
		},
		{
			name:       "public: SELECT denied on pii_vault_keys",
			sql:        "SELECT * FROM production_data.public.pii_vault_keys",
			wantCount:  1,
			wantSubstr: "SELECT on production_data.public.pii_vault_keys",
		},
		{
			name:      "public: CREATE INDEX allowed by management",
			sql:       "CREATE INDEX idx_users_name ON production_data.public.users (name)",
			wantCount: 0,
		},
		{
			name:       "public: ALTER TABLE denied (allow_ddl=false)",
			sql:        "ALTER TABLE production_data.public.users ADD COLUMN email TEXT",
			wantCount:  1,
			wantSubstr: "ALTER TABLE on production_data.public.users",
		},
		// ---- staging_area schema: base_access=read_write, management: allow_ddl, can_drop ----
		{
			name:      "staging: SELECT allowed",
			sql:       "SELECT * FROM production_data.staging_area.temp",
			wantCount: 0,
		},
		{
			name:      "staging: INSERT allowed",
			sql:       "INSERT INTO production_data.staging_area.temp (id) VALUES (1)",
			wantCount: 0,
		},
		{
			name:      "staging: UPDATE allowed",
			sql:       "UPDATE production_data.staging_area.temp SET x = 1",
			wantCount: 0,
		},
		{
			name:      "staging: DELETE allowed",
			sql:       "DELETE FROM production_data.staging_area.temp WHERE id = 1",
			wantCount: 0,
		},
		{
			name:      "staging: ALTER TABLE allowed (allow_ddl=true)",
			sql:       "ALTER TABLE production_data.staging_area.temp ADD COLUMN y INT",
			wantCount: 0,
		},
		{
			name:      "staging: DROP TABLE allowed (can_drop=true)",
			sql:       "DROP TABLE production_data.staging_area.temp",
			wantCount: 0,
		},
		{
			name:      "staging: TRUNCATE allowed (can_drop=true)",
			sql:       "TRUNCATE production_data.staging_area.temp",
			wantCount: 0,
		},
		{
			name:       "staging: read_only override blocks UPDATE",
			sql:        "UPDATE production_data.staging_area.reference_master_data SET x = 1",
			wantCount:  1,
			wantSubstr: "UPDATE on production_data.staging_area.reference_master_data",
		},
		{
			name:      "staging: read_only override allows SELECT",
			sql:       "SELECT * FROM production_data.staging_area.reference_master_data",
			wantCount: 0,
		},
		// ---- finance_reports schema: base_access=none, granular override ----
		{
			name:       "finance: SELECT denied on unlisted table",
			sql:        "SELECT * FROM production_data.finance_reports.secret_budget",
			wantCount:  1,
			wantSubstr: "SELECT on production_data.finance_reports.secret_budget",
		},
		{
			name:      "finance: SELECT allowed on read_only override",
			sql:       "SELECT * FROM production_data.finance_reports.public_revenue_summary",
			wantCount: 0,
		},
		{
			name:       "finance: INSERT denied on read_only override",
			sql:        "INSERT INTO production_data.finance_reports.public_revenue_summary (x) VALUES (1)",
			wantCount:  1,
			wantSubstr: "INSERT on production_data.finance_reports.public_revenue_summary",
		},
		{
			name:      "finance: granular SELECT allowed on ledger_entries",
			sql:       "SELECT * FROM production_data.finance_reports.ledger_entries",
			wantCount: 0,
		},
		{
			name:      "finance: granular INSERT allowed on ledger_entries",
			sql:       "INSERT INTO production_data.finance_reports.ledger_entries (id) VALUES (1)",
			wantCount: 0,
		},
		{
			name:       "finance: granular UPDATE denied on ledger_entries",
			sql:        "UPDATE production_data.finance_reports.ledger_entries SET id = 2",
			wantCount:  1,
			wantSubstr: "UPDATE on production_data.finance_reports.ledger_entries",
		},
		{
			name:       "finance: granular DELETE denied on ledger_entries",
			sql:        "DELETE FROM production_data.finance_reports.ledger_entries WHERE id = 1",
			wantCount:  1,
			wantSubstr: "DELETE on production_data.finance_reports.ledger_entries",
		},
		{
			name:       "finance: granular ALTER TABLE denied on ledger_entries (allow_ddl=false)",
			sql:        "ALTER TABLE production_data.finance_reports.ledger_entries ADD COLUMN y INT",
			wantCount:  1,
			wantSubstr: "ALTER TABLE on production_data.finance_reports.ledger_entries",
		},
		// ---- access level: append ----
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
		// ---- access level: full_dml ----
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
		// ---- compound statements ----
		{
			name:      "JOIN both allowed",
			sql:       "SELECT * FROM production_data.public.users u JOIN production_data.public.application_logs l ON u.id = l.user_id",
			wantCount: 0,
		},
		{
			name:       "JOIN one denied",
			sql:        "SELECT * FROM production_data.public.users u JOIN production_data.public.pii_vault_keys p ON u.id = p.user_id",
			wantCount:  1,
			wantSubstr: "pii_vault_keys",
		},
		{
			name:       "INSERT SELECT from denied table",
			sql:        "INSERT INTO production_data.public.application_logs (id) SELECT id FROM production_data.public.pii_vault_keys",
			wantCount:  1,
			wantSubstr: "pii_vault_keys",
		},
		{
			name:      "CTE reading allowed tables",
			sql:       "WITH cte AS (SELECT * FROM production_data.public.users) SELECT * FROM cte",
			wantCount: 0,
		},
		{
			name:       "subquery in WHERE accessing denied table",
			sql:        "SELECT * FROM production_data.public.users WHERE id IN (SELECT user_id FROM production_data.public.pii_vault_keys)",
			wantCount:  1,
			wantSubstr: "pii_vault_keys",
		},
		{
			name:       "unknown catalog denied",
			sql:        "SELECT * FROM unknown_db.public.t1",
			wantCount:  1,
			wantSubstr: "SELECT on unknown_db.public.t1",
		},
		// ---- MERGE ----
		{
			name: "MERGE with INSERT+UPDATE on staging allowed",
			sql: `MERGE INTO production_data.staging_area.temp t
				  USING production_data.staging_area.src s ON t.id = s.id
				  WHEN MATCHED THEN UPDATE SET val = s.val
				  WHEN NOT MATCHED THEN INSERT (id, val) VALUES (s.id, s.val)`,
			wantCount: 0,
		},
		{
			name: "MERGE target on read-only table denied",
			sql: `MERGE INTO production_data.public.users t
				  USING production_data.public.application_logs s ON t.id = s.user_id
				  WHEN MATCHED THEN UPDATE SET name = s.msg`,
			wantCount:  1,
			wantSubstr: "UPDATE on production_data.public.users",
		},
		// ---- multiple violations ----
		{
			name:      "multiple DDL+DML violations",
			sql:       "ALTER TABLE production_data.public.users ADD COLUMN x INT; DROP TABLE production_data.public.users",
			wantCount: 2,
		},
	}

	// Add the append/full_dml policy for those tests.
	appendFullDMLPolicy := &Policy{
		Permissions: []CatalogPermission{
			{
				Catalog: "mydb",
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
	sql := "SELECT * FROM production_data.public.pii_vault_keys p JOIN production_data.finance_reports.secret_budget s ON p.id = s.id"

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
		"version": "1.2",
		"permissions": [
			{
				"catalog": "mydb",
				"schemas": [
					{
						"schema_name": "public",
						"base_access": "read",
						"all_tables": true,
						"management": {
							"allow_ddl": true,
							"allow_index": true
						},
						"overrides": {
							"denied": ["secret"],
							"granular": [
								{
									"tables": ["audit"],
									"actions": ["SELECT", "INSERT"],
									"allow_ddl": false
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
	if p.Version != "1.2" {
		t.Errorf("Version = %q, want %q", p.Version, "1.2")
	}
	if len(p.Permissions) != 1 {
		t.Fatalf("len(Permissions) = %d, want 1", len(p.Permissions))
	}
	schema := p.Permissions[0].Schemas[0]
	if schema.Management == nil {
		t.Fatal("expected non-nil Management")
	}
	if !schema.Management.AllowDDL {
		t.Error("expected AllowDDL=true")
	}
	if len(schema.Overrides.Granular) != 1 {
		t.Fatalf("expected 1 granular override, got %d", len(schema.Overrides.Granular))
	}
	if len(schema.Overrides.Granular[0].Actions) != 2 {
		t.Errorf("expected 2 granular actions, got %d", len(schema.Overrides.Granular[0].Actions))
	}
}

func TestParsePolicyInvalidAccess(t *testing.T) {
	data := []byte(`{
		"permissions": [
			{
				"catalog": "mydb",
				"schemas": [
					{
						"schema_name": "public",
						"base_access": "invalid_level",
						"all_tables": true
					}
				]
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
