package proof

import (
	"testing"
	"time"

	"github.com/alexandrmotologa/queryproof/internal/config"
)

func TestSignAndVerifyVoucher(t *testing.T) {
	secretKey := "super-secure-enterprise-key-99"

	voucher := &ProofVoucher{
		Version:   "queryproof/v1",
		VoucherID: "vch_test_12345",
		IssuedAt:  time.Now().UTC().Format(time.RFC3339),
		Engine:    "queryproof-v1.0.0",
		Dialect:   config.DialectPostgres,
		CandidateQuery: QueryMetadata{
			RawSQL: "SELECT * FROM orders",
			SHA256: "abc1234567890",
		},
		Status:         StatusCertified,
		IntegrityScore: 1.0,
	}

	if err := SignVoucher(voucher, secretKey); err != nil {
		t.Fatalf("sign error: %v", err)
	}

	if voucher.Signature == nil || voucher.Signature.Signature == "" {
		t.Fatalf("expected signature in voucher")
	}

	// Verify valid voucher
	valid, err := VerifyVoucher(voucher, secretKey)
	if err != nil || !valid {
		t.Fatalf("expected valid voucher verification, got valid=%v, err=%v", valid, err)
	}

	// Tamper with voucher
	voucher.IntegrityScore = 0.5
	tampered, err := VerifyVoucher(voucher, secretKey)
	if tampered || err == nil {
		t.Fatalf("expected tampered voucher to fail verification")
	}
}
