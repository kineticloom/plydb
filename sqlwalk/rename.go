package sqlwalk

import (
	pg_query "github.com/pganalyze/pg_query_go/v6"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// TableName identifies a table by its catalog, schema, and name.
// Used as a map key for rename operations.
type TableName struct {
	Catalog string
	Schema  string
	Name    string
}

// RenameTables walks the pg_query AST and renames every RangeVar whose
// (catalog, schema, name) matches a key in the renames map. The
// corresponding value supplies the new catalog, schema, and name.
// Column references (e.g. tbl.col, tbl.*, schema.tbl.col) for unaliased
// renamed tables are updated as well. The AST is mutated in place and
// also returned for convenience.
func RenameTables(parsed *pg_query.ParseResult, renames map[TableName]TableName) *pg_query.ParseResult {
	// First pass: collect CTE names so we can skip them during renaming.
	cteNames := make(map[string]struct{})
	for _, stmt := range parsed.GetStmts() {
		if stmt.GetStmt() != nil {
			collectCTENames(stmt.GetStmt(), cteNames)
		}
	}

	// Second pass: rename RangeVars and collect column-ref prefix renames
	// for unaliased tables.
	r := &renamer{
		renames:       renames,
		cteNames:      cteNames,
		colRefRenames: make(map[colRefPrefix]colRefPrefix),
	}
	for _, stmt := range parsed.GetStmts() {
		if stmt.GetStmt() != nil {
			walkNodeRename(stmt.GetStmt(), r)
		}
	}

	// Third pass: rename ColumnRef prefixes for unaliased renamed tables.
	if len(r.colRefRenames) > 0 {
		for _, stmt := range parsed.GetStmts() {
			if stmt.GetStmt() != nil {
				walkNodeColRef(stmt.GetStmt(), r)
			}
		}
	}

	return parsed
}

// colRefPrefix represents the qualifier portion of a ColumnRef
// (e.g. for "schema.tbl.col" it would be {A: "schema", B: "tbl"}).
type colRefPrefix struct {
	A, B string // A is schema (or table if single-part), B is table (or empty)
}

type renamer struct {
	renames       map[TableName]TableName
	cteNames      map[string]struct{}
	colRefRenames map[colRefPrefix]colRefPrefix
}

func (r *renamer) applyRangeVar(rv *pg_query.RangeVar) {
	if rv == nil {
		return
	}
	// Skip CTE references — an unqualified name matching a CTE is not a real table.
	if rv.GetCatalogname() == "" && rv.GetSchemaname() == "" {
		if _, isCTE := r.cteNames[rv.GetRelname()]; isCTE {
			return
		}
	}

	key := TableName{
		Catalog: rv.GetCatalogname(),
		Schema:  rv.GetSchemaname(),
		Name:    rv.GetRelname(),
	}
	if newName, ok := r.renames[key]; ok {
		oldRel := rv.GetRelname()
		oldSchema := rv.GetSchemaname()

		rv.Catalogname = newName.Catalog
		rv.Schemaname = newName.Schema
		rv.Relname = newName.Name

		// Track column-ref prefix renames for unaliased tables.
		if rv.GetAlias() == nil {
			if oldSchema != "" {
				r.colRefRenames[colRefPrefix{A: oldSchema, B: oldRel}] =
					colRefPrefix{A: newName.Schema, B: newName.Name}
			} else {
				r.colRefRenames[colRefPrefix{A: oldRel}] =
					colRefPrefix{A: newName.Name}
			}
		}
	}
}

func walkNodeRename(node *pg_query.Node, r *renamer) {
	if node == nil {
		return
	}

	switch v := node.GetNode().(type) {
	case *pg_query.Node_RangeVar:
		r.applyRangeVar(v.RangeVar)
		walkMessageRename(v.RangeVar.ProtoReflect(), r)
	default:
		walkMessageRename(node.ProtoReflect(), r)
	}
}

func walkMessageRename(msg protoreflect.Message, r *renamer) {
	msg.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		switch {
		case fd.IsList() && fd.Kind() == protoreflect.MessageKind:
			list := v.List()
			for i := 0; i < list.Len(); i++ {
				walkReflectMessageRename(list.Get(i).Message(), r)
			}
		case fd.Kind() == protoreflect.MessageKind:
			walkReflectMessageRename(v.Message(), r)
		}
		return true
	})
}

