# Table Access Policy

> NOTE: The foundations of this are in the codebase, but this functionality is
> not yet fully enabled for end users

This specification defines a JSON-based schema for managing database access
control. It utilizes a hierarchical inheritance model that allows for broad,
high-level defaults at the **Catalog** level, with the ability to define
granular overrides at the **Schema** and **Table** levels.

---

## 1. Core Concepts

### 1.1 Inheritance Hierarchy

Permissions flow from top to bottom. A more specific definition always overrides
a more general one: `Catalog Default` → `Schema Override` → `Table Override`.

### 1.2 Access Levels (Shorthands)

To ensure clarity, the following shorthand strings map to specific SQL privilege
sets:

| Level        | SQL Privileges                             | Intended Use                          |
| ------------ | ------------------------------------------ | ------------------------------------- |
| `none`       | `REVOKE ALL`                               | Complete isolation/removal of access. |
| `read`       | `SELECT`                                   | Standard data viewing.                |
| `append`     | `SELECT, INSERT`                           | Logging and audit trails.             |
| `read_write` | `SELECT, INSERT, UPDATE, DELETE`           | Standard application/DML access.      |
| `full_dml`   | `SELECT, INSERT, UPDATE, DELETE, TRUNCATE` | High-level data manipulation.         |

### 1.3 Management (DDL) Capabilities

DDL (Data Definition Language) permissions are handled via the `management`
object.

- **`allow_ddl`**: Grants the ability to `ALTER` existing table structures.
- **`allow_index`**: Grants the ability to `CREATE` or `DROP` indexes.
- **`can_drop`**: Grants the ability to `DROP` tables or the schema itself.

---

## 2. Data Dictionary

### Catalog Level (Root)

| Property      | Type   | Description                                                |
| ------------- | ------ | ---------------------------------------------------------- |
| `catalog`     | String | Name of the database.                                      |
| `base_access` | Enum   | Default access level for every schema and table in the DB. |
| `management`  | Object | (Optional) Default DDL settings for the entire DB.         |
| `schemas`     | Array  | List of schema-specific overrides.                         |

### Schema Level

| Property      | Type    | Description                                                                    |
| ------------- | ------- | ------------------------------------------------------------------------------ |
| `schema_name` | String  | Name of the schema.                                                            |
| `base_access` | Enum    | (Optional) Overrides the Catalog-level default for this schema.                |
| `all_tables`  | Boolean | If `true`, applies the `base_access` to all tables in the schema.              |
| `management`  | Object  | (Optional) Overrides the Catalog-level DDL settings.                           |
| `overrides`   | Object  | Contains `read_only`, `read_write`, `append`, `granular`, and `denied` arrays. |

---

## 3. Comprehensive Example Policy

This policy demonstrates a multi-database environment with global defaults and
surgical exceptions.

```json
{
  "policy_id": "global_enterprise_policy_2026",
  "version": "1.3",
  "permissions": [
    {
      "catalog": "production_db",
      "base_access": "read",
      "management": { "allow_ddl": false, "allow_index": false },
      "comment": "Default to read-only for safety.",
      "schemas": [
        {
          "schema_name": "app_data",
          "base_access": "read_write",
          "all_tables": true,
          "management": { "allow_index": true },
          "overrides": {
            "denied": ["user_passwords", "credit_card_numbers"],
            "read_only": ["app_config_immutable"]
          }
        },
        {
          "schema_name": "reporting_sandbox",
          "base_access": "full_dml",
          "all_tables": true,
          "management": { "allow_ddl": true, "can_drop": true },
          "comment": "Full control within this specific schema only."
        }
      ]
    },
    {
      "catalog": "staging_db",
      "base_access": "read_write",
      "management": {
        "allow_ddl": true,
        "allow_index": true
      },
      "comment": "Broad access for developers in staging environment.",
      "schemas": [
        {
          "schema_name": "finance_audit",
          "base_access": "read",
          "all_tables": true,
          "management": { "allow_ddl": false },
          "overrides": {
            "granular": [
              {
                "tables": ["audit_trail"],
                "actions": ["SELECT", "INSERT"],
                "comment": "Promotion to append-only for the audit table."
              }
            ]
          }
        }
      ]
    }
  ]
}
```

---

## 4. Implementation Rules & Logic Flow

The Policy Engine must resolve permissions in the following order to ensure
security:

1. **Catalog Defaulting**: Start with the `catalog.base_access`.
2. **Schema Merging**: If a schema entry exists, its `base_access` and
   `management` settings overwrite the catalog defaults.
3. **Table Resolution**:

- Apply `GRANT` for the resolved `base_access` to all tables in the schema.
- Process `overrides` arrays to specific tables.
- **Crucial**: Tables in the `denied` array must receive an explicit
  `REVOKE ALL PRIVILEGES` to override any schema-level grants.

4. **DDL Execution**:

- If `allow_ddl` is true, the script must ensure the target role is either the
  `OWNER` of the table or a member of a role with `CREATE` privileges on the
  schema.
