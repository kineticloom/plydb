// Copyright 2026 Paul Tzen
// SPDX-License-Identifier: Apache-2.0

package queryengine

import (
	"encoding/json"
	"fmt"
	"strings"
)

// DatabaseType identifies the kind of data source.
type DatabaseType string

const (
	PostgreSQL DatabaseType = "postgresql"
	MySQL      DatabaseType = "mysql"
	SQLServer  DatabaseType = "sqlserver"
	File       DatabaseType = "file"
	S3         DatabaseType = "s3"
	GSheet     DatabaseType = "gsheet"
)

// SemanticContextConfig holds semantic context configuration.
type SemanticContextConfig struct {
	Overlays []string `json:"overlays,omitempty"`
}

// Config is the top-level configuration for the query engine.
type Config struct {
	Credentials     map[string]Credential     `json:"credentials"`
	Databases       map[string]DatabaseConfig `json:"databases"`
	SemanticContext SemanticContextConfig     `json:"semanticContext,omitempty"`
}

// Credential holds authentication fields for cloud providers.
// Each credential must contain either S3 fields (access_key_env + secret_key_env)
// or the gsheet field (key_file), not both.
type Credential struct {
	AccessKeyEnv string `json:"access_key_env,omitempty"`
	SecretKeyEnv string `json:"secret_key_env,omitempty"`
	KeyFile      string `json:"key_file,omitempty"`
}

// Metadata describes a data source.
type Metadata struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// DatabaseConfig is a flat struct covering all source types.
// Type-specific fields use omitempty.
type DatabaseConfig struct {
	Metadata Metadata     `json:"metadata"`
	Type     DatabaseType `json:"type"`

	// Networked database fields (postgresql, mysql)
	Host           string `json:"host,omitempty"`
	Port           int    `json:"port,omitempty"`
	DatabaseName   string `json:"database_name,omitempty"`
	Username       string `json:"username,omitempty"`
	PasswordEnvVar string `json:"password_env_var,omitempty"`

	// File fields
	Path string `json:"path,omitempty"`

	// S3 fields
	URI               string `json:"uri,omitempty"`
	CredentialProfile string `json:"credential_profile,omitempty"`
	Region            string `json:"region,omitempty"`

	// GSheet fields
	SpreadsheetID string `json:"spreadsheet_id,omitempty"`

	// Shared file/S3/GSheet fields
	Format    string `json:"format,omitempty"`
	Delimiter string `json:"delimiter,omitempty"`
	HeaderRow *bool  `json:"header_row,omitempty"`
	SheetName string `json:"sheet_name,omitempty"`
}

