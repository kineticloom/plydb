package main

import (
	"fmt"
	"os"

	"github.com/ypt/experiment-nexus/cmd"
)

const usage = `Usage: nexus <command> [arguments] [flags]

Commands:
  query          Execute a SQL query
  scan-context   Scan all data sources and output semantic context as YAML

Run "nexus <command> -h" for command-specific usage.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "query":
		cmd.RunQuery(os.Args[2:])
	case "scan-context":
		cmd.RunScanContext(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n%s", os.Args[1], usage)
		os.Exit(1)
	}
}
