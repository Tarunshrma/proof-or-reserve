package service

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"time"

	"github.com/Tarunshrma/proof-or-reserve/internal/storage"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

type SignatureService struct {
	privateKey     *ecdsa.PrivateKey
	signatureStore *storage.SignatureStorage
}

func NewSignatureService(privateKeyHex string, store *storage.SignatureStorage) (*SignatureService, error) {
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %v", err)
	}

	return &SignatureService{
		privateKey:     privateKey,
		signatureStore: store,
	}, nil
}

func (s *SignatureService) GenerateSignature(token, wallet string) error {
	// Convert addresses to checksum format
	tokenAddr := common.HexToAddress(token)
	walletAddr := common.HexToAddress(wallet)

	// Set validity period (30 days from now)
	validUntil := uint64(time.Now().Add(30 * 24 * time.Hour).Unix())

	// Create message hash as per smart contract
	message := crypto.Keccak256(
		common.LeftPadBytes(tokenAddr.Bytes(), 32),
		common.LeftPadBytes(walletAddr.Bytes(), 32),
		common.LeftPadBytes(big.NewInt(int64(validUntil)).Bytes(), 32),
	)

	// Create Ethereum signed message hash
	ethMessage := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), message)
	ethHash := crypto.Keccak256Hash([]byte(ethMessage))

	// Sign the hash
	signature, err := crypto.Sign(ethHash.Bytes(), s.privateKey)
	if err != nil {
		return fmt.Errorf("failed to sign message: %v", err)
	}

	// Store the signature
	record := &storage.SignatureRecord{
		Token:       token,
		Wallet:      wallet,
		Signature:   hexutil.Encode(signature),
		ValidUntil:  validUntil,
		GeneratedAt: time.Now(),
	}

	return s.signatureStore.SaveSignature(record)
}

func (s *SignatureService) GetSignature(token, wallet string) (*storage.SignatureRecord, error) {
	return s.signatureStore.GetSignature(token, wallet)
}

func (s *SignatureService) IsSignatureValid(token, wallet string) (bool, uint64, error) {
	record, err := s.signatureStore.GetSignature(token, wallet)
	if err != nil {
		return false, 0, err
	}

	if record == nil {
		return false, 0, nil
	}

	// Check if signature is still valid
	now := uint64(time.Now().Unix())
	if now > record.ValidUntil {
		return false, record.ValidUntil, nil
	}

	return true, record.ValidUntil, nil
}
