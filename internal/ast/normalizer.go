package ast

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/alexandrmotologa/queryproof/internal/config"
)

// NormalizeQuery takes raw SQL or an AST and outputs a canonical formatted SQL string.
func NormalizeQuery(sql string, dialect config.Dialect) (string, error) {
	stmt, err := ParseSQL(sql)
	if err != nil {
		return "", fmt.Errorf("normalize parse: %w", err)
	}
	return FormatDialect(stmt, dialect), nil
}

// ComputeQueryHash calculates SHA-256 fingerprint of the normalized SQL query.
func ComputeQueryHash(canonicalSQL string) string {
	hasher := sha256.New()
	hasher.Write([]byte(strings.TrimSpace(canonicalSQL)))
	return hex.EncodeToString(hasher.Sum(nil))
}

// FormatDialect formats an AST statement tailored to the given SQL dialect.
func FormatDialect(stmt *SelectStmt, dialect config.Dialect) string {
	if stmt == nil {
		return ""
	}
	raw := stmt.ToSQL()
	switch dialect {
	case config.DialectBigQuery:
		// Replace standard double quotes with backticks if present
		return strings.ReplaceAll(raw, `"`, "`")
	case config.DialectSnowflake:
		// Snowflake standard
		return raw
	case config.DialectPostgres:
		return raw
	default:
		return raw
	}
}
