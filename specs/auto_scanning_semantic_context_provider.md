# Auto scanning semantic context provider

The auto scanning semantic context provider is intended to provide AI's with
context about the semantics of underlying connected data sources made available
by the service so the AI has context to write SQL queries.

For an example of types of data sources, see the spec for configuration of data
sources in specs/database_connection_config_schema.md.

At a high level, the auto scanning semantic context provider should:

1. Automatically harvest the metadata it is able to from what's available from
   the underlying data sources. For example: Postgres table and column metadata
   and COMMENT annotations, CSV headers, JSON fields, Parquet columns.
2. Output a Struct that can be later serialized to yaml that conforms to the OSI
   spec (https://github.com/open-semantic-interchange/OSI). OSI essentially
   requires a mapping between Physical Data (tables/columns) and Logical
   Concepts (entities/metrics).

## Implementation notes

- In this project, the queryengine package internally leverages
  github.com/duckdb/duckdb-go/v2 to connect to and query underlying data
  sources. The queryengine package may already contain the foundations for
  building semantic context harvesting functionality.
- Out of scope: later, we will want to layer in additional semantic context on
  top of what is automatically scanned and harvested by the auto scanning
  semantic context provider. For example: User provided keys and values that add
  to or override what was automatically harvested. Or additional automatically
  collected context from other types of semantic context providers. Multiple
  semantic context providers should be chainable and composable, working
  together to construct the final output data.
