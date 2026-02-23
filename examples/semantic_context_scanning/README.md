# Semantic Context Scanning

This example demonstrates how PlyDB uses PostgreSQL `COMMENT` metadata to give
AI agents the semantic context they need to understand your data. When an agent
calls the `get_semantic_context` MCP tool, PlyDB introspects the configured data
sources and returns a structured YAML description of every table, column, and
relationship — including any human-written comments that explain what the data
actually means.

## Why This Matters

The schema here tracks a fictional multidimensional power grid. The terminology
is intentionally esoteric (e.g. `vortex_anchor`, `flux_telemetry`,
`syn_link_01`) so an LLM cannot rely on "common sense" to generate queries.

PostgreSQL `COMMENT` statements provide the semantic mapping. PlyDB extracts
these comments alongside column types and foreign keys, producing structured
YAML that follows the
[Open Semantic Interchange (OSI)](https://github.com/open-semantic-interchange/OSI)
specification. This gives the agent the context it needs to understand the
domain and write correct SQL.

## Data Sources

- **grid** (PostgreSQL) — `vortex_anchor`, `flux_telemetry`, and `syn_link_01`
  tables with `COMMENT` metadata describing the fictional domain.

## Prerequisites

- Docker installed and running
- [Install or build](/README.md#installation) `plydb` if you have not already.

## Setup

### 1. Start PostgreSQL

Start and pre-seed the database with the esoteric schema and data:

```bash
docker run -d \
  --rm \
  --name plydb-postgres \
  -e POSTGRES_USER=plydb \
  -e POSTGRES_PASSWORD=plydb \
  -e POSTGRES_DB=grid \
  -p 5432:5432 \
  -v $PWD/examples/semantic_context_scanning/seed.sql:/docker-entrypoint-initdb.d/seed.sql \
  postgres:17-alpine
```

If needed, access the database via psql like so:

```bash
docker exec -it plydb-postgres psql -U plydb -d grid
```

### 2. Set the Password Environment Variable

```bash
export PLYDB_PG_PASSWORD=plydb
```

## Getting Semantic Context via MCP

When PlyDB runs as an MCP server, it provides a `get_semantic_context` tool
alongside the `query` tool. An AI agent (Claude, ChatGPT, etc.) can call
`get_semantic_context` at any time to retrieve a full semantic model of the
configured data sources. The agent then uses this context to understand
domain-specific terminology and write accurate SQL.

For example, after connecting Claude Desktop to this example's config (see the
[Claude Desktop tutorial](../connect_to_claude_desktop/README.md) for setup
steps), Claude would:

1. Call `get_semantic_context` to learn that `oscill_rate` means "the frequency
   of energy vibration" and that `syn_link_01` "maps the entanglement between
   two different vortex anchors."
2. Use that understanding to translate natural-language questions into correct
   SQL via the `query` tool.

### Example Prompts

Try asking an agent connected to this data source:

1. **"Which energy anchor is currently at risk of collapsing?"** — Requires
   joining `vortex_anchor` to `flux_telemetry` and comparing
   `stability_threshold` to `entropy_delta`.
2. **"What is the average vibration frequency of Obsidian-Nine over the last
   hour?"** — Requires mapping "vibration frequency" to `oscill_rate`.
3. **"List all pairs of nodes that have a high ability to share energy."** —
   Requires identifying that `conductivity_ratio` in `syn_link_01` represents
   "sharing energy".

Without the semantic context from `COMMENT` metadata, an agent would have no way
to connect these natural-language concepts to the underlying column names.

## Getting Semantic Context via CLI

Instead of MCP, you or an AI agent can also get the semantic context via the
PlyDB CLI `semantic-context` command directly:

```bash
plydb semantic-context --config examples/semantic_context_scanning/config.json
```

This outputs the same OSI YAML that the MCP tool returns to agents:

```yaml
semantic_model:
  name: Auto-scanned Semantic Model
  datasets:
    - name: grid.public.flux_telemetry
      description: Time-series log of multidimensional energy fluctuations.
      source: grid.public.flux_telemetry
      fields:
        - name: telemetry_id
          expression:
            dialects:
              - dialect: ANSI_SQL
                expression: telemetry_id
        - name: anchor_ref
          expression:
            dialects:
              - dialect: ANSI_SQL
                expression: anchor_ref
        - name: recorded_at
          expression:
            dialects:
              - dialect: ANSI_SQL
                expression: recorded_at
          dimension:
            is_time: true
        - name: oscill_rate
          description:
            The frequency of energy vibration. Optimal range is between 400 and
            600 mHz.
          expression:
            dialects:
              - dialect: ANSI_SQL
                expression: oscill_rate
        - name: entropy_delta
          description:
            The rate of energy decay. Positive values indicate system leakage.
          expression:
            dialects:
              - dialect: ANSI_SQL
                expression: entropy_delta
    - name: grid.public.syn_link_01
      description:
        Maps the entanglement between two different vortex anchors. High
        conductivity allows for energy sharing.
      source: grid.public.syn_link_01
      fields:
        - name: link_id
          expression:
            dialects:
              - dialect: ANSI_SQL
                expression: link_id
        - name: alpha_node
          expression:
            dialects:
              - dialect: ANSI_SQL
                expression: alpha_node
        - name: beta_node
          expression:
            dialects:
              - dialect: ANSI_SQL
                expression: beta_node
        - name: conductivity_ratio
          expression:
            dialects:
              - dialect: ANSI_SQL
                expression: conductivity_ratio
    - name: grid.public.vortex_anchor
      description:
        Primary stability points for aetheric harvesting. Anchors must remain
        above their stability_threshold to prevent collapse.
      source: grid.public.vortex_anchor
      fields:
        - name: anchor_id
          expression:
            dialects:
              - dialect: ANSI_SQL
                expression: anchor_id
        - name: designation
          description: The unique resonant name of the anchor.
          expression:
            dialects:
              - dialect: ANSI_SQL
                expression: designation
        - name: stability_threshold
          expression:
            dialects:
              - dialect: ANSI_SQL
                expression: stability_threshold
```

Notice how the PostgreSQL `COMMENT` metadata appears as `description` fields in
the YAML output. Without these comments, an LLM would have no way to know that
`oscill_rate` is "the frequency of energy vibration" or that `syn_link_01` "maps
the entanglement between two different vortex anchors."

## Layering Additional Semantic Context

Auto-scanning captures what the database already knows. The
`--semantic-context-overlay` flag lets you supply one or more OSI YAML files
that enrich the auto-scanned model with additional descriptions, relationships,
and metrics — without changing the source database.

**Constraints:** overlays cannot add new datasets (tables) or new fields
(columns). They only enrich what was already discovered by the scanner.

### Example overlay

The file [`overlay.yaml`](overlay.yaml) in this directory is a ready-to-use
example. It adds a description to `flux_telemetry.anchor_ref`, defines the
relationship between `flux_telemetry` and `vortex_anchor`, and adds an
`avg_entropy` metric.

### CLI usage

```bash
plydb semantic-context \
  --config examples/semantic_context_scanning/config.json \
  --semantic-context-overlay examples/semantic_context_scanning/overlay.yaml
```

The flag is repeatable — multiple overlays are applied in order:

```bash
plydb semantic-context \
  --config examples/semantic_context_scanning/config.json \
  --semantic-context-overlay base_overlay.yaml \
  --semantic-context-overlay team_overlay.yaml
```

### MCP usage

The same flag works with `plydb mcp`:

```bash
plydb mcp \
  --config examples/semantic_context_scanning/config.json \
  --semantic-context-overlay examples/semantic_context_scanning/overlay.yaml
```

When an agent calls `get_semantic_context`, it receives the enriched model with
the overlay applied.
