package ledger

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/alexandrmotologa/queryproof/internal/engine"
	"github.com/alexandrmotologa/queryproof/internal/probes"
	_ "modernc.org/sqlite"
)

// LedgerEntry represents a persisted verification event in the ledger.
type LedgerEntry struct {
	ID              int64              `json:"id"`
	VoucherID       string             `json:"voucher_id"`
	Timestamp       string             `json:"timestamp"`
	CandidateSQL    string             `json:"candidate_sql"`
	RewrittenSQL    string             `json:"rewritten_sql,omitempty"`
	CandidateHash   string             `json:"candidate_hash"`
	RewrittenHash   string             `json:"rewritten_hash,omitempty"`
	Status          string             `json:"status"`
	IntegrityScore  float64            `json:"integrity_score"`
	ViolationsCount int                `json:"violations_count"`
	Violations      []probes.Violation `json:"violations"`
	LatencyMs       float64            `json:"latency_ms"`
}

// LedgerStats aggregates historical verification and hallucination trends.
type LedgerStats struct {
	TotalQueries     int64              `json:"total_queries"`
	CertifiedCount   int64              `json:"certified_count"`
	FalsifiedCount   int64              `json:"falsified_count"`
	FalsifyRatePct   float64            `json:"falsify_rate_pct"`
	AvgIntegrity     float64            `json:"avg_integrity_score"`
	AvgLatencyMs     float64            `json:"avg_latency_ms"`
	ProbeBreakdown   map[string]int     `json:"probe_breakdown"`
	RecentViolations []probes.Violation `json:"recent_violations"`
}

// AuditLedger manages historical verification logs using embedded SQLite.
type AuditLedger struct {
	db *sql.DB
	mu sync.RWMutex
}

// OpenLedger opens or creates the audit ledger at the given path.
func OpenLedger(dbPath string) (*AuditLedger, error) {
	if dbPath != ":memory:" {
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create ledger dir: %w", err)
		}
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open ledger db: %w", err)
	}

	createTableSQL := `
		CREATE TABLE IF NOT EXISTS audit_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			voucher_id TEXT NOT NULL,
			timestamp DATETIME NOT NULL,
			candidate_sql TEXT NOT NULL,
			rewritten_sql TEXT,
			candidate_hash TEXT NOT NULL,
			rewritten_hash TEXT,
			status TEXT NOT NULL,
			integrity_score REAL NOT NULL,
			violations_count INTEGER NOT NULL,
			violations_json TEXT,
			latency_ms REAL NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_events(timestamp DESC);
		CREATE INDEX IF NOT EXISTS idx_audit_status ON audit_events(status);
	`
	if _, err := db.Exec(createTableSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("init ledger schema: %w", err)
	}

	return &AuditLedger{db: db}, nil
}

// RecordEvent records a verification report into the ledger.
func (l *AuditLedger) RecordEvent(rep *engine.VerificationReport) error {
	if l == nil || l.db == nil || rep == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	voucherID := ""
	status := "CERTIFIED"
	if rep.Voucher != nil {
		voucherID = rep.Voucher.VoucherID
		status = string(rep.Voucher.Status)
	} else if !rep.IsClean {
		status = "FALSIFIED_AND_REWRITTEN"
	}

	var allViolations []probes.Violation
	if rep.SuiteResult != nil {
		allViolations = rep.SuiteResult.AllViolations
	}

	violJSON, _ := json.Marshal(allViolations)
	now := time.Now().UTC().Format(time.RFC3339)

	score := 1.0
	if rep.SuiteResult != nil {
		score = rep.SuiteResult.OverallScore
	}

	query := `
		INSERT INTO audit_events (
			voucher_id, timestamp, candidate_sql, rewritten_sql, candidate_hash,
			rewritten_hash, status, integrity_score, violations_count, violations_json, latency_ms
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`
	_, err := l.db.Exec(query,
		voucherID,
		now,
		rep.OriginalSQL,
		rep.RewrittenSQL,
		rep.OriginalHash,
		rep.RewrittenHash,
		status,
		score,
		len(allViolations),
		string(violJSON),
		rep.VerificationMs,
	)
	return err
}

// ListEvents returns paginated audit events.
func (l *AuditLedger) ListEvents(limit, offset int) ([]LedgerEntry, int64, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var total int64
	_ = l.db.QueryRow("SELECT COUNT(*) FROM audit_events;").Scan(&total)

	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT id, voucher_id, timestamp, candidate_sql, rewritten_sql, candidate_hash,
		       rewritten_hash, status, integrity_score, violations_count, violations_json, latency_ms
		FROM audit_events
		ORDER BY id DESC
		LIMIT ? OFFSET ?;
	`
	rows, err := l.db.Query(query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var entries []LedgerEntry
	for rows.Next() {
		var e LedgerEntry
		var rewSQL, rewHash sql.NullString
		var violStr string

		err := rows.Scan(
			&e.ID,
			&e.VoucherID,
			&e.Timestamp,
			&e.CandidateSQL,
			&rewSQL,
			&e.CandidateHash,
			&rewHash,
			&e.Status,
			&e.IntegrityScore,
			&e.ViolationsCount,
			&violStr,
			&e.LatencyMs,
		)
		if err == nil {
			if rewSQL.Valid {
				e.RewrittenSQL = rewSQL.String
			}
			if rewHash.Valid {
				e.RewrittenHash = rewHash.String
			}
			if violStr != "" {
				_ = json.Unmarshal([]byte(violStr), &e.Violations)
			}
			entries = append(entries, e)
		}
	}

	return entries, total, nil
}

// GetStats computes aggregate metrics across all audit events.
func (l *AuditLedger) GetStats() (*LedgerStats, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	stats := &LedgerStats{
		ProbeBreakdown: make(map[string]int),
	}

	row := l.db.QueryRow(`
		SELECT COUNT(*),
		       COALESCE(SUM(CASE WHEN status = 'CERTIFIED' THEN 1 ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN status != 'CERTIFIED' THEN 1 ELSE 0 END), 0),
		       COALESCE(AVG(integrity_score), 1.0),
		       COALESCE(AVG(latency_ms), 0.0)
		FROM audit_events;
	`)
	if err := row.Scan(&stats.TotalQueries, &stats.CertifiedCount, &stats.FalsifiedCount, &stats.AvgIntegrity, &stats.AvgLatencyMs); err != nil {
		return stats, err
	}

	if stats.TotalQueries > 0 {
		stats.FalsifyRatePct = (float64(stats.FalsifiedCount) / float64(stats.TotalQueries)) * 100.0
	}

	// Breakdown of violated probes
	rows, err := l.db.Query("SELECT violations_json FROM audit_events WHERE violations_count > 0 ORDER BY id DESC LIMIT 100;")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var vStr string
			if err := rows.Scan(&vStr); err == nil && vStr != "" {
				var viols []probes.Violation
				if err := json.Unmarshal([]byte(vStr), &viols); err == nil {
					for _, v := range viols {
						stats.ProbeBreakdown[v.ProbeName]++
						if len(stats.RecentViolations) < 5 {
							stats.RecentViolations = append(stats.RecentViolations, v)
						}
					}
				}
			}
		}
	}

	return stats, nil
}

// Close closes the database connection.
func (l *AuditLedger) Close() error {
	if l != nil && l.db != nil {
		return l.db.Close()
	}
	return nil
}
