package ast

import (
	"strings"
)

// Visitor interface for traversing AST nodes.
type Visitor interface {
	Visit(node Node) (w Visitor)
}

// Walk traverses an AST in depth-first order.
func Walk(v Visitor, node Node) {
	if node == nil {
		return
	}
	w := v.Visit(node)
	if w == nil {
		return
	}

	switch n := node.(type) {
	case *SelectStmt:
		if n.With != nil {
			Walk(w, n.With)
		}
		for _, p := range n.Projections {
			Walk(w, p)
		}
		if n.From != nil {
			Walk(w, n.From)
		}
		if n.Where != nil {
			Walk(w, n.Where)
		}
		for _, g := range n.GroupBy {
			Walk(w, g)
		}
		if n.Having != nil {
			Walk(w, n.Having)
		}
		for _, o := range n.OrderBy {
			Walk(w, o)
		}
	case *WithClause:
		for _, cte := range n.CTEs {
			Walk(w, cte)
		}
	case *CTE:
		if n.Query != nil {
			Walk(w, n.Query)
		}
	case *Projection:
		Walk(w, n.Expr)
	case *FromClause:
		if n.Primary != nil {
			Walk(w, n.Primary)
		}
		for _, j := range n.Joins {
			Walk(w, j)
		}
	case *TableRef:
		if n.Subquery != nil {
			Walk(w, n.Subquery)
		}
	case *JoinClause:
		if n.Table != nil {
			Walk(w, n.Table)
		}
		if n.Condition != nil {
			Walk(w, n.Condition)
		}
	case *BinaryExpr:
		Walk(w, n.Left)
		Walk(w, n.Right)
	case *UnaryExpr:
		Walk(w, n.Expr)
	case *FunctionCall:
		for _, arg := range n.Args {
			Walk(w, arg)
		}
	case *ParenExpr:
		Walk(w, n.Inner)
	case *IsNullExpr:
		Walk(w, n.Expr)
	case *InExpr:
		Walk(w, n.Expr)
		for _, item := range n.List {
			Walk(w, item)
		}
		if n.Subquery != nil {
			Walk(w, n.Subquery)
		}
	case *CaseExpr:
		if n.BaseExpr != nil {
			Walk(w, n.BaseExpr)
		}
		for _, wc := range n.Whens {
			Walk(w, wc.When)
			Walk(w, wc.Then)
		}
		if n.ElseExpr != nil {
			Walk(w, n.ElseExpr)
		}
	case *CastExpr:
		Walk(w, n.Expr)
	case *OrderByItem:
		Walk(w, n.Expr)
	}
}

// TableExtractor collects all referenced tables in a query.
type TableExtractor struct {
	Tables []*TableRef
}

func (t *TableExtractor) Visit(node Node) Visitor {
	if tbl, ok := node.(*TableRef); ok {
		t.Tables = append(t.Tables, tbl)
	}
	return t
}

// ExtractTables returns all tables referenced in stmt.
func ExtractTables(stmt *SelectStmt) []*TableRef {
	ext := &TableExtractor{}
	Walk(ext, stmt)
	return ext.Tables
}

// ExtractAliases builds a map of alias/table -> actual table name.
func ExtractAliases(stmt *SelectStmt) map[string]string {
	tables := ExtractTables(stmt)
	res := make(map[string]string)
	for _, tbl := range tables {
		if tbl.Name != "" {
			res[strings.ToLower(tbl.Name)] = tbl.Name
			if tbl.Alias != "" {
				res[strings.ToLower(tbl.Alias)] = tbl.Name
			}
		}
	}
	return res
}

// AggregateExtractor collects all aggregate function calls.
type AggregateExtractor struct {
	Aggregates []*FunctionCall
}

func (a *AggregateExtractor) Visit(node Node) Visitor {
	if fn, ok := node.(*FunctionCall); ok {
		switch strings.ToUpper(fn.Name) {
		case "SUM", "AVG", "COUNT", "MIN", "MAX", "STDDEV", "VARIANCE":
			a.Aggregates = append(a.Aggregates, fn)
		}
	}
	return a
}

// ExtractAggregates returns all aggregate function calls in stmt.
func ExtractAggregates(stmt *SelectStmt) []*FunctionCall {
	ext := &AggregateExtractor{}
	Walk(ext, stmt)
	return ext.Aggregates
}

// ColumnExtractor collects all column references from an expression.
type ColumnExtractor struct {
	Columns []*ColumnRef
}

func (c *ColumnExtractor) Visit(node Node) Visitor {
	if col, ok := node.(*ColumnRef); ok {
		c.Columns = append(c.Columns, col)
	}
	return c
}

// ExtractColumns returns all ColumnRefs under expr.
func ExtractColumns(expr Node) []*ColumnRef {
	ext := &ColumnExtractor{}
	Walk(ext, expr)
	return ext.Columns
}

// ExtractWhereConjunctions splits a WHERE expression into a slice of AND-ed conditions.
func ExtractWhereConjunctions(expr Expr) []Expr {
	if expr == nil {
		return nil
	}
	var res []Expr
	var flatten func(e Expr)
	flatten = func(e Expr) {
		if bin, ok := e.(*BinaryExpr); ok && strings.EqualFold(bin.Op, "AND") {
			flatten(bin.Left)
			flatten(bin.Right)
		} else if paren, ok := e.(*ParenExpr); ok {
			flatten(paren.Inner)
		} else {
			res = append(res, e)
		}
	}
	flatten(expr)
	return res
}

// InspectJoinCondition inspects an equality join predicate (e.g. a.id = b.cust_id).
func InspectJoinCondition(cond Expr) (leftCol *ColumnRef, rightCol *ColumnRef, ok bool) {
	if cond == nil {
		return nil, nil, false
	}
	bin, isBin := cond.(*BinaryExpr)
	if !isBin || bin.Op != "=" {
		return nil, nil, false
	}

	lCol, lOk := bin.Left.(*ColumnRef)
	rCol, rOk := bin.Right.(*ColumnRef)
	if lOk && rOk {
		return lCol, rCol, true
	}
	return nil, nil, false
}
