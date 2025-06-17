package types

import (
	"math/big"
)

// ReserveDetailsOutput represents the output from GetReserveDetails
type ReserveDetailsOutput struct {
	IsConfigured     bool   `json:"isConfigured"`
	Name             string `json:"name"`
	Symbol           string `json:"symbol"`
	Balance          string `json:"balance"`
	LastVerified     string `json:"lastVerified"`
	Target           string `json:"target"`
	ThresholdPercent int    `json:"thresholdPercent"`
}

// BlockchainService defines the interface for blockchain interactions
type BlockchainService interface {
	// IsReserveWallet checks if a wallet is configured as a reserve for a token
	IsReserveWallet(token, wallet string) (bool, error)

	// GetReserveDetails fetches comprehensive details about a reserve
	GetReserveDetails(token, wallet string) (*ReserveDetailsOutput, error)

	// VerifyProof verifies an EIP-712 signature proof
	VerifyProof(token, wallet string, validUntil uint64, signature string) (bool, error)

	// GetReserveBalance gets the current balance of a reserve wallet
	GetReserveBalance(token, wallet string) (*big.Int, error)

	// GetChainID returns the current chain ID
	GetChainID() (*big.Int, error)

	// GetContractAddress returns the contract address
	GetContractAddress() string

	// SetPrivateKey sets the private key for sending transactions
	SetPrivateKey(privateKeyHex string) error

	// ConfigureReserve configures a wallet as a reserve for a token
	ConfigureReserve(token, wallet string, target *big.Int, thresholdPercent int) error
}
