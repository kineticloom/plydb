package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/kineticloom/plydb/queryengine"
)

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
