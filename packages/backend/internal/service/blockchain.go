package service

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"reflect"
	"strings"
	"time"

	"bytes"

	"github.com/Tarunshrma/proof-or-reserve/internal/types"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// blockchainServiceImpl is the concrete implementation of the types.BlockchainService interface.
type blockchainServiceImpl struct {
	client           *ethclient.Client
	contractAddr     common.Address
	contractABI      abi.ABI
	privateKeyString string
	defaultTimeout   time.Duration
}

// Struct to help parse the Hardhat artifact JSON
type ContractArtifact struct {
	Abi json.RawMessage `json:"abi"` // We only need the abi part as raw message first
	// Add other fields like Bytecode if needed elsewhere, but not for current ABI parsing.
}

// NewBlockchainService creates a new instance of BlockchainService.
// privateKeyHex is optional and only required for sending transactions.
func NewBlockchainService(rpcURL string, contractAddress string, abiFilePath string) (types.BlockchainService, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum client: %v", err)
	}

	abiFileBytes, err := ioutil.ReadFile(abiFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read ABI file from %s: %v", abiFilePath, err)
	}

	var artifact ContractArtifact
	if err := json.Unmarshal(abiFileBytes, &artifact); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ABI artifact JSON from %s: %v", abiFilePath, err)
	}

	abiJsonString := string(artifact.Abi)
	if abiJsonString == "null" || abiJsonString == "" {
		return nil, fmt.Errorf("ABI data not found or empty in artifact file %s", abiFilePath)
	}

	parsedABI, err := abi.JSON(strings.NewReader(abiJsonString))
	if err != nil {
		return nil, fmt.Errorf("failed to parse contract ABI from string: %v", err)
	}

	return &blockchainServiceImpl{
		client:         client,
		contractAddr:   common.HexToAddress(contractAddress),
		contractABI:    parsedABI,
		defaultTimeout: 30 * time.Second,
	}, nil
}

// SetPrivateKey sets the private key for sending transactions
func (s *blockchainServiceImpl) SetPrivateKey(privateKeyHex string) error {
	if privateKeyHex == "" {
		return fmt.Errorf("private key cannot be empty")
	}

	pk := privateKeyHex
	if strings.HasPrefix(pk, "0x") {
		pk = pk[2:]
	}
	_, err := ethcrypto.HexToECDSA(pk)
	if err != nil {
		return fmt.Errorf("invalid private key: %v", err)
	}

	s.privateKeyString = privateKeyHex
	return nil
}

