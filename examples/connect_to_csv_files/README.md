# Connect to CSV files

This is a simple example that connects two CSV files as data sources.

Build `plydb` if you don't have it already.

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
./plydb query \
  --config examples/connect_to_csv_files/config.json \
  "SELECT * FROM customers.default.customers c
   JOIN orders.default.orders o
   ON c.id = o.customer_id"
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
