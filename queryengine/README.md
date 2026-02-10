# queryengine

A Go package that wraps DuckDB (via `github.com/duckdb/duckdb-go/v2`) to provide a unified query engine over heterogeneous data sources.

## Architecture

The engine manages a single in-memory DuckDB instance. On initialization it:

1. **Opens DuckDB** — `sql.Open("duckdb", "")` for an in-memory instance
2. **Loads extensions** — `postgres`, `mysql`, `httpfs` as needed based on configured source types
3. **Configures S3 credentials** — Sets `s3_access_key_id`, `s3_secret_access_key`, `s3_region` via resolved environment variables
4. **Attaches networked databases** — PostgreSQL and MySQL sources are mounted as read-only DuckDB catalogs via `ATTACH`
5. **Records file/S3 entries** — Tracked in `activeConnections` but no ATTACH is needed; users query these via DuckDB table functions

## Design Decisions

- **sqlserver**: Rejected at config parse time with a clear error. DuckDB does not support SQL Server attachment.
- **Single S3 credential profile**: All S3 sources must share one credential profile. DuckDB's S3 configuration is session-global, so multiple credential sets cannot coexist. An error is returned at parse time if multiple profiles differ.
- **No auto-created VIEWs for file/S3**: Users use DuckDB table functions (`read_csv`, `read_parquet`, `read_json`, etc.) directly in their queries.
- **Read-only enforcement**: All `ATTACH` commands include `READ_ONLY` to prevent accidental writes to source databases.
- **Deterministic bootstrapping**: Databases are processed in sorted key order.

## Configuration

The engine accepts a JSON configuration with two top-level objects:

```json
{
  "credentials": {
    "aws-profile": {
      "access_key_env": "AWS_ACCESS_KEY_ID",
      "secret_key_env": "AWS_SECRET_ACCESS_KEY"
    }
  },
  "databases": {
    "db-prod": {
      "metadata": { "name": "Production DB", "description": "Read replica" },
      "type": "postgresql",
      "host": "db.example.com",
      "port": 5432,
      "database_name": "analytics",
      "username": "reader",
      "password_env_var": "DB_PROD_PASSWORD"
    },
    "local-csv": {
      "metadata": { "name": "Report", "description": "Local CSV file" },
      "type": "file",
      "path": "/data/report.csv",
      "format": "csv",
      "delimiter": ",",
      "header_row": true
    },
    "s3-data": {
      "metadata": { "name": "S3 Data", "description": "Parquet on S3" },
      "type": "s3",
      "uri": "s3://bucket/data/*.parquet",
      "credential_profile": "aws-profile",
      "region": "us-east-1",
      "format": "parquet"
    }
  }
}
```

### Supported types

| Type | Required fields |
|------|----------------|
| `postgresql` | `host`, `port`, `database_name`, `username`, `password_env_var` |
| `mysql` | `host`, `port`, `database_name`, `username`, `password_env_var` |
| `file` | `path` |
| `s3` | `uri`, `credential_profile`, `region`, `format` |
| `sqlserver` | Not supported — rejected at parse time |

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

rows, err := engine.Query(ctx, `SELECT * FROM "db-prod".public.users LIMIT 10`)
```

## Query Interface

- `ParseConfig(data []byte) (*Config, error)` — Parse and validate JSON configuration
- `New(cfg *Config) (*QueryEngine, error)` — Create and bootstrap the engine
- `(*QueryEngine).Query(ctx, sql) (*sql.Rows, error)` — Execute a SQL query
- `(*QueryEngine).Close() error` — Shut down the DuckDB instance

## Error Handling

- **Missing environment variables**: If a `password_env_var`, `access_key_env`, or `secret_key_env` references an unset or empty environment variable, `New` returns an error.
- **Connection validation**: After `ATTACH`, the engine runs a validation query. Unreachable hosts cause `New` to fail.
- **Cleanup on failure**: If any step in `New` fails, the DuckDB connection is closed before returning the error.

## Limitations

- SQL Server (`sqlserver`) is not supported
- All S3 sources must use the same credential profile (DuckDB limitation: session-global S3 config)
- File and S3 sources are not auto-mounted as views; use DuckDB table functions directly
