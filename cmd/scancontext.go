package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/ypt/experiment-nexus/semanticcontext"
	"go.yaml.in/yaml/v4"
)

func RunScanContext(args []string) {
	fs := flag.NewFlagSet("scan-context", flag.ExitOnError)
	configPath := fs.String("config", "", "path to the connection config JSON file")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `Usage: nexus scan-context [flags]

Flags:`)
		fs.PrintDefaults()
	}
	fs.Parse(args)

	if *configPath == "" {
		fmt.Fprintln(os.Stderr, "error: --config is required")
		fs.Usage()
		os.Exit(1)
	}

	cfg, engine := LoadConfigAndEngine(*configPath)
	defer engine.Close()

	provider := semanticcontext.NewAutoScanProvider(cfg, engine)
	model, err := provider.Provide(context.Background(), nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error scanning context: %v\n", err)
		os.Exit(1)
	}

	enc := yaml.NewEncoder(os.Stdout)
	if err := enc.Encode(model); err != nil {
		fmt.Fprintf(os.Stderr, "error encoding YAML: %v\n", err)
		os.Exit(1)
	}
}