// ParseConfig unmarshals JSON data into a Config and validates it.
func ParseConfig(data []byte) (*Config, error) {
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	if cfg.Credentials == nil {
		cfg.Credentials = make(map[string]Credential)
	}
	if cfg.Databases == nil {
		cfg.Databases = make(map[string]DatabaseConfig)
	}

	// Validate credentials: each must have either S3 fields or key_file, not both.
	for key, cred := range cfg.Credentials {
		hasS3 := strings.TrimSpace(cred.AccessKeyEnv) != "" || strings.TrimSpace(cred.SecretKeyEnv) != ""
		hasKeyFile := strings.TrimSpace(cred.KeyFile) != ""
		if hasS3 && hasKeyFile {
			return nil, fmt.Errorf("credential %q: cannot mix S3 fields (access_key_env/secret_key_env) with gsheet field (key_file)", key)
		}
		if !hasS3 && !hasKeyFile {
			return nil, fmt.Errorf("credential %q: must provide either S3 fields (access_key_env + secret_key_env) or gsheet field (key_file)", key)
		}
		if hasS3 {
			if strings.TrimSpace(cred.AccessKeyEnv) == "" {
				return nil, fmt.Errorf("credential %q: access_key_env is required", key)
			}
			if strings.TrimSpace(cred.SecretKeyEnv) == "" {
				return nil, fmt.Errorf("credential %q: secret_key_env is required", key)
			}
		}
	}

	// Track credential profiles for single-profile enforcement.
	var s3CredProfile string
	var gsheetCredProfile string
	gsheetCredProfileSet := false

	for key, db := range cfg.Databases {
		switch db.Type {
		case SQLServer:
			return nil, fmt.Errorf("database %q: sqlserver is not supported", key)

		case PostgreSQL, MySQL:
			if err := validateNetworkedDB(key, db); err != nil {
				return nil, err
			}

		case File:
			if strings.TrimSpace(db.Path) == "" {
				return nil, fmt.Errorf("database %q: path is required for file type", key)
			}

		case S3:
			if err := validateS3(key, db, cfg.Credentials); err != nil {
				return nil, err
			}
			if s3CredProfile == "" {
				s3CredProfile = db.CredentialProfile
			} else if s3CredProfile != db.CredentialProfile {
				return nil, fmt.Errorf(
					"database %q: all S3 sources must share the same credential_profile (found %q and %q)",
					key, s3CredProfile, db.CredentialProfile,
				)
			}

		case GSheet:
			if err := validateGSheet(key, db, cfg.Credentials); err != nil {
				return nil, err
			}
			cp := db.CredentialProfile
			if !gsheetCredProfileSet {
				gsheetCredProfile = cp
				gsheetCredProfileSet = true
			} else if gsheetCredProfile != cp {
				return nil, fmt.Errorf(
					"database %q: all gsheet sources must share the same credential_profile (found %q and %q)",
					key, gsheetCredProfile, cp,
				)
			}

		default:
			return nil, fmt.Errorf("database %q: unknown type %q", key, db.Type)
		}
	}

	return &cfg, nil
}

func validateNetworkedDB(key string, db DatabaseConfig) error {
	if strings.TrimSpace(db.Host) == "" {
		return fmt.Errorf("database %q: host is required for %s type", key, db.Type)
	}
	if db.Port == 0 {
		return fmt.Errorf("database %q: port is required for %s type", key, db.Type)
	}
	if strings.TrimSpace(db.DatabaseName) == "" {
		return fmt.Errorf("database %q: database_name is required for %s type", key, db.Type)
	}
	if strings.TrimSpace(db.Username) == "" {
		return fmt.Errorf("database %q: username is required for %s type", key, db.Type)
	}
	if strings.TrimSpace(db.PasswordEnvVar) == "" {
		return fmt.Errorf("database %q: password_env_var is required for %s type", key, db.Type)
	}
	return nil
}

func validateGSheet(key string, db DatabaseConfig, creds map[string]Credential) error {
	if strings.TrimSpace(db.SpreadsheetID) == "" {
		return fmt.Errorf("database %q: spreadsheet_id is required for gsheet type", key)
	}
	if cp := strings.TrimSpace(db.CredentialProfile); cp != "" {
		cred, ok := creds[cp]
		if !ok {
			return fmt.Errorf("database %q: credential_profile %q not found in credentials", key, cp)
		}
		if strings.TrimSpace(cred.KeyFile) == "" {
			return fmt.Errorf("database %q: credential_profile %q must have key_file for gsheet type", key, cp)
		}
	}
	return nil
}

func validateS3(key string, db DatabaseConfig, creds map[string]Credential) error {
	if strings.TrimSpace(db.URI) == "" {
		return fmt.Errorf("database %q: uri is required for s3 type", key)
	}
	if strings.TrimSpace(db.CredentialProfile) == "" {
		return fmt.Errorf("database %q: credential_profile is required for s3 type", key)
	}
	if strings.TrimSpace(db.Region) == "" {
		return fmt.Errorf("database %q: region is required for s3 type", key)
	}
	if strings.TrimSpace(db.Format) == "" {
		return fmt.Errorf("database %q: format is required for s3 type", key)
	}
	if _, ok := creds[db.CredentialProfile]; !ok {
		return fmt.Errorf("database %q: credential_profile %q not found in credentials", key, db.CredentialProfile)
	}
	return nil
}
