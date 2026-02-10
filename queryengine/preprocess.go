package queryengine

import (
	"fmt"
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v6"
	"github.com/ypt/experiment-nexus/sqlwalk"
)

// PreprocessQuery parses a SQL query, validates that all table references are
// fully qualified 3-part names (catalog.schema.table) matching a configured
// database, and rewrites file/S3 references to DuckDB-native form.
// PostgreSQL and MySQL references are left unchanged (they are already
// attached as DuckDB catalogs).
func PreprocessQuery(query string, cfg *Config) (string, error) {
	parsed, err := pg_query.Parse(query)
	if err != nil {
		return "", fmt.Errorf("parse error: %w", err)
	}

	refs := sqlwalk.Extract(parsed)

	renames := make(map[sqlwalk.TableName]sqlwalk.TableName)
	funcReplacements := make(map[sqlwalk.TableName]sqlwalk.FuncReplace)

	for _, ref := range refs.Tables {
		if ref.Catalog == "" || ref.Schema == "" || ref.Name == "" {
			return "", fmt.Errorf("table reference %q is not fully qualified (expected catalog.schema.table)",
				formatRef(ref))
		}

		dbCfg, ok := cfg.Databases[ref.Catalog]
		if !ok {
			return "", fmt.Errorf("unknown catalog %q in table reference %q",
				ref.Catalog, formatRef(ref))
		}

		key := sqlwalk.TableName{
			Catalog: ref.Catalog,
			Schema:  ref.Schema,
			Name:    ref.Name,
		}

		switch dbCfg.Type {
		case PostgreSQL, MySQL:
			// Already attached as a DuckDB catalog — no rewrite needed.
			continue
		case File:
			if strings.HasSuffix(strings.ToLower(dbCfg.Path), ".xlsx") {
				funcReplacements[key] = sqlwalk.FuncReplace{
					FuncName:  "read_xlsx",
					Args:      []string{dbCfg.Path},
					NamedArgs: [][2]string{{"sheet", ref.Name}},
				}
			} else {
				renames[key] = sqlwalk.TableName{Name: dbCfg.Path}
			}
		case S3:
			renames[key] = sqlwalk.TableName{Name: dbCfg.URI}
		default:
			return "", fmt.Errorf("unsupported database type %q for catalog %q", dbCfg.Type, ref.Catalog)
		}
	}

	if len(funcReplacements) > 0 {
		sqlwalk.ReplaceTablesWithFunctions(parsed, funcReplacements)
	}
	if len(renames) > 0 {
		sqlwalk.RenameTables(parsed, renames)
	}

	result, err := pg_query.Deparse(parsed)
	if err != nil {
		return "", fmt.Errorf("deparse error: %w", err)
	}

	return result, nil
}

func formatRef(ref sqlwalk.TableRef) string {
	if ref.Catalog != "" && ref.Schema != "" {
		return ref.Catalog + "." + ref.Schema + "." + ref.Name
	}
	if ref.Schema != "" {
		return ref.Schema + "." + ref.Name
	}
	return ref.Name
}