// GetReserveBalance is a method of blockchainServiceImpl.
func (s *blockchainServiceImpl) GetReserveBalance(token, wallet string) (*big.Int, error) {
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

// VerifySignature sends a transaction to the verifyProof method on the smart contract.
func (s *blockchainServiceImpl) VerifySignature(token, wallet, signature string) (bool, error) {
	if s.privateKeyString == "" {
		return false, fmt.Errorf("private key not set, required for sending transactions")
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.defaultTimeout+30*time.Second)
	defer cancel()

	pkHex := s.privateKeyString
	if strings.HasPrefix(pkHex, "0x") {
		pkHex = pkHex[2:]
	}
	privateKeyECDSA, err := ethcrypto.HexToECDSA(pkHex)
	if err != nil {
		return false, fmt.Errorf("failed to parse private key: %v", err)
	}

	publicKeyECDSA, ok := privateKeyECDSA.Public().(*ecdsa.PublicKey)
	if !ok {
		return false, fmt.Errorf("failed to derive public key")
	}
	fromAddress := ethcrypto.PubkeyToAddress(*publicKeyECDSA)

	chainID, err := s.client.ChainID(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get chain ID: %v", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKeyECDSA, chainID)
	if err != nil {
		return false, fmt.Errorf("failed to create transactor: %v", err)
	}

	nonce, err := s.client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		return false, fmt.Errorf("failed to get pending nonce: %v", err)
	}
	auth.Nonce = big.NewInt(int64(nonce))

	gasPrice, err := s.client.SuggestGasPrice(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to suggest gas price: %v", err)
	}
	auth.GasPrice = gasPrice
	auth.GasLimit = uint64(300000)
	auth.Value = big.NewInt(0)

	tokenAddr := common.HexToAddress(token)
	walletAddr := common.HexToAddress(wallet)

	// Convert signature to bytes
	var signatureBytes []byte
	if strings.HasPrefix(signature, "0x") {
		signatureBytes = common.FromHex(signature)
	} else {
		signatureBytes = common.FromHex("0x" + signature)
	}

	// Validate signature length
	if len(signatureBytes) != 65 {
		return false, fmt.Errorf("invalid signature length: must be 65 bytes (got %d bytes)", len(signatureBytes))
	}

	// Pack parameters for verifyProof
	data, err := s.contractABI.Pack("verifyProof", tokenAddr, walletAddr, signatureBytes)
	if err != nil {
		return false, fmt.Errorf("failed to pack parameters for verifyProof: %v", err)
	}

	// Create and send transaction
	tx := ethtypes.NewTransaction(auth.Nonce.Uint64(), s.contractAddr, auth.Value, auth.GasLimit, auth.GasPrice, data)

	signedTx, err := ethtypes.SignTx(tx, ethtypes.NewEIP155Signer(chainID), privateKeyECDSA)
	if err != nil {
		return false, fmt.Errorf("failed to sign transaction: %v", err)
	}

	err = s.client.SendTransaction(ctx, signedTx)
	if err != nil {
		return false, fmt.Errorf("failed to send transaction: %v", err)
	}

	// Wait for transaction to be mined
	receipt, err := bind.WaitMined(ctx, s.client, signedTx)
	if err != nil {
		return false, fmt.Errorf("failed to mine transaction: %v", err)
	}

	// Check transaction status
	if receipt.Status == ethtypes.ReceiptStatusFailed {
		callErr := checkTxFailureReason(s.client, fromAddress, signedTx, receipt.BlockNumber)
		log.Printf("On-chain transaction %s failed. Block: %s. Revert Reason: %v", signedTx.Hash().Hex(), receipt.BlockNumber.String(), callErr)
		return false, fmt.Errorf("transaction failed on-chain: %v", callErr)
	}

	// Check if the transaction was successful
	var success bool
	if len(receipt.Logs) > 0 {
		for _, log := range receipt.Logs {
			if len(log.Topics) > 0 && log.Topics[0] == common.HexToHash("0x7e99f890d99474682ed3b6b99077a328caa8be3436fc94725d48a8f6ab34d723") {
				// This is the ProofVerified event
				success = true
				break
			}
		}
	}

	return success, nil
}

// Helper function to try and get a revert reason
func checkTxFailureReason(client *ethclient.Client, from common.Address, tx *ethtypes.Transaction, blockNumber *big.Int) error {
	msg := ethereum.CallMsg{
		From:     from,
		To:       tx.To(),
		Gas:      tx.Gas(),
		GasPrice: tx.GasPrice(),
		Value:    tx.Value(),
		Data:     tx.Data(),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, callErr := client.CallContract(ctx, msg, blockNumber)
	if callErr == nil {
		return fmt.Errorf("revert reason in response: %s", string(res))
	}
	// If CallContract errors, the error message itself might contain the revert reason.
	return fmt.Errorf("call to check revert reason failed: %w (raw call output: %s)", callErr, string(res))
}

// IsReserveWallet checks if a wallet is configured as a reserve for a token
func (s *blockchainServiceImpl) IsReserveWallet(token, wallet string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.defaultTimeout)
	defer cancel()

	// Get the reserve config from the contract
	data, err := s.contractABI.Pack("reserveConfigs", common.HexToAddress(token), common.HexToAddress(wallet))
	if err != nil {
		return false, fmt.Errorf("failed to pack data for reserveConfigs: %v", err)
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
		// No config exists yet
		return false, nil
	}
	type ReserveConfig struct {
		IsConfigured          bool
		Target                *big.Int
		ThresholdPercent      *big.Int
		LastVerifiedTimestamp *big.Int
	}

	var config ReserveConfig
	if err := s.contractABI.UnpackIntoInterface(&config, "reserveConfigs", output); err != nil {
		return false, fmt.Errorf("failed to unpack result: %v", err)
	}

	return config.IsConfigured, nil
}

// GetReserveDetails fetches comprehensive details about a reserve
func (s *blockchainServiceImpl) GetReserveDetails(token, wallet string) (*types.ReserveDetailsOutput, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.defaultTimeout)
	defer cancel()

	data, err := s.contractABI.Pack("getReserveDetails", common.HexToAddress(token), common.HexToAddress(wallet))
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

	// Get the reserve config to include target and threshold
	configData, err := s.contractABI.Pack("reserveConfigs", common.HexToAddress(token), common.HexToAddress(wallet))
	if err != nil {
		return nil, fmt.Errorf("failed to pack data for reserveConfigs: %v", err)
	}

	configMsg := ethereum.CallMsg{
		To:   &s.contractAddr,
		Data: configData,
	}

	configOutput, err := s.client.CallContract(ctx, configMsg, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call contract for config: %v", err)
	}

	type ReserveConfig struct {
		IsConfigured          bool
		Target                *big.Int
		ThresholdPercent      *big.Int
		LastVerifiedTimestamp *big.Int
	}

	var config ReserveConfig
	if err := s.contractABI.UnpackIntoInterface(&config, "reserveConfigs", configOutput); err != nil {
		return nil, fmt.Errorf("failed to unpack config result: %v", err)
	}

	// Unpack the details result (Solidity returns a tuple as a single []interface{} in Go)
	type ReserveDetailsResult struct {
		IsConfigured     bool
		Name             string
		Symbol           string
		Balance          *big.Int
		LastVerified     *big.Int
		Target           *big.Int
		ThresholdPercent *big.Int
	}

	// Use Unpack to get a []interface{} for the tuple
	unpacked, err := s.contractABI.Unpack("getReserveDetails", output)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack details result: %v", err)
	}
	if len(unpacked) == 0 {
		return nil, fmt.Errorf("no data returned from getReserveDetails")
	}

	var details ReserveDetailsResult
	if tuple, ok := unpacked[0].([]interface{}); ok && len(tuple) == 7 {
		details = ReserveDetailsResult{
			IsConfigured:     tuple[0].(bool),
			Name:             tuple[1].(string),
			Symbol:           tuple[2].(string),
			Balance:          tuple[3].(*big.Int),
			LastVerified:     tuple[4].(*big.Int),
			Target:           tuple[5].(*big.Int),
			ThresholdPercent: tuple[6].(*big.Int),
		}
	} else {
		log.Printf("[DEBUG] getReserveDetails: unpacked[0] type: %T, value: %+v", unpacked[0], unpacked[0])
		v := reflect.ValueOf(unpacked[0])
		details = ReserveDetailsResult{
			IsConfigured:     v.FieldByName("IsConfigured").Bool(),
			Name:             v.FieldByName("Name").String(),
			Symbol:           v.FieldByName("Symbol").String(),
			Balance:          v.FieldByName("Balance").Interface().(*big.Int),
			LastVerified:     v.FieldByName("LastVerified").Interface().(*big.Int),
			Target:           v.FieldByName("Target").Interface().(*big.Int),
			ThresholdPercent: v.FieldByName("ThresholdPercent").Interface().(*big.Int),
		}
	}

	return &types.ReserveDetailsOutput{
		IsConfigured:     details.IsConfigured,
		Name:             details.Name,
		Symbol:           details.Symbol,
		Balance:          details.Balance.String(),
		LastVerified:     details.LastVerified.String(),
		Target:           details.Target.String(),
		ThresholdPercent: int(details.ThresholdPercent.Int64()),
	}, nil
}

