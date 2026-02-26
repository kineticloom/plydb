// Copyright 2026 Paul Tzen
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/kineticloom/plydb/queryengine"
)

func RunAuth(args []string) {
	fs := flag.NewFlagSet("auth", flag.ExitOnError)
	configPath := fs.String("config", "", "path to the connection config JSON file")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `Usage: plydb auth [flags]

Flags:`)
		fs.PrintDefaults()
	}
	fs.Parse(reorderArgs(args))

	if *configPath == "" {
		fmt.Fprintln(os.Stderr, "error: --config is required")
		fs.Usage()
		os.Exit(1)
	}

	data, err := os.ReadFile(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading config file: %v\n", err)
		os.Exit(1)
	}
	cfg, err := queryengine.ParseConfig(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing config: %v\n", err)
		os.Exit(1)
	}

	if !hasGSheetOAuth(cfg) {
		fmt.Fprintln(os.Stderr, "error: no gsheet data sources requiring browser OAuth found in config")
		os.Exit(1)
	}

	if err := queryengine.AuthGSheet(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("gsheet authentication complete.")
}

// hasGSheetOAuth reports whether cfg contains at least one gsheet database
// configured for browser OAuth (i.e. no service account key_file).
func hasGSheetOAuth(cfg *queryengine.Config) bool {
	for _, db := range cfg.Databases {
		if db.Type != queryengine.GSheet {
			continue
		}
		if db.CredentialProfile == "" {
			return true
		}
		if cfg.Credentials[db.CredentialProfile].KeyFile == "" {
			return true
		}
	}
	return false
}
