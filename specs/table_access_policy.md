## Specification: Database Hierarchical Access Policy

This specification defines a JSON schema for managing database access permissions, covering **DQL (Data Query Language)**, **DML (Data Manipulation Language)**, and **DDL (Data Definition Language)**. It supports a "Base-Override" inheritance model to minimize configuration verbosity.

---

### 1. Hierarchy & Inheritance

The policy follows a strict top-down inheritance:

1. **Catalog (Database)**: The top-level container.
2. **Schema**: Defines the default (`base_access`) for all contained objects.
3. **Table Overrides**: Explicitly modifies permissions for specific tables, deviating from the schema base.

---

### 2. Key Definitions

#### A. Access Shorthands

To simplify the JSON, the following shorthands map to specific SQL privilege sets:

* `none`: Revokes all privileges.
* `read`: `SELECT`.
* `append`: `SELECT`, `INSERT`.
* `read_write`: `SELECT`, `INSERT`, `UPDATE`, `DELETE`.
* `full_dml`: All of the above plus `TRUNCATE`.

#### B. DDL & Management

Because DDL (`ALTER`, `DROP`) usually requires ownership in Postgres, the policy engine interprets these flags to either change object ownership or grant membership in a "Manager" role.

* **`allow_ddl`**: Enables `ALTER TABLE` (adding/modifying columns).
* **`allow_index`**: Enables `CREATE INDEX`.
* **`can_drop`**: Enables `DROP TABLE` and `TRUNCATE`.

---

### 3. JSON Schema Specification

| Path | Type | Description |
| --- | --- | --- |
| `permissions[].catalog` | String | The Postgres database name. |
| `permissions[].all_schemas` | Boolean | If `true`, applies `base_access` to all schemas. |
| `schemas[].base_access` | Enum | The default DQL/DML level for the schema. |
| `schemas[].all_tables` | Boolean | If `true`, enables the allowlist for the schema. |
| `schemas[].management` | Object | Defines DDL-level permissions for the schema. |
| `schemas[].overrides` | Object | Arrays defining tables that differ from `base_access`. |

---

### 4. Comprehensive Example Policy

```json
{
  "policy_id": "pol_eng_admin_2026",
  "version": "1.2",
  "permissions": [
    {
      "catalog": "production_data",
      "schemas": [
        {
          "schema_name": "public",
          "base_access": "read",
          "all_tables": true,
          "management": {
            "allow_ddl": false,
            "allow_index": true,
            "comment": "Users can tune performance via indexes but not change schema."
          },
          "overrides": {
            "read_write": ["application_logs", "session_cache"],
            "denied": ["pii_vault_keys"]
          }
        },
        {
          "schema_name": "staging_area",
          "base_access": "read_write",
          "all_tables": true,
          "management": {
            "allow_ddl": true,
            "can_drop": true,
            "comment": "Full DDL/DML control for staging tables."
          },
          "overrides": {
            "read_only": ["reference_master_data"]
          }
        },
        {
          "schema_name": "finance_reports",
          "base_access": "none",
          "all_tables": false,
          "overrides": {
            "read_only": ["public_revenue_summary"],
            "granular": [
              {
                "tables": ["ledger_entries"],
                "actions": ["SELECT", "INSERT"],
                "allow_ddl": false,
                "comment": "Append-only access to ledger; no modifications allowed."
              }
            ]
          }
        }
      ]
    }
  ]
}

```
