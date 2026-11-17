package ast

import (
	"fmt"
	"strings"
)

// Node is the interface implemented by all AST nodes.
type Node interface {
	ToSQL() string
}

// Expr represents any SQL expression.
type Expr interface {
	Node
	exprNode()
}

type exprBase struct{}

func (exprBase) exprNode() {}

// SelectStmt represents a SQL SELECT statement.
type SelectStmt struct {
	With        *WithClause
	Distinct    bool
	Projections []*Projection
	From        *FromClause
	Where       Expr
	GroupBy     []Expr
	Having      Expr
	OrderBy     []*OrderByItem
	Limit       Expr
	Offset      Expr
}

func (s *SelectStmt) ToSQL() string {
	var b strings.Builder

	if s.With != nil && len(s.With.CTEs) > 0 {
		b.WriteString(s.With.ToSQL())
		b.WriteString("\n")
	}

	b.WriteString("SELECT ")
	if s.Distinct {
		b.WriteString("DISTINCT ")
	}

	for i, p := range s.Projections {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(p.ToSQL())
	}

	if s.From != nil {
		b.WriteString("\nFROM ")
		b.WriteString(s.From.ToSQL())
	}

	if s.Where != nil {
		b.WriteString("\nWHERE ")
		b.WriteString(s.Where.ToSQL())
	}

	if len(s.GroupBy) > 0 {
		b.WriteString("\nGROUP BY ")
		for i, g := range s.GroupBy {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(g.ToSQL())
		}
	}

	if s.Having != nil {
		b.WriteString("\nHAVING ")
		b.WriteString(s.Having.ToSQL())
	}

	if len(s.OrderBy) > 0 {
		b.WriteString("\nORDER BY ")
		for i, o := range s.OrderBy {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(o.ToSQL())
		}
	}

	if s.Limit != nil {
		b.WriteString("\nLIMIT ")
		b.WriteString(s.Limit.ToSQL())
	}

	if s.Offset != nil {
		b.WriteString(" OFFSET ")
		b.WriteString(s.Offset.ToSQL())
	}

	return b.String()
}

// WithClause represents CTE with statements.
type WithClause struct {
	Recursive bool
	CTEs      []*CTE
}

func (w *WithClause) ToSQL() string {
	var b strings.Builder
	b.WriteString("WITH ")
	if w.Recursive {
		b.WriteString("RECURSIVE ")
	}
	for i, c := range w.CTEs {
		if i > 0 {
			b.WriteString(",\n")
		}
		b.WriteString(c.ToSQL())
	}
	return b.String()
}

// CTE represents a single Common Table Expression.
type CTE struct {
	Name    string
	Columns []string
	Query   *SelectStmt
}

func (c *CTE) ToSQL() string {
	var b strings.Builder
	b.WriteString(c.Name)
	if len(c.Columns) > 0 {
		b.WriteString(" (")
		b.WriteString(strings.Join(c.Columns, ", "))
		b.WriteString(")")
	}
	b.WriteString(" AS (\n")
	b.WriteString(c.Query.ToSQL())
	b.WriteString("\n)")
	return b.String()
}

// Projection represents a projected column expression with an optional alias.
type Projection struct {
	Expr  Expr
	Alias string
}

func (p *Projection) ToSQL() string {
	sql := p.Expr.ToSQL()
	if p.Alias != "" {
		return fmt.Sprintf("%s AS %s", sql, p.Alias)
	}
	return sql
}

// FromClause represents the FROM source and joins.
type FromClause struct {
	Primary *TableRef
	Joins   []*JoinClause
}

func (f *FromClause) ToSQL() string {
	var b strings.Builder
	b.WriteString(f.Primary.ToSQL())
	for _, j := range f.Joins {
		b.WriteString("\n")
		b.WriteString(j.ToSQL())
	}
	return b.String()
}

// TableRef represents a referenced table or subquery.
type TableRef struct {
	Schema   string
	Name     string
	Alias    string
	Subquery *SelectStmt
}