func walkReflectMessageRename(msg protoreflect.Message, r *renamer) {
	iface := msg.Interface()

	if node, ok := iface.(*pg_query.Node); ok {
		walkNodeRename(node, r)
		return
	}

	if rv, ok := iface.(*pg_query.RangeVar); ok {
		r.applyRangeVar(rv)
		return
	}

	walkMessageRename(msg, r)
}

// walkNodeColRef walks the AST looking for ColumnRef nodes whose leading
// field(s) match an unaliased renamed table, and updates them.
func walkNodeColRef(node *pg_query.Node, r *renamer) {
	if node == nil {
		return
	}

	if cr, ok := node.GetNode().(*pg_query.Node_ColumnRef); ok {
		r.applyColumnRef(cr.ColumnRef)
		return
	}

	walkMessageColRef(node.ProtoReflect(), r)
}

func walkMessageColRef(msg protoreflect.Message, r *renamer) {
	msg.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		switch {
		case fd.IsList() && fd.Kind() == protoreflect.MessageKind:
			list := v.List()
			for i := 0; i < list.Len(); i++ {
				walkReflectMessageColRef(list.Get(i).Message(), r)
			}
		case fd.Kind() == protoreflect.MessageKind:
			walkReflectMessageColRef(v.Message(), r)
		}
		return true
	})
}

func walkReflectMessageColRef(msg protoreflect.Message, r *renamer) {
	iface := msg.Interface()
	if node, ok := iface.(*pg_query.Node); ok {
		walkNodeColRef(node, r)
		return
	}
	walkMessageColRef(msg, r)
}

func (r *renamer) applyColumnRef(cr *pg_query.ColumnRef) {
	if cr == nil {
		return
	}
	fields := cr.GetFields()
	if len(fields) < 2 {
		return // unqualified column, nothing to rename
	}

	// Try two-part prefix match: fields[0].fields[1] = schema.table
	if len(fields) >= 3 {
		s0 := nodeString(fields[0])
		s1 := nodeString(fields[1])
		if s0 != "" && s1 != "" {
			key := colRefPrefix{A: s0, B: s1}
			if newP, ok := r.colRefRenames[key]; ok {
				setNodeString(fields[0], newP.A)
				setNodeString(fields[1], newP.B)
				return
			}
		}
	}

	// Try single-part prefix match: fields[0] = table
	s0 := nodeString(fields[0])
	if s0 != "" {
		key := colRefPrefix{A: s0}
		if newP, ok := r.colRefRenames[key]; ok {
			setNodeString(fields[0], newP.A)
			return
		}
	}
}

func nodeString(n *pg_query.Node) string {
	if s, ok := n.GetNode().(*pg_query.Node_String_); ok {
		return s.String_.GetSval()
	}
	return ""
}

func setNodeString(n *pg_query.Node, val string) {
	if s, ok := n.GetNode().(*pg_query.Node_String_); ok {
		s.String_.Sval = val
	}
}

// collectCTENames walks the AST and records all CTE definition names.
func collectCTENames(node *pg_query.Node, names map[string]struct{}) {
	if node == nil {
		return
	}
	walkMessageCollectCTEs(node.ProtoReflect(), names)
}

func walkMessageCollectCTEs(msg protoreflect.Message, names map[string]struct{}) {
	iface := msg.Interface()
	if cte, ok := iface.(*pg_query.CommonTableExpr); ok {
		names[cte.GetCtename()] = struct{}{}
	}
	msg.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		switch {
		case fd.IsList() && fd.Kind() == protoreflect.MessageKind:
			list := v.List()
			for i := 0; i < list.Len(); i++ {
				walkMessageCollectCTEs(list.Get(i).Message(), names)
			}
		case fd.Kind() == protoreflect.MessageKind:
			walkMessageCollectCTEs(v.Message(), names)
		}
		return true
	})
}
