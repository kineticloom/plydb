// Copyright 2026 Paul Tzen
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/kineticloom/plydb/semanticcontext"
	"go.yaml.in/yaml/v4"
)

func RunScanContext(args []string) {
	fs := flag.NewFlagSet("semantic-context", flag.ExitOnError)
	configPath := fs.String("config", "", "path to the connection config JSON file")
	var overlayFiles stringSliceFlag
	fs.Var(&overlayFiles, "semantic-context-overlay", "path to an Open Semantic Interchage OSI (https://github.com/open-semantic-interchange/OSI) YAML overlay file (repeatable)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `Usage: plydb semantic-context [flags]

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

	allOverlays := append(cfg.SemanticContext.Overlays, []string(overlayFiles)...)
	if len(allOverlays) > 0 {
		overlay := semanticcontext.NewOverlayProvider(allOverlays)
		model, err = overlay.Provide(context.Background(), model)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error applying overlay: %v\n", err)
			os.Exit(1)
		}
	}

	enc := yaml.NewEncoder(os.Stdout)
	if err := enc.Encode(model); err != nil {
		fmt.Fprintf(os.Stderr, "error encoding YAML: %v\n", err)
		os.Exit(1)
	}
}
