package sqlwalk

import (
	"testing"

	pg_query "github.com/pganalyze/pg_query_go/v6"
)

func TestRenameTables(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		renames map[TableName]TableName
		want    string
	}{
		{
			name: "simple rename",
			sql:  "SELECT a, b FROM t1",
			renames: map[TableName]TableName{
				{Name: "t1"}: {Name: "t1_renamed"},
			},
			want: "SELECT a, b FROM t1_renamed",
		},
		{
			name: "rename with schema",
			sql:  "SELECT x FROM myschema.tbl",
			renames: map[TableName]TableName{
				{Schema: "myschema", Name: "tbl"}: {Schema: "newschema", Name: "new_tbl"},
			},
			want: "SELECT x FROM newschema.new_tbl",
		},
		{
			name: "rename preserves alias",
			sql:  "SELECT t.x FROM myschema.tbl AS t",
			renames: map[TableName]TableName{
				{Schema: "myschema", Name: "tbl"}: {Schema: "other", Name: "other_tbl"},
			},
			want: "SELECT t.x FROM other.other_tbl t",
		},
		{
			name: "rename three-part name",
			sql:  "SELECT x FROM db.schema.tbl",
			renames: map[TableName]TableName{
				{Catalog: "db", Schema: "schema", Name: "tbl"}: {Catalog: "db2", Schema: "schema2", Name: "tbl2"},
			},
			want: "SELECT x FROM db2.schema2.tbl2",
		},
		{
			name: "rename three-part name only when Catalog, Schema, and Name all match",
			sql:  "SELECT x FROM db.schema.tbl",
			renames: map[TableName]TableName{
				{Catalog: "db", Schema: "some_other_schema", Name: "tbl"}: {Catalog: "db2", Schema: "schema2", Name: "tbl2"},
			},
			want: "SELECT x FROM db.schema.tbl",
		},
		{
			name: "rename only matching table in JOIN",
			sql:  "SELECT a.id FROM alpha a JOIN beta b ON a.id = b.alpha_id",
			renames: map[TableName]TableName{
				{Name: "alpha"}: {Name: "alpha_v2"},
			},
			want: "SELECT a.id FROM alpha_v2 a JOIN beta b ON a.id = b.alpha_id",
		},
		{
			name: "rename in CTE body",
			sql:  "WITH cte AS (SELECT id FROM src) SELECT id FROM cte",
			renames: map[TableName]TableName{
				{Name: "src"}: {Name: "src_new"},
			},
			want: "WITH cte AS (SELECT id FROM src_new) SELECT id FROM cte",
		},
		{
			name: "rename INSERT target",
			sql:  "INSERT INTO dst (col1) SELECT col1 FROM src",
			renames: map[TableName]TableName{
				{Name: "dst"}: {Name: "dst_new"},
				{Name: "src"}: {Name: "src_new"},
			},
			want: "INSERT INTO dst_new (col1) SELECT col1 FROM src_new",
		},
		{
			name: "rename UPDATE target",
			sql:  "UPDATE t1 SET x = 1",
			renames: map[TableName]TableName{
				{Name: "t1"}: {Name: "t1_new"},
			},
			want: "UPDATE t1_new SET x = 1",
		},
		{
			name: "CTE reference not renamed even if name matches",
			sql:  "WITH src AS (SELECT id FROM real_src) SELECT id FROM src",
			renames: map[TableName]TableName{
				{Name: "src"}:      {Name: "src_renamed"},
				{Name: "real_src"}: {Name: "real_src_renamed"},
			},
			want: "WITH src AS (SELECT id FROM real_src_renamed) SELECT id FROM src",
		},
		{
			name: "nested CTE references not renamed",
			sql:  "WITH a AS (SELECT 1 FROM t1), b AS (SELECT * FROM a) SELECT * FROM b",
			renames: map[TableName]TableName{
				{Name: "a"}:  {Name: "a_renamed"},
				{Name: "b"}:  {Name: "b_renamed"},
				{Name: "t1"}: {Name: "t1_renamed"},
			},
			want: "WITH a AS (SELECT 1 FROM t1_renamed), b AS (SELECT * FROM a) SELECT * FROM b",
		},
		{
			name: "unaliased table.* renamed",
			sql:  "SELECT tbl.* FROM tbl",
			renames: map[TableName]TableName{
				{Name: "tbl"}: {Name: "tbl_new"},
			},
			want: "SELECT tbl_new.* FROM tbl_new",
		},
		{
			name: "unaliased table.col renamed",
			sql:  "SELECT tbl.col FROM tbl",
			renames: map[TableName]TableName{
				{Name: "tbl"}: {Name: "tbl_new"},
			},
			want: "SELECT tbl_new.col FROM tbl_new",
		},
		{
			name: "aliased table.* not renamed in column ref",
			sql:  "SELECT t.* FROM tbl AS t",
			renames: map[TableName]TableName{
				{Name: "tbl"}: {Name: "tbl_new"},
			},
			want: "SELECT t.* FROM tbl_new t",
		},
		{
			name: "schema-qualified column ref renamed",
			sql:  "SELECT s.tbl.col FROM s.tbl",
			renames: map[TableName]TableName{
				{Schema: "s", Name: "tbl"}: {Schema: "s2", Name: "tbl2"},
			},
			want: "SELECT s2.tbl2.col FROM s2.tbl2",
		},
		{
			name: "schema-qualified star renamed",
			sql:  "SELECT s.tbl.* FROM s.tbl",
			renames: map[TableName]TableName{
				{Schema: "s", Name: "tbl"}: {Schema: "s2", Name: "tbl2"},
			},
			want: "SELECT s2.tbl2.* FROM s2.tbl2",
		},
		{
			name: "column ref in WHERE renamed",
			sql:  "SELECT tbl.a FROM tbl WHERE tbl.b > 1",
			renames: map[TableName]TableName{
				{Name: "tbl"}: {Name: "tbl_new"},
			},
			want: "SELECT tbl_new.a FROM tbl_new WHERE tbl_new.b > 1",
		},
		{
			name: "UPDATE with FROM renames table and column refs",
			sql:  "UPDATE users SET name = 'test' FROM orders WHERE users.id = orders.u_id",
			renames: map[TableName]TableName{
				{Name: "users"}:  {Name: "users_v2"},
				{Name: "orders"}: {Name: "orders_v2"},
			},
			want: "UPDATE users_v2 SET name = 'test' FROM orders_v2 WHERE users_v2.id = orders_v2.u_id",
		},
		{
			name: "UPDATE with FROM aliased table keeps alias in column refs",
			sql:  "UPDATE users SET name = o.name FROM orders o WHERE users.id = o.u_id",
			renames: map[TableName]TableName{
				{Name: "users"}:  {Name: "users_v2"},
				{Name: "orders"}: {Name: "orders_v2"},
			},
			want: "UPDATE users_v2 SET name = o.name FROM orders_v2 o WHERE users_v2.id = o.u_id",
		},
		{
			name: "upsert with RETURNING leaves EXCLUDED untouched",
			sql:  "INSERT INTO users (id, name) VALUES (1, 'Gemini') ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name RETURNING id",
			renames: map[TableName]TableName{
				{Name: "users"}: {Name: "users_v2"},
			},
			want: "INSERT INTO users_v2 (id, name) VALUES (1, 'Gemini') ON CONFLICT (id) DO UPDATE SET name = excluded.name RETURNING id",
		},
		{
			name: "upsert with qualified RETURNING column refs renamed",
			sql:  "INSERT INTO users (id, name) VALUES (1, 'Gemini') ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name RETURNING users.id, users.name",
			renames: map[TableName]TableName{
				{Name: "users"}: {Name: "users_v2"},
			},
			want: "INSERT INTO users_v2 (id, name) VALUES (1, 'Gemini') ON CONFLICT (id) DO UPDATE SET name = excluded.name RETURNING users_v2.id, users_v2.name",
		},
		{
			name: "UNION same table renamed on both sides",
			sql:  "SELECT id FROM users UNION SELECT id FROM users",
			renames: map[TableName]TableName{
				{Name: "users"}: {Name: "users_v2"},
			},
			want: "SELECT id FROM users_v2 UNION SELECT id FROM users_v2",
		},
		{
			name: "UNION different tables with qualified column refs",
			sql:  "SELECT users.id FROM users UNION SELECT orders.id FROM orders",
			renames: map[TableName]TableName{
				{Name: "users"}:  {Name: "users_v2"},
				{Name: "orders"}: {Name: "orders_v2"},
			},
			want: "SELECT users_v2.id FROM users_v2 UNION SELECT orders_v2.id FROM orders_v2",
		},
		{
			name: "triple UNION",
			sql:  "SELECT id FROM t1 UNION SELECT id FROM t2 UNION SELECT id FROM t3",
			renames: map[TableName]TableName{
				{Name: "t1"}: {Name: "t1_new"},
				{Name: "t2"}: {Name: "t2_new"},
				{Name: "t3"}: {Name: "t3_new"},
			},
			want: "(SELECT id FROM t1_new UNION SELECT id FROM t2_new) UNION SELECT id FROM t3_new",
		},
		{
			name: "nested SELECT",
			sql:  "SELECT id FROM (SELECT id FROM t2) t1",
			renames: map[TableName]TableName{
				{Name: "t1"}: {Name: "t1_new"},
				{Name: "t2"}: {Name: "t2_new"},
			},
			want: "SELECT id FROM (SELECT id FROM t2_new) t1",
		},
		{
			name: "nested in WHERE",
			sql:  "SELECT id FROM t1 WHERE id IN (SELECT id FROM t2)",
			renames: map[TableName]TableName{
				{Name: "t1"}: {Name: "t1_new"},
				{Name: "t2"}: {Name: "t2_new"},
			},
			want: "SELECT id FROM t1_new WHERE id IN (SELECT id FROM t2_new)",
		},
		{
			name: "nested in EXISTS",
			sql:  "SELECT id FROM t1 WHERE EXISTS (SELECT 1 FROM t2)",
			renames: map[TableName]TableName{
				{Name: "t1"}: {Name: "t1_new"},
				{Name: "t2"}: {Name: "t2_new"},
			},
			want: "SELECT id FROM t1_new WHERE EXISTS (SELECT 1 FROM t2_new)",
		},
		{
			name: "renaming table to a file path",
			sql:  "SELECT id FROM t1",
			renames: map[TableName]TableName{
				{Name: "t1"}: {Name: "/some/file/path.json"},
			},
			want: "SELECT id FROM \"/some/file/path.json\"",
		},
		{
			name: "renaming table to an S3 path",
			sql:  "SELECT id FROM t1",
			renames: map[TableName]TableName{
				{Name: "t1"}: {Name: "s3://bucket-name/key/name/with/prefixes"},
			},
			want: "SELECT id FROM \"s3://bucket-name/key/name/with/prefixes\"",
		},
		{
			name: "no match leaves SQL unchanged",
			sql:  "SELECT a FROM t1",
			renames: map[TableName]TableName{
				{Name: "no_match"}: {Name: "whatever"},
			},
			want: "SELECT a FROM t1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := pg_query.Parse(tt.sql)
			if err != nil {
				t.Fatalf("Parse(%q): %v", tt.sql, err)
			}

			RenameTables(parsed, tt.renames)

			deparsed, err := pg_query.Deparse(parsed)
			if err != nil {
				t.Fatalf("Deparse: %v", err)
			}

			if deparsed != tt.want {
				t.Errorf("got %q, want %q", deparsed, tt.want)
			}
		})
	}
}
