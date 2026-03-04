# PlyDB: The Universal Database Gateway for AI Agents

- [Website](https://www.plydb.com/)
- [Documentation](https://www.plydb.com/docs/)

---

Real-time conversational analytics with zero data movement. Bridge your AI to
live sources without the ETL tax. Immediate insights, where your data lives.

PlyDB is a secure, unified access point for AI agents to query data in-place.
From SQL databases like Postgres and MySQL, flat files like CSV and Excel, or
cloud sources like Google Sheets and S3, PlyDB can query across them all with
standard SQL - no data warehouse necessary.

PlyDB is:

- **Simple:** Deploy in minutes on a local machine with no additional
  infrastructure. Connect your agents to live data without the friction of
  building ETL pipelines.
- **Secure:** Control which data sources your agents can access. Read-only by
  default.
- **Versatile:** Query across different [data sources](#supported-data-sources)
  through a single interface. Integrate your agents via
  [CLI](#ai-agents--plydb-via-cli-agent-skill) or
  [Model Context Protocol (MCP)](https://modelcontextprotocol.io).

---

## Demo

PlyDB can connect with a wide variety of [data sources](#supported-data-sources)
and integrate with any AI agent that supports
[CLI tools](#ai-agents--plydb-via-cli-agent-skill) or
[MCP](#ai-agents--plydb-via-mcp).

Here is a demo with Claude Desktop + PlyDB + Revenue Data:

<div align="center">
  <video src="https://github.com/user-attachments/assets/1d11208f-502e-4436-865b-7413196ed861" width="400" controls muted autoplay loop>
  </video>
</div>

---

## Where does PlyDB fit in?

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

## Why PlyDB?

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
  in a single query.
- **Operational Simplicity:** Designed to be up and running in minutes without
  additional infrastructure dependencies.
- **Open Source & Extensible:** Built on an open-source foundation, PlyDB
  ensures transparency, security, and no vendor lock-in.

---

## Example Use Cases

When your AI agent has a secure, real-time view of your data, it evolves from a
chatbot into a **Strategic Partner** — one that can answer complex questions
about your business the moment you ask them.

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

### DevOps

**Prompt:** "Review our app logs in S3 for errors from the past week.
Cross-reference with our codebase and database replica to identify affected
customers and diagnose root causes. **Open PRs with fixes** for the most severe
issues, and draft a summary our PM and CSM teams can use."

### Genomics & Bioinformatics

**Prompt:** "Query the variant_calls table to find the top 10 most frequent SNPs
found in samples labeled as 'resistant' to Penicillin, excluding variants found
in the 'control' group."

### Public Health & Epidemiology

**Prompt:** "Calculate the rolling 12-month average of ER admissions for asthma
by zip code".

---

## Supported Data Sources

PlyDB abstracts the complexity of different storage formats into a single
relational view:

| Category                | Supported Sources                 |
| :---------------------- | :-------------------------------- |
| **SQL Databases**       | PostgreSQL, MySQL, SQLite, DuckDB |
| **File Formats**        | CSV, JSON, Parquet, Excel (.xlsx) |
| **Object Storage**      | S3                                |
| **SaaS**                | Google Sheets                     |
| **Data Lake** (planned) | Apache Iceberg, Delta Lake        |

---

## Installation

### Quick install (macOS / Linux)

```sh
curl -fsSL https://raw.githubusercontent.com/kineticloom/plydb/main/install.sh | sh
```

### Quick install (Windows — PowerShell)

```powershell
irm https://raw.githubusercontent.com/kineticloom/plydb/main/install.ps1 | iex
```

### Options

| Variable            | Description                            | Default        |
| :------------------ | :------------------------------------- | :------------- |
| `PLYDB_INSTALL_DIR` | Where to place the binary              | `~/.local/bin` |
| `PLYDB_VERSION`     | Version tag to install (e.g. `v0.1.0`) | latest         |

### Manual download

Pre-built binaries for all platforms are available on the
[Releases](https://github.com/kineticloom/plydb/releases) page.

### Build from source

```sh
make build
```

---

## AI agents + PlyDB via MCP

> Deciding between MCP vs CLI? See the [FAQ](/FAQ.md#should-i-use-mcp-or-cli)
> for recommendations.

AI agents can connect to PlyDB via [MCP](https://modelcontextprotocol.io).

Install PlyDB (we recommend using the [quick install](#installation) process).

Then follow your specific agent's instructions for configuring MCP:

- [Claude Desktop + PlyDB tutorial](examples/connect_to_claude_desktop/README.md) -
  full tutorial querying CSV files in Claude Desktop via MCP
- [ChatGPT](https://platform.openai.com/docs/guides/developer-mode)
- [OpenCode](https://opencode.ai/docs/mcp-servers/)
- [Gemini](https://geminicli.com/docs/tools/mcp-server/)

## AI agents + PlyDB via CLI Agent Skill

> Deciding between MCP vs CLI? See the [FAQ](/FAQ.md#should-i-use-mcp-or-cli)
> for recommendations.

AI agents can also use PlyDB directly via the `plydb` CLI when provided context
on how to do so via an [Agent Skill](https://agentskills.io).

Install PlyDB (we recommend using the [quick install](#installation) process).

Then download the Agent Skill bundle
([plydb_skill.zip](https://github.com/kineticloom/plydb/releases)) and follow
your specific agent's instructions for installing skills:

- [Claude Code + PlyDB tutorial](examples/connect_to_claude_code/README.md) -
  full tutorial querying CSV files in Claude Code via Agent Skill + CLI
- [Claude Code](https://code.claude.com/docs/en/skills) - Agent Skill
  configuration
- [Claude](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/overview#claude-ai)
- [Codex](https://developers.openai.com/codex/skills)
- [Gemini CLI](https://geminicli.com/docs/cli/skills/)
- [OpenClaw](https://docs.openclaw.ai/tools/skills)
- [OpenCode](https://opencode.ai/docs/skills/)

## Configuring data sources

> Pro tip: If you have the
> [PlyDB Agent Skill](#ai-agents--plydb-via-cli-agent-skill) installed, you can
> ask your agent to work with you on data source configuration instead of
> writing a config file manually.

Data sources are configured via the
[PlyDB config file](/specs/config_schema.md).

You can configure more than one type of data source in a config file, depending
on your needs, and query across all of them.

To guide your AI agent's understanding of the semantics of your data, PlyDB can
automatically scan your data sources and provide your AI agent with
[semantic context](/examples/semantic_context_scanning/README.md) - schema,
tables, columns, and comment metadata. You can further enrich this context by
[overlaying](/examples/semantic_context_scanning/README.md#layering-additional-semantic-context)
your own descriptions or AI-generated annotations.

Examples of configuring data sources:

- [Query CSV files](examples/connect_to_csv_files/README.md)
- [Query CSV files + PostgreSQL](examples/connect_to_csv_and_postgres/README.md)
- [Query Google Sheets](examples/connect_to_google_sheets/README.md)
- [Query DuckDB databases](examples/connect_to_duckdb/README.md)
- [Query SQLite databases](examples/connect_to_sqlite/README.md)
- [Providing semantic context](examples/semantic_context_scanning/README.md)

## FAQ

- [Do I need to write SQL myself or can my AI agent do that?](/FAQ.md#do-i-need-to-write-sql-myself-or-can-my-ai-agent-do-that)
- [How organized should my data be?](/FAQ.md#how-organized-should-my-data-be)
- [More...](/FAQ.md)

## Contributing

We love contributions! However, before getting too deep into implementation,
please first check our [Roadmap](/TODO.md) and start a discussion so we can
align on a direction.

Features to improve PlyDB for individuals can and should be a part of this open
source version, while features aimed to improve usage at scale, as in an
enterprise, should be reserved for the proprietary version. Doing so helps keep
the project sustainable. If you aren't sure where a feature fits, feel free to
open a discussion first!

See [CONTRIBUTING.md](/CONTRIBUTING.md) for details.

## License

This project is licensed under the Apache License Version 2.0. See the
[LICENSE](/LICENSE) file for details.

All code contributed prior to 02/23/2026 is also licensed under Apache License
Version 2.0.
