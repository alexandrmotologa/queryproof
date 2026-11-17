package ledger

import (
	"testing"

	"github.com/alexandrmotologa/queryproof/internal/engine"
	"github.com/alexandrmotologa/queryproof/internal/probes"
	"github.com/alexandrmotologa/queryproof/internal/proof"
)

func TestAuditLedgerPersistence(t *testing.T) {
	led, err := OpenLedger(":memory:")
	if err != nil {
		t.Fatalf("failed to open memory ledger: %v", err)
	}
	defer led.Close()

	// 1. Record clean event
	cleanRep := &engine.VerificationReport{
		OriginalSQL:  "SELECT id FROM customers",
		OriginalHash: "hash123",
		IsClean:      true,
		SuiteResult: &probes.SuiteResult{
			Passed:       true,
			OverallScore: 1.0,
		},
		Voucher: &proof.ProofVoucher{
			VoucherID:      "vch_clean_1",
			Status:         proof.StatusCertified,
			IntegrityScore: 1.0,
		},
		VerificationMs: 0.45,
	}
	if err := led.RecordEvent(cleanRep); err != nil {
		t.Fatalf("record clean error: %v", err)
	}

	// 2. Record falsified event
	falsifiedRep := &engine.VerificationReport{
		OriginalSQL:   "SELECT c.id, SUM(c.mrr) FROM customers c JOIN orders o ON c.id=o.customer_id",
		OriginalHash:  "hash456",
		RewrittenSQL:  "WITH orders_agg AS (...) SELECT ...",
		RewrittenHash: "hash789",
		IsClean:       false,
		SuiteResult: &probes.SuiteResult{
			Passed:        false,
			OverallScore:  0.8,
			CriticalCount: 1,
			AllViolations: []probes.Violation{
				{
					ProbeName: "JoinFanoutGuard",
					Severity:  probes.SeverityCritical,
					Title:     "Cartesian Metric Inflation",
				},
			},
		},
		Voucher: &proof.ProofVoucher{
			VoucherID:      "vch_falsified_1",
			Status:         proof.StatusFalsifiedRewritten,
			IntegrityScore: 0.98,
		},
		VerificationMs: 0.85,
	}
	if err := led.RecordEvent(falsifiedRep); err != nil {
		t.Fatalf("record falsified error: %v", err)
	}

	// 3. List events
	events, total, err := led.ListEvents(10, 0)
	if err != nil {
		t.Fatalf("list events error: %v", err)
	}
	if total != 2 || len(events) != 2 {
		t.Errorf("expected 2 total events, got total=%d, len=%d", total, len(events))
	}

	// 4. Check stats
	stats, err := led.GetStats()
	if err != nil {
		t.Fatalf("stats error: %v", err)
	}
	if stats.TotalQueries != 2 {
		t.Errorf("expected 2 total queries, got %d", stats.TotalQueries)
	}
	if stats.CertifiedCount != 1 || stats.FalsifiedCount != 1 {
		t.Errorf("expected 1 certified and 1 falsified, got %d and %d", stats.CertifiedCount, stats.FalsifiedCount)
	}
	if stats.ProbeBreakdown["JoinFanoutGuard"] != 1 {
		t.Errorf("expected 1 JoinFanoutGuard violation in breakdown")
	}
}
