// Copyright 2026 Paul Tzen
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/kineticloom/plydb/queryengine"
	"github.com/kineticloom/plydb/queryresult"
)

func RunQuery(args []string) {
	fs := flag.NewFlagSet("query", flag.ExitOnError)
	configPath := fs.String("config", "", "path to the connection config JSON file")
	skipPreprocess := fs.Bool("skip-query-preprocessing", false, "skip query preprocessing (table reference rewriting)")
	debug := fs.Bool("debug", false, "print the query after preprocessing")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `Usage: plydb query <sql> [flags]

Arguments:
  <sql>    SQL query to execute

Flags:`)
		fs.PrintDefaults()
	}
	fs.Parse(reorderArgs(args))

	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(1)
	}
	sqlQuery := fs.Arg(0)

	if *configPath == "" {
		fmt.Fprintln(os.Stderr, "error: --config is required")
		fs.Usage()
		os.Exit(1)
	}

	cfg, engine := LoadConfigAndEngine(*configPath)
	defer engine.Close()

	query := sqlQuery
	var err error
	if !*skipPreprocess {
		policy := queryengine.ReadOnlyPolicy(cfg)
		validator := queryengine.NewPolicyValidator(policy)
		query, err = queryengine.PreprocessQuery(query, cfg, validator)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error preprocessing query: %v\n", err)
			os.Exit(1)
		}
	}

	if *debug {
		fmt.Fprintf(os.Stderr, "[debug] query: %s\n", query)
	}

	rows, err := engine.Query(context.Background(), query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error executing query: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	result, err := queryresult.BuildQueryResult(rows)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error building result: %v\n", err)
		os.Exit(1)
	}
	text, err := queryresult.MarshalResult(result)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error marshaling result: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(text)
}
