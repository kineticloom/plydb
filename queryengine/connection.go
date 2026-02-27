// Copyright 2026 Paul Tzen
// SPDX-License-Identifier: Apache-2.0

package queryengine

import (
	"fmt"
	"os"
	"sort"
)

// requiredExtensions returns deduplicated INSTALL/LOAD SQL statements
// based on the database types present in the configuration.
// The map value is the install source ("" for default, "community" for community).
func requiredExtensions(cfg *Config) []string {
	need := make(map[string]string) // extension name -> install source
	for _, db := range cfg.Databases {
		switch db.Type {
		case PostgreSQL:
			need["postgres"] = ""
		case MySQL:
			need["mysql"] = ""
		case SQLite:
			need["sqlite"] = ""
		case S3:
			need["httpfs"] = ""
		case GSheet:
			need["gsheets"] = "community"
		}
	}

	// Sort for deterministic ordering.
	exts := make([]string, 0, len(need))
	for ext := range need {
		exts = append(exts, ext)
	}
	sort.Strings(exts)

	stmts := make([]string, 0, len(exts)*2)
	for _, ext := range exts {
		source := need[ext]
		if source != "" {
			stmts = append(stmts, fmt.Sprintf("INSTALL %s FROM %s;", ext, source))
		} else {
			stmts = append(stmts, fmt.Sprintf("INSTALL %s;", ext))
		}
		stmts = append(stmts, fmt.Sprintf("LOAD %s;", ext))
	}
	return stmts
}

// resolveEnvVar reads an environment variable by name and returns an error
// if it is unset or empty.
func resolveEnvVar(name string) (string, error) {
	val, ok := os.LookupEnv(name)
	if !ok || val == "" {
		return "", fmt.Errorf("environment variable %q is not set or empty", name)
	}
	return val, nil
}

// s3ConfigSQL returns SET statements to configure S3 credentials and region
// in the DuckDB session.
func s3ConfigSQL(cred Credential, region string) ([]string, error) {
	accessKey, err := resolveEnvVar(cred.AccessKeyEnv)
	if err != nil {
		return nil, fmt.Errorf("resolving S3 access key: %w", err)
	}
	secretKey, err := resolveEnvVar(cred.SecretKeyEnv)
	if err != nil {
		return nil, fmt.Errorf("resolving S3 secret key: %w", err)
	}

	return []string{
		fmt.Sprintf("SET s3_access_key_id='%s';", accessKey),
		fmt.Sprintf("SET s3_secret_access_key='%s';", secretKey),
		fmt.Sprintf("SET s3_region='%s';", region),
	}, nil
}

// gsheetSecretSQL returns a CREATE SECRET statement for Google Sheets authentication.
// If keyFilePath is non-empty, service account key file auth is used.
// If empty, browser-based OAuth is used.
func gsheetSecretSQL(keyFilePath string) string {
	if keyFilePath != "" {
		return fmt.Sprintf("CREATE SECRET (TYPE gsheet, PROVIDER key_file, FILEPATH '%s');", keyFilePath)
	}
	return "CREATE OR REPLACE PERSISTENT SECRET __plydb_gsheet (TYPE gsheet);"
}

// attachSQL returns an ATTACH statement for a networked database (postgresql or mysql).
// The key is double-quoted as the DuckDB alias.
func attachSQL(key string, db DatabaseConfig) (string, error) {
	password, err := resolveEnvVar(db.PasswordEnvVar)
	if err != nil {
		return "", fmt.Errorf("resolving password for %q: %w", key, err)
	}

	var connStr string
	var dbType string
	switch db.Type {
	case PostgreSQL:
		connStr = fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s",
			db.Host, db.Port, db.DatabaseName, db.Username, password)
		dbType = "POSTGRES"
	case MySQL:
		connStr = fmt.Sprintf("host=%s port=%d user=%s password=%s database=%s",
			db.Host, db.Port, db.Username, password, db.DatabaseName)
		dbType = "MYSQL"
	default:
		return "", fmt.Errorf("attachSQL: unsupported type %q", db.Type)
	}

	return fmt.Sprintf(`ATTACH '%s' AS "%s" (TYPE %s, READ_ONLY);`, connStr, key, dbType), nil
}

// attachSQLiteSQL returns an ATTACH statement for a SQLite database file.
// The key is double-quoted as the DuckDB alias.
func attachSQLiteSQL(key string, db DatabaseConfig) string {
	return fmt.Sprintf(`ATTACH '%s' AS "%s" (TYPE SQLITE, READ_ONLY);`, db.Path, key)
}
