// Copyright 2026 Paul Tzen
// SPDX-License-Identifier: Apache-2.0

package queryengine

import (
	"strings"
	"testing"
)

func TestRequiredExtensions(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want []string
	}{
		{
			name: "no databases",
			cfg:  &Config{Databases: map[string]DatabaseConfig{}},
			want: nil,
		},
		{
			name: "postgresql only",
			cfg: &Config{Databases: map[string]DatabaseConfig{
				"pg1": {Type: PostgreSQL},
			}},
			want: []string{"INSTALL postgres;", "LOAD postgres;"},
		},
		{
			name: "mysql only",
			cfg: &Config{Databases: map[string]DatabaseConfig{
				"my1": {Type: MySQL},
			}},
			want: []string{"INSTALL mysql;", "LOAD mysql;"},
		},
		{
			name: "s3 only",
			cfg: &Config{Databases: map[string]DatabaseConfig{
				"s1": {Type: S3},
			}},
			want: []string{"INSTALL httpfs;", "LOAD httpfs;"},
		},
		{
			name: "all types deduped and sorted",
			cfg: &Config{Databases: map[string]DatabaseConfig{
				"pg1": {Type: PostgreSQL},
				"pg2": {Type: PostgreSQL},
				"my1": {Type: MySQL},
				"s1":  {Type: S3},
				"f1":  {Type: File},
			}},
			want: []string{
				"INSTALL httpfs;", "LOAD httpfs;",
				"INSTALL mysql;", "LOAD mysql;",
				"INSTALL postgres;", "LOAD postgres;",
			},
		},
		{
			name: "file only produces no extensions",
			cfg: &Config{Databases: map[string]DatabaseConfig{
				"f1": {Type: File},
			}},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := requiredExtensions(tt.cfg)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("index %d: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestRequiredExtensionsGSheet(t *testing.T) {
	cfg := &Config{Databases: map[string]DatabaseConfig{
		"gs1": {Type: GSheet},
	}}
	got := requiredExtensions(cfg)
	want := []string{"INSTALL gsheets FROM community;", "LOAD gsheets;"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("index %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestGsheetSecretSQL(t *testing.T) {
	t.Run("key_file auth", func(t *testing.T) {
		sql := gsheetSecretSQL("/path/to/key.json")
		want := "CREATE SECRET (TYPE gsheet, PROVIDER key_file, FILEPATH '/path/to/key.json');"
		if sql != want {
			t.Fatalf("got %q, want %q", sql, want)
		}
	})

	t.Run("browser OAuth", func(t *testing.T) {
		sql := gsheetSecretSQL("")
		want := "CREATE PERSISTENT SECRET __plydb_gsheet (TYPE gsheet);"
		if sql != want {
			t.Fatalf("got %q, want %q", sql, want)
		}
	})
}

func TestResolveEnvVar(t *testing.T) {
	t.Run("set and non-empty", func(t *testing.T) {
		t.Setenv("TEST_RESOLVE_VAR", "myvalue")
		val, err := resolveEnvVar("TEST_RESOLVE_VAR")
		if err != nil {
			t.Fatal(err)
		}
		if val != "myvalue" {
			t.Fatalf("got %q, want %q", val, "myvalue")
		}
	})

	t.Run("empty value", func(t *testing.T) {
		t.Setenv("TEST_RESOLVE_EMPTY", "")
		_, err := resolveEnvVar("TEST_RESOLVE_EMPTY")
		if err == nil {
			t.Fatal("expected error for empty env var")
		}
	})

	t.Run("unset", func(t *testing.T) {
		_, err := resolveEnvVar("TEST_RESOLVE_UNSET_XYZ_999")
		if err == nil {
			t.Fatal("expected error for unset env var")
		}
	})
}

func TestS3ConfigSQL(t *testing.T) {
	t.Setenv("MY_ACCESS_KEY", "AKIA123")
	t.Setenv("MY_SECRET_KEY", "secret456")

	cred := Credential{
		AccessKeyEnv: "MY_ACCESS_KEY",
		SecretKeyEnv: "MY_SECRET_KEY",
	}
	stmts, err := s3ConfigSQL(cred, "us-west-2")
	if err != nil {
		t.Fatal(err)
	}

	if len(stmts) != 3 {
		t.Fatalf("expected 3 statements, got %d", len(stmts))
	}
	if !strings.Contains(stmts[0], "AKIA123") {
		t.Errorf("expected access key in statement, got %q", stmts[0])
	}
	if !strings.Contains(stmts[1], "secret456") {
		t.Errorf("expected secret key in statement, got %q", stmts[1])
	}
	if !strings.Contains(stmts[2], "us-west-2") {
		t.Errorf("expected region in statement, got %q", stmts[2])
	}
}

func TestS3ConfigSQLMissingEnv(t *testing.T) {
	cred := Credential{
		AccessKeyEnv: "MISSING_S3_KEY_XYZ",
		SecretKeyEnv: "MISSING_S3_SECRET_XYZ",
	}
	_, err := s3ConfigSQL(cred, "us-east-1")
	if err == nil {
		t.Fatal("expected error for missing env var")
	}
}

func TestAttachSQL(t *testing.T) {
	t.Run("postgresql", func(t *testing.T) {
		t.Setenv("PG_PASS", "pgpass123")
		db := DatabaseConfig{
			Type:           PostgreSQL,
			Host:           "db.example.com",
			Port:           5432,
			DatabaseName:   "analytics",
			Username:       "reader",
			PasswordEnvVar: "PG_PASS",
		}
		sql, err := attachSQL("db-prod", db)
		if err != nil {
			t.Fatal(err)
		}
		expected := `ATTACH 'host=db.example.com port=5432 dbname=analytics user=reader password=pgpass123' AS "db-prod" (TYPE POSTGRES, READ_ONLY);`
		if sql != expected {
			t.Fatalf("got:\n  %s\nwant:\n  %s", sql, expected)
		}
	})

	t.Run("mysql", func(t *testing.T) {
		t.Setenv("MY_PASS", "mypass456")
		db := DatabaseConfig{
			Type:           MySQL,
			Host:           "mysql.example.com",
			Port:           3306,
			DatabaseName:   "appdb",
			Username:       "admin",
			PasswordEnvVar: "MY_PASS",
		}
		sql, err := attachSQL("my-prod", db)
		if err != nil {
			t.Fatal(err)
		}
		expected := `ATTACH 'host=mysql.example.com port=3306 user=admin password=mypass456 database=appdb' AS "my-prod" (TYPE MYSQL, READ_ONLY);`
		if sql != expected {
			t.Fatalf("got:\n  %s\nwant:\n  %s", sql, expected)
		}
	})

	t.Run("missing password env", func(t *testing.T) {
		db := DatabaseConfig{
			Type:           PostgreSQL,
			Host:           "localhost",
			Port:           5432,
			DatabaseName:   "test",
			Username:       "user",
			PasswordEnvVar: "MISSING_PASSWORD_XYZ_999",
		}
		_, err := attachSQL("test-db", db)
		if err == nil {
			t.Fatal("expected error for missing password env var")
		}
	})
}
