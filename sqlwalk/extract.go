package sqlwalk

import (
	pg_query "github.com/pganalyze/pg_query_go/v6"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// collector accumulates table references and CTE names during the walk.
type collector struct {
	tables   []TableRef
	cteNames map[string]struct{}
}

// walkNode inspects a single AST Node. If the node wraps a RangeVar it
// extracts the table reference; otherwise it recurses via the generic
// protobuf reflection walker.
func walkNode(node *pg_query.Node, c *collector) {
	if node == nil {
		return
	}

	switch v := node.GetNode().(type) {
	case *pg_query.Node_RangeVar:
		rv := v.RangeVar
		c.tables = append(c.tables, TableRef{
			Catalog: rv.GetCatalogname(),
			Schema:  rv.GetSchemaname(),
			Name:    rv.GetRelname(),
			Alias:   aliasName(rv.GetAlias()),
		})
		walkMessage(rv.ProtoReflect(), c)

	default:
		walkMessage(node.ProtoReflect(), c)
	}
}

// walkMessage generically walks all fields of a protobuf message,
// recursing into sub-messages and repeated message fields.
func walkMessage(msg protoreflect.Message, c *collector) {
	msg.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		switch {
		case fd.IsList() && fd.Kind() == protoreflect.MessageKind:
			list := v.List()
			for i := 0; i < list.Len(); i++ {
				walkReflectMessage(list.Get(i).Message(), c)
			}
		case fd.Kind() == protoreflect.MessageKind:
			walkReflectMessage(v.Message(), c)
		}
		return true
	})
}

// walkReflectMessage converts a reflected message back to a concrete Go
// type. If it is a *pg_query.Node we dispatch to walkNode so that
// RangeVar extraction fires; everything else just recurses through
// walkMessage.
func walkReflectMessage(msg protoreflect.Message, c *collector) {
	iface := msg.Interface()

	if node, ok := iface.(*pg_query.Node); ok {
		walkNode(node, c)
		return
	}

	// Some statements (INSERT, UPDATE, DELETE, MERGE) store the target
	// table as a *RangeVar field directly, not wrapped in a Node.
	if rv, ok := iface.(*pg_query.RangeVar); ok {
		c.tables = append(c.tables, TableRef{
			Catalog: rv.GetCatalogname(),
			Schema:  rv.GetSchemaname(),
			Name:    rv.GetRelname(),
			Alias:   aliasName(rv.GetAlias()),
		})
		return
	}

	// Record CTE names so we can filter them from the final result.
	if cte, ok := iface.(*pg_query.CommonTableExpr); ok {
		c.cteNames[cte.GetCtename()] = struct{}{}
		walkMessage(msg, c)
		return
	}

	walkMessage(msg, c)
}

// aliasName returns the alias name or "" if the alias is nil.
func aliasName(a *pg_query.Alias) string {
	if a == nil {
		return ""
	}
	return a.GetAliasname()
}
