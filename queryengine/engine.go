// Copyright 2026 Paul Tzen
// SPDX-License-Identifier: Apache-2.0

package queryengine

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	_ "github.com/duckdb/duckdb-go/v2"
)

// ConnectionType tracks the kind of active connection.
type ConnectionType string

const (
	ConnPostgres ConnectionType = "postgresql"
	ConnMySQL    ConnectionType = "mysql"
	ConnS3       ConnectionType = "s3"
	ConnFile     ConnectionType = "file"
	ConnGSheet   ConnectionType = "gsheet"
)

// QueryEngine wraps a DuckDB instance with attached remote sources.
type QueryEngine struct {
	db                *sql.DB
	activeConnections map[string]ConnectionType
}

// New creates a QueryEngine from the given Config. It opens an in-memory
// DuckDB instance, loads required extensions, configures S3 credentials,
// and attaches networked databases.
func New(cfg *Config) (*QueryEngine, error) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("opening duckdb: %w", err)
	}

	// On any failure after open, ensure we close.
	success := false
	defer func() {
		if !success {
			db.Close()
		}
	}()

	// Load extensions.
	for _, stmt := range requiredExtensions(cfg) {
		if _, err := db.Exec(stmt); err != nil {
			return nil, fmt.Errorf("executing %q: %w", stmt, err)
		}
	}

	// Configure S3 if any S3 sources exist.
	if err := configureS3(db, cfg); err != nil {
		return nil, err
	}

	// Configure GSheet if any gsheet sources exist.
	if err := configureGSheet(db, cfg); err != nil {
		return nil, err
	}

	active := make(map[string]ConnectionType)

	// Iterate databases in sorted key order for deterministic bootstrapping.
	keys := make([]string, 0, len(cfg.Databases))
	for k := range cfg.Databases {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		dbCfg := cfg.Databases[key]
		switch dbCfg.Type {
		case PostgreSQL, MySQL:
			stmt, err := attachSQL(key, dbCfg)
			if err != nil {
				return nil, err
			}
			if _, err := db.Exec(stmt); err != nil {
				return nil, fmt.Errorf("attaching %q: %w", key, err)
			}
			// Validate the connection.
			if _, err := db.Exec(fmt.Sprintf(`SELECT 1 FROM "%s".information_schema.tables LIMIT 0`, key)); err != nil {
				return nil, fmt.Errorf("validating connection %q: %w", key, err)
			}
			active[key] = ConnectionType(dbCfg.Type)

		case File:
			active[key] = ConnFile

		case S3:
			active[key] = ConnS3

		case GSheet:
			active[key] = ConnGSheet
		}
	}

	success = true
	return &QueryEngine{
		db:                db,
		activeConnections: active,
	}, nil
}

// configureS3 sets S3 credentials in the DuckDB session if any S3 sources
// are present. All S3 sources must share the same credential profile
// (enforced at parse time).
func configureS3(db *sql.DB, cfg *Config) error {
	// Find first S3 source to get credential profile and region.
	for _, dbCfg := range cfg.Databases {
		if dbCfg.Type != S3 {
			continue
		}
		cred := cfg.Credentials[dbCfg.CredentialProfile]
		stmts, err := s3ConfigSQL(cred, dbCfg.Region)
		if err != nil {
			return fmt.Errorf("configuring S3 credentials: %w", err)
		}
		for _, stmt := range stmts {
			if _, err := db.Exec(stmt); err != nil {
				return fmt.Errorf("executing S3 config %q: %w", stmt, err)
			}
		}
		return nil
	}
	return nil
}

// configureGSheet sets up Google Sheets authentication in the DuckDB session
// if any gsheet sources are present. All gsheet sources must share the same
// credential profile (enforced at parse time).
func configureGSheet(db *sql.DB, cfg *Config) error {
	for _, dbCfg := range cfg.Databases {
		if dbCfg.Type != GSheet {
			continue
		}
		var keyFile string
		if dbCfg.CredentialProfile != "" {
			cred := cfg.Credentials[dbCfg.CredentialProfile]
			keyFile = cred.KeyFile
		}
		stmt := gsheetSecretSQL(keyFile)
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("configuring gsheet credentials: %w", err)
		}
		return nil
	}
	return nil
}

// Query executes the provided SQL and returns the result rows.
func (e *QueryEngine) Query(ctx context.Context, sqlQuery string) (*sql.Rows, error) {
	return e.db.QueryContext(ctx, sqlQuery)
}

// Close shuts down the DuckDB instance.
func (e *QueryEngine) Close() error {
	return e.db.Close()
}
