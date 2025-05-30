package service

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"time"

	"github.com/Tarunshrma/proof-or-reserve/internal/storage"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type SignatureService struct {
	privateKey *ecdsa.PrivateKey
	storage    *storage.SignatureStorage
}

func NewSignatureService(privateKeyHex string, storage *storage.SignatureStorage) (*SignatureService, error) {
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %v", err)
	}

	return &SignatureService{
		privateKey: privateKey,
		storage:    storage,
	}, nil
}

func (s *SignatureService) GenerateSignature(token, wallet string) error {
	// Calculate validUntil (30 days from now)
	validUntil := uint64(time.Now().Add(30 * 24 * time.Hour).Unix())

	// Create message hash (matching the smart contract's getMessageHash function)
	msg := crypto.Keccak256(
		common.HexToAddress(token).Bytes(),
		common.HexToAddress(wallet).Bytes(),
		new(big.Int).SetUint64(validUntil).Bytes(),
	)

	// Add Ethereum specific prefix
	prefixedMsg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(msg), msg)
	finalMsg := crypto.Keccak256([]byte(prefixedMsg))

	// Sign the message
	signature, err := crypto.Sign(finalMsg, s.privateKey)
	if err != nil {
		return fmt.Errorf("failed to sign message: %v", err)
	}

	// Store the signature
	record := &storage.SignatureRecord{
		Token:       token,
		Wallet:      wallet,
		Signature:   common.Bytes2Hex(signature),
		ValidUntil:  validUntil,
		GeneratedAt: time.Now(),
	}

	return s.storage.Store(record)
}