func (t *TableRef) FullName() string {
	if t.Schema != "" {
		return t.Schema + "." + t.Name
	}
	return t.Name
}

func (t *TableRef) EffectiveName() string {
	if t.Alias != "" {
		return t.Alias
	}
	return t.Name
}

func (t *TableRef) ToSQL() string {
	var b strings.Builder
	if t.Subquery != nil {
		b.WriteString("(\n")
		b.WriteString(t.Subquery.ToSQL())
		b.WriteString("\n)")
	} else if t.Schema != "" {
		b.WriteString(t.Schema + "." + t.Name)
	} else {
		b.WriteString(t.Name)
	}

	if t.Alias != "" {
		b.WriteString(" AS " + t.Alias)
	}
	return b.String()
}

// JoinType denotes the type of join.
type JoinType string

const (
	JoinInner JoinType = "INNER JOIN"
	JoinLeft  JoinType = "LEFT JOIN"
	JoinRight JoinType = "RIGHT JOIN"
	JoinFull  JoinType = "FULL JOIN"
	JoinCross JoinType = "CROSS JOIN"
)

// JoinClause represents a SQL JOIN.
type JoinClause struct {
	Type      JoinType
	Table     *TableRef
	Condition Expr
	Using     []string
}

func (j *JoinClause) ToSQL() string {
	var b strings.Builder
	b.WriteString(string(j.Type))
	b.WriteString(" ")
	b.WriteString(j.Table.ToSQL())

	if j.Condition != nil {
		b.WriteString(" ON ")
		b.WriteString(j.Condition.ToSQL())
	} else if len(j.Using) > 0 {
		b.WriteString(" USING (")
		b.WriteString(strings.Join(j.Using, ", "))
		b.WriteString(")")
	}
	return b.String()
}

// BinaryExpr represents binary operations (A + B, A = B, A AND B).
type BinaryExpr struct {
	exprBase
	Left  Expr
	Op    string
	Right Expr
}

func (b *BinaryExpr) ToSQL() string {
	return fmt.Sprintf("%s %s %s", b.Left.ToSQL(), b.Op, b.Right.ToSQL())
}

// UnaryExpr represents unary operations (NOT A, -A).
type UnaryExpr struct {
	exprBase
	Op   string
	Expr Expr
}

func (u *UnaryExpr) ToSQL() string {
	return fmt.Sprintf("%s %s", u.Op, u.Expr.ToSQL())
}

// ColumnRef represents a reference to a table column (t.col or col).
type ColumnRef struct {
	exprBase
	Table  string
	Column string
}

func (c *ColumnRef) ToSQL() string {
	if c.Table != "" {
		return fmt.Sprintf("%s.%s", c.Table, c.Column)
	}
	return c.Column
}

// Literal represents a constant value in SQL.
type Literal struct {
	exprBase
	Value string
	IsStr bool
}

func (l *Literal) ToSQL() string {
	if l.IsStr {
		return fmt.Sprintf("'%s'", strings.ReplaceAll(l.Value, "'", "''"))
	}
	return l.Value
}

// FunctionCall represents SQL function invocations like SUM(x), AVG(x), COALESCE(a, b).
type FunctionCall struct {
	exprBase
	Name     string
	Distinct bool
	Args     []Expr
	Over     *WindowSpec
}

func (f *FunctionCall) ToSQL() string {
	var b strings.Builder
	b.WriteString(strings.ToUpper(f.Name))
	b.WriteString("(")
	if f.Distinct {
		b.WriteString("DISTINCT ")
	}
	for i, arg := range f.Args {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(arg.ToSQL())
	}
	b.WriteString(")")

	if f.Over != nil {
		b.WriteString(" OVER (")
		b.WriteString(f.Over.ToSQL())
		b.WriteString(")")
	}
	return b.String()
}

// WindowSpec represents OVER (PARTITION BY ... ORDER BY ...).
type WindowSpec struct {
	PartitionBy []Expr
	OrderBy     []*OrderByItem
}

