// Copyright 2026 Paul Tzen
// SPDX-License-Identifier: Apache-2.0

package queryengine

import (
	"strings"
	"testing"
)

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr string // empty means expect success
	}{
		{
			name: "valid postgresql",
			json: `{
				"databases": {
					"pg1": {
						"metadata": {"name": "PG", "description": "test"},
						"type": "postgresql",
						"host": "localhost",
						"port": 5432,
						"database_name": "mydb",
						"username": "user",
						"password_env_var": "PG_PASS"
					}
				}
			}`,
		},
		{
			name: "valid mysql",
			json: `{
				"databases": {
					"my1": {
						"metadata": {"name": "MySQL", "description": "test"},
						"type": "mysql",
						"host": "localhost",
						"port": 3306,
						"database_name": "mydb",
						"username": "user",
						"password_env_var": "MY_PASS"
					}
				}
			}`,
		},
		{
			name: "valid file",
			json: `{
				"databases": {
					"f1": {
						"metadata": {"name": "CSV", "description": "test"},
						"type": "file",
						"path": "/data/test.csv",
						"format": "csv",
						"delimiter": ",",
						"header_row": true
					}
				}
			}`,
		},
		{
			name: "valid file with header_row false",
			json: `{
				"databases": {
					"f1": {
						"metadata": {"name": "CSV", "description": "test"},
						"type": "file",
						"path": "/data/test.csv",
						"format": "csv",
						"header_row": false
					}
				}
			}`,
		},
		{
			name: "valid s3",
			json: `{
				"credentials": {
					"aws1": {
						"access_key_env": "AWS_KEY",
						"secret_key_env": "AWS_SECRET"
					}
				},
				"databases": {
					"s1": {
						"metadata": {"name": "S3 data", "description": "test"},
						"type": "s3",
						"uri": "s3://bucket/path/",
						"credential_profile": "aws1",
						"region": "us-east-1",
						"format": "parquet"
					}
				}
			}`,
		},
		{
			name: "empty config",
			json: `{}`,
		},
		{
			name:    "sqlserver rejected",
			wantErr: "sqlserver is not supported",
			json: `{
				"databases": {
					"ss1": {
						"metadata": {"name": "SS", "description": "test"},
						"type": "sqlserver",
						"host": "localhost",
						"port": 1433,
						"database_name": "mydb",
						"username": "user",
						"password_env_var": "SS_PASS"
					}
				}
			}`,
		},
		{
			name:    "unknown type",
			wantErr: `unknown type "oracle"`,
			json: `{
				"databases": {
					"o1": {
						"metadata": {"name": "Oracle", "description": "test"},
						"type": "oracle"
					}
				}
			}`,
		},
		{
			name:    "postgresql missing host",
			wantErr: "host is required",
			json: `{
				"databases": {
					"pg1": {
						"metadata": {"name": "PG", "description": "test"},
						"type": "postgresql",
						"port": 5432,
						"database_name": "mydb",
						"username": "user",
						"password_env_var": "PG_PASS"
					}
				}
			}`,
		},
		{
			name:    "postgresql missing port",
			wantErr: "port is required",
			json: `{
				"databases": {
					"pg1": {
						"metadata": {"name": "PG", "description": "test"},
						"type": "postgresql",
						"host": "localhost",
						"database_name": "mydb",
						"username": "user",
						"password_env_var": "PG_PASS"
					}
				}
			}`,
		},
		{
			name:    "postgresql missing database_name",
			wantErr: "database_name is required",
			json: `{
				"databases": {
					"pg1": {
						"metadata": {"name": "PG", "description": "test"},
						"type": "postgresql",
						"host": "localhost",
						"port": 5432,
						"username": "user",
						"password_env_var": "PG_PASS"
					}
				}
			}`,
		},
		{
			name:    "postgresql missing username",
			wantErr: "username is required",
			json: `{
				"databases": {
					"pg1": {
						"metadata": {"name": "PG", "description": "test"},
						"type": "postgresql",
						"host": "localhost",
						"port": 5432,
						"database_name": "mydb",
						"password_env_var": "PG_PASS"
					}
				}
			}`,
		},
		{
			name:    "postgresql missing password_env_var",
			wantErr: "password_env_var is required",
			json: `{
				"databases": {
					"pg1": {
						"metadata": {"name": "PG", "description": "test"},
						"type": "postgresql",
						"host": "localhost",
						"port": 5432,
						"database_name": "mydb",
						"username": "user"
					}
				}
			}`,
		},
		{
			name:    "file missing path",
			wantErr: "path is required",
			json: `{
				"databases": {
					"f1": {
						"metadata": {"name": "CSV", "description": "test"},
						"type": "file"
					}
				}
			}`,
		},
		{
			name:    "s3 missing uri",
			wantErr: "uri is required",
			json: `{
				"credentials": {
					"aws1": {"access_key_env": "K", "secret_key_env": "S"}
				},
				"databases": {
					"s1": {
						"metadata": {"name": "S3", "description": "test"},
						"type": "s3",
						"credential_profile": "aws1",
						"region": "us-east-1",
						"format": "csv"
					}
				}
			}`,
		},
		{
			name:    "s3 missing credential_profile",
			wantErr: "credential_profile is required",
			json: `{
				"databases": {
					"s1": {
						"metadata": {"name": "S3", "description": "test"},
						"type": "s3",
						"uri": "s3://bucket/path/",
						"region": "us-east-1",
						"format": "csv"
					}
				}
			}`,
		},
		{
			name:    "s3 missing region",
			wantErr: "region is required",
			json: `{
				"credentials": {
					"aws1": {"access_key_env": "K", "secret_key_env": "S"}
				},
				"databases": {
					"s1": {
						"metadata": {"name": "S3", "description": "test"},
						"type": "s3",
						"uri": "s3://bucket/path/",
						"credential_profile": "aws1",
						"format": "csv"
					}
				}
			}`,
		},
		{
			name:    "s3 missing format",
			wantErr: "format is required",
			json: `{
				"credentials": {
					"aws1": {"access_key_env": "K", "secret_key_env": "S"}
				},
				"databases": {
					"s1": {
						"metadata": {"name": "S3", "description": "test"},
						"type": "s3",
						"uri": "s3://bucket/path/",
						"credential_profile": "aws1",
						"region": "us-east-1"
					}
				}
			}`,
		},
		{
			name:    "s3 bad credential reference",
			wantErr: `credential_profile "nope" not found`,
			json: `{
				"databases": {
					"s1": {
						"metadata": {"name": "S3", "description": "test"},
						"type": "s3",
						"uri": "s3://bucket/path/",
						"credential_profile": "nope",
						"region": "us-east-1",
						"format": "csv"
					}
				}
			}`,
		},
		{
			name:    "multiple s3 credential profiles",
			wantErr: "all S3 sources must share the same credential_profile",
			json: `{
				"credentials": {
					"aws1": {"access_key_env": "K1", "secret_key_env": "S1"},
					"aws2": {"access_key_env": "K2", "secret_key_env": "S2"}
				},
				"databases": {
					"s1": {
						"metadata": {"name": "S3a", "description": "test"},
						"type": "s3",
						"uri": "s3://bucket1/",
						"credential_profile": "aws1",
						"region": "us-east-1",
						"format": "csv"
					},
					"s2": {
						"metadata": {"name": "S3b", "description": "test"},
						"type": "s3",
						"uri": "s3://bucket2/",
						"credential_profile": "aws2",
						"region": "us-east-1",
						"format": "csv"
					}
				}
			}`,
		},
		{
			name:    "credential missing access_key_env",
			wantErr: "access_key_env is required",
			json: `{
				"credentials": {
					"bad": {"access_key_env": "", "secret_key_env": "S"}
				}
			}`,
		},
		{
			name:    "credential missing secret_key_env",
			wantErr: "secret_key_env is required",
			json: `{
				"credentials": {
					"bad": {"access_key_env": "K", "secret_key_env": ""}
				}
			}`,
		},
		{
			name:    "invalid JSON",
			wantErr: "invalid JSON",
			json:    `{not json`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ParseConfig([]byte(tt.json))
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg == nil {
				t.Fatal("expected non-nil config")
			}
		})
	}
}

