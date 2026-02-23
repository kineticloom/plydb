// Copyright 2026 Paul Tzen
// SPDX-License-Identifier: Apache-2.0

package queryengine

import (
	"strings"
	"testing"

	pg_query "github.com/pganalyze/pg_query_go/v6"
)

func TestNewPolicyValidator_AllowsSelect(t *testing.T) {
	cfg := testConfig()
	policy := ReadOnlyPolicy(cfg)
	validator := NewPolicyValidator(policy)

	parsed, err := pg_query.Parse("SELECT * FROM my_pg.public.users")
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if err := validator(parsed); err != nil {
		t.Fatalf("expected SELECT to be allowed, got: %v", err)
	}
}

func TestNewPolicyValidator_DeniesInsert(t *testing.T) {
	cfg := testConfig()
	policy := ReadOnlyPolicy(cfg)
	validator := NewPolicyValidator(policy)

	parsed, err := pg_query.Parse("INSERT INTO my_pg.public.users (name) VALUES ('alice')")
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	err = validator(parsed)
	if err == nil {
		t.Fatal("expected INSERT to be denied, got nil")
	}
	if !strings.Contains(err.Error(), "denied") {
		t.Fatalf("expected denial error, got: %v", err)
	}
}

func TestNewPolicyValidator_DeniesUpdate(t *testing.T) {
	cfg := testConfig()
	policy := ReadOnlyPolicy(cfg)
	validator := NewPolicyValidator(policy)

	parsed, err := pg_query.Parse("UPDATE my_pg.public.users SET name = 'bob' WHERE id = 1")
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	err = validator(parsed)
	if err == nil {
		t.Fatal("expected UPDATE to be denied, got nil")
	}
	if !strings.Contains(err.Error(), "denied") {
		t.Fatalf("expected denial error, got: %v", err)
	}
}

func TestReadOnlyPolicy_CoversAllCatalogs(t *testing.T) {
	cfg := testConfig()
	policy := ReadOnlyPolicy(cfg)

	catalogsInConfig := make(map[string]bool)
	for name := range cfg.Databases {
		catalogsInConfig[name] = true
	}

	catalogsInPolicy := make(map[string]bool)
	for _, perm := range policy.Permissions {
		catalogsInPolicy[perm.Catalog] = true
		if perm.BaseAccess != "read" {
			t.Errorf("catalog %q: expected base_access \"read\", got %q", perm.Catalog, perm.BaseAccess)
		}
	}

	if len(catalogsInPolicy) != len(catalogsInConfig) {
		t.Errorf("expected %d catalogs in policy, got %d", len(catalogsInConfig), len(catalogsInPolicy))
	}

	for name := range catalogsInConfig {
		if !catalogsInPolicy[name] {
			t.Errorf("catalog %q missing from policy", name)
		}
	}
}