// GetChainID returns the current chain ID
func (s *blockchainServiceImpl) GetChainID() (*big.Int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.defaultTimeout)
	defer cancel()
	return s.client.ChainID(ctx)
}

// GetContractAddress returns the contract address
func (s *blockchainServiceImpl) GetContractAddress() string {
	return s.contractAddr.Hex()
}

// VerifySignatureOffchain verifies a signature without submitting a transaction
func (s *blockchainServiceImpl) VerifySignatureOffchain(token, wallet, signature, payload string) (bool, error) {
	fmt.Printf("\n=== Debug: Signature Verification ===\n")
	fmt.Printf("Token: %s\n", token)
	fmt.Printf("Wallet: %s\n", wallet)
	fmt.Printf("Signature: %s\n", signature)
	fmt.Printf("Signature length: %d chars\n", len(signature))
	fmt.Printf("Payload: %s\n", payload)

	// 1. Verify the payload format matches what we expect
	tokenAddr := common.HexToAddress(token)
	walletAddr := common.HexToAddress(wallet)

	// Reconstruct the expected payload
	prefix := []byte("ProofOfReserve:")
	expectedPayload := append(prefix, tokenAddr.Bytes()...)
	expectedPayload = append(expectedPayload, walletAddr.Bytes()...)

	// Convert the provided payload from hex to bytes
	providedPayload := common.FromHex(payload)
	if !bytes.Equal(expectedPayload, providedPayload) {
		fmt.Printf("Payload mismatch!\n")
		fmt.Printf("Expected: %x\n", expectedPayload)
		fmt.Printf("Provided: %x\n", providedPayload)
		return false, fmt.Errorf("payload mismatch")
	}
	fmt.Printf("Payload verification passed\n")

	// 2. Get the signer's address from the signature
	sig := common.FromHex(signature)
	if len(sig) != 65 {
		fmt.Printf("Invalid signature length: %d bytes (expected 65 bytes)\n", len(sig))
		fmt.Printf("Signature hex length: %d chars (expected 132 chars + 0x prefix)\n", len(signature))
		fmt.Printf("Raw signature: %s\n", signature)
		fmt.Printf("Signature bytes: %x\n", sig)
		return false, fmt.Errorf("invalid signature length: must be 65 bytes")
	}
	fmt.Printf("Signature length check passed\n")
	fmt.Printf("Signature bytes: %x\n", sig)
	fmt.Printf("V value before adjustment: %d\n", sig[64])

	// Adjust V value for Ethereum's personal_sign
	// If V is 27/28, convert to 0/1
	if sig[64] == 27 || sig[64] == 28 {
		sig[64] -= 27
	}
	fmt.Printf("V value after adjustment: %d\n", sig[64])

	// For personal_sign, we need to match the smart contract's verification:
	// 1. First hash the message: keccak256(abi.encodePacked("ProofOfReserve:", token, wallet))
	// 2. Then add the Ethereum signed message prefix to the hash
	// 3. Then hash again
	messageHash := ethcrypto.Keccak256(expectedPayload)
	fmt.Printf("Message hash: %x\n", messageHash)

	// Add Ethereum signed message prefix to the hash
	prefix = []byte(fmt.Sprintf("\x19Ethereum Signed Message:\n32"))
	msg := append(prefix, messageHash...)
	fmt.Printf("Complete message (hex): %x\n", msg)

	// Hash the complete message
	ethSignedMessageHash := ethcrypto.Keccak256(msg)
	fmt.Printf("Eth signed message hash: %x\n", ethSignedMessageHash)

	// Recover the signer's address
	pubKey, err := ethcrypto.Ecrecover(ethSignedMessageHash, sig)
	if err != nil {
		fmt.Printf("Failed to recover public key: %v\n", err)
		return false, fmt.Errorf("failed to recover public key: %v", err)
	}

	// Convert public key to address
	recoveredAddr := common.BytesToAddress(ethcrypto.Keccak256(pubKey[1:])[12:])
	fmt.Printf("Recovered address: %s\n", recoveredAddr.Hex())
	fmt.Printf("Expected wallet: %s\n", wallet)

	// Compare addresses
	return strings.EqualFold(recoveredAddr.Hex(), wallet), nil
}

