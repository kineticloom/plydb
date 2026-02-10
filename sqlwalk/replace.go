package sqlwalk

import (
	pg_query "github.com/pganalyze/pg_query_go/v6"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// FuncReplace describes a function call that should replace a table reference.
type FuncReplace struct {
	FuncName  string      // e.g. "read_xlsx"
	Args      []string    // positional string arguments
	NamedArgs [][2]string // ordered key-value pairs (deparsed as key => 'value')
}

// ReplaceTablesWithFunctions walks the pg_query AST and replaces every
// RangeVar whose (catalog, schema, name) matches a key in the replacements
// map with a RangeFunction containing the specified function call.
// The AST is mutated in place and also returned for convenience.
func ReplaceTablesWithFunctions(parsed *pg_query.ParseResult, replacements map[TableName]FuncReplace) *pg_query.ParseResult {
	// Collect CTE names so we can skip them.
	cteNames := make(map[string]struct{})
	for _, stmt := range parsed.GetStmts() {
		if stmt.GetStmt() != nil {
			collectCTENames(stmt.GetStmt(), cteNames)
		}
	}

	rep := &replacer{
		replacements: replacements,
		cteNames:     cteNames,
	}
	for _, stmt := range parsed.GetStmts() {
		if stmt.GetStmt() != nil {
			walkNodeReplace(stmt.GetStmt(), rep)
		}
	}
	return parsed
}

type replacer struct {
	replacements map[TableName]FuncReplace
	cteNames     map[string]struct{}
}

func walkNodeReplace(node *pg_query.Node, rep *replacer) {
	if node == nil {
		return
	}

	if rv, ok := node.GetNode().(*pg_query.Node_RangeVar); ok {
		if tryReplace(node, rv.RangeVar, rep) {
			return
		}
	}

	walkMessageReplace(node.ProtoReflect(), rep)
}

func tryReplace(node *pg_query.Node, rv *pg_query.RangeVar, rep *replacer) bool {
	if rv == nil {
		return false
	}
	// Skip CTE references.
	if rv.GetCatalogname() == "" && rv.GetSchemaname() == "" {
		if _, isCTE := rep.cteNames[rv.GetRelname()]; isCTE {
			return false
		}
	}

	key := TableName{
		Catalog: rv.GetCatalogname(),
		Schema:  rv.GetSchemaname(),
		Name:    rv.GetRelname(),
	}
	fr, ok := rep.replacements[key]
	if !ok {
		return false
	}

	// Build function call arguments.
	var args []*pg_query.Node
	for _, a := range fr.Args {
		args = append(args, pg_query.MakeAConstStrNode(a, 0))
	}
	for _, na := range fr.NamedArgs {
		args = append(args, &pg_query.Node{
			Node: &pg_query.Node_NamedArgExpr{
				NamedArgExpr: &pg_query.NamedArgExpr{
					Arg:       pg_query.MakeAConstStrNode(na[1], 0),
					Name:      na[0],
					Argnumber: -1,
				},
			},
		})
	}

	funcCall := pg_query.MakeFuncCallNode(
		[]*pg_query.Node{pg_query.MakeStrNode(fr.FuncName)},
		args,
		0,
	)

	// RangeFunction.Functions is a list of two-element lists: [funcexpr, coldeflist].
	funcItem := pg_query.MakeListNode([]*pg_query.Node{funcCall, {Node: nil}})

	rf := &pg_query.RangeFunction{
		Functions: []*pg_query.Node{funcItem},
		Alias:     rv.GetAlias(),
	}

	node.Node = &pg_query.Node_RangeFunction{RangeFunction: rf}
	return true
}

func walkMessageReplace(msg protoreflect.Message, rep *replacer) {
	msg.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		switch {
		case fd.IsList() && fd.Kind() == protoreflect.MessageKind:
			list := v.List()
			for i := 0; i < list.Len(); i++ {
				walkReflectMessageReplace(list.Get(i).Message(), rep)
			}
		case fd.Kind() == protoreflect.MessageKind:
			walkReflectMessageReplace(v.Message(), rep)
		}
		return true
	})
}

func walkReflectMessageReplace(msg protoreflect.Message, rep *replacer) {
	iface := msg.Interface()
	if node, ok := iface.(*pg_query.Node); ok {
		walkNodeReplace(node, rep)
		return
	}
	walkMessageReplace(msg, rep)
}
