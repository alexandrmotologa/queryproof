package proof

import (
	"encoding/json"
	"os"

	"github.com/alexandrmotologa/queryproof/internal/ast"
	"github.com/alexandrmotologa/queryproof/internal/config"
	"github.com/alexandrmotologa/queryproof/internal/probes"
)

// VerificationStatus represents the cryptographic passport outcome.
type VerificationStatus string

const (
	StatusCertified           VerificationStatus = "CERTIFIED"
	StatusFalsifiedRewritten VerificationStatus = "FALSIFIED_AND_REWRITTEN"
	StatusRejected            VerificationStatus = "REJECTED"
)

// QueryMetadata holds query fingerprint and properties.
type QueryMetadata struct {
	RawSQL     string `json:"raw_sql"`
	SHA256     string `json:"sha256"`
	NodeCount  int    `json:"node_count,omitempty"`
	TableCount int    `json:"table_count,omitempty"`
}

// RewrittenResult holds mutated SQL and recorded AST transformations.
type RewrittenResult struct {
	RawSQL           string               `json:"raw_sql"`
	SHA256           string               `json:"sha256"`
	MutationsApplied []ast.MutationRecord `json:"mutations_applied"`
}

// SignatureBlock stores the cryptographic seal of integrity.
type SignatureBlock struct {
	Algorithm string `json:"algorithm"`
	KeyID     string `json:"key_id"`
	Digest    string `json:"digest"`
	Signature string `json:"signature"`
	SignedAt  string `json:"signed_at"`
}

// ProofVoucher represents the complete signed proof.json manifest.
type ProofVoucher struct {
	Version            string              `json:"version"`
	VoucherID          string              `json:"voucher_id"`
	IssuedAt           string              `json:"issued_at"`
	Engine             string              `json:"engine"`
	Dialect            config.Dialect      `json:"dialect"`
	CandidateQuery     QueryMetadata       `json:"candidate_query"`
	Status             VerificationStatus  `json:"status"`
	IntegrityScore     float64             `json:"integrity_score"`
	ViolationsDetected []probes.Violation  `json:"violations_detected"`
	RewrittenQuery     *RewrittenResult    `json:"rewritten_query,omitempty"`
	Signature          *SignatureBlock     `json:"signature"`
}

// ToJSON converts the voucher into pretty formatted JSON.
func (p *ProofVoucher) ToJSON() ([]byte, error) {
	return json.MarshalIndent(p, "", "  ")
}

// SaveToFile writes proof.json to disk.
func (p *ProofVoucher) SaveToFile(path string) error {
	data, err := p.ToJSON()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LoadFromFile loads proof.json from disk.
func LoadFromFile(path string) (*ProofVoucher, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var v ProofVoucher
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return &v, nil
}
