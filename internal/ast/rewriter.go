package ast

import (
	"fmt"
	"strings"

	"github.com/alexandrmotologa/queryproof/internal/config"
)

// MutationRecord details an AST transformation made by the rewriter.
type MutationRecord struct {
	ProbeName   string `json:"probe_name"`
	Target      string `json:"target"`
	Description string `json:"description"`
	BeforeExpr  string `json:"before_expr"`
	AfterExpr   string `json:"after_expr"`
}

// Rewriter applies deterministic transformations to a SelectStmt.
type Rewriter struct {
	catalog   *config.Catalog
	mutations []MutationRecord
}

// NewRewriter creates a rewriter instance.
func NewRewriter(cat *config.Catalog) *Rewriter {
	return &Rewriter{
		catalog:   cat,
		mutations: make([]MutationRecord, 0),
	}
}

// GetMutations returns all applied mutations.
func (r *Rewriter) GetMutations() []MutationRecord {
	return r.mutations
}

// RewriteFanoutJoin transforms a fan-out 1:N join into a pre-aggregated CTE join.
func (r *Rewriter) RewriteFanoutJoin(stmt *SelectStmt, parentTbl, childTbl, parentCol, childCol string) bool {
	if stmt == nil || stmt.From == nil || len(stmt.From.Joins) == 0 {
		return false
	}

	_ = parentTbl
	childClean := strings.ToLower(childTbl)

	// Locate join on child table
	var targetJoin *JoinClause
	for _, j := range stmt.From.Joins {
		if strings.EqualFold(j.Table.Name, childClean) || strings.EqualFold(j.Table.EffectiveName(), childClean) {
			targetJoin = j
			break
		}
	}

	if targetJoin == nil {
		return false
	}

	childAlias := targetJoin.Table.EffectiveName()
	cteName := fmt.Sprintf("%s_agg", targetJoin.Table.Name)

	// Collect all columns referenced from child table in projections
	childColsReferenced := make(map[string]bool)

	for _, p := range stmt.Projections {
		cols := ExtractColumns(p.Expr)
		for _, c := range cols {
			if strings.EqualFold(c.Table, childAlias) {
				childColsReferenced[strings.ToLower(c.Column)] = true
			}
		}
	}

	// Create CTE:
	// WITH <child>_agg AS (SELECT <childCol>, SUM(...) AS ..., COUNT(*) AS child_cnt FROM <child> GROUP BY <childCol>)
	cteSelect := &SelectStmt{
		Projections: []*Projection{
			{
				Expr:  &ColumnRef{Column: childCol},
				Alias: childCol,
			},
		},
		From: &FromClause{
			Primary: &TableRef{Name: targetJoin.Table.Name},
		},
		GroupBy: []Expr{
			&ColumnRef{Column: childCol},
		},
	}

	// For any measure on child table, aggregate it inside CTE
	for colName := range childColsReferenced {
		if strings.EqualFold(colName, childCol) {
			continue
		}
		aggAlias := fmt.Sprintf("total_%s", colName)
		cteSelect.Projections = append(cteSelect.Projections, &Projection{
			Expr: &FunctionCall{
				Name: "SUM",
				Args: []Expr{&ColumnRef{Column: colName}},
			},
			Alias: aggAlias,
		})
	}

	// Prepend CTE to statement
	if stmt.With == nil {
		stmt.With = &WithClause{}
	}
	stmt.With.CTEs = append(stmt.With.CTEs, &CTE{
		Name:  cteName,
		Query: cteSelect,
	})

	// Replace join table with CTE table
	oldJoinTarget := targetJoin.Table.ToSQL()
	targetJoin.Table = &TableRef{
		Name:  cteName,
		Alias: childAlias,
	}

	r.mutations = append(r.mutations, MutationRecord{
		ProbeName:   "JoinFanoutGuard",
		Target:      childTbl,
		Description: fmt.Sprintf("Pre-aggregated child relation %q into CTE %q on key %q to prevent 1:N Cartesian multiplication", childTbl, cteName, childCol),
		BeforeExpr:  oldJoinTarget,
		AfterExpr:   targetJoin.Table.ToSQL(),
	})

	return true
}

// RewriteNullSafeDivision wraps division operators with NULLIF and COALESCE.
func (r *Rewriter) RewriteNullSafeDivision(stmt *SelectStmt) {
	for _, proj := range stmt.Projections {
		proj.Expr = r.wrapDivision(proj.Expr)
	}
}

