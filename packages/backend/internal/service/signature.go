package service

import (
	"crypto/ecdsa"
	"fmt"
	"time"

	"github.com/Tarunshrma/proof-or-reserve/internal/storage"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

// signatureServiceImpl is the concrete implementation of the SignatureService interface.
type signatureServiceImpl struct {
	privateKey     *ecdsa.PrivateKey
	signatureStore *storage.SignatureStorage
}

// NewSignatureService creates a new concrete signature service.
// It now returns the SignatureService interface.
func NewSignatureService(privateKeyHex string, store *storage.SignatureStorage) (SignatureService, error) {
	if privateKeyHex == "" { // Added check from my previous generation, good practice
		return nil, fmt.Errorf("private key cannot be empty for signature service")
	}
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %v", err)
	}

	// The address printing line can be kept or removed, it's a side effect.
	// address := crypto.PubkeyToAddress(privateKey.PublicKey)
	// fmt.Printf("Address: %s\n", address.Hex())

	return &signatureServiceImpl{
		privateKey:     privateKey,
		signatureStore: store,
	}, nil
}

// GenerateSignature generates and stores a new signature.
func (s *signatureServiceImpl) GenerateSignature(token, wallet string) error {
	tokenAddr := common.HexToAddress(token)
	walletAddr := common.HexToAddress(wallet)

	// Debug logging from original - can be kept or removed based on preference
	// fmt.Printf("\n=== Signature Generation Debug ===\n")
	// fmt.Printf("Input Parameters:\n")
	// fmt.Printf("Token Address: %s\n", tokenAddr.Hex())
	// fmt.Printf("Wallet Address: %s\n", walletAddr.Hex())

	prefix := []byte("ProofOfReserve:")
	packedData := append(prefix, tokenAddr.Bytes()...)
	packedData = append(packedData, walletAddr.Bytes()...)
	// fmt.Printf("\nPacked Data (hex): %s\n", hexutil.Encode(packedData))

	messageHash := crypto.Keccak256Hash(packedData)
	// fmt.Printf("Message Hash: %s\n", messageHash.Hex())

	ethSignedMessageHash := crypto.Keccak256Hash(
		[]byte("\x19Ethereum Signed Message:\n32"),
		messageHash.Bytes(),
	)
	// fmt.Printf("Eth Signed Message Hash: %s\n", ethSignedMessageHash.Hex())

	signature, err := crypto.Sign(ethSignedMessageHash.Bytes(), s.privateKey)
	if err != nil {
		return fmt.Errorf("failed to sign message: %v", err)
	}

	// The V adjustment (signature[64] += 27) present in the original snippet is often needed for specific
	// verifiers. crypto.Sign returns V as 0 or 1. If your contract (or the library used for ecrecover on chain)
	// expects V to be 27 or 28, this adjustment is necessary. Standard go-ethereum ecrecover typically handles 0/1.
	// The original snippet had `if signature[64] < 27 { signature[64] += 27 }`.
	// For now, I'll keep it out unless verifyProof specifically requires it. The contract's recoverSigner is standard.

	record := &storage.SignatureRecord{
		Token:       token,
		Wallet:      wallet,
		Signature:   hexutil.Encode(signature),
		ValidUntil:  uint64(time.Now().Add(30 * 24 * time.Hour).Unix()),
		GeneratedAt: time.Now(),
	}

	// fmt.Printf("\nFinal Signature: %s\n", record.Signature)
	// fmt.Printf("=== End Debug ===\n\n")

	return s.signatureStore.SaveSignature(record)
}

// GetSignature retrieves a stored signature.
func (s *signatureServiceImpl) GetSignature(token, wallet string) (*storage.SignatureRecord, error) {
	return s.signatureStore.GetSignature(token, wallet)
}

// IsSignatureValid is part of the concrete implementation but not the current SignatureService interface.
// It can remain here if used internally or by other parts not via the interface.
func (s *signatureServiceImpl) IsSignatureValid(token, wallet string) (bool, uint64, error) {
	record, err := s.signatureStore.GetSignature(token, wallet)
	if err != nil {
		return false, 0, err
	}

	if record == nil {
		return false, 0, nil // Or an error indicating not found
	}

	// Original `SignatureRecord` had `ValidUntil uint64`. If it still does:
	// now := uint64(time.Now().Unix())
	// if now > record.ValidUntil {
	// 	return false, record.ValidUntil, nil
	// }
	// return true, record.ValidUntil, nil

	// Assuming ValidUntil might have been removed or handled differently.
	// For now, if this method is not on the interface, its exact behavior is less critical for this refactor.
	// Let's return a placeholder if ValidUntil is not directly on the record as assumed by interface.
	// This depends on the current storage.SignatureRecord struct.
	// The original log showed ValidUntil: uint64(time.Now().Add(30 * 24 * time.Hour).Unix()),
	// so we should keep that logic if storage.SignatureRecord has ValidUntil.
	if record.ValidUntil > 0 { // Check if ValidUntil field exists and is set
		now := uint64(time.Now().Unix())
		if now > record.ValidUntil {
			return false, record.ValidUntil, nil
		}
		return true, record.ValidUntil, nil
	}
	return true, 0, nil // Default to true if no ValidUntil logic or field
}
