# Connect to a SQLite database

> Pro tip: If you have the
> [PlyDB Agent Skill](/README.md#ai-agents--plydb-via-cli-agent-skill)
> installed, you can ask your agent to work with you on data source
> configuration instead of writing a config file manually.

This example connects a SQLite database file as a data source.

## Prerequisites

- [Install or build](/README.md#installation) `plydb` if you have not already.
- [SQLite](https://sqlite.org/index.html)

---

## 1. Create the sample SQLite database

Use the `sqlite3` CLI to create a database from the provided seed script. The
generated file goes into the gitignored `demo_sandbox/` directory:

```sh
mkdir -p demo_sandbox
sqlite3 demo_sandbox/shop.sqlite < examples/connect_to_sqlite/setup.sql
```

## 2. Query the data

The config file `examples/connect_to_sqlite/config.json` points to the
`demo_sandbox/shop.sqlite` file. SQLite tables live under the `main` schema, so
fully-qualified table names use the form `catalog.main.table`.

```sh
plydb query \
  --config examples/connect_to_sqlite/config.json \
  "SELECT c.name, o.product, o.amount
   FROM shop.main.customers c
   JOIN shop.main.orders o ON c.id = o.customer_id
   ORDER BY c.name, o.product"
```

Expected output:

```
{
  "success": true,
  "columns": ["name", "product", "amount"],
  "column_types": ["VARCHAR", "VARCHAR", "BIGINT"],
  "rows": [
    ["Alice", "Doohickey", 3],
    ["Alice", "Widget", 2],
    ["Bob", "Widget", 5],
    ["Carol", "Gadget", 1],
    ["Dave", "Widget", 1],
    ["Eve", "Gadget", 2]
  ],
  "row_count": 6,
  "truncated": false
}
```

> **Note:** SQLite uses `main` as its default schema, not `public` (PostgreSQL)
> or the database name (MySQL).

## Next Steps

- **Add a database:** See the
  [CSV + PostgreSQL example](../connect_to_csv_and_postgres/README.md) to learn
  how to connect a PostgreSQL database alongside file sources and run
  cross-source joins.
- **Semantic context:** See the
  [semantic context scanning example](../semantic_context_scanning/README.md) to
  learn how to annotate your data sources with descriptions that help AI agents
  understand your schema.
- **Connect to Claude Desktop:** PlyDB works with any MCP-compatible client. See
  the
  [connecting to Claude Desktop to PlyDB example](../connect_to_claude_desktop/README.md)
  for a step by step tutorial that demonstrates how to connect Claude Desktop to
  PlyDB to unlock AI agent powered data analysis.
