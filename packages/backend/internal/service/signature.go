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

type SignatureService struct {
	privateKey     *ecdsa.PrivateKey
	signatureStore *storage.SignatureStorage
}

func NewSignatureService(privateKeyHex string, store *storage.SignatureStorage) (*SignatureService, error) {
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %v", err)
	}

	address := crypto.PubkeyToAddress(privateKey.PublicKey)
	fmt.Printf("Address: %s\n", address.Hex())

	return &SignatureService{
		privateKey:     privateKey,
		signatureStore: store,
	}, nil
}

func (s *SignatureService) GenerateSignature(token, wallet string) error {
	// Convert addresses to checksum format
	tokenAddr := common.HexToAddress(token)
	walletAddr := common.HexToAddress(wallet)

	// Debug logging
	fmt.Printf("\n=== Signature Generation Debug ===\n")
	fmt.Printf("Input Parameters:\n")
	fmt.Printf("Token Address: %s\n", tokenAddr.Hex())
	fmt.Printf("Wallet Address: %s\n", walletAddr.Hex())

	// Pack parameters with a simple prefix
	prefix := []byte("ProofOfReserve:")
	packedData := append(prefix, tokenAddr.Bytes()...)
	packedData = append(packedData, walletAddr.Bytes()...)

	fmt.Printf("\nPacked Data (hex): %s\n", hexutil.Encode(packedData))

	// Create message hash
	messageHash := crypto.Keccak256Hash(packedData)
	fmt.Printf("Message Hash: %s\n", messageHash.Hex())

	// Create Ethereum signed message hash
	ethSignedMessageHash := crypto.Keccak256Hash(
		[]byte("\x19Ethereum Signed Message:\n32"),
		messageHash.Bytes(),
	)
	fmt.Printf("Eth Signed Message Hash: %s\n", ethSignedMessageHash.Hex())

	// Sign the hash
	signature, err := crypto.Sign(ethSignedMessageHash.Bytes(), s.privateKey)
	if err != nil {
		return fmt.Errorf("failed to sign message: %v", err)
	}

	// Get the signer's address for verification
	publicKey, err := crypto.Ecrecover(ethSignedMessageHash.Bytes(), signature)
	if err != nil {
		return fmt.Errorf("failed to recover public key: %v", err)
	}

	pubKey, err := crypto.UnmarshalPubkey(publicKey)
	if err != nil {
		return fmt.Errorf("failed to unmarshal public key: %v", err)
	}

	recoveredAddr := crypto.PubkeyToAddress(*pubKey)
	fmt.Printf("\nSignature Components:\n")
	fmt.Printf("R: %s\n", hexutil.Encode(signature[:32]))
	fmt.Printf("S: %s\n", hexutil.Encode(signature[32:64]))
	fmt.Printf("V (before adjustment): %d\n", signature[64])

	// Fix v value for Ethereum's EIP-155
	if signature[64] < 27 {
		signature[64] += 27
	}
	fmt.Printf("V (after adjustment): %d\n", signature[64])

	fmt.Printf("\nVerification:\n")
	fmt.Printf("Recovered signer address: %s\n", recoveredAddr.Hex())
	fmt.Printf("Expected wallet address: %s\n", walletAddr.Hex())
	fmt.Printf("Addresses match: %v\n", recoveredAddr == walletAddr)

	// Store the signature
	record := &storage.SignatureRecord{
		Token:       token,
		Wallet:      wallet,
		Signature:   hexutil.Encode(signature),
		ValidUntil:  uint64(time.Now().Add(30 * 24 * time.Hour).Unix()), // Keep this for compatibility
		GeneratedAt: time.Now(),
	}

	fmt.Printf("\nFinal Signature: %s\n", record.Signature)
	fmt.Printf("=== End Debug ===\n\n")

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
