# Connect to CSV + PostgreSQL

This example demonstrates querying across a local CSV file and a PostgreSQL database using PlyDB.

## Data Sources

- **products** (CSV) — Product catalog with id, name, category, and price.
- **store** (PostgreSQL) — Store database with `customers` and `orders` tables.

Orders reference `product_id` which maps to the CSV product catalog, so you can join across both sources in a single query.

## Prerequisites

- Docker installed and running
- PlyDB binary built (`go build .` from the project root)

## Setup

### 1. Start PostgreSQL

Start and pre-seed the database with data

```bash
docker run -d \
  --rm \
  --name plydb-postgres \
  -e POSTGRES_USER=plydb \
  -e POSTGRES_PASSWORD=plydb \
  -e POSTGRES_DB=store \
  -p 5432:5432 \
  -v $PWD/examples/connect_to_csv_and_postgres/seed.sql:/docker-entrypoint-initdb.d/seed.sql \
  postgres:17-alpine
```

If needed, access the database via psql like so:

```bash
docker exec -it plydb-postgres psql -U plydb -d store
```

### 2. Set the Password Environment Variable

```bash
export PLYDB_PG_PASSWORD=plydb
```

## Usage

### Example 1: Cross-Source Query with the CLI

Join orders from PostgreSQL with the product catalog from CSV to see total revenue per product:

```bash
./plydb query \
  "SELECT
      p.name AS product,
      p.category,
      SUM(o.quantity) AS total_quantity,
      SUM(o.quantity * p.price) AS total_revenue
   FROM store.public.orders o
   JOIN products.default.\"table\" p ON o.product_id = p.id
   GROUP BY p.name, p.category
   ORDER BY total_revenue DESC" \
  --config examples/connect_to_csv_and_postgres/config.json
```

Expected output:

```
{
  "success": true,
  "columns": ["product", "category", "total_quantity", "total_revenue"],
  "column_types": ["VARCHAR", "VARCHAR", "HUGEINT", "DOUBLE"],
  "rows": [
    ["Thingamajig", "Electronics", 4, 199.96],
    ["Gadget", "Electronics", 3, 74.97],
    ["Widget", "Hardware", 7, 69.93],
    ["Sprocket", "Hardware", 10, 27.5],
    ["Doohickey", "Hardware", 1, 4.5]
  ],
  "row_count": 5,
  "truncated": false
}
```

### Example 2: MCP Server over stdio

Start the MCP server with the stdio transport:

```bash
./plydb mcp --config examples/connect_to_csv_and_postgres/config.json --transport stdio
```

The server reads JSON-RPC messages from stdin. In another terminal (or by piping input), send an `initialize` request followed by a `tools/call` request:

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"demo","version":"0.1.0"}}}
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"query","arguments":{"sql":"SELECT c.name, c.city, p.name AS product, o.quantity, o.order_date FROM store.public.orders o JOIN store.public.customers c ON o.customer_id = c.id JOIN products.default.\"table\" p ON o.product_id = p.id ORDER BY o.order_date DESC LIMIT 5"}}}' \
  | ./plydb mcp --config examples/connect_to_csv_and_postgres/config.json --transport stdio
```

The server responds with JSON-RPC messages on stdout. The `tools/call` response contains a `QueryResult` JSON object with columns, rows, and metadata.
