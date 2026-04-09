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
	configPath := fs.String("config", "", "path to the data source config JSON file")
	configEnvVar := fs.String("config-env-var", "", "name of env var containing config JSON (alternative to --config)")
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
	rawQuery := fs.Arg(0)

	cfg, engine := LoadConfigAndEngineFromFlags(*configPath, *configEnvVar, fs.Usage)
	defer engine.Close()

	policy := queryengine.ReadOnlyPolicy(cfg)
	validator := queryengine.NewPolicyValidator(policy)
	query, err := queryengine.PreprocessQuery(rawQuery, cfg, validator)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error preprocessing query: %v\n", err)
		os.Exit(1)
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
