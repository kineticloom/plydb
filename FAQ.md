# FAQ

## Should I use MCP or CLI?

It depends. There are tradeoffs and limitations depending on your situation.

Having an AI agent use PlyDB as a
[CLI tool](/README.md#ai-agents--plydb-via-cli-agent-skill) can be a more
dynamic workflow. For example, AI agents can reconfigure `plydb` from call to
call - changing config files, or evolving semantic context overlays on the fly.
In contrast, PlyDB's MCP server will need to be restarted to change its
configuration. Usually this means restarting your agent, too. (Though we might
[address this](/TODO.md#feat---dynamic-data-source-configuration) in the future)

Your choice may also be informed by what your agent's limitations are around CLI
tool calling. For example, OpenClaw allows very permissive access to your
system, while Claude Desktop Cowork operates in an isolated VM sandbox that
strictly limits allowed tools and outbound networking, and Claude Code is
somewhere in the middle.

See also:

- [Does the PlyDB CLI work with Claude Desktop Cowork?](#does-the-plydb-cli-work-with-claude-desktop-cowork)
- [Does the PlyDB CLI work with Claude Code?](#does-the-plydb-cli-work-with-claude-code)

## Can I have an AI agent write my PlyDB config file for me?

Yes! Install the
[PlyDB Agent Skill](/README.md#ai-agents--plydb-via-cli-agent-skill) or point
your agent at the [PlyDB config schema](/specs/config_schema.md) and tell it
about the data sources you want to configure.

## Can I have an AI agent write my PlyDB semantic context overlays for me?

Yes! Install the
[PlyDB Agent Skill](/README.md#ai-agents--plydb-via-cli-agent-skill) to teach it
about PlyDB
[semantic context overlays](/examples/semantic_context_scanning/README.md#example-overlay).

Try asking your AI agent write an overlay file after a session of data analysis,
that is a particularly good opportunity to record some learnings into an overlay
file for future sessions.

## Does the PlyDB CLI work with Claude Desktop Cowork?

Yes, but with some limitations when using PlyDB via the CLI (not MCP).

Limitations:

- Claude Desktop Cowork's security sandboxing requires that the `plydb` binary
  be available in a directory that you've granted Claude access to. For example,
  you can place it in your project's workspace.
- Also, Claude Desktop Cowork's sandboxing limits network access from within the
  sandbox. This means that in this scenario, PlyDB is not able to connect to
  networked data sources nor download extensions for its data connectors.

When working with local CVS files as data sources, you should not run into the
networking restrictions.

However, when connecting to networked data sources such as Postgres, MySQL, S3,
or Google Sheets, it is recommended that you integrate PlyDB via
[MCP](/README.md#ai-agents--plydb-via-mcp) instead of CLI.

For more information, you can ask Claude Cowork.

## Does the PlyDB CLI work with Claude Code?

Yes. Claude Code does not use the same
[security sandboxing](#does-the-plydb-cli-work-with-claude-desktop-cowork) as
Claude Desktop Cowork. Claude Code can run any tool available on your system
(with your permission).

Limitations:

- A Google Sheet configured as data source with OAuth requires an interactive
  login workflow. This interactivity does not work with Claude Code's subprocess
  execution. Instead, in this scenario, a Google Sheet data source should be
  configured with a
  [service account token](examples/connect_to_google_sheets/README.md#option-a-service-account-authentication).
