package ast

import (
	"fmt"
	"strconv"
	"strings"
)

// Parser parses a SQL token stream into an AST.
type Parser struct {
	tokens []Token
	pos    int
}

// NewParser creates a new parser from a token slice.
func NewParser(tokens []Token) *Parser {
	return &Parser{tokens: tokens, pos: 0}
}

// ParseSQL parses raw SQL text into a SelectStmt.
func ParseSQL(sql string) (*SelectStmt, error) {
	tokens, err := Tokenize(sql)
	if err != nil {
		return nil, fmt.Errorf("lex error: %w", err)
	}
	p := NewParser(tokens)
	return p.ParseSelect()
}

func (p *Parser) current() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) peek(offset int) Token {
	idx := p.pos + offset
	if idx >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[idx]
}

func (p *Parser) advance() Token {
	tok := p.current()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return tok
}

func (p *Parser) matchKeyword(kw string) bool {
	tok := p.current()
	if tok.Type == TokenKeyword && strings.EqualFold(tok.Value, kw) {
		p.advance()
		return true
	}
	return false
}

func (p *Parser) matchPunct(punct string) bool {
	tok := p.current()
	if tok.Type == TokenPunctuation && tok.Value == punct {
		p.advance()
		return true
	}
	return false
}

func (p *Parser) expectKeyword(kw string) error {
	tok := p.current()
	if tok.Type != TokenKeyword || !strings.EqualFold(tok.Value, kw) {
		return fmt.Errorf("expected keyword %s, got %s (type %v) at pos %d", kw, tok.Value, tok.Type, tok.Pos)
	}
	p.advance()
	return nil
}

func (p *Parser) expectPunct(punct string) error {
	tok := p.current()
	if tok.Type != TokenPunctuation || tok.Value != punct {
		return fmt.Errorf("expected punctuation %q, got %q at pos %d", punct, tok.Value, tok.Pos)
	}
	p.advance()
	return nil
}

// ParseSelect parses a SELECT statement with optional WITH clause.
func (p *Parser) ParseSelect() (*SelectStmt, error) {
	stmt := &SelectStmt{}

	// Optional WITH clause
	if p.matchKeyword("WITH") {
		with := &WithClause{}
		if p.matchKeyword("RECURSIVE") {
			with.Recursive = true
		}
		for {
			cte, err := p.parseCTE()
			if err != nil {
				return nil, err
			}
			with.CTEs = append(with.CTEs, cte)
			if !p.matchPunct(",") {
				break
			}
		}
		stmt.With = with
	}

	if err := p.expectKeyword("SELECT"); err != nil {
		return nil, err
	}

	if p.matchKeyword("DISTINCT") {
		stmt.Distinct = true
	}

	// Projections
	for {
		proj, err := p.parseProjection()
		if err != nil {
			return nil, err
		}
		stmt.Projections = append(stmt.Projections, proj)
		if !p.matchPunct(",") {
			break
		}
	}

	// FROM clause
	if p.matchKeyword("FROM") {
		from, err := p.parseFrom()
		if err != nil {
			return nil, err
		}
		stmt.From = from
	}

	// WHERE clause
	if p.matchKeyword("WHERE") {
		whereExpr, err := p.parseExpression(0)
		if err != nil {
			return nil, fmt.Errorf("parse WHERE clause: %w", err)
		}
		stmt.Where = whereExpr
	}

	// GROUP BY clause
	if p.matchKeyword("GROUP") {
		if err := p.expectKeyword("BY"); err != nil {
			return nil, err
		}
		for {
			grpExpr, err := p.parseExpression(0)
			if err != nil {
				return nil, fmt.Errorf("parse GROUP BY: %w", err)
			}
			stmt.GroupBy = append(stmt.GroupBy, grpExpr)
			if !p.matchPunct(",") {
				break
			}
		}
	}

	// HAVING clause
	if p.matchKeyword("HAVING") {
		havingExpr, err := p.parseExpression(0)
		if err != nil {
			return nil, fmt.Errorf("parse HAVING: %w", err)
		}
		stmt.Having = havingExpr
	}

	// ORDER BY clause
	if p.matchKeyword("ORDER") {
		if err := p.expectKeyword("BY"); err != nil {
			return nil, err
		}
		for {
			item, err := p.parseOrderByItem()
			if err != nil {
				return nil, err
			}
			stmt.OrderBy = append(stmt.OrderBy, item)
			if !p.matchPunct(",") {
				break
			}
		}
	}

	// LIMIT
	if p.matchKeyword("LIMIT") {
		lim, err := p.parsePrimary()
		if err != nil {
			return nil, fmt.Errorf("parse LIMIT: %w", err)
		}
		stmt.Limit = lim
	}

	// OFFSET
	if p.matchKeyword("OFFSET") {
		off, err := p.parsePrimary()
		if err != nil {
			return nil, fmt.Errorf("parse OFFSET: %w", err)
		}
		stmt.Offset = off
	}

	return stmt, nil
}

