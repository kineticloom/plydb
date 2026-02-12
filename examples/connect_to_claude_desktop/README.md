# Connecting Claude Desktop to Nexus

This tutorial walks you through connecting
[Claude Desktop](https://claude.ai/download) to Nexus so that Claude can
autonomously query your data using SQL. By the end you will have Claude
answering natural-language questions against a pair of CSV files — no database
server required.

## Prerequisites

- **Go+** installed ([download](https://go.dev/dl/))
- **Claude Desktop** installed ([download](https://claude.ai/download))

## Quick Start

### 1. Build Nexus

Clone the repository (if you haven't already) and build the binary:

```bash
git clone https://github.com/ypt/experiment-nexus.git
cd experiment-nexus
go build -o nexus .
```

After the build completes you should have a `nexus` binary in the project root.
Verify it works:

```bash
./nexus --help
```

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

### 3. Review the config file

The config file tells Nexus which data sources to expose. Open
`examples/connect_to_csv_files/config.json`:

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
      "path": "examples/connect_to_csv_files/customers.csv",
      "format": "csv",
      "delimiter": ",",
      "header_row": true
    },
    "orders": {
      "metadata": {
        "name": "Orders",
        "description": "Customer order history."
      },
      "type": "file",
      "path": "examples/connect_to_csv_files/orders.csv",
      "format": "csv",
      "delimiter": ",",
      "header_row": true
    }
  }
}
```

Each key under `databases` becomes a **catalog** in SQL. CSV files are exposed
as a table named `"table"` under the `default` schema, so the fully-qualified
table names are:

- `customers.default."table"` — the customers data
- `orders.default."table"` — the orders data

### 4. (Optional) Test with the CLI

Before connecting Claude Desktop, you can verify everything works from the
command line:

```bash
./nexus query \
  'SELECT * FROM customers.default."table" LIMIT 3' \
  --config examples/connect_to_csv_files/config.json
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

Add (or merge) the following into the config file. Replace
`/absolute/path/to/experiment-nexus` with the actual absolute path to your
cloned repository:

```json
{
  "mcpServers": {
    "nexus": {
      "command": "/absolute/path/to/experiment-nexus/nexus",
      "args": [
        "mcp",
        "--config",
        "/absolute/path/to/experiment-nexus/examples/connect_to_csv_files/config.json"
      ]
    }
  }
}
```

For example, if you cloned the repo to `~/projects/experiment-nexus`:

```json
{
  "mcpServers": {
    "nexus": {
      "command": "/Users/you/projects/experiment-nexus/nexus",
      "args": [
        "mcp",
        "--config",
        "/Users/you/projects/experiment-nexus/examples/connect_to_csv_files/config.json"
      ]
    }
  }
}
```

> **Important:** All paths must be **absolute**. Relative paths will not work
> because Claude Desktop does not run from the Nexus project directory.

### 6. Restart Claude Desktop

Quit Claude Desktop completely and reopen it. On the new-chat screen you should
see a **hammer icon** (tools) in the bottom-right of the message input area.
Click it to confirm that the `query` and `get_semantic_context` tools from Nexus
are listed.

If the tools don't appear, check the MCP server logs:

| OS      | Log path                                     |
| ------- | -------------------------------------------- |
| macOS   | `~/Library/Logs/Claude/mcp-server-nexus.log` |
| Windows | `%APPDATA%\Claude\logs\mcp-server-nexus.log` |

### 7. Try it out

Start a new conversation in Claude Desktop and try the prompts below. Claude
will use the Nexus MCP tools to discover schemas and run SQL against your CSV
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
- **Other AI agents:** Nexus works with any MCP-compatible client. See the main
  [README](../../README.md) for links to setup guides for ChatGPT, OpenCode, and
  Gemini.
