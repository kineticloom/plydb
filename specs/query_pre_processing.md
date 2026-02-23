# Query pre-processing

## Overview

A SQL query string goes through a few validation and formatting steps before the
string is finally passed into engine.Query.

### 1. Query validation

Physical table references in MUST reference a table by catalog name, schema
name, and table name - e.g. this is ALLOWED:
`SELECT * FROM my_catalog.my_schema.my_table`, while this is NOT ALLOWED:
`SELECT * FROM my_schema.my_table`. This removes ambiguity in the table
references.

Catalog name values:

The catalog name value is from the database connection configuration described
in the database connection configuration spec (specs/config_schema.md) - it
should be one of the keys under the "databases" key.

Schema name values:

For source systems that support namespacing by schemas, such as Postgres, the
schema value should be from the database itself.

For source systems that do not support namespacing by schemas, such as CSV or
Excel files, the schema value should be a static value, such as "default".

Table name values:

For source systems that support logical tables, such as Postgres, the table name
value should be from the database itself.

For source systems that do not support logical tables, such as CSV files, the
table name should be a static value, such as "table".

For Excel files, the table name should be the name of a sheet in the Excel file.

### 2. Catalog, schema, and table name rewriting

Once the catalog, schema, and table names have been validated, we will need to
rewrite them to align with how duckdb actually queries the underlying data
sources.

For example, if the configuration references a CSV file under the key
databases.my_csv, and the input query queries the CSV like
`SELECT * FROM my_csv.default.table`, then we will need to rewrite this to
duckdb form, e.g. `SELECT * FROM "/path/to/my_csv.csv"`

## Implementation notes

The sqlwalk package in the repo contains core functions that can be used for
query parsing, validation and table name rewriting that can be used for the
foundation of this specific query pre-processing implementation.

## TODO - later

There are plans for more extensive query validation (e.g. permission checks
based on defined policies), but for now, let the scope of query-preprocessing be
the scope that's outlined in this document.