func (p *Parser) parseCTE() (*CTE, error) {
	tok := p.advance()
	if tok.Type != TokenIdentifier && tok.Type != TokenKeyword {
		return nil, fmt.Errorf("expected CTE name, got %s", tok.Value)
	}
	cte := &CTE{Name: tok.Value}

	// Optional column list: cte(col1, col2)
	if p.matchPunct("(") {
		for {
			cTok := p.advance()
			cte.Columns = append(cte.Columns, cTok.Value)
			if p.matchPunct(")") {
				break
			}
			if !p.matchPunct(",") {
				return nil, fmt.Errorf("expected ',' or ')' in CTE column list")
			}
		}
	}

	if err := p.expectKeyword("AS"); err != nil {
		return nil, err
	}
	if err := p.expectPunct("("); err != nil {
		return nil, err
	}

	subquery, err := p.ParseSelect()
	if err != nil {
		return nil, fmt.Errorf("parse CTE subquery %s: %w", cte.Name, err)
	}
	cte.Query = subquery

	if err := p.expectPunct(")"); err != nil {
		return nil, err
	}
	return cte, nil
}

func (p *Parser) parseProjection() (*Projection, error) {
	expr, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}

	proj := &Projection{Expr: expr}

	// Check for AS alias or bare identifier alias
	if p.matchKeyword("AS") {
		aliasTok := p.advance()
		proj.Alias = aliasTok.Value
	} else {
		next := p.current()
		// If next token is an identifier and NOT a keyword starting next clause
		if next.Type == TokenIdentifier && !isClauseKeyword(next.Value) {
			proj.Alias = next.Value
			p.advance()
		}
	}

	return proj, nil
}

func isClauseKeyword(val string) bool {
	upper := strings.ToUpper(val)
	switch upper {
	case "FROM", "WHERE", "GROUP", "HAVING", "ORDER", "LIMIT", "OFFSET", "JOIN", "LEFT", "RIGHT", "INNER", "FULL", "CROSS", "UNION", "EXCEPT", "INTERSECT":
		return true
	default:
		return false
	}
}

func (p *Parser) parseFrom() (*FromClause, error) {
	primary, err := p.parseTableRef()
	if err != nil {
		return nil, err
	}

	from := &FromClause{Primary: primary}

	for {
		joinType, hasJoin := p.matchJoinType()
		if !hasJoin {
			break
		}

		tbl, err := p.parseTableRef()
		if err != nil {
			return nil, fmt.Errorf("parse joined table: %w", err)
		}

		join := &JoinClause{Type: joinType, Table: tbl}

		if p.matchKeyword("ON") {
			cond, err := p.parseExpression(0)
			if err != nil {
				return nil, fmt.Errorf("parse join ON condition: %w", err)
			}
			join.Condition = cond
		} else if p.matchKeyword("USING") {
			if err := p.expectPunct("("); err != nil {
				return nil, err
			}
			for {
				uTok := p.advance()
				join.Using = append(join.Using, uTok.Value)
				if p.matchPunct(")") {
					break
				}
				p.matchPunct(",")
			}
		}

		from.Joins = append(from.Joins, join)
	}

	return from, nil
}

