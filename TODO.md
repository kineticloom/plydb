# TODO list for PlyDB

## Category reference

The following tags are used in this document to categorize the type of work that
needs to be done:

- FEAT: New functionality
- FIX: Solving a bug
- DOCS: Documentation only
- TECH DEBT: Code change that neither fixes a bug nor adds a feature

## Short to mid term priorities

### DOCS - PlyDB CLI Skill

Add Skill for users that choose to integrate PlyDB with their AI agent via CLI
instead of MCP.

### FEAT - PlyDB read-write access controls

See specs/table_access_policy.md and sqlwalk.Validate for foundations. End user
code paths will need to be wired up and tested. Data sources are currently
hardcoded to read-only by default. Due to duckdb constraints, some types of data
sources will only support read-only (e.g. CSV), while others may support
read-write (e.g. Postgres). The configuration interface for end users may need
some discovery and thought - to find a good balance between ease of use and
flexibility.

### FEAT - Reduce friction of install and config of PlyDB with AI agents

The current MCP setup process involves editing a handful of json config files in
a text editor. This isn't ideal for less technical users. Look into better
distribution and config options. See mcpb
https://github.com/modelcontextprotocol/mcpb which seems to be Claude's answer
to this challenge, however it fairly new and it remains to be seen if the
ecosystem has embraced this standard.

Note, there is also an alternative way for agents to use PlyDB - via CLI,
instead of MCP.

Related to:

- [Dynamic data source configuration](#feat---dynamic-data-source-configuration)
- [Auto-updates](#feat---auto-updates)
- [PlyDB CLI Skill](#docs---plydb-cli-skill)

### FEAT - Auto-updates

See auto update functionality of mcpb:
https://github.com/modelcontextprotocol/mcpb

Related to:

- [Reduce friction of install and config of PlyDB with AI agents](#feat---reduce-friction-of-install-and-config-of-plydb-with-ai-agents)

### FEAT - Build and release binaries

Release binaries to GitHub. Remember to look into signing best practices.

### DOCS - Licensing

Decide on licensing approach and update repo with appropriate docs.

### FEAT - Connecting to Cloud storage systems (S3, Iceberg)

Some foundations for this are in place, but this functionality has not yet been
tested and validated as working.

## Backlog priority

Stories are in the backlog because the need or requirements are not yet clear.
They're documented here so we can monitor and prioritize them for action as we
learn more. They're listed in no particular order.

### TECH DEBT - Add a Makefile

To encapsulate common dev or release tasks

### FEAT - Revisit logging

First the logging requirements need to be better defined.

### FEAT - Advanced data security

Column hiding, column masking, PII scanning. TBD whether which parts of this
should be in scope or out of scope for PlyDB. There are general purpose tools
that do this that can be plugged into the system next to PlyDB.

### FEAT - Semantic context layering

Currently PlyDB automatically scans for semantic context, using table schema,
and table and column `COMMENT` metadata to construct and provide data compliant
with the
[Open Semantic Interchange (OSI)](https://github.com/open-semantic-interchange/OSI)
spec to AI agents.

This is an ok foundation for an AI agent to start understanding the semantics of
the data sources, however there may be cases where additional business semantics
need to be provided that are not available to be automatically scanned.

For example, if a library's database of books has a `checked_out_date` column,
but does not have an `is_overdue` column, we need a mechanism to provide the
additional context that "a book is overdue if it was checked out more than 2
weeks ago".

There are mechanisms in OSI to provide these semantics. However, PlyDB does not
_yet_ have a means of loading user customized OSI. A first pass could be to
layer the auto scanned OSI data with user provided customized OSI data.

A first step here is to better understand users need for this.

### FEAT - Providing semantic context for large datasets

Currently fetching semantic context returns semantic context for all tables and
columns on connected databases and files. However, this may be less than ideal
for scenarios where there are a large number of tables or columns - hundreds to
thousands. Before prioritizing this, we should first monitor whether this is a
concern that real users are surfacing.

### FEAT - Dynamic data source configuration

Currently data sources are configured prior to the execution of PlyDB. This
configuration remains static for the lifecycle of PlyDB. However, there may be
cases where the user will want to configure data sources on the fly during run
time. Web UI, and or MCP tool? Note: Because AI agents can execute tools
autonomously, we will need to consider security if we go the MCP tool route.

Related to:

- [Reduce friction of install and config of PlyDB with AI agents](#feat---reduce-friction-of-install-and-config-of-plydb-with-ai-agents)

### FEAT - Requirements for usage as a remote service

This will require some discovery first. While technically the PlyDB MCP service
can be used remotely via HTTP transport, before remote usage is officially
encouraged, we should revisit concerns important for that use case, particularly
auth, multi-tenancy, control plane, but there may be others.

### FEAT - Reduce friction around connecting to cloud resources in private cloud

Most likely best left out of scope for PlyDB and best handled by an
organization's established practices - e.g. Cloud auth, ssh tunneling, zero
trust networking solutions, etc.
