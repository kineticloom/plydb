# queryengine

A Go package that wraps DuckDB (via `github.com/duckdb/duckdb-go/v2`) to provide
a unified query engine over heterogeneous data sources.

## Architecture

The engine manages a single in-memory DuckDB instance. On initialization it:

1. **Opens DuckDB** — `sql.Open("duckdb", "")` for an in-memory instance
2. **Loads extensions** — `postgres`, `mysql`, `sqlite`, `httpfs` as needed
   based on configured source types
3. **Configures S3 credentials** — Sets `s3_access_key_id`,
   `s3_secret_access_key`, `s3_region` via resolved environment variables
4. **Attaches databases** — PostgreSQL, MySQL, and SQLite sources are mounted as
   read-only DuckDB catalogs via `ATTACH`
5. **Records file/S3 entries** — Tracked in `activeConnections` but no ATTACH is
   needed; users query these via DuckDB table functions

## Design Decisions

- **Single S3 credential profile**: All S3 sources must share one credential
  profile. DuckDB's S3 configuration is session-global, so multiple credential
  sets cannot coexist. An error is returned at parse time if multiple profiles
  differ.
- **No auto-created VIEWs for file/S3**: Users use DuckDB table functions
  (`read_csv`, `read_parquet`, `read_json`, etc.) directly in their queries.
- **Read-only enforcement**: All `ATTACH` commands include `READ_ONLY` to
  prevent accidental writes to source databases.
- **SQLite type affinity**: DuckDB maps SQLite column types through SQLite's
  type affinity rules (e.g., `TIMESTAMP` → `VARCHAR`). Time dimension detection
  does not apply to SQLite sources.
- **Deterministic bootstrapping**: Databases are processed in sorted key order.

## Configuration

The engine accepts JSON configuration per the
[config schema specification](/specs/config_schema.md).

````

### Supported types

| Type         | Required fields                                                 | Notes                                                  |
| ------------ | --------------------------------------------------------------- | ------------------------------------------------------ |
| `postgresql` | `host`, `port`, `database_name`, `username`, `password_env_var` | ATTACHed as DuckDB catalog                             |
| `mysql`      | `host`, `port`, `database_name`, `username`, `password_env_var` | ATTACHed as DuckDB catalog                             |
| `sqlite`     | `path`                                                          | ATTACHed as DuckDB catalog; default schema is `main`   |
| `file`       | `path`                                                          | CSV, Parquet, XLSX, JSON                               |
| `s3`         | `uri`, `credential_profile`, `region`, `format`                 | Glob patterns supported                                |
| `gsheet`     | `spreadsheet_id`                                                | Optional `credential_profile` for service account auth |

## Usage

```go
data, _ := os.ReadFile("connections.json")
cfg, err := queryengine.ParseConfig(data)
if err != nil {
    log.Fatal(err)
}

engine, err := queryengine.New(cfg)
if err != nil {
    log.Fatal(err)
}
defer engine.Close()

// PostgreSQL: catalog.schema.table
rows, err := engine.Query(ctx, `SELECT * FROM "db-prod".public.users LIMIT 10`)

// SQLite: catalog.main.table  (SQLite's default schema is always "main")
rows, err := engine.Query(ctx, `SELECT * FROM "app-sqlite".main.events LIMIT 10`)
````

## Query Interface

- `ParseConfig(data []byte) (*Config, error)` — Parse and validate JSON
  configuration
- `New(cfg *Config) (*QueryEngine, error)` — Create and bootstrap the engine
- `(*QueryEngine).Query(ctx, sql) (*sql.Rows, error)` — Execute a SQL query
- `(*QueryEngine).Close() error` — Shut down the DuckDB instance

## Error Handling

- **Missing environment variables**: If a `password_env_var`, `access_key_env`,
  or `secret_key_env` references an unset or empty environment variable, `New`
  returns an error.
- **Connection validation**: After `ATTACH`, the engine runs a validation query.
  Unreachable hosts cause `New` to fail.
- **Cleanup on failure**: If any step in `New` fails, the DuckDB connection is
  closed before returning the error.

## Limitations

- All S3 sources must use the same credential profile (DuckDB limitation:
  session-global S3 config)
- File and S3 sources are not auto-mounted as views; use DuckDB table functions
  directly
- SQLite columns do not carry typed date/time information — DuckDB resolves all
  SQLite types through type affinity, mapping `TIMESTAMP` to `VARCHAR`
