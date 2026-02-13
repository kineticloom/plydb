# Query engine connection lifecycle

This specification defines the connection architecture for a Go-based query
engine using `github.com/duckdb/duckdb-go/v2`. The scope is strictly limited to
**session initialization, extension loading, and remote source attachment**.

---

## 1. Engine Core Structure

The engine manages a single persistent DuckDB instance. It resolves
configuration secrets into an active state before accepting queries.

```go
type QueryEngine struct {
    // Internal sql.DB handle using the duckdb driver
    db *sql.DB

    // Tracks which extensions and remote attachments are active
    activeConnections map[string]ConnectionType
}

type ConnectionType string
const (
    ConnPostgres ConnectionType = "postgresql"
    ConnMySQL    ConnectionType = "mysql"
    ConnS3       ConnectionType = "s3"
    ConnFile     ConnectionType = "file"
)

```

---

## 2. Bootstrapping Sequence

When the engine is initialized with the connection config, it must perform these
four steps in order:

### 2.1 Driver Initialization

The engine opens a connection using the `duckdb` driver. In-memory mode is
preferred for a query engine unless persistence of metadata is required.

```go
db, err := sql.Open("duckdb", "") // In-memory

```

### 2.2 Extension Pre-loading

DuckDB requires specific extensions to handle networked databases and cloud
storage. The engine executes these as raw SQL commands immediately after
opening:

- `INSTALL httpfs; LOAD httpfs;` (For S3 and HTTP)
- `INSTALL postgres; LOAD postgres;` (For PostgreSQL)
- `INSTALL mysql; LOAD mysql;` (For MySQL)

### 2.3 Authentication & Secrets Resolution

The engine iterates through the `credentials` and `databases` objects to resolve
environment variables into the DuckDB session.

#### A. S3/Cloud Secrets

For each `s3` source, the engine maps the `credential_profile` to the
corresponding `credentials` entry and sets global session variables:

```sql
SET s3_access_key_id='<resolved_access_key>';
SET s3_secret_access_key='<resolved_secret_key>';
SET s3_region='<region_field>';

```

#### B. Networked DB Secrets

For `postgresql` and `mysql`, the `password_env_var` is read. These are not set
globally but injected into the `ATTACH` string.

### 2.4 Remote Database Attachment

The engine loops through the `databases` map and "mounts" networked databases as
internal DuckDB catalogs.

| Type           | Connection String Format                                                        |
| -------------- | ------------------------------------------------------------------------------- |
| **PostgreSQL** | `ATTACH 'host=H port=P dbname=D user=U password=PWD' AS <key> (TYPE POSTGRES);` |
| **MySQL**      | `ATTACH 'host=H port=P user=U password=PWD database=D' AS <key> (TYPE MYSQL);`  |

> **Note:** The `<key>` is the unique identifier from your JSON (e.g.,
> `db-prod-analytics`). Users will query these using
> `SELECT * FROM "db-prod-analytics".table_name`.

---

## 3. Configuration Mapping Table

The following table defines how your JSON spec fields map to DuckDB's internal
connection parameters.

| Spec Field         | DuckDB Implementation Method            | Scope                        |
| ------------------ | --------------------------------------- | ---------------------------- |
| `type: s3`         | `httpfs` extension + `SET` variables    | Session-wide / Profile-based |
| `type: postgresql` | `postgres` extension + `ATTACH` command | Per-Database (as a Schema)   |
| `type: mysql`      | `mysql` extension + `ATTACH` command    | Per-Database (as a Schema)   |
| `type: file`       | Direct filesystem access                | Native                       |

---

## 4. The Functional Interface

The engine provides a single primary method for executing raw SQL queries.

```go
// Query executes the provided SQL.
// Because of the ATTACH step, the user can now join across sources:
// e.g., "SELECT * FROM 'db-prod-analytics'.users JOIN 's3_data_view'..."
func (e *QueryEngine) Query(ctx context.Context, sqlQuery string) (*sql.Rows, error) {
    return e.db.QueryContext(ctx, sqlQuery)
}

```

In practice, queries should have been pre-processed as specified by
specs/query_pre_processing.md prior to being used as input for `Query`.

---

## 5. Error Handling & Safety

- **Resolution Errors**: If a `password_env_var` or `access_key_env` is defined
  in the spec but missing in the host OS environment, the `Connect` function
  must return an error and abort.
- **Connection Validation**: After an `ATTACH` command, the engine should run a
  lightweight validation query (e.g., `SELECT 1`) to ensure the remote host is
  reachable.
- **Read-Only Enforcement**: To prevent data corruption on source systems, all
  `ATTACH` commands should include the `READ_ONLY` flag by default.
