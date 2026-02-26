# FAQ

## Do I need to write SQL myself or can my AI agent do that?

While you are welcome to manually use PlyDB with your hand crafted SQL, AI
agents can be very good at writing SQL too - exploring, understanding, and
analysing datasets - even if your data is
[less organized](#how-organized-should-my-data-be) than you'd like it to be.

## Should I use MCP or CLI?

It depends. There are tradeoffs and limitations depending on your situation.

Having an AI agent use PlyDB as a
[CLI tool](/README.md#ai-agents--plydb-via-cli-agent-skill) can be a more
dynamic workflow. For example, AI agents can reconfigure `plydb` between calls —
changing config files, or evolving semantic context overlays on the fly. In
contrast, PlyDB's MCP server will need to be restarted to change its
configuration. Usually this means restarting your agent, too. (Though we might
[address this](/TODO.md#feat---dynamic-data-source-configuration) in the
future.)

Your choice may also be informed by what your agent's limitations are around CLI
tool calling. For example, OpenClaw allows very permissive access to your
system, while Claude Cowork operates in an isolated VM sandbox that strictly
limits allowed tools and outbound networking, and Claude Code is somewhere in
the middle.

See also:

- [Does the PlyDB CLI work with Claude Code?](#does-the-plydb-cli-work-with-claude-code)
- [Does the PlyDB CLI work with Claude Cowork?](#does-the-plydb-cli-work-with-claude-cowork)

## How organized should my data be?

Even if your data isn't well organized, you may be pleasantly surprised at how
capable AI agents are at making sense of it. We suggest you give it a shot!

Pro tip: After a session of data analysis, ask your AI agent to distill its
learnings about your data's semantics and
[write a semantic context overlay](#can-i-have-an-ai-agent-write-my-plydb-semantic-context-overlays-for-me)
to record its findings for future sessions.

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

Try asking your AI agent to write an overlay file after a data analysis session
— it's a particularly good opportunity to capture learnings for future sessions.

## Does the PlyDB CLI work with Claude Cowork?

Yes, but with some limitations when using PlyDB via the CLI (not MCP).

Limitations:

- Claude Cowork's security sandboxing requires that the `plydb` binary be
  available in a directory that you've granted Claude access to. For example,
  you can place it in your project's workspace.
- Claude Cowork's sandboxing also limits network access from within the sandbox.
  This means PlyDB is not able to connect to networked data sources or download
  extensions for its data connectors.

When working with local CSV files as data sources, you should not run into the
networking restrictions.

However, when connecting to networked data sources such as Postgres, MySQL, S3,
or Google Sheets, it is recommended that you integrate PlyDB via
[MCP](/README.md#ai-agents--plydb-via-mcp) instead of CLI when using Claude
Cowork.

## Does the PlyDB CLI work with Claude Code?

Yes. Claude Code does not use the same
[security sandboxing](#does-the-plydb-cli-work-with-claude-cowork) as Claude
Cowork. Claude Code can run any tool available on your system (with your
permission).
