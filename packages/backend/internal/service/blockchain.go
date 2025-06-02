package service

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
)

// ProofOfReserveABI is the input ABI used to generate the binding from
const ProofOfReserveABI = `[
    {"inputs":[],"stateMutability":"nonpayable","type":"constructor"},
    {"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"previousOwner","type":"address"},{"indexed":true,"internalType":"address","name":"newOwner","type":"address"}],"name":"OwnershipTransferred","type":"event"},
    {"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"token","type":"address"},{"indexed":true,"internalType":"address","name":"wallet","type":"address"},{"indexed":false,"internalType":"uint256","name":"validUntil","type":"uint256"},{"indexed":false,"internalType":"bool","name":"success","type":"bool"}],"name":"ProofVerified","type":"event"},
    {"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"token","type":"address"},{"indexed":true,"internalType":"address","name":"wallet","type":"address"}],"name":"ReserveConfigured","type":"event"},
    {"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"token","type":"address"},{"indexed":true,"internalType":"address","name":"wallet","type":"address"}],"name":"ReserveDeactivated","type":"event"},
    {"inputs":[{"internalType":"address","name":"token","type":"address"},{"internalType":"address","name":"wallet","type":"address"}],"name":"configureReserve","outputs":[],"stateMutability":"nonpayable","type":"function"},
    {"inputs":[{"internalType":"address","name":"token","type":"address"},{"internalType":"address","name":"wallet","type":"address"}],"name":"deactivateReserve","outputs":[],"stateMutability":"nonpayable","type":"function"},
    {"inputs":[{"internalType":"bytes32","name":"messageHash","type":"bytes32"}],"name":"getEthSignedMessageHash","outputs":[{"internalType":"bytes32","name":"","type":"bytes32"}],"stateMutability":"pure","type":"function"},
    {"inputs":[{"internalType":"address","name":"token","type":"address"},{"internalType":"address","name":"wallet","type":"address"}],"name":"getReserveBalance","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
    {"inputs":[{"internalType":"address","name":"token","type":"address"},{"internalType":"address","name":"wallet","type":"address"}],"name":"isProofValid","outputs":[{"internalType":"bool","name":"isValid","type":"bool"},{"internalType":"uint256","name":"validUntil","type":"uint256"}],"stateMutability":"view","type":"function"},
    {"inputs":[{"internalType":"address","name":"","type":"address"},{"internalType":"address","name":"","type":"address"}],"name":"isReserveWallet","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},
    {"inputs":[{"internalType":"address","name":"token","type":"address"},{"internalType":"address","name":"wallet","type":"address"}],"name":"lastVerifiedTimestamp","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
    {"inputs":[],"name":"owner","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
    {"inputs":[{"internalType":"bytes32","name":"ethSignedMessageHash","type":"bytes32"},{"internalType":"bytes","name":"signature","type":"bytes"}],"name":"recoverSigner","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"pure","type":"function"},
    {"inputs":[{"internalType":"address","name":"newOwner","type":"address"}],"name":"transferOwnership","outputs":[],"stateMutability":"nonpayable","type":"function"},
    {"inputs":[{"internalType":"address","name":"token","type":"address"},{"internalType":"address","name":"wallet","type":"address"}],"name":"validUntilTimestamp","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
    {"inputs":[{"internalType":"address","name":"token","type":"address"},{"internalType":"address","name":"wallet","type":"address"},{"internalType":"bytes","name":"signature","type":"bytes"}],"name":"verifyProof","outputs":[{"internalType":"bool","name":"success","type":"bool"}],"stateMutability":"nonpayable","type":"function"}
]`

type BlockchainService struct {
	client         *ethclient.Client
	contractAddr   common.Address
	contractABI    abi.ABI
	defaultTimeout time.Duration
}

func NewBlockchainService(rpcURL string, contractAddress string) (*BlockchainService, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum client: %v", err)
	}

	contractABI, err := abi.JSON(strings.NewReader(ProofOfReserveABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse contract ABI: %v", err)
	}

	return &BlockchainService{
		client:         client,
		contractAddr:   common.HexToAddress(contractAddress),
		contractABI:    contractABI,
		defaultTimeout: 30 * time.Second,
	}, nil
}

func (s *BlockchainService) GetReserveBalance(token, wallet string) (*big.Int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.defaultTimeout)
	defer cancel()

	data, err := s.contractABI.Pack("getReserveBalance", common.HexToAddress(token), common.HexToAddress(wallet))
	if err != nil {
		return nil, fmt.Errorf("failed to pack data: %v", err)
	}

	msg := ethereum.CallMsg{
		To:   &s.contractAddr,
		Data: data,
	}

	output, err := s.client.CallContract(ctx, msg, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call contract: %v", err)
	}

	result := new(big.Int)
	if err := s.contractABI.UnpackIntoInterface(&result, "getReserveBalance", output); err != nil {
		return nil, fmt.Errorf("failed to unpack result: %v", err)
	}

	return result, nil
}