func (p *Parser) matchJoinType() (JoinType, bool) {
	tok := p.current()
	if tok.Type != TokenKeyword {
		return "", false
	}

	switch strings.ToUpper(tok.Value) {
	case "JOIN":
		p.advance()
		return JoinInner, true
	case "INNER":
		p.advance()
		p.matchKeyword("JOIN")
		return JoinInner, true
	case "LEFT":
		p.advance()
		p.matchKeyword("OUTER")
		p.matchKeyword("JOIN")
		return JoinLeft, true
	case "RIGHT":
		p.advance()
		p.matchKeyword("OUTER")
		p.matchKeyword("JOIN")
		return JoinRight, true
	case "FULL":
		p.advance()
		p.matchKeyword("OUTER")
		p.matchKeyword("JOIN")
		return JoinFull, true
	case "CROSS":
		p.advance()
		p.matchKeyword("JOIN")
		return JoinCross, true
	}
	return "", false
}

func (p *Parser) parseTableRef() (*TableRef, error) {
	// Check for subquery table: (SELECT ...) [AS] alias
	if p.matchPunct("(") {
		subquery, err := p.ParseSelect()
		if err != nil {
			return nil, err
		}
		if err := p.expectPunct(")"); err != nil {
			return nil, err
		}
		ref := &TableRef{Subquery: subquery}
		p.matchKeyword("AS")
		if p.current().Type == TokenIdentifier {
			ref.Alias = p.advance().Value
		}
		return ref, nil
	}

	tok := p.advance()
	if tok.Type != TokenIdentifier && tok.Type != TokenKeyword {
		return nil, fmt.Errorf("expected table name, got %s at pos %d", tok.Value, tok.Pos)
	}

	ref := &TableRef{Name: tok.Value}

	// Schema qualified table: schema.table
	if p.matchPunct(".") {
		tblTok := p.advance()
		ref.Schema = ref.Name
		ref.Name = tblTok.Value
	}

	// Optional alias
	if p.matchKeyword("AS") {
		ref.Alias = p.advance().Value
	} else {
		next := p.current()
		if next.Type == TokenIdentifier && !isClauseKeyword(next.Value) {
			ref.Alias = next.Value
			p.advance()
		}
	}

	return ref, nil
}

func (p *Parser) parseOrderByItem() (*OrderByItem, error) {
	expr, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	item := &OrderByItem{Expr: expr}
	if p.matchKeyword("DESC") {
		item.Desc = true
	} else {
		p.matchKeyword("ASC")
	}
	return item, nil
}

// Expression parsing with precedence climbing
func (p *Parser) parseExpression(minPrec int) (Expr, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}

	for {
		op, prec := p.getBinaryOpPrec()
		if prec < minPrec {
			break
		}

		p.advance() // consume operator

		// Special handling for IS NULL / IS NOT NULL
		if op == "IS" {
			isNot := false
			if p.matchKeyword("NOT") {
				isNot = true
			}
			if err := p.expectKeyword("NULL"); err != nil {
				return nil, err
			}
			left = &IsNullExpr{Expr: left, Isnot: isNot}
			continue
		}

		// Special handling for IN
		if op == "IN" || op == "NOT IN" {
			inExpr := &InExpr{Expr: left, Not: (op == "NOT IN")}
			if err := p.expectPunct("("); err != nil {
				return nil, fmt.Errorf("expected '(' after IN")
			}
			if p.current().Type == TokenKeyword && strings.EqualFold(p.current().Value, "SELECT") {
				subquery, err := p.ParseSelect()
				if err != nil {
					return nil, err
				}
				inExpr.Subquery = subquery
			} else {
				for {
					item, err := p.parseExpression(0)
					if err != nil {
						return nil, err
					}
					inExpr.List = append(inExpr.List, item)
					if !p.matchPunct(",") {
						break
					}
				}
			}
			if err := p.expectPunct(")"); err != nil {
				return nil, err
			}
			left = inExpr
			continue
		}

		// Double colon :: cast operator
		if op == "::" {
			typeTok := p.advance()
			left = &CastExpr{Expr: left, Type: typeTok.Value}
			continue
		}

		right, err := p.parseExpression(prec + 1)
		if err != nil {
			return nil, err
		}

		left = &BinaryExpr{
			Left:  left,
			Op:    op,
			Right: right,
		}
	}

	return left, nil
}

