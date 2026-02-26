# Connect to CSV files

> Pro tip: If you have the
> [PlyDB Agent Skill](/README.md#ai-agents--plydb-via-cli-agent-skill)
> installed, you can ask your agent to work with you on data source
> configuration instead of writing a config file manually.

This is a simple example that connects two CSV files as data sources.

[Install or build](/README.md#installation) `plydb` if you have not already.

```sh
go build .
```

For this example, we will use this PlyDB config file
`examples/connect_to_csv_files/config.json` which configures two CSV files as
data sources.

With this config file, PlyDB can run queries that span across both of the CSV's.
For example, here's a query that returns data from the two CSV's joined on
customer id:

```
plydb query \
  --config examples/connect_to_csv_files/config.json \
  "SELECT * FROM customers.default.customers c
   JOIN orders.default.orders o
   ON c.id = o.customer_id"
```

Expected output:

```
{
  "success": true,
  "columns": [
    "id",
    "name",
    "email",
    "city",
    "id",
    "customer_id",
    "product",
    "amount",
    "order_date"
  ],
  "column_types": [
    "BIGINT",
    "VARCHAR",
    "VARCHAR",
    "VARCHAR",
    "BIGINT",
    "BIGINT",
    "VARCHAR",
    "BIGINT",
    "DATE"
  ],
  "rows": [
    [
      1,
      "Alice",
      "alice@example.com",
      "Seattle",
      4,
      1,
      "Doohickey",
      3,
      "2026-02-05T00:00:00Z"
    ],
    [
      2,
      "Bob",
      "bob@example.com",
      "Portland",
      3,
      2,
      "Widget",
      5,
      "2026-02-01T00:00:00Z"
    ],
    [
      3,
      "Carol",
      "carol@example.com",
      "Seattle",
      2,
      3,
      "Gadget",
      1,
      "2026-01-20T00:00:00Z"
    ],
    [
      4,
      "Dave",
      "dave@example.com",
      "Denver",
      6,
      4,
      "Widget",
      1,
      "2026-02-10T00:00:00Z"
    ],
    [
      5,
      "Eve",
      "eve@example.com",
      "Portland",
      5,
      5,
      "Gadget",
      2,
      "2026-02-08T00:00:00Z"
    ],
    [
      1,
      "Alice",
      "alice@example.com",
      "Seattle",
      1,
      1,
      "Widget",
      2,
      "2026-01-15T00:00:00Z"
    ]
  ],
  "row_count": 6,
  "truncated": false
}
```

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
