# PlyDB: The Universal Database Gateway for AI Agents

PlyDB securely bridges the gap between your AI agents and your fragmented data
sources. It provides a single controlled access point for AI agents to query
databases and flat files, such as Postgres, MySQL, CSV, Excel, and Parquet, with
SQL, wherever the data lives.

PlyDB is:

- **Simple:** Get up and running in minutes on your personal computer. No
  complex infrastructure or heavy dependencies. Connect your agents to the data
  they need, wherever it lives - no data movement (ETL) required.
- **Secure:** You choose which data sources your agents are allowed to access.
  Read-only by default.
- **Versatile** Integrate as either a CLI or
  [Model Context Protocol (MCP)](https://modelcontextprotocol.io) server.

---

```
 ┌─────────────────┐ SQL ┌──────────────────────────────────────┐
 │     AI Agent    │────▶│               PlyDB                  │
 │(Claude, ChatGPT,│◀────│                                      │
 │ etc.)           │     │  ┌─────────────────────────────────┐ │
 └─────────────────┘     │  │          Query Engine           │ │
                         │  │   (Access control, Planning,    │ │
                         │  │    Optimization & Execution)    │ │
                         │  └───────────────┬─────────────────┘ │
                         │                  │                   │
                         │     ┌────────────┼────────────┐      │
                         └─────┼────────────┼────────────┼──────┘
                               ▼            ▼            ▼
                          ┌──────────┐ ┌──────────┐ ┌──────────┐
                          │PostgreSQL│ │  MySQL   │ │ S3/Local │
                          │ Database │ │ Database │ │  Files   │
                          └──────────┘ └──────────┘ └──────────┘
```

---

## Empowerment Meets Security

- **Agentic Data Analysis:** Unleash the full potential of AI agents by allowing
  them to write sophisticated SQL and perform complex data analysis
  autonomously. Agents can use either the PlyDB CLI or MCP server to discover
  tables, inspect schemas, and understand your data's semantics.
- **Zero ETL (Query In-Place):** Eliminate the need for expensive and brittle
  ETL pipelines. PlyDB lets your agents query your data exactly where it lives -
  whether it's a production database, a cloud-hosted spreadsheet, or a data
  lake.
- **Read-Only Guardrails:** PlyDB is a read-only "look, don't touch" system by
  default. Your AI can analyze information and find patterns, but it cannot
  delete, edit, or alter your original records unless you explicitly allow it
  to.
- **Cross-Source Queries:** Join tables across MySQL, PostgreSQL, CSV, and more
  in a single query
- **Operational Simplicity:** Designed to be up and running in minutes without
  additional infrastructure dependencies.
- **Deploy Anywhere:** Run it locally for personal productivity or deploy it as
  a stateless service in the cloud (AWS, GCP, Azure) to power enterprise-grade
  agentic workflows.
- **Open Source & Extensible:** Built on an open-source foundation, PlyDB
  ensures transparency, security, and no vendor lock-in. Easily extend the
  gateway with custom connectors or contribute to the community-driven core.

---

## Turning Data into Decisions

When your AI agent has a secure, real-time view of your data, it evolves from a
chatbot into a **Strategic Partner**.

### Strategic Sales & Retention

**Prompt:** "Analyze our top 20 accounts by revenue. Cross-reference their
support tickets with their recent product usage. **Generate a churn-risk
dashboard** and draft personalized 'Value Review' emails for the three accounts
with the lowest activity."

### Marketing Performance & Optimization

**Prompt:** "Look at our ad spend across Google and Facebook, then compare it
with our actual transaction data. **Create a chart showing the ROAS trend** over
the last 90 days and identify which specific campaign we should move budget into
to maximize next month's yield."

### Revenue Operations (RevOps)

**Prompt:** "Audit our active seat counts against our signed contracts in the
Google Sheet. **Identify all overages**, calculate the total unbilled revenue,
and build a summary table that the billing team can use to issue invoices."

### Executive Insights

**Prompt:** "I need a high-level summary of our business health. Pull the MRR
from the CRM, the infrastructure costs from our logs, and the headcount from the
HR spreadsheet. **Build a financial health dashboard** and suggest three areas
where we can improve our operating margin."

### Genomics & Bioinformatics

**Prompt:** "Query the variant_calls table to find the top 10 most frequent SNPs
found in samples labeled as 'resistant' to Penicillin, excluding variants found
in the 'control' group."

### Public Health & Epidemiology

**Prompt:** "Calculate the rolling 12-month average of ER admissions for asthma
by zip code"

---

## Supported Data Sources

PlyDB abstracts the complexity of different storage formats into a single
relational view:

| Category               | Supported Sources                 |
| :--------------------- | :-------------------------------- |
| **SQL Databases**      | PostgreSQL, MySQL, SQLite         |
| **File Formats**       | CSV, JSON, Parquet, Excel (.xlsx) |
| **Cloud / SaaS** (WIP) | Google Sheets, S3                 |
| **Data Lake** (WIP)    | Apache Iceberg, Delta Lake        |

## Connecting AI agents to PlyDB via MCP

To connect an AI agent to PlyDB via MCP, follow the step-by-step tutorial for
your preferred client:

- **[Claude Desktop](examples/connect_to_claude_desktop/README.md)** - full
  walkthrough querying CSV files in Claude Desktop via MCP

PlyDB uses the [Model Context Protocol](https://modelcontextprotocol.io) and
works with any MCP-compatible client. For other agents, see their MCP setup
guides:

- [ChatGPT](https://platform.openai.com/docs/guides/developer-mode)
- [OpenCode](https://opencode.ai/docs/mcp-servers/)
- [Gemini](https://geminicli.com/docs/tools/mcp-server/)

## Connecting AI agents to PlyDB via CLI

TODO

## Configuring data sources

Examples of configuring data sources:

- [Query CSV files](examples/connect_to_csv_files/README.md)
- [Query CSV files + PostgreSQL](examples/connect_to_csv_and_postgres/README.md)
- [Providing semantic context](examples/semantic_context_scanning/README.md)

## Build

To build a `plydb` binary:

```sh
go build .
```

## Testing

To run unit tests:

```sh
go test ./...
```

To run integration tests (requires Docker):

```sh
go test -tags=integration -v -timeout 300s ./...
```

## Contributing

TODO

## License

TODO
