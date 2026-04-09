// Copyright 2026 Paul Tzen
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/kineticloom/plydb/queryengine"
)

// stringSliceFlag is a repeatable string flag (implements flag.Value).
// Use with fs.Var to allow --flag val1 --flag val2 patterns.
type stringSliceFlag []string

func (f *stringSliceFlag) String() string { return strings.Join(*f, ",") }
func (f *stringSliceFlag) Set(s string) error {
	*f = append(*f, s)
	return nil
}

// reorderArgs moves flag-like tokens (--key / --key=val / -key) before
// positional args so that Go's flag package can parse them all.
func reorderArgs(args []string) []string {
	var flags, positional []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "-") {
			flags = append(flags, a)
			// If it looks like "--key val" (no '='), consume the next arg as its value.
			if !strings.Contains(a, "=") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				flags = append(flags, args[i])
			}
		} else {
			positional = append(positional, a)
		}
	}
	return append(flags, positional...)
}

func LoadConfigAndEngine(configPath string) (*queryengine.Config, *queryengine.QueryEngine) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading config file: %v\n", err)
		os.Exit(1)
	}
	return loadConfigAndEngineFromBytes(data)
}

func loadConfigAndEngineFromBytes(data []byte) (*queryengine.Config, *queryengine.QueryEngine) {
	cfg, err := queryengine.ParseConfig(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing config: %v\n", err)
		os.Exit(1)
	}

	engine, err := queryengine.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating query engine: %v\n", err)
		os.Exit(1)
	}

	return cfg, engine
}

// resolveConfigData returns raw JSON config bytes from either a file path or
// an env var name. Exits on error or if neither/both are provided.
func resolveConfigData(configPath, configEnvVar string, usage func()) []byte {
	switch {
	case configPath != "" && configEnvVar != "":
		fmt.Fprintln(os.Stderr, "error: --config and --config-env-var are mutually exclusive")
		os.Exit(1)
	case configPath != "":
		data, err := os.ReadFile(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading config file: %v\n", err)
			os.Exit(1)
		}
		return data
	case configEnvVar != "":
		val := os.Getenv(configEnvVar)
		if val == "" {
			fmt.Fprintf(os.Stderr, "error: env var %q is not set or empty\n", configEnvVar)
			os.Exit(1)
		}
		return []byte(val)
	default:
		fmt.Fprintln(os.Stderr, "error: one of --config or --config-env-var is required")
		usage()
		os.Exit(1)
	}
	return nil // unreachable
}

// LoadConfigAndEngineFromFlags loads config from --config (file path) or
// --config-env-var (env var name), then creates and returns a QueryEngine.
func LoadConfigAndEngineFromFlags(configPath, configEnvVar string, usage func()) (*queryengine.Config, *queryengine.QueryEngine) {
	return loadConfigAndEngineFromBytes(resolveConfigData(configPath, configEnvVar, usage))
}
