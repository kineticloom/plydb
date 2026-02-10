package sqlwalk

import (
	"fmt"
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v6"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// OpType describes how a table is being accessed.
type OpType int

const (
	OpSelect      OpType = iota // SELECT / FROM / JOIN / USING
	OpInsert                    // INSERT target
	OpUpdate                    // UPDATE target
	OpDelete                    // DELETE target
	OpTruncate                  // TRUNCATE target
	OpAlterTable                // ALTER TABLE target
	OpCreateIndex               // CREATE INDEX target
	OpDropIndex                 // DROP INDEX target
	OpDropTable                 // DROP TABLE target
)

func (o OpType) String() string {
	switch o {
	case OpSelect:
		return "SELECT"
	case OpInsert:
		return "INSERT"
	case OpUpdate:
		return "UPDATE"
	case OpDelete:
		return "DELETE"
	case OpTruncate:
		return "TRUNCATE"
	case OpAlterTable:
		return "ALTER TABLE"
	case OpCreateIndex:
		return "CREATE INDEX"
	case OpDropIndex:
		return "DROP INDEX"
	case OpDropTable:
		return "DROP TABLE"
	default:
		return "unknown"
	}
}

// ActionString returns the SQL action verb used for matching granular actions.
func (o OpType) ActionString() string {
	switch o {
	case OpSelect:
		return "SELECT"
	case OpInsert:
		return "INSERT"
	case OpUpdate:
		return "UPDATE"
	case OpDelete:
		return "DELETE"
	case OpTruncate:
		return "TRUNCATE"
	default:
		return ""
	}
}

// tableAccess pairs a table reference with the operation being performed.
type tableAccess struct {
	ref TableRef
	op  OpType
}

// Violation describes a single access policy violation.
type Violation struct {
	Table TableRef
	Op    OpType
}

func (v Violation) Error() string {
	parts := []string{}
	if v.Table.Catalog != "" {
		parts = append(parts, v.Table.Catalog)
	}
	if v.Table.Schema != "" {
		parts = append(parts, v.Table.Schema)
	}
	parts = append(parts, v.Table.Name)
	name := strings.Join(parts, ".")
	return fmt.Sprintf("%s on %s denied", v.Op, name)
}

// ValidateOption configures the behavior of Validate.
type ValidateOption func(*validateConfig)

type validateConfig struct {
	failFast bool
}

// FailFast causes Validate to return immediately after the first violation.
func FailFast() ValidateOption {
	return func(c *validateConfig) { c.failFast = true }
}

// Validate checks that the parsed SQL query only accesses tables in ways
// permitted by the given policy. It returns a Violation for each table
// access that violates the policy.
func Validate(parsed *pg_query.ParseResult, policy *Policy, opts ...ValidateOption) ([]Violation, error) {
	var cfg validateConfig
	for _, o := range opts {
		o(&cfg)
	}

	rp, err := resolve(policy)
	if err != nil {
		return nil, fmt.Errorf("resolving policy: %w", err)
	}

	accesses := extractAccesses(parsed)

	var violations []Violation
	for _, a := range accesses {
		perm := rp.lookup(a.ref.Catalog, a.ref.Schema, a.ref.Name)
		if !perm.allows(a.op) {
			violations = append(violations, Violation{
				Table: a.ref,
				Op:    a.op,
			})
			if cfg.failFast {
				return violations, nil
			}
		}
	}
	return violations, nil
}

// extractAccesses walks the AST and returns every table access with its
// operation type. CTE names are excluded.
func extractAccesses(parsed *pg_query.ParseResult) []tableAccess {
	v := &accessCollector{
		cteNames: make(map[string]struct{}),
	}

	// First pass: collect CTE names.
	for _, stmt := range parsed.GetStmts() {
		if stmt.GetStmt() != nil {
			collectCTENames(stmt.GetStmt(), v.cteNames)
		}
	}

	// Second pass: collect accesses.
	for _, stmt := range parsed.GetStmts() {
		if stmt.GetStmt() != nil {
			v.walkTopLevel(stmt.GetStmt())
		}
	}

	// Filter out CTE references.
	var result []tableAccess
	for _, a := range v.accesses {
		if a.ref.Catalog == "" && a.ref.Schema == "" {
			if _, isCTE := v.cteNames[a.ref.Name]; isCTE {
				continue
			}
		}
		result = append(result, a)
	}
	return result
}

type accessCollector struct {
	accesses []tableAccess
	cteNames map[string]struct{}
}

func (ac *accessCollector) addTable(rv *pg_query.RangeVar, op OpType) {
	if rv == nil {
		return
	}
	ac.accesses = append(ac.accesses, tableAccess{
		ref: TableRef{
			Catalog: rv.GetCatalogname(),
			Schema:  rv.GetSchemaname(),
			Name:    rv.GetRelname(),
			Alias:   aliasName(rv.GetAlias()),
		},
		op: op,
	})
}

func (ac *accessCollector) addTableDirect(catalog, schema, name string, op OpType) {
	ac.accesses = append(ac.accesses, tableAccess{
		ref: TableRef{
			Catalog: catalog,
			Schema:  schema,
			Name:    name,
		},
		op: op,
	})
}

// walkTopLevel dispatches on the top-level statement type to correctly
// classify target tables by their operation type.
func (ac *accessCollector) walkTopLevel(node *pg_query.Node) {
	if node == nil {
		return
	}

	switch v := node.GetNode().(type) {
	case *pg_query.Node_SelectStmt:
		ac.walkSelectStmt(v.SelectStmt)

	case *pg_query.Node_InsertStmt:
		ac.addTable(v.InsertStmt.GetRelation(), OpInsert)
		if v.InsertStmt.GetSelectStmt() != nil {
			ac.walkAllReadsNode(v.InsertStmt.GetSelectStmt())
		}
		for _, w := range v.InsertStmt.GetWithClause().GetCtes() {
			ac.walkAllReadsNode(w)
		}

	case *pg_query.Node_UpdateStmt:
		ac.addTable(v.UpdateStmt.GetRelation(), OpUpdate)
		for _, f := range v.UpdateStmt.GetFromClause() {
			ac.walkAllReadsNode(f)
		}
		ac.walkAllReadsNode(v.UpdateStmt.GetWhereClause())
		for _, w := range v.UpdateStmt.GetWithClause().GetCtes() {
			ac.walkAllReadsNode(w)
		}

	case *pg_query.Node_DeleteStmt:
		ac.addTable(v.DeleteStmt.GetRelation(), OpDelete)
		for _, u := range v.DeleteStmt.GetUsingClause() {
			ac.walkAllReadsNode(u)
		}
		ac.walkAllReadsNode(v.DeleteStmt.GetWhereClause())
		for _, w := range v.DeleteStmt.GetWithClause().GetCtes() {
			ac.walkAllReadsNode(w)
		}

	case *pg_query.Node_MergeStmt:
		ac.walkMergeStmt(v.MergeStmt)

	case *pg_query.Node_TruncateStmt:
		for _, rel := range v.TruncateStmt.GetRelations() {
			if rv, ok := rel.GetNode().(*pg_query.Node_RangeVar); ok {
				ac.addTable(rv.RangeVar, OpTruncate)
			}
		}

	case *pg_query.Node_AlterTableStmt:
		ac.addTable(v.AlterTableStmt.GetRelation(), OpAlterTable)

	case *pg_query.Node_IndexStmt:
		ac.addTable(v.IndexStmt.GetRelation(), OpCreateIndex)

	case *pg_query.Node_DropStmt:
		ac.walkDropStmt(v.DropStmt)

	default:
		ac.walkAllReadsNode(node)
	}
}

// walkMergeStmt inspects MERGE WHEN clauses to determine the precise
// operations performed on the target table.
func (ac *accessCollector) walkMergeStmt(stmt *pg_query.MergeStmt) {
	if stmt == nil {
		return
	}

	// Determine which DML ops the MERGE performs from its WHEN clauses.
	ops := make(map[OpType]struct{})
	for _, clause := range stmt.GetMergeWhenClauses() {
		if mwc, ok := clause.GetNode().(*pg_query.Node_MergeWhenClause); ok {
			switch mwc.MergeWhenClause.GetCommandType() {
			case pg_query.CmdType_CMD_INSERT:
				ops[OpInsert] = struct{}{}
			case pg_query.CmdType_CMD_UPDATE:
				ops[OpUpdate] = struct{}{}
			case pg_query.CmdType_CMD_DELETE:
				ops[OpDelete] = struct{}{}
			}
		}
	}

	// Emit one access per distinct operation type on the target.
	rel := stmt.GetRelation()
	for op := range ops {
		ac.addTable(rel, op)
	}
	// The target is always read (for the ON condition).
	ac.addTable(rel, OpSelect)

	// Source relation is a read.
	ac.walkAllReadsNode(stmt.GetSourceRelation())
	for _, w := range stmt.GetWithClause().GetCtes() {
		ac.walkAllReadsNode(w)
	}
	for _, clause := range stmt.GetMergeWhenClauses() {
		ac.walkAllReadsNode(clause)
	}
}

// walkDropStmt handles DROP TABLE and DROP INDEX statements by extracting
// object names from the objects list.
func (ac *accessCollector) walkDropStmt(stmt *pg_query.DropStmt) {
	if stmt == nil {
		return
	}
	var op OpType
	switch stmt.GetRemoveType() {
	case pg_query.ObjectType_OBJECT_TABLE:
		op = OpDropTable
	case pg_query.ObjectType_OBJECT_INDEX:
		op = OpDropIndex
	default:
		return
	}
	for _, obj := range stmt.GetObjects() {
		list, ok := obj.GetNode().(*pg_query.Node_List)
		if !ok {
			continue
		}
		var catalog, schema, name string
		items := list.List.GetItems()
		switch len(items) {
		case 1:
			name = nodeString(items[0])
		case 2:
			schema = nodeString(items[0])
			name = nodeString(items[1])
		case 3:
			catalog = nodeString(items[0])
			schema = nodeString(items[1])
			name = nodeString(items[2])
		}
		if name != "" {
			ac.addTableDirect(catalog, schema, name, op)
		}
	}
}

// walkSelectStmt handles SELECT, which may be a UNION/INTERSECT/EXCEPT.
func (ac *accessCollector) walkSelectStmt(sel *pg_query.SelectStmt) {
	if sel == nil {
		return
	}
	if sel.GetLarg() != nil || sel.GetRarg() != nil {
		ac.walkSelectStmt(sel.GetLarg())
		ac.walkSelectStmt(sel.GetRarg())
		return
	}
	for _, f := range sel.GetFromClause() {
		ac.walkAllReadsNode(f)
	}
	ac.walkAllReadsNode(sel.GetWhereClause())
	for _, w := range sel.GetWithClause().GetCtes() {
		ac.walkAllReadsNode(w)
	}
	for _, t := range sel.GetTargetList() {
		ac.walkAllReadsNode(t)
	}
	ac.walkAllReadsNode(sel.GetHavingClause())
}

// walkAllReadsNode walks the AST treating every RangeVar encountered as a
// read (SELECT) operation.
func (ac *accessCollector) walkAllReadsNode(node *pg_query.Node) {
	if node == nil {
		return
	}

	switch v := node.GetNode().(type) {
	case *pg_query.Node_RangeVar:
		ac.addTable(v.RangeVar, OpSelect)

	case *pg_query.Node_SelectStmt:
		ac.walkSelectStmt(v.SelectStmt)

	case *pg_query.Node_InsertStmt:
		ac.addTable(v.InsertStmt.GetRelation(), OpInsert)
		if v.InsertStmt.GetSelectStmt() != nil {
			ac.walkAllReadsNode(v.InsertStmt.GetSelectStmt())
		}

	case *pg_query.Node_UpdateStmt:
		ac.addTable(v.UpdateStmt.GetRelation(), OpUpdate)
		for _, f := range v.UpdateStmt.GetFromClause() {
			ac.walkAllReadsNode(f)
		}

	case *pg_query.Node_DeleteStmt:
		ac.addTable(v.DeleteStmt.GetRelation(), OpDelete)
		for _, u := range v.DeleteStmt.GetUsingClause() {
			ac.walkAllReadsNode(u)
		}

	default:
		ac.walkAllReadsMessage(node.ProtoReflect())
	}
}

// walkAllReadsMessage generically walks protobuf fields looking for nodes.
func (ac *accessCollector) walkAllReadsMessage(msg protoreflect.Message) {
	msg.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		switch {
		case fd.IsList() && fd.Kind() == protoreflect.MessageKind:
			list := v.List()
			for i := 0; i < list.Len(); i++ {
				ac.walkAllReadsReflect(list.Get(i).Message())
			}
		case fd.Kind() == protoreflect.MessageKind:
			ac.walkAllReadsReflect(v.Message())
		}
		return true
	})
}

func (ac *accessCollector) walkAllReadsReflect(msg protoreflect.Message) {
	iface := msg.Interface()

	if node, ok := iface.(*pg_query.Node); ok {
		ac.walkAllReadsNode(node)
		return
	}

	if rv, ok := iface.(*pg_query.RangeVar); ok {
		ac.addTable(rv, OpSelect)
		return
	}

	ac.walkAllReadsMessage(msg)
}
