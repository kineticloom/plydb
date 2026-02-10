package sqlwalk

import (
	"testing"

	pg_query "github.com/pganalyze/pg_query_go/v6"
)

func TestReplaceTablesWithFunctions(t *testing.T) {
	tests := []struct {
		name         string
		sql          string
		replacements map[TableName]FuncReplace
		want         string
	}{
		{
			name: "basic replacement with named arg",
			sql:  "SELECT * FROM cat.sch.t1",
			replacements: map[TableName]FuncReplace{
				{Catalog: "cat", Schema: "sch", Name: "t1"}: {
					FuncName:  "read_xlsx",
					Args:      []string{"/path/to/file.xlsx"},
					NamedArgs: [][2]string{{"sheet", "s1"}},
				},
			},
			want: "SELECT * FROM read_xlsx('/path/to/file.xlsx', sheet := 's1')",
		},
		{
			name: "alias preservation",
			sql:  "SELECT t.col FROM cat.sch.t1 AS t",
			replacements: map[TableName]FuncReplace{
				{Catalog: "cat", Schema: "sch", Name: "t1"}: {
					FuncName:  "read_xlsx",
					Args:      []string{"/data/report.xlsx"},
					NamedArgs: [][2]string{{"sheet", "sheet1"}},
				},
			},
			want: "SELECT t.col FROM read_xlsx('/data/report.xlsx', sheet := 'sheet1') t",
		},
		{
			name: "no match passthrough",
			sql:  "SELECT a FROM cat.sch.t1",
			replacements: map[TableName]FuncReplace{
				{Catalog: "other", Schema: "sch", Name: "t1"}: {
					FuncName: "read_xlsx",
					Args:     []string{"/path"},
				},
			},
			want: "SELECT a FROM cat.sch.t1",
		},
		{
			name: "positional args only",
			sql:  "SELECT * FROM cat.sch.t1",
			replacements: map[TableName]FuncReplace{
				{Catalog: "cat", Schema: "sch", Name: "t1"}: {
					FuncName: "read_csv",
					Args:     []string{"/data/file.csv"},
				},
			},
			want: "SELECT * FROM read_csv('/data/file.csv')",
		},
		{
			name: "mixed replace and no-match in join",
			sql:  "SELECT * FROM cat.sch.t1 JOIN cat.sch.t2 ON true",
			replacements: map[TableName]FuncReplace{
				{Catalog: "cat", Schema: "sch", Name: "t1"}: {
					FuncName:  "read_xlsx",
					Args:      []string{"/file.xlsx"},
					NamedArgs: [][2]string{{"sheet", "data"}},
				},
			},
			want: "SELECT * FROM read_xlsx('/file.xlsx', sheet := 'data') JOIN cat.sch.t2 ON true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := pg_query.Parse(tt.sql)
			if err != nil {
				t.Fatalf("Parse(%q): %v", tt.sql, err)
			}

			ReplaceTablesWithFunctions(parsed, tt.replacements)

			deparsed, err := pg_query.Deparse(parsed)
			if err != nil {
				t.Fatalf("Deparse: %v", err)
			}

			if deparsed != tt.want {
				t.Errorf("got  %q\nwant %q", deparsed, tt.want)
			}
		})
	}
}
