// Copyright 2026 Paul Tzen
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"

	"github.com/kineticloom/plydb/cmd"
)

const usage = `Usage: plydb <command> [arguments] [flags]

Commands:
  query              Execute a SQL query
  semantic-context   Scan all data sources and output semantic context as YAML
  mcp                Start an MCP server exposing a SQL query tool
  version            Print version information
  license            Print license information

Run "plydb <command> -h" for command-specific usage.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "query":
		cmd.RunQuery(os.Args[2:])
	case "semantic-context":
		cmd.RunScanContext(os.Args[2:])
	case "mcp":
		cmd.RunMCP(os.Args[2:])
	case "version", "--version", "-v":
		cmd.RunVersion()
	case "license":
		cmd.RunLicense(licenseText)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n%s", os.Args[1], usage)
		os.Exit(1)
	}
}
