package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/kineticloom/plydb/queryengine"
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
		query, err = queryengine.PreprocessQuery(query, cfg)
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

	cols, err := rows.Columns()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading columns: %v\n", err)
		os.Exit(1)
	}

	// Print header.
	for i, col := range cols {
		if i > 0 {
			fmt.Print("\t")
		}
		fmt.Print(col)
	}
	fmt.Println()

	// Print rows.
	vals := make([]interface{}, len(cols))
	ptrs := make([]interface{}, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}

	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			fmt.Fprintf(os.Stderr, "error scanning row: %v\n", err)
			os.Exit(1)
		}
		for i, v := range vals {
			if i > 0 {
				fmt.Print("\t")
			}
			if b, ok := v.([]byte); ok {
				fmt.Print(string(b))
			} else {
				fmt.Print(v)
			}
		}
		fmt.Println()
	}

	if err := rows.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "error iterating rows: %v\n", err)
		os.Exit(1)
	}
}