func (p *Parser) getBinaryOpPrec() (string, int) {
	tok := p.current()

	if tok.Type == TokenOperator {
		switch tok.Value {
		case "::":
			return "::", 8
		case "*", "/":
			return tok.Value, 6
		case "+", "-":
			return tok.Value, 5
		case "=", "!=", "<>", "<", "<=", ">", ">=":
			return tok.Value, 4
		case "||":
			return "||", 5
		}
	}

	if tok.Type == TokenKeyword {
		upper := strings.ToUpper(tok.Value)
		switch upper {
		case "LIKE", "ILIKE":
			return upper, 4
		case "IS":
			return "IS", 4
		case "IN":
			return "IN", 4
		case "AND":
			return "AND", 2
		case "OR":
			return "OR", 1
		case "NOT":
			if p.peek(1).Type == TokenKeyword && strings.EqualFold(p.peek(1).Value, "IN") {
				p.advance() // consume NOT
				return "NOT IN", 4
			}
		}
	}

	return "", -1
}

func (p *Parser) parseUnary() (Expr, error) {
	tok := p.current()
	if tok.Type == TokenKeyword && strings.EqualFold(tok.Value, "NOT") {
		p.advance()
		inner, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{Op: "NOT", Expr: inner}, nil
	}
	if tok.Type == TokenOperator && (tok.Value == "-" || tok.Value == "+") {
		p.advance()
		inner, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{Op: tok.Value, Expr: inner}, nil
	}
	return p.parsePrimary()
}

func (p *Parser) parsePrimary() (Expr, error) {
	tok := p.current()

	// Parentheses (expr) or (SELECT ...)
	if p.matchPunct("(") {
		if p.current().Type == TokenKeyword && strings.EqualFold(p.current().Value, "SELECT") {
			subquery, err := p.ParseSelect()
			if err != nil {
				return nil, err
			}
			if err := p.expectPunct(")"); err != nil {
				return nil, err
			}
			return &SubqueryExpr{Query: subquery}, nil
		}
		inner, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}
		if err := p.expectPunct(")"); err != nil {
			return nil, err
		}
		return &ParenExpr{Inner: inner}, nil
	}

	// String literal
	if tok.Type == TokenString {
		p.advance()
		return &Literal{Value: tok.Value, IsStr: true}, nil
	}

	// Number literal
	if tok.Type == TokenNumber {
		p.advance()
		return &Literal{Value: tok.Value, IsStr: false}, nil
	}

	// NULL, TRUE, FALSE
	if tok.Type == TokenKeyword {
		upper := strings.ToUpper(tok.Value)
		switch upper {
		case "NULL", "TRUE", "FALSE":
			p.advance()
			return &Literal{Value: upper, IsStr: false}, nil
		case "CASE":
			return p.parseCaseExpr()
		case "CAST":
			return p.parseCastFunction()
		}
	}

	// Function call or Column reference or Identifier
	if tok.Type == TokenIdentifier || tok.Type == TokenKeyword {
		name := tok.Value
		p.advance()

		// Dot notation for schema/table/column: a.b
		if p.matchPunct(".") {
			colTok := p.advance()
			return &ColumnRef{Table: name, Column: colTok.Value}, nil
		}

		// Function call: name(...)
		if p.matchPunct("(") {
			fn := &FunctionCall{Name: strings.ToUpper(name)}
			if p.matchKeyword("DISTINCT") {
				fn.Distinct = true
			}

			// Check for COUNT(*)
			if p.current().Type == TokenOperator && p.current().Value == "*" {
				p.advance()
				fn.Args = append(fn.Args, &Literal{Value: "*", IsStr: false})
			} else if !p.matchPunct(")") {
				for {
					arg, err := p.parseExpression(0)
					if err != nil {
						return nil, err
					}
					fn.Args = append(fn.Args, arg)
					if p.matchPunct(")") {
						break
					}
					if !p.matchPunct(",") {
						return nil, fmt.Errorf("expected ',' or ')' in argument list of %s", name)
					}
				}
			}

			// Window function OVER (...)
			if p.matchKeyword("OVER") {
				ws, err := p.parseWindowSpec()
				if err != nil {
					return nil, err
				}
				fn.Over = ws
			}

			return fn, nil
		}

		// Bare column or identifier
		return &ColumnRef{Column: name}, nil
	}

	// Star operator e.g. in SELECT *
	if tok.Type == TokenOperator && tok.Value == "*" {
		p.advance()
		return &Literal{Value: "*", IsStr: false}, nil
	}

	return nil, fmt.Errorf("unexpected token in expression: %s (type %v) at pos %d", tok.Value, tok.Type, tok.Pos)
}