func (r *Rewriter) wrapDivision(expr Expr) Expr {
	if expr == nil {
		return nil
	}

	if bin, ok := expr.(*BinaryExpr); ok {
		bin.Left = r.wrapDivision(bin.Left)
		bin.Right = r.wrapDivision(bin.Right)

		if bin.Op == "/" {
			// Wrap right side in NULLIF(right, 0)
			nullifCall := &FunctionCall{
				Name: "NULLIF",
				Args: []Expr{bin.Right, &Literal{Value: "0", IsStr: false}},
			}
			safeBin := &BinaryExpr{
				Left:  bin.Left,
				Op:    "/",
				Right: nullifCall,
			}
			// Wrap entire division in COALESCE(left / NULLIF(right, 0), 0)
			coalesceCall := &FunctionCall{
				Name: "COALESCE",
				Args: []Expr{safeBin, &Literal{Value: "0", IsStr: false}},
			}

			r.mutations = append(r.mutations, MutationRecord{
				ProbeName:   "NullabilityGuard",
				Target:      bin.ToSQL(),
				Description: "Wrapped arithmetic division with NULLIF(denominator, 0) and COALESCE to prevent runtime div-by-zero crashes",
				BeforeExpr:  bin.ToSQL(),
				AfterExpr:   coalesceCall.ToSQL(),
			})
			return coalesceCall
		}
		return bin
	}

	if paren, ok := expr.(*ParenExpr); ok {
		paren.Inner = r.wrapDivision(paren.Inner)
		return paren
	}

	if fn, ok := expr.(*FunctionCall); ok {
		for i, arg := range fn.Args {
			fn.Args[i] = r.wrapDivision(arg)
		}
		return fn
	}

	return expr
}

// RewriteNullSafeAvg wraps AVG(col) into AVG(COALESCE(col, 0)) for nullable columns.
func (r *Rewriter) RewriteNullSafeAvg(stmt *SelectStmt, nullableCols map[string]bool) {
	for _, proj := range stmt.Projections {
		proj.Expr = r.wrapAvg(proj.Expr, nullableCols)
	}
}

func (r *Rewriter) wrapAvg(expr Expr, nullableCols map[string]bool) Expr {
	if expr == nil {
		return nil
	}

	if fn, ok := expr.(*FunctionCall); ok && strings.EqualFold(fn.Name, "AVG") {
		if len(fn.Args) == 1 {
			if col, isCol := fn.Args[0].(*ColumnRef); isCol {
				key := strings.ToLower(col.Column)
				if nullableCols[key] {
					oldExpr := fn.ToSQL()
					fn.Args[0] = &FunctionCall{
						Name: "COALESCE",
						Args: []Expr{col, &Literal{Value: "0", IsStr: false}},
					}
					r.mutations = append(r.mutations, MutationRecord{
						ProbeName:   "NullabilityGuard",
						Target:      col.ToSQL(),
						Description: fmt.Sprintf("Wrapped nullable column %q inside AVG() with COALESCE to prevent silent upward mean distortion", col.ToSQL()),
						BeforeExpr:  oldExpr,
						AfterExpr:   fn.ToSQL(),
					})
				}
			}
		}
		return fn
	}

	if bin, ok := expr.(*BinaryExpr); ok {
		bin.Left = r.wrapAvg(bin.Left, nullableCols)
		bin.Right = r.wrapAvg(bin.Right, nullableCols)
		return bin
	}

	if paren, ok := expr.(*ParenExpr); ok {
		paren.Inner = r.wrapAvg(paren.Inner, nullableCols)
		return paren
	}

	return expr
}

// InjectGovernanceFilters injects missing soft-delete and tenant filters into WHERE clause.
func (r *Rewriter) InjectGovernanceFilters(stmt *SelectStmt, missingFilters []string) {
	for _, filter := range missingFilters {
		parsedFilter, err := ParseExpression(filter)
		if err != nil {
			continue
		}

		if stmt.Where == nil {
			stmt.Where = parsedFilter
		} else {
			stmt.Where = &BinaryExpr{
				Left:  stmt.Where,
				Op:    "AND",
				Right: parsedFilter,
			}
		}

		r.mutations = append(r.mutations, MutationRecord{
			ProbeName:   "GovernanceGuard",
			Target:      "WHERE",
			Description: fmt.Sprintf("Injected mandatory compliance filter: %s", filter),
			BeforeExpr:  "Omitted",
			AfterExpr:   filter,
		})
	}
}

// ParseExpression parses a standalone expression string.
func ParseExpression(exprStr string) (Expr, error) {
	fakeSQL := fmt.Sprintf("SELECT %s", exprStr)
	stmt, err := ParseSQL(fakeSQL)
	if err != nil {
		return nil, err
	}
	if len(stmt.Projections) > 0 {
		return stmt.Projections[0].Expr, nil
	}
	return nil, fmt.Errorf("no expression parsed")
}
