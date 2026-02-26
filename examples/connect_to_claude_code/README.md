# Connecting Claude Code to PlyDB

> Pro tip: Once the
> [PlyDB Agent Skill](/README.md#ai-agents--plydb-via-cli-agent-skill) is
> installed, you can ask Claude Code to help you write and evolve your PlyDB
> config files instead of editing them manually.

This tutorial walks you through connecting
[Claude Code](https://claude.ai/claude-code) to PlyDB so that Claude can
autonomously query your data using SQL. Unlike the MCP approach used with Claude
Desktop, Claude Code can integrate with PlyDB through the **Agent Skill + CLI**
— a more dynamic workflow that lets Claude adapt its own configuration on the
fly.

By the end you will have Claude Code answering natural-language questions
against a pair of CSV files — no database server required.

## Why CLI instead of MCP?

Both integration paths work with Claude Code. The CLI approach has a few
advantages worth knowing about:

- **Dynamic reconfiguration:** Claude can edit PlyDB config files, swap data
  sources, or evolve
  [semantic context overlays](../semantic_context_scanning/README.md) between
  queries — all without restarting anything. MCP servers, by contrast, require a
  restart to pick up configuration changes.
- **No sandboxing restrictions:** Claude Code can run any tool available on your
  system (with your permission). Unlike some agent environments, such as Claude
  Cowork, there are no network restrictions that would prevent PlyDB from
  reaching databases or cloud-hosted data sources.
- **Richer agent context:** The PlyDB Agent Skill teaches Claude how to use
  `plydb` CLI commands, understand PlyDB config files, and work with semantic
  context overlays — so Claude can reason about your PlyDB setup.

See the [FAQ](/FAQ.md#should-i-use-mcp-or-cli) for a fuller comparison.

## Prerequisites

- **PlyDB** installed ([installation instructions](/README.md#installation))
- **Claude Code** installed ([download](https://claude.ai/claude-code))

## Quick Start

### 1. Understand the sample data

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

### 2. Set up a PlyDB config file

The config file tells PlyDB which data sources to allow.

Make a copy of the example config file to edit for this demo:

```sh
mkdir demo_sandbox
cp examples/connect_to_csv_files/config.json demo_sandbox/my_config.json
```

Because Claude Code runs from your project directory, **relative paths work
fine**. The config file you just copied is ready to use as-is:

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
      "format": "csv"
    },
    "orders": {
      "metadata": {
        "name": "Orders",
        "description": "Customer order history."
      },
      "type": "file",
      "path": "examples/connect_to_csv_files/orders.csv",
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

### 3. Test with the CLI

Verify everything works from the command line before involving Claude:

```bash
plydb query \
  'SELECT * FROM customers.default."table" LIMIT 3' \
  --config demo_sandbox/my_config.json
```

You should see tab-separated output with the first three customers.

### 4. Install the PlyDB Agent Skill

Download the Agent Skill bundle (`plydb_skill.zip`) from the
[Releases](https://github.com/kineticloom/plydb/releases) page, then follow
[Claude Code's skill installation instructions](https://code.claude.com/docs/en/skills)
to add it to your Claude Code setup.

The skill gives Claude Code built-in knowledge of:

- The `plydb query` and `plydb semantic-context` CLI commands and their flags
- The PlyDB [config file schema](/specs/config_schema.md), and how to write them
- How to read and write
  [semantic context overlays](/examples/semantic_context_scanning/README.md#layering-additional-semantic-context)

> **Tip:** You do not need the skill installed to run `plydb` CLI commands —
> Claude Code can invoke any binary on your system. The skill simply gives
> Claude guidance on how to configure and use PlyDB.

### 5. Try it out

Open a Claude Code session and try the prompts below. Claude will invoke
`plydb query` as a shell command, discover schemas, and run SQL against your CSV
files autonomously.

---

**Explore the data:**

> What data sources are available in demo_sandbox/my_config.json? List all
> tables and their columns.

Claude will run `plydb semantic-context` or a series of `plydb query` calls to
inspect the available schemas and describe what data is present.

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

---

**Dynamic reconfiguration (CLI advantage):**

> I have another CSV at `examples/connect_to_csv_and_postgres/products.csv`. Add
> it to my PlyDB config as a new data source and then query it to see what
> columns it has.

Because Claude Code runs CLI commands directly, it can edit `my_config.json` and
immediately run a new query against the updated config — no restart required.
This kind of on-the-fly reconfiguration is not possible with a running MCP
server.

---

**Distill learnings into a semantic context overlay:**

After a data analysis session, Claude has built up an understanding of what your
data actually means — which columns matter, how tables relate, what quirks
exist. You can ask it to write that understanding into a
[semantic context overlay](/examples/semantic_context_scanning/README.md#layering-additional-semantic-context)
file so future sessions start with that context already in place:

> Based on what you've learned about this data, write or update a semantic
> context overlay file that records your learnings and update my config file to
> reference it.

Claude will produce an
[Open Semantic Interchange (OSI)](https://github.com/open-semantic-interchange/OSI)
YAML file capturing descriptions and relationships. You can then reference it in
your config so PlyDB automatically includes the enriched context on every
subsequent query:

```json
{
  "credentials": {},
  "databases": { ... },
  "semantic_context": {
    "overlays": [
      "demo_sandbox/overlay.yaml"
    ]
  }
}
```

Or pass it as a flag if you prefer to keep the config file unchanged:

```bash
plydb query \
  'SELECT * FROM customers.default."table"' \
  --config demo_sandbox/my_config.json \
  --semantic-context-overlay demo_sandbox/overlay.yaml
```

Over time, overlays accumulate the institutional knowledge your agent builds up
about your data.

## Next Steps

- **Add a database:** See the
  [CSV + PostgreSQL example](../connect_to_csv_and_postgres/README.md) to learn
  how to connect a PostgreSQL database alongside file sources and run
  cross-source joins.
- **Semantic context:** See the
  [semantic context scanning example](../semantic_context_scanning/README.md) to
  learn how to annotate your data sources with descriptions that help Claude
  understand your schema. After a data analysis session, ask Claude Code to
  distill its learnings into a semantic context overlay file for future
  sessions.
- **MCP alternative:** If you prefer a persistent server rather than CLI calls,
  see the [Claude Desktop tutorial](../connect_to_claude_desktop/README.md) for
  the MCP-based setup.
