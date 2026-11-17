package ast

import (
	"fmt"
	"strings"
	"unicode"
)

// TokenType represents lexical token types.
type TokenType int

const (
	TokenEOF TokenType = iota
	TokenKeyword
	TokenIdentifier
	TokenString
	TokenNumber
	TokenOperator
	TokenPunctuation
)

// Token represents a single lexical token.
type Token struct {
	Type  TokenType
	Value string
	Pos   int
}

// Lexer breaks raw SQL string into tokens.
type Lexer struct {
	input []rune
	pos   int
}

// NewLexer creates a new lexer.
func NewLexer(sql string) *Lexer {
	return &Lexer{
		input: []rune(sql),
		pos:   0,
	}
}

var keywords = map[string]bool{
	"SELECT": true, "FROM": true, "WHERE": true, "GROUP": true, "BY": true,
	"HAVING": true, "ORDER": true, "ASC": true, "DESC": true, "LIMIT": true,
	"OFFSET": true, "WITH": true, "RECURSIVE": true, "AS": true, "ON": true,
	"JOIN": true, "INNER": true, "LEFT": true, "RIGHT": true, "FULL": true,
	"CROSS": true, "OUTER": true, "USING": true, "AND": true, "OR": true,
	"NOT": true, "IN": true, "IS": true, "NULL": true, "CASE": true,
	"WHEN": true, "THEN": true, "ELSE": true, "END": true, "CAST": true,
	"DISTINCT": true, "OVER": true, "PARTITION": true, "BETWEEN": true,
	"LIKE": true, "ILIKE": true,
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) {
		r := l.input[l.pos]
		if unicode.IsSpace(r) {
			l.pos++
			continue
		}
		// Single-line comment --
		if r == '-' && l.pos+1 < len(l.input) && l.input[l.pos+1] == '-' {
			l.pos += 2
			for l.pos < len(l.input) && l.input[l.pos] != '\n' {
				l.pos++
			}
			continue
		}
		// Multi-line comment /* ... */
		if r == '/' && l.pos+1 < len(l.input) && l.input[l.pos+1] == '*' {
			l.pos += 2
			for l.pos+1 < len(l.input) && !(l.input[l.pos] == '*' && l.input[l.pos+1] == '/') {
				l.pos++
			}
			if l.pos+1 < len(l.input) {
				l.pos += 2
			}
			continue
		}
		break
	}
}

// NextToken returns the next token from input.
func (l *Lexer) NextToken() (Token, error) {
	l.skipWhitespace()
	if l.pos >= len(l.input) {
		return Token{Type: TokenEOF, Value: "", Pos: l.pos}, nil
	}

	start := l.pos
	r := l.input[l.pos]

	// Punctuation
	if strings.ContainsRune("(),;.", r) {
		l.pos++
		return Token{Type: TokenPunctuation, Value: string(r), Pos: start}, nil
	}

	// Double colon :: cast operator
	if r == ':' && l.pos+1 < len(l.input) && l.input[l.pos+1] == ':' {
		l.pos += 2
		return Token{Type: TokenOperator, Value: "::", Pos: start}, nil
	}

	// Operators
	if strings.ContainsRune("=+-*/<>!|", r) {
		l.pos++
		if l.pos < len(l.input) {
			next := l.input[l.pos]
			combined := string([]rune{r, next})
			if combined == "<=" || combined == ">=" || combined == "!=" || combined == "<>" || combined == "||" {
				l.pos++
				return Token{Type: TokenOperator, Value: combined, Pos: start}, nil
			}
		}
		return Token{Type: TokenOperator, Value: string(r), Pos: start}, nil
	}

	// String literal '...'
	if r == '\'' {
		l.pos++
		var str strings.Builder
		for l.pos < len(l.input) {
			cur := l.input[l.pos]
			if cur == '\'' {
				if l.pos+1 < len(l.input) && l.input[l.pos+1] == '\'' {
					str.WriteRune('\'')
					l.pos += 2
					continue
				}
				l.pos++
				return Token{Type: TokenString, Value: str.String(), Pos: start}, nil
			}
			str.WriteRune(cur)
			l.pos++
		}
		return Token{}, fmt.Errorf("unterminated string literal starting at position %d", start)
	}

	// Quoted identifier "..." or `...`
	if r == '"' || r == '`' {
		quote := r
		l.pos++
		var id strings.Builder
		for l.pos < len(l.input) && l.input[l.pos] != quote {
			id.WriteRune(l.input[l.pos])
			l.pos++
		}
		if l.pos < len(l.input) && l.input[l.pos] == quote {
			l.pos++
		}
		return Token{Type: TokenIdentifier, Value: id.String(), Pos: start}, nil
	}

	// Numbers
	if unicode.IsDigit(r) || (r == '.' && l.pos+1 < len(l.input) && unicode.IsDigit(l.input[l.pos+1])) {
		var num strings.Builder
		hasDot := false
		for l.pos < len(l.input) {
			cur := l.input[l.pos]
			if unicode.IsDigit(cur) {
				num.WriteRune(cur)
				l.pos++
			} else if cur == '.' && !hasDot {
				hasDot = true
				num.WriteRune(cur)
				l.pos++
			} else {
				break
			}
		}
		return Token{Type: TokenNumber, Value: num.String(), Pos: start}, nil
	}

	// Identifiers or Keywords
	if unicode.IsLetter(r) || r == '_' {
		var id strings.Builder
		for l.pos < len(l.input) {
			cur := l.input[l.pos]
			if unicode.IsLetter(cur) || unicode.IsDigit(cur) || cur == '_' {
				id.WriteRune(cur)
				l.pos++
			} else {
				break
			}
		}
		val := id.String()
		upper := strings.ToUpper(val)
		if keywords[upper] {
			return Token{Type: TokenKeyword, Value: upper, Pos: start}, nil
		}
		return Token{Type: TokenIdentifier, Value: val, Pos: start}, nil
	}

	l.pos++
	return Token{Type: TokenPunctuation, Value: string(r), Pos: start}, nil
}

// TokenizeAll reads all tokens into a slice.
func Tokenize(sql string) ([]Token, error) {
	lex := NewLexer(sql)
	var tokens []Token
	for {
		tok, err := lex.NextToken()
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, tok)
		if tok.Type == TokenEOF {
			break
		}
	}
	return tokens, nil
}
