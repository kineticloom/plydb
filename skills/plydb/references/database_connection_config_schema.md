# Database connections schema

This specification defines a JSON-based schema for managing a heterogeneous
registry of data sources. It supports traditional networked databases, local
flat files, and cloud-hosted objects (S3) with support for shared credential
profiles and globbing patterns.

---

## 1. Structural Overview

The schema is composed of two primary top-level objects:

1. **`credentials`**: A map of authentication profiles used by cloud providers.
2. **`databases`**: A map of data source configurations where the key is a
   unique identifier.

---

## 2. Field Definitions

### 2.1 The `credentials` Object

Stores shared authentication metadata to prevent redundancy across multiple
sources.

| Field            | Type   | Description                                                 |
| ---------------- | ------ | ----------------------------------------------------------- |
| `access_key_env` | String | Name of the environment variable for the Access Key ID.     |
| `secret_key_env` | String | Name of the environment variable for the Secret Access Key. |

### 2.2 The `databases` Object

Each entry in this map contains common metadata and type-specific connection
details.

#### A. Common Fields (All Types)

- **`metadata`**: Object containing `name` (String) and `description` (String).
- **`type`**: String. One of: `postgresql`, `mysql`, `sqlserver`, `file`, or
  `s3`.

#### B. Networked Database Fields (`type: "postgresql" | "mysql" | etc.`)

- **`host`**: Server address.
- **`port`**: Network port (Integer).
- **`database_name`**: The target schema/database name.
- **`username`**: Login identity.
- **`password_env_var`**: Name of the env var holding the secret.

#### C. Local File Fields (`type: "file"`)

- **`path`**: Unix-style path to the file.
- **`format`**: (Optional if inferred from extension) `csv`, `xlsx`, `parquet`,
  `json`.
- **`delimiter`**: (CSV only) The separator character.
- **`header_row`**: (CSV/XLSX) Boolean; indicates if row 1 is the header.
- **`sheet_name`**: (XLSX only) The tab to read.

#### D. S3 Cloud Storage Fields (`type: "s3"`)

- **`uri`**: S3 URI (e.g., `s3://bucket/path/`). Supports globbing patterns
  (`*`, `?`, `[]`).
- **`credential_profile`**: Key matching an entry in the top-level `credentials`
  map.
- **`region`**: AWS region (e.g., `us-east-1`).
- **`format`**: **Required.** The file format (`csv`, `parquet`, etc.).
- **`delimiter` / `header_row` / `sheet_name**`: Same as Local File fields.

---

## 3. Reference Implementation

```json
{
  "credentials": {
    "aws-marketing-user": {
      "access_key_env": "AWS_ACCESS_KEY_ID",
      "secret_key_env": "AWS_SECRET_ACCESS_KEY"
    }
  },
  "databases": {
    "db-prod-analytics": {
      "metadata": {
        "name": "Production Analytics",
        "description": "Primary read-replica for data warehousing."
      },
      "type": "postgresql",
      "host": "db-prod-01.example.com",
      "port": 5432,
      "database_name": "analytics_main",
      "username": "bi_user",
      "password_env_var": "DB_PROD_PASSWORD"
    },
    "local-budget-report": {
      "metadata": {
        "name": "FY2026 Budget Plan",
        "description": "Local Excel workbook for department budget allocations."
      },
      "type": "file",
      "path": "/Users/sarah/Documents/Finance/budget_2026.xlsx",
      "sheet_name": "Final_Approval",
      "header_row": true
    },
    "s3-sensor-data-glob": {
      "metadata": {
        "name": "IoT Sensor Data",
        "description": "Partitioned sensor data using glob patterns."
      },
      "type": "s3",
      "credential_profile": "aws-marketing-user",
      "uri": "s3://iot-bucket/2026/*/device_id_100?/sensor_*.parquet",
      "format": "parquet",
      "region": "us-west-2"
    },
    "inventory-snapshot": {
      "metadata": {
        "name": "Warehouse Inventory",
        "description": "Global inventory levels exported as CSV to S3."
      },
      "type": "s3",
      "credential_profile": "aws-marketing-user",
      "uri": "s3://corporate-reports/inventory/daily_snapshot.csv",
      "format": "csv",
      "delimiter": ",",
      "header_row": true,
      "region": "us-east-1"
    }
  }
}
```