func TestParseConfigHeaderRowPointer(t *testing.T) {
	jsonWithTrue := `{
		"databases": {
			"f1": {
				"metadata": {"name": "CSV", "description": "test"},
				"type": "file",
				"path": "/data/test.csv",
				"header_row": true
			}
		}
	}`
	cfg, err := ParseConfig([]byte(jsonWithTrue))
	if err != nil {
		t.Fatal(err)
	}
	db := cfg.Databases["f1"]
	if db.HeaderRow == nil {
		t.Fatal("expected header_row to be non-nil")
	}
	if *db.HeaderRow != true {
		t.Fatal("expected header_row to be true")
	}

	jsonWithFalse := `{
		"databases": {
			"f1": {
				"metadata": {"name": "CSV", "description": "test"},
				"type": "file",
				"path": "/data/test.csv",
				"header_row": false
			}
		}
	}`
	cfg, err = ParseConfig([]byte(jsonWithFalse))
	if err != nil {
		t.Fatal(err)
	}
	db = cfg.Databases["f1"]
	if db.HeaderRow == nil {
		t.Fatal("expected header_row to be non-nil")
	}
	if *db.HeaderRow != false {
		t.Fatal("expected header_row to be false")
	}

	jsonAbsent := `{
		"databases": {
			"f1": {
				"metadata": {"name": "CSV", "description": "test"},
				"type": "file",
				"path": "/data/test.csv"
			}
		}
	}`
	cfg, err = ParseConfig([]byte(jsonAbsent))
	if err != nil {
		t.Fatal(err)
	}
	db = cfg.Databases["f1"]
	if db.HeaderRow != nil {
		t.Fatal("expected header_row to be nil when absent")
	}
}
