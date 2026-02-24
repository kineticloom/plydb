# Connect to Google Sheets

This example demonstrates querying a Google Sheets spreadsheet with PlyDB.

Two authentication methods are supported:

- **Service account key file** - for server-side, non-interactive use.
- **Browser-based OAuth** - for interactive/ad-hoc use. DuckDB opens a browser
  for Google login; no credentials needed in the config.

## Prerequisites

- [Install or build](/README.md#installation) `plydb` if you have not already.
- A Google Sheets spreadsheet with data you want to query. You'll need the
  spreadsheet ID from the URL:
  `https://docs.google.com/spreadsheets/d/SPREADSHEET_ID/edit`

## Setup

### Option A: Service Account Authentication

Use this method for automated or server-side access.

1. Create a service account in the
   [Google Cloud Console](https://console.cloud.google.com/iam-admin/serviceaccounts)
   and download the JSON key file.
2. Share the spreadsheet with the service account's email address (found in the
   key file under `client_email`).
3. Edit `config_service_account.json`:
   - Set `key_file` to the path to your downloaded JSON key file.
   - Set `spreadsheet_id` to your spreadsheet's ID.
   - Set `sheet_name` to the tab you want to query (or remove it to use the
     table name from your SQL query).

### Option B: Browser OAuth Authentication

Use this method for interactive, ad-hoc querying.

1. Edit `config_browser_oauth.json`:
   - Set `spreadsheet_id` to your spreadsheet's ID.
2. On first query, DuckDB will open your browser for Google login. No service
   account or key file is needed.
3. The OAuth token is automatically persisted to `~/.duckdb/stored_secrets/`.
   Subsequent runs reuse the cached token without re-authenticating.
4. To force re-authentication, delete the cached secret:
   `rm -rf ~/.duckdb/stored_secrets/`

## Usage

### Query with Service Account

```bash
plydb query \
  --config examples/connect_to_google_sheets/config_service_account.json \
  "SELECT * FROM sales.default.data LIMIT 10"
```

The `sheet_name` in the config is set to `Q1`, so regardless of the table name
used in the SQL (`data` above), PlyDB reads from the `Q1` tab.

### Query with Browser OAuth

```bash
plydb query \
  --config examples/connect_to_google_sheets/config_browser_oauth.json \
  "SELECT * FROM sales.default.\"Q1\" LIMIT 10"
```

Since no `sheet_name` is set in the config, PlyDB uses the table name from the
SQL query as the sheet name. Here `Q1` is used as both the SQL table name and
the Google Sheets tab name.

### Dynamic Sheet Names

When `sheet_name` is omitted from the config, the table name in your SQL query
determines which tab is read. This lets you query different tabs without
changing the config:

```bash
# Read from the "January" tab
plydb query \
  --config examples/connect_to_google_sheets/config_browser_oauth.json \
  "SELECT * FROM sales.default.\"January\""

# Read from the "February" tab
plydb query \
  --config examples/connect_to_google_sheets/config_browser_oauth.json \
  "SELECT * FROM sales.default.\"February\""
```

**Note on case sensitivity:** SQL lowercases unquoted identifiers per the SQL
standard. If your Google Sheets tab name contains uppercase letters (e.g.,
`Revenue`), an unquoted reference like `FROM sales.default.Revenue` will look
for a tab named `revenue`, which won't match. Two workarounds:

- **Quote the identifier** with double quotes to preserve case:
  `FROM sales.default."Revenue"`
- **Set `sheet_name` in the config** to bypass SQL identifier parsing entirely.

### Cross-Source Join

Google Sheets sources can be joined with any other PlyDB data source. For
example, to join a Google Sheet with a local CSV, add both to the same config
file and query across them:

```bash
plydb query \
  --config your_config.json \
  "SELECT s.product, s.revenue, t.target
   FROM sales.default.\"Q1\" AS s
   JOIN targets.default.targets AS t ON s.product = t.product"
```
