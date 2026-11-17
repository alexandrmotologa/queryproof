package proof

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// DefaultKeyID is the default key label for integrity signing.
const DefaultKeyID = "queryproof-root-authority"

// SignVoucher computes HMAC-SHA256 integrity signature across all core proof fields.
func SignVoucher(voucher *ProofVoucher, secretKey string) error {
	if voucher == nil {
		return fmt.Errorf("voucher cannot be nil")
	}

	digest := computeDigest(voucher)
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(digest))
	sigHex := hex.EncodeToString(h.Sum(nil))

	voucher.Signature = &SignatureBlock{
		Algorithm: "HMAC-SHA256",
		KeyID:     DefaultKeyID,
		Digest:    digest,
		Signature: sigHex,
		SignedAt:  time.Now().UTC().Format(time.RFC3339),
	}

	return nil
}

// VerifyVoucher checks whether voucher contents match the signature and digest.
func VerifyVoucher(voucher *ProofVoucher, secretKey string) (bool, error) {
	if voucher == nil || voucher.Signature == nil {
		return false, fmt.Errorf("voucher or signature missing")
	}

	expectedDigest := computeDigest(voucher)
	if voucher.Signature.Digest != expectedDigest {
		return false, fmt.Errorf("voucher digest mismatch: document has been modified after issuance")
	}

	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(expectedDigest))
	expectedSig := hex.EncodeToString(h.Sum(nil))

	if !hmac.Equal([]byte(voucher.Signature.Signature), []byte(expectedSig)) {
		return false, fmt.Errorf("cryptographic signature mismatch")
	}

	return true, nil
}

func computeDigest(v *ProofVoucher) string {
	hasher := sha256.New()
	hasher.Write([]byte(v.VoucherID))
	hasher.Write([]byte("|"))
	hasher.Write([]byte(v.IssuedAt))
	hasher.Write([]byte("|"))
	hasher.Write([]byte(v.CandidateQuery.SHA256))
	hasher.Write([]byte("|"))
	hasher.Write([]byte(v.Status))
	hasher.Write([]byte("|"))
	hasher.Write([]byte(fmt.Sprintf("%.4f", v.IntegrityScore)))

	if v.RewrittenQuery != nil {
		hasher.Write([]byte("|"))
		hasher.Write([]byte(v.RewrittenQuery.SHA256))
	}

	return hex.EncodeToString(hasher.Sum(nil))
}
