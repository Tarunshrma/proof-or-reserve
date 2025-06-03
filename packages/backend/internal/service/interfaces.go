package service

import (
	"math/big"

	// We need to know the actual return type of GetSignature.
	// Assuming it's from storage for now as per the mock.
	// If it's a type defined in the service package itself, adjust the import.
	"github.com/Tarunshrma/proof-or-reserve/internal/storage"
)

// BlockchainService defines the interface for blockchain operations required by the API handler.
type BlockchainService interface {
	GetReserveDetails(token, wallet string) (*ReserveDetailsOutput, error)
	IsReserveWallet(token, wallet string) (bool, error)
	VerifySignature(token, wallet, signature string) (bool, error)
	GetReserveBalance(token, wallet string) (*big.Int, error)
}

// SignatureService defines the interface for signature operations required by the API handler.
type SignatureService interface {
	GenerateSignature(token, wallet string) error
	GetSignature(token, wallet string) (*storage.SignatureRecord, error) // Ensure SignatureRecord is the correct type and package
}
