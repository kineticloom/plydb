# Connecting Claude Desktop to PlyDB

This tutorial walks you through connecting
[Claude Desktop](https://claude.ai/download) to PlyDB so that Claude can
autonomously query your data using SQL. By the end you will have Claude
answering natural-language questions against a pair of CSV files — no database
server required.

## Prerequisites

- **Go+** installed ([download](https://go.dev/dl/))
- **Claude Desktop** installed ([download](https://claude.ai/download))

## Quick Start

### 1. Install or Build PlyDB

[Install or build](/README.md#installation) `plydb` if you have not already.

### 2. Understand the sample data

This tutorial uses two CSV files that ship with the repository under
`examples/connect_to_csv_files/`:

**customers.csv** — five customers with contact info:

| id  | name  | email             | city     |
| --- | ----- | ----------------- | -------- |
| 1   | Alice | alice@example.com | Seattle  |
| 2   | Bob   | bob@example.com   | Portland |
| 3   | Carol | carol@example.com | Seattle  |
| 4   | Dave  | dave@example.com  | Denver   |
| 5   | Eve   | eve@example.com   | Portland |

**orders.csv** — six orders referencing those customers:

| id  | customer_id | product   | amount | order_date |
| --- | ----------- | --------- | ------ | ---------- |
| 1   | 1           | Widget    | 2      | 2026-01-15 |
| 2   | 3           | Gadget    | 1      | 2026-01-20 |
| 3   | 2           | Widget    | 5      | 2026-02-01 |
| 4   | 1           | Doohickey | 3      | 2026-02-05 |
| 5   | 5           | Gadget    | 2      | 2026-02-08 |
| 6   | 4           | Widget    | 1      | 2026-02-10 |

### 3. Set up a PlyDB config file

The config file tells PlyDB which data sources to allow.

Make a copy of the example config file to edit for this demo.

```sh
mkdir demo_sandbox
cp examples/connect_to_csv_files/config.json demo_sandbox/my_config.json
```

> **Important:** All paths must be **absolute**. Relative paths will not work
> because Claude Desktop does not run from the PlyDB project directory.

So that Claude Desktop knows the exact location of your files, we will need to
make a slight edit `demo_sandbox/my_config.json` - changing the `path` values
from relative paths to absolute paths.

It should look something like this when you're done editing it:

```json
{
  "credentials": {},
  "databases": {
    "customers": {
      "metadata": {
        "name": "Customers",
        "description": "Customer contact information."
      },
      "type": "file",
      "path": "/absolute/path/to/plydb/examples/connect_to_csv_files/customers.csv",
      "format": "csv"
    },
    "orders": {
      "metadata": {
        "name": "Orders",
        "description": "Customer order history."
      },
      "type": "file",
      "path": "/absolute/path/to/plydb/examples/connect_to_csv_files/orders.csv",
      "format": "csv"
    }
  }
}
```

Each key under `databases` is mapped to a **catalog** in SQL. CSV files are
registered as a table named `"table"` under the `default` schema, so the
fully-qualified table names are:

- `customers.default."table"` — the customers data
- `orders.default."table"` — the orders data

### 4. (Optional) Test with the CLI

Before connecting Claude Desktop, you can verify everything works from the
command line:

```bash
plydb query \
  'SELECT * FROM customers.default."table" LIMIT 3' \
  --config demo_sandbox/my_config.json
```

You should see tab-separated output with the first three customers.

### 5. Configure Claude Desktop

Open the Claude Desktop configuration file in your editor. The file location
depends on your OS:

| OS      | Path                                                              |
| ------- | ----------------------------------------------------------------- |
| macOS   | `~/Library/Application Support/Claude/claude_desktop_config.json` |
| Windows | `%APPDATA%\Claude\claude_desktop_config.json`                     |

> **Tip:** You can also open it from Claude Desktop via **Settings > Developer >
> Edit Config**.

```json
{
  "mcpServers": {
    "plydb": {
      "command": "plydb",
      "args": [
        "mcp",
        "--config",
        "/absolute/path/to/plydb/demo_sandbox/my_config.json"
      ]
    }
  }
}
```

> **Important:** All paths must be **absolute**. Relative paths will not work
> because Claude Desktop does not run from the PlyDB project directory.

### 6. Restart Claude Desktop

Quit Claude Desktop completely and reopen it. On the new-chat screen you should
see a **Plus (+) icon** on the bottom-left of the message input area. Click it,
and look under **Connectors** to confirm that the `plydb` is listed and enabled.

If `plydb` does not appear, check the MCP server logs:

| OS      | Log path                                     |
| ------- | -------------------------------------------- |
| macOS   | `~/Library/Logs/Claude/mcp-server-plydb.log` |
| Windows | `%APPDATA%\Claude\logs\mcp-server-plydb.log` |

### 7. Try it out

Start a new conversation in Claude Desktop and try the prompts below. Claude
will use the PlyDB MCP tools to discover schemas and run SQL against your CSV
files autonomously.

---

**Explore the data:**

> What data sources are available? List all tables and their columns.

Claude will call `get_semantic_context` and/or the `query` tool to inspect the
available schemas and describe what data is present.

---

**Simple query:**

> How many customers are in each city?

Expected result: Seattle 2, Portland 2, Denver 1.

---

**Cross-source join:**

> Which customer placed the most orders? Show their name, email, and total
> number of orders.

Claude will join `customers.default."table"` with `orders.default."table"` on
`customer_id` and aggregate the results. (Answer: Alice, with 2 orders.)

---

**Analytical question:**

> What is the total amount ordered for each product? Rank them from highest to
> lowest.

Expected result: Widget 8, Doohickey 3, Gadget 3.

---

**Open-ended analysis:**

> Analyze the order data and give me insights about purchasing trends — which
> products are popular, which cities generate the most orders, and any patterns
> you notice.

Claude will run multiple queries, correlate the results, and provide a narrative
summary of trends across the dataset.

## Next Steps

- **Add a database:** See the
  [CSV + PostgreSQL example](../connect_to_csv_and_postgres/README.md) to learn
  how to connect a PostgreSQL database alongside file sources and run
  cross-source joins.
- **Semantic context:** See the
  [semantic context scanning example](../semantic_context_scanning/README.md) to
  learn how to annotate your data sources with descriptions that help AI agents
  understand your schema.
- **Other AI agents:** PlyDB works with any MCP-compatible client. See the main
  [README](../../README.md) for links to setup guides for ChatGPT, OpenCode, and
  Gemini.
