---
name: plydb
description: Skill for using the PlyDB CLI to perform SQL analysis of connected data sources. Use for SQL queries across heterogeneous databases and files such as Postgres, MySQL, CSV, Parquet, JSON, Excel. Triggers on "plydb", "sql", "query", "data analysis", "parquet", "csv", "excel", "database".
---

# PlyDB CLI skill

The `plydb` CLI can be used to query across heterogenous data sources. Look in
`assets/` for a pre-built binary for your OS and architecture.

## Dependencies

The `plydb` binary must be available.

## Instructions

### Configure data sources

First, the data sources to make available to PlyDB must be configured in a
config file as per the specification in
`references\database_connection_config_schema.md`.

### Query with SQL

Once you have a data source config file, PlyDB can query across all of the
configured data sources. Use fully qualified table names: catalog.schema.table.

```sh
./plydb query \
  --config path/to/config/file/config.json \
  "SELECT * FROM customers.default.customers c
   JOIN orders.default.orders o
   ON c.id = o.customer_id"
```

### Fetching semantic context of the data

To provide context to understand the domain and write correct SQL - PlyDB can
build and provide semantic context from database `COMMENT` metadata alongside
column types and foreign keys as structured YAML that follows the
[Open Semantic Interchange (OSI)](https://github.com/open-semantic-interchange/OSI)
specification.

```sh
./plydb scan-context --config path/to/config/file/config.json
```
