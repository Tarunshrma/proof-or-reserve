package service

import (
	"math/big"

	"github.com/Tarunshrma/proof-or-reserve/internal/storage"
)

// BlockchainService defines the interface for blockchain interactions
type BlockchainService interface {
	GetReserveBalance(token, wallet string) (*big.Int, error)
	VerifySignature(token, wallet, signature string) (bool, error)
	IsReserveWallet(token, wallet string) (bool, error)
	GetReserveDetails(token, wallet string) (*ReserveDetailsOutput, error)
	GetChainID() (*big.Int, error)
	GetContractAddress() string
	VerifySignatureOffchain(token, wallet, signature, payload string) (bool, error)
	SetPrivateKey(privateKeyHex string) error
}

// SignatureService defines the interface for signature operations
type SignatureService interface {
	StoreSignature(token, wallet, signature string, validUntil uint64) error
	GetSignature(token, wallet string) (*storage.SignatureRecord, error)
	IsSignatureValid(token, wallet string) (bool, uint64, error)
}
