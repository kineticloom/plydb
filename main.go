package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/ypt/experiment-nexus/queryengine"
)

func main() {
	configPath := flag.String("config", "", "path to the connection config JSON file")
	sqlQuery := flag.String("sql", "", "SQL query to execute")
	skipPreprocess := flag.Bool("skip-query-preprocessing", false, "skip query preprocessing (table reference rewriting)")
	debug := flag.Bool("debug", false, "print the query after preprocessing")
	flag.Parse()

	if *configPath == "" || *sqlQuery == "" {
		fmt.Fprintln(os.Stderr, "usage: nexus --config <config.json> --sql <query>")
		flag.PrintDefaults()
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

	engine, err := queryengine.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating query engine: %v\n", err)
		os.Exit(1)
	}
	defer engine.Close()

	query := *sqlQuery
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
