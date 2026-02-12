# Nexus: The Unified Database Gateway for AI Agents

Unlock the power of AI-driven analytics.

Nexus is a lightweight
[Model Context Protocol (MCP)](https://modelcontextprotocol.io) gateway that
bridges your AI agents with your data. It provides a single, unified SQL
interface to query across databases and files - including Postgres, MySQL, CSV,
Excel, and Parquet - effortlessly.

Get up and running in minutes without complex infrastructure dependencies.

```
 ┌─────────────────┐ SQL ┌──────────────────────────────────────┐
 │     AI Agent    │────▶│               Nexus                  │
 │(Claude, ChatGPT,│◀────│                                      │
 │ etc.)           │     │  ┌─────────────────────────────────┐ │
 └─────────────────┘     │  │          Query Engine           │ │
                         │  │   (Planning, Optimization &     │ │
                         │  │         Execution)              │ │
                         │  └──────────────┬──────────────────┘ │
                         │                 │                    │
                         │    ┌────────────┼────────────┐       │
                         │    ▼            ▼            ▼       │
                         │ ┌─────────┐ ┌─────────┐   ┌───────┐  │
                         │ │Postgres │ │  MySQL  │   │ Files │  │
                         │ │Connector│ │Connector│   │CSV/etc│  │
                         │ └───┬─────┘ └────┬────┘   └───┬───┘  │
                         └─────┼────────────┼────────────┼──────┘
                               ▼            ▼            ▼
                         ┌──────────┐ ┌──────────┐ ┌──────────┐
                         │PostgreSQL│ │  MySQL   │ │S3/Local  │
                         │ Database │ │ Database │ │  Files   │
                         └──────────┘ └──────────┘ └──────────┘

```

## Why Nexus?

- **Agentic Data Analysis:** Unleash the full potential of AI agents by allowing
  them to write sophisticated SQL and perform complex data analysis
  autonomously. Agents use MCP tools to discover tables, inspect schemas, and
  understand your data's semantics.
- **Zero ETL (Query In-Place):** Eliminate the need for expensive and brittle
  ETL pipelines. Nexus lets you query your data exactly where it lives-whether
  it's a production database, a cloud-hosted spreadsheet, or a data lake.
- **Cross-Source Queries** - Join tables across MySQL, PostgreSQL, CSV, and more
  in a single query
- **Operational Simplicity:** Designed to be up and running in minutes without
  additional infrastructure dependencies.
- **Deploy Anywhere:** Run it locally for personal productivity or deploy it as
  a stateless service in the cloud (AWS, GCP, Azure) to power enterprise-grade
  agentic workflows.
- **Open Source & Extensible:** Built on an open-source foundation, Nexus
  ensures transparency, security, and no vendor lock-in. Easily extend the
  gateway with custom connectors or contribute to the community-driven core.

---

## Use Cases

Because Nexus exposes schema metadata and relational power via MCP, capable AI
agents can perform iterative reasoning - querying data, finding patterns,
generating visualizations, and recommending specific business actions.

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

Nexus abstracts the complexity of different storage formats into a single
relational view:

| Category               | Supported Sources                 |
| :--------------------- | :-------------------------------- |
| **SQL Databases**      | PostgreSQL, MySQL, SQLite         |
| **File Formats**       | CSV, JSON, Parquet, Excel (.xlsx) |
| **Cloud / SaaS** (WIP) | Google Sheets, S3                 |
| **Data Lake** (WIP)    | Apache Iceberg, Delta Lake        |

## Quick start

TODO: link to examples

## Testing

Run unit tests for a package

```
go test ./somepackage/...
```

Run integration tests for a package (requires Docker)

```
go test -tags=integration -v -timeout 300s ./somepackage/...
```

## Contributing

TODO