func (p *Parser) parseCaseExpr() (*CaseExpr, error) {
	p.advance() // consume CASE
	caseExpr := &CaseExpr{}

	// Optional base expression: CASE col WHEN ...
	if p.current().Type != TokenKeyword || !strings.EqualFold(p.current().Value, "WHEN") {
		base, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}
		caseExpr.BaseExpr = base
	}

	for p.matchKeyword("WHEN") {
		cond, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}
		if err := p.expectKeyword("THEN"); err != nil {
			return nil, err
		}
		then, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}
		caseExpr.Whens = append(caseExpr.Whens, &WhenClause{When: cond, Then: then})
	}

	if p.matchKeyword("ELSE") {
		elseExpr, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}
		caseExpr.ElseExpr = elseExpr
	}

	if err := p.expectKeyword("END"); err != nil {
		return nil, err
	}
	return caseExpr, nil
}

func (p *Parser) parseCastFunction() (*CastExpr, error) {
	p.advance() // consume CAST
	if err := p.expectPunct("("); err != nil {
		return nil, err
	}
	expr, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	if err := p.expectKeyword("AS"); err != nil {
		return nil, err
	}
	typeTok := p.advance()
	castType := typeTok.Value
	if p.matchPunct("(") {
		numTok := p.advance()
		castType = fmt.Sprintf("%s(%s)", castType, numTok.Value)
		p.matchPunct(")")
	}
	if err := p.expectPunct(")"); err != nil {
		return nil, err
	}
	return &CastExpr{Expr: expr, Type: castType}, nil
}

func (p *Parser) parseWindowSpec() (*WindowSpec, error) {
	if err := p.expectPunct("("); err != nil {
		return nil, err
	}
	ws := &WindowSpec{}
	if p.matchKeyword("PARTITION") {
		if err := p.expectKeyword("BY"); err != nil {
			return nil, err
		}
		for {
			item, err := p.parseExpression(0)
			if err != nil {
				return nil, err
			}
			ws.PartitionBy = append(ws.PartitionBy, item)
			if !p.matchPunct(",") {
				break
			}
		}
	}
	if p.matchKeyword("ORDER") {
		if err := p.expectKeyword("BY"); err != nil {
			return nil, err
		}
		for {
			item, err := p.parseOrderByItem()
			if err != nil {
				return nil, err
			}
			ws.OrderBy = append(ws.OrderBy, item)
			if !p.matchPunct(",") {
				break
			}
		}
	}
	if err := p.expectPunct(")"); err != nil {
		return nil, err
	}
	return ws, nil
}

// FormatQuery pretty formats an AST SelectStmt.
func FormatQuery(stmt *SelectStmt) string {
	if stmt == nil {
		return ""
	}
	return stmt.ToSQL()
}

func (s *SelectStmt) String() string {
	return s.ToSQL()
}

var _ = strconv.Itoa