// VerifyProof verifies an EIP-712 signature proof
func (s *blockchainServiceImpl) VerifyProof(token, wallet string, validUntil uint64, signature string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.defaultTimeout)
	defer cancel()

	tokenAddr := common.HexToAddress(token)
	walletAddr := common.HexToAddress(wallet)

	data, err := s.contractABI.Pack("verifyProof", tokenAddr, walletAddr, big.NewInt(int64(validUntil)), common.FromHex(signature))
	if err != nil {
		return false, fmt.Errorf("failed to pack data for verifyProof: %v", err)
	}

	msg := ethereum.CallMsg{
		To:   &s.contractAddr,
		Data: data,
	}

	output, err := s.client.CallContract(ctx, msg, nil)
	if err != nil {
		return false, fmt.Errorf("failed to call contract for verifyProof: %v", err)
	}

	var success bool
	err = s.contractABI.UnpackIntoInterface(&success, "verifyProof", output)
	if err != nil {
		return false, fmt.Errorf("failed to unpack verifyProof result: %v", err)
	}

	return success, nil
}

// ConfigureReserve configures a wallet as a reserve for a token
func (s *blockchainServiceImpl) ConfigureReserve(token, wallet string, target *big.Int, thresholdPercent int) error {
	if s.privateKeyString == "" {
		return fmt.Errorf("private key not set, required for sending transactions")
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.defaultTimeout+30*time.Second)
	defer cancel()

	pkHex := s.privateKeyString
	if strings.HasPrefix(pkHex, "0x") {
		pkHex = pkHex[2:]
	}
	privateKeyECDSA, err := ethcrypto.HexToECDSA(pkHex)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %v", err)
	}

	publicKeyECDSA, ok := privateKeyECDSA.Public().(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("failed to derive public key")
	}
	fromAddress := ethcrypto.PubkeyToAddress(*publicKeyECDSA)

	chainID, err := s.client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get chain ID: %v", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKeyECDSA, chainID)
	if err != nil {
		return fmt.Errorf("failed to create transactor: %v", err)
	}

	nonce, err := s.client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		return fmt.Errorf("failed to get pending nonce: %v", err)
	}
	auth.Nonce = big.NewInt(int64(nonce))

	gasPrice, err := s.client.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("failed to suggest gas price: %v", err)
	}
	auth.GasPrice = gasPrice
	auth.GasLimit = uint64(300000)
	auth.Value = big.NewInt(0)

	tokenAddr := common.HexToAddress(token)
	walletAddr := common.HexToAddress(wallet)

	// Pack parameters for configureReserve
	data, err := s.contractABI.Pack("configureReserve", tokenAddr, walletAddr, target, big.NewInt(int64(thresholdPercent)))
	if err != nil {
		return fmt.Errorf("failed to pack parameters for configureReserve: %v", err)
	}

	// Create and send transaction
	tx := ethtypes.NewTransaction(auth.Nonce.Uint64(), s.contractAddr, auth.Value, auth.GasLimit, auth.GasPrice, data)

	signedTx, err := ethtypes.SignTx(tx, ethtypes.NewEIP155Signer(chainID), privateKeyECDSA)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %v", err)
	}

	err = s.client.SendTransaction(ctx, signedTx)
	if err != nil {
		return fmt.Errorf("failed to send transaction: %v", err)
	}

	// Wait for transaction receipt
	receipt, err := bind.WaitMined(ctx, s.client, signedTx)
	if err != nil {
		return fmt.Errorf("failed to get transaction receipt: %v", err)
	}

	if receipt.Status == 0 {
		return fmt.Errorf("transaction failed")
	}

	return nil
}