func (s *BlockchainService) VerifySignature(token, wallet, signature string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.defaultTimeout)
	defer cancel()

	// First check if contract exists
	code, err := s.client.CodeAt(ctx, s.contractAddr, nil)
	if err != nil {
		return false, fmt.Errorf("failed to check contract code: %v", err)
	}
	if len(code) == 0 {
		return false, fmt.Errorf("no contract found at address %s", s.contractAddr.Hex())
	}

	fmt.Printf("\n=== Contract Verification Debug ===\n")
	fmt.Printf("Contract Address: %s\n", s.contractAddr.Hex())
	fmt.Printf("Token: %s\n", token)
	fmt.Printf("Wallet: %s\n", wallet)
	fmt.Printf("Signature: %s\n", signature)

	// Check if reserve is active
	isActive, err := s.IsReserveWallet(token, wallet)
	if err != nil {
		return false, fmt.Errorf("failed to check reserve status: %v", err)
	}
	fmt.Printf("Is Reserve Active: %v\n", isActive)

	if !isActive {
		return false, fmt.Errorf("reserve not active")
	}

	// Convert parameters
	tokenAddr := common.HexToAddress(token)
	walletAddr := common.HexToAddress(wallet)
	signatureBytes := hexutil.MustDecode(signature)

	// Call verifyProof
	data, err := s.contractABI.Pack("verifyProof", tokenAddr, walletAddr, signatureBytes)
	if err != nil {
		return false, fmt.Errorf("failed to pack parameters: %v", err)
	}

	fmt.Printf("\nContract Call Data:\n")
	fmt.Printf("Method: verifyProof\n")
	fmt.Printf("Packed Data: %s\n", hexutil.Encode(data))

	// Make the call
	result, err := s.client.CallContract(ctx, ethereum.CallMsg{
		To:   &s.contractAddr,
		Data: data,
	}, nil)
	if err != nil {
		return false, fmt.Errorf("contract call failed: %v", err)
	}

	fmt.Printf("Contract Call Result: %s\n", hexutil.Encode(result))

	// Unpack result
	var success bool
	if err := s.contractABI.UnpackIntoInterface(&success, "verifyProof", result); err != nil {
		return false, fmt.Errorf("failed to unpack result: %v", err)
	}

	fmt.Printf("Verification Result: %v\n", success)
	fmt.Printf("=== End Debug ===\n\n")

	return success, nil
}

type proofValidityResult struct {
	IsValid    bool     `abi:"isValid"`
	ValidUntil *big.Int `abi:"validUntil"`
}

func (s *BlockchainService) IsProofValid(token, wallet string) (bool, uint64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.defaultTimeout)
	defer cancel()

	data, err := s.contractABI.Pack("isProofValid",
		common.HexToAddress(token),
		common.HexToAddress(wallet),
	)
	if err != nil {
		return false, 0, fmt.Errorf("failed to pack data: %v", err)
	}

	msg := ethereum.CallMsg{
		To:   &s.contractAddr,
		Data: data,
	}

	output, err := s.client.CallContract(ctx, msg, nil)
	if err != nil {
		return false, 0, fmt.Errorf("failed to call contract: %v", err)
	}

	result := new(proofValidityResult)
	if err := s.contractABI.UnpackIntoInterface(result, "isProofValid", output); err != nil {
		return false, 0, fmt.Errorf("failed to unpack result: %v", err)
	}

	return result.IsValid, result.ValidUntil.Uint64(), nil
}

func (s *BlockchainService) GetLastVerified(token, wallet string) (uint64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.defaultTimeout)
	defer cancel()

	data, err := s.contractABI.Pack("lastVerifiedTimestamp",
		common.HexToAddress(token),
		common.HexToAddress(wallet),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to pack data: %v", err)
	}

	msg := ethereum.CallMsg{
		To:   &s.contractAddr,
		Data: data,
	}

	output, err := s.client.CallContract(ctx, msg, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to call contract: %v", err)
	}

	timestamp := new(big.Int)
	if err := s.contractABI.UnpackIntoInterface(&timestamp, "lastVerifiedTimestamp", output); err != nil {
		return 0, fmt.Errorf("failed to unpack result: %v", err)
	}

	return timestamp.Uint64(), nil
}

func (s *BlockchainService) IsReserveWallet(token, wallet string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.defaultTimeout)
	defer cancel()

	// First check if contract exists
	code, err := s.client.CodeAt(ctx, s.contractAddr, nil)
	if err != nil {
		return false, fmt.Errorf("failed to check contract code: %v", err)
	}
	if len(code) == 0 {
		return false, fmt.Errorf("no contract found at address %s", s.contractAddr.Hex())
	}

	// Use the mapping getter instead of the function
	tokenAddr := common.HexToAddress(token)
	walletAddr := common.HexToAddress(wallet)

	data, err := s.contractABI.Pack("isReserveWallet", tokenAddr, walletAddr)
	if err != nil {
		return false, fmt.Errorf("failed to pack data: %v", err)
	}

	msg := ethereum.CallMsg{
		To:   &s.contractAddr,
		Data: data,
	}

	output, err := s.client.CallContract(ctx, msg, nil)
	if err != nil {
		return false, fmt.Errorf("failed to call contract: %v", err)
	}

	if len(output) == 0 {
		return false, fmt.Errorf("empty response from contract")
	}

	var isActive bool
	if err := s.contractABI.UnpackIntoInterface(&isActive, "isReserveWallet", output); err != nil {
		// Try unpacking as a mapping getter
		if err2 := s.contractABI.UnpackIntoInterface(&isActive, "", output); err2 != nil {
			return false, fmt.Errorf("failed to unpack result: %v (output length: %d)", err, len(output))
		}
	}

	return isActive, nil
}