func (w *WindowSpec) ToSQL() string {
	var parts []string
	if len(w.PartitionBy) > 0 {
		var p []string
		for _, item := range w.PartitionBy {
			p = append(p, item.ToSQL())
		}
		parts = append(parts, "PARTITION BY "+strings.Join(p, ", "))
	}
	if len(w.OrderBy) > 0 {
		var o []string
		for _, item := range w.OrderBy {
			o = append(o, item.ToSQL())
		}
		parts = append(parts, "ORDER BY "+strings.Join(o, ", "))
	}
	return strings.Join(parts, " ")
}

// OrderByItem represents an expression in ORDER BY with direction.
type OrderByItem struct {
	Expr Expr
	Desc bool
}

func (o *OrderByItem) ToSQL() string {
	dir := "ASC"
	if o.Desc {
		dir = "DESC"
	}
	return fmt.Sprintf("%s %s", o.Expr.ToSQL(), dir)
}

// ParenExpr represents parenthesized expressions (A + B).
type ParenExpr struct {
	exprBase
	Inner Expr
}

func (p *ParenExpr) ToSQL() string {
	return fmt.Sprintf("(%s)", p.Inner.ToSQL())
}

// IsNullExpr represents col IS NULL or col IS NOT NULL.
type IsNullExpr struct {
	exprBase
	Expr  Expr
	Isnot bool
}

func (i *IsNullExpr) ToSQL() string {
	if i.Isnot {
		return fmt.Sprintf("%s IS NOT NULL", i.Expr.ToSQL())
	}
	return fmt.Sprintf("%s IS NULL", i.Expr.ToSQL())
}

// InExpr represents col IN (1, 2, 3) or col IN (SELECT ...).
type InExpr struct {
	exprBase
	Expr     Expr
	Not      bool
	List     []Expr
	Subquery *SelectStmt
}

func (in *InExpr) ToSQL() string {
	var b strings.Builder
	b.WriteString(in.Expr.ToSQL())
	if in.Not {
		b.WriteString(" NOT IN ")
	} else {
		b.WriteString(" IN ")
	}

	if in.Subquery != nil {
		b.WriteString("(\n")
		b.WriteString(in.Subquery.ToSQL())
		b.WriteString("\n)")
	} else {
		b.WriteString("(")
		for i, item := range in.List {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(item.ToSQL())
		}
		b.WriteString(")")
	}
	return b.String()
}

// CaseExpr represents CASE WHEN ... THEN ... ELSE ... END.
type CaseExpr struct {
	exprBase
	BaseExpr Expr
	Whens    []*WhenClause
	ElseExpr Expr
}

type WhenClause struct {
	When Expr
	Then Expr
}

func (c *CaseExpr) ToSQL() string {
	var b strings.Builder
	b.WriteString("CASE")
	if c.BaseExpr != nil {
		b.WriteString(" " + c.BaseExpr.ToSQL())
	}
	for _, w := range c.Whens {
		b.WriteString(fmt.Sprintf(" WHEN %s THEN %s", w.When.ToSQL(), w.Then.ToSQL()))
	}
	if c.ElseExpr != nil {
		b.WriteString(fmt.Sprintf(" ELSE %s", c.ElseExpr.ToSQL()))
	}
	b.WriteString(" END")
	return b.String()
}

// SubqueryExpr represents a subquery wrapped in an expression.
type SubqueryExpr struct {
	exprBase
	Query *SelectStmt
}

func (s *SubqueryExpr) ToSQL() string {
	return fmt.Sprintf("(\n%s\n)", s.Query.ToSQL())
}

// CastExpr represents CAST(expr AS type) or expr::type.
type CastExpr struct {
	exprBase
	Expr Expr
	Type string
}

func (c *CastExpr) ToSQL() string {
	return fmt.Sprintf("CAST(%s AS %s)", c.Expr.ToSQL(), c.Type)
}
