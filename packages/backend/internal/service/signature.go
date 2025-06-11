package service

import (
	"fmt"
	"time"

	"github.com/Tarunshrma/proof-or-reserve/internal/storage"
	"github.com/ethereum/go-ethereum/common"
)

// signatureServiceImpl is the concrete implementation of the SignatureService interface.
type signatureServiceImpl struct {
	signatureStore *storage.SignatureStorage
}

// NewSignatureService creates a new concrete signature service.
func NewSignatureService(store *storage.SignatureStorage) SignatureService {
	return &signatureServiceImpl{
		signatureStore: store,
	}
}

// StoreSignature stores a signature submitted by the frontend.
func (s *signatureServiceImpl) StoreSignature(token, wallet, signature string, validUntil uint64) error {
	// Validate signature format
	if len(signature) < 2 || signature[:2] != "0x" {
		return fmt.Errorf("invalid signature format: must be hex string starting with 0x")
	}

	// Convert hex string to bytes to check length
	sigBytes := common.FromHex(signature)
	if len(sigBytes) != 65 {
		fmt.Printf("Invalid signature length: %d bytes (expected 65 bytes)\n", len(sigBytes))
		fmt.Printf("Signature hex length: %d chars (expected 132 chars + 0x prefix)\n", len(signature))
		fmt.Printf("Raw signature: %s\n", signature)
		fmt.Printf("Signature bytes: %x\n", sigBytes)
		return fmt.Errorf("invalid signature length: must be 65 bytes (got %d bytes)", len(sigBytes))
	}

	record := &storage.SignatureRecord{
		Token:       token,
		Wallet:      wallet,
		Signature:   signature,
		ValidUntil:  validUntil,
		GeneratedAt: time.Now(),
	}

	return s.signatureStore.SaveSignature(record)
}

// GetSignature retrieves a stored signature.
func (s *signatureServiceImpl) GetSignature(token, wallet string) (*storage.SignatureRecord, error) {
	return s.signatureStore.GetSignature(token, wallet)
}

// IsSignatureValid checks if a signature exists and is still valid.
func (s *signatureServiceImpl) IsSignatureValid(token, wallet string) (bool, uint64, error) {
	record, err := s.signatureStore.GetSignature(token, wallet)
	if err != nil {
		return false, 0, err
	}

	if record == nil {
		return false, 0, nil
	}

	now := uint64(time.Now().Unix())
	if now > record.ValidUntil {
		return false, record.ValidUntil, nil
	}
	return true, record.ValidUntil, nil
}
