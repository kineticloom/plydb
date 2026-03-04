// Copyright 2026 Paul Tzen
// SPDX-License-Identifier: Apache-2.0

package main

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/kineticloom/plydb/cmd"
)

//go:embed LICENSE
var licenseText string

func main() {
	if len(os.Args) < 2 {
		cmd.RunHelp(os.Stderr)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "auth":
		cmd.RunAuth(os.Args[2:])
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
	case "help", "-h", "--help":
		cmd.RunHelp(os.Stdout)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		cmd.RunHelp(os.Stderr)
		os.Exit(1)
	}
}
