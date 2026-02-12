# Connecting Claude Desktop to Nexus

## Building or Installation

To build Nexus

```
go build -o nexus .
```

TODO: downloadable binary, mcpb packaging

## Quick Start

### 1. Configure your data sources

Create a `config.yaml` file to configure your data sources.

TODO: instructions based on examples/connect_to_csv_and_postgres/README.md

### 2. Connecting to Your AI Agent

To connect an AI agent, like **Claude** or **ChatGPT**, to Nexus, follow the MCP
connection instructions for your preferred AI agent:

- [Claude](https://support.claude.com/en/articles/11175166-getting-started-with-custom-connectors-using-remote-mcp)
- [ChatGPT](https://platform.openai.com/docs/guides/developer-mode)
- [OpenCode](https://opencode.ai/docs/mcp-servers/)
- [Gemini](https://geminicli.com/docs/tools/mcp-server/)

Once configured, the agent will have access to tools that allow it to list
tables, describe columns, and execute SQL across all defined data sources
autonomously.

TODO: walk through of example of Claude Desktop setup with MCP on STDIO
transport - editing claude config file, starting claude Desktop

```json
{
  "mcpServers": {
    "nexus": {
      "command": "/path/to/nexus",
      "args": ["mcp", "--config", "/path/to/config/file/config.json"],
      "env": {
        "NEXUS_PG_PASSWORD": "nexus"
      }
    }
  }
}
```

TODO: a few demo prompts user can try in Claude Desktop
