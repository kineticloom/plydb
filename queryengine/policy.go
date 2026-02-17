package queryengine

import (
	"fmt"

	"github.com/kineticloom/plydb/sqlwalk"
	pg_query "github.com/pganalyze/pg_query_go/v6"
)

// NewPolicyValidator returns a ValidateFunc that checks a parsed SQL AST
// against the given sqlwalk.Policy using fail-fast mode.
func NewPolicyValidator(policy *sqlwalk.Policy) ValidateFunc {
	return func(parsed *pg_query.ParseResult) error {
		violations, err := sqlwalk.Validate(parsed, policy, sqlwalk.FailFast())
		if err != nil {
			return fmt.Errorf("policy validation: %w", err)
		}
		if len(violations) > 0 {
			return fmt.Errorf("%s", violations[0].Error())
		}
		return nil
	}
}

// ReadOnlyPolicy constructs a sqlwalk.Policy where every catalog from
// cfg.Databases gets base_access "read", allowing SELECT and denying
// INSERT/UPDATE/DELETE/TRUNCATE/DDL.
func ReadOnlyPolicy(cfg *Config) *sqlwalk.Policy {
	perms := make([]sqlwalk.CatalogPermission, 0, len(cfg.Databases))
	for name := range cfg.Databases {
		perms = append(perms, sqlwalk.CatalogPermission{
			Catalog:    name,
			BaseAccess: "read",
		})
	}
	return &sqlwalk.Policy{
		Permissions: perms,
	}
}
