package service

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"strings"
	"time"

	"bytes"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// blockchainServiceImpl is the concrete implementation of the BlockchainService interface.
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

// ReserveDetailsOutput matches the structure returned by the getReserveDetails function in the smart contract.
// Fields must be exported (start with a capital letter) for the ABI unpacker.
// Added abi tags for explicit field mapping.
type ReserveDetailsOutput struct {
	IsConfigured bool     `abi:"isConfigured" json:"isConfigured"`
	Name         string   `abi:"name" json:"name"`
	Symbol       string   `abi:"symbol" json:"symbol"`
	Balance      *big.Int `abi:"balance" json:"balance"`
	LastVerified *big.Int `abi:"lastVerified" json:"lastVerified"`
}

// NewBlockchainService creates a new instance of BlockchainService (the interface).
func NewBlockchainService(rpcURL string, contractAddress string, abiFilePath string, privateKeyHex string) (BlockchainService, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum client: %v", err)
	}

	pk := privateKeyHex
	if strings.HasPrefix(pk, "0x") {
		pk = pk[2:]
	}
	_, err = ethcrypto.HexToECDSA(pk)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %v", err)
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
		client:           client,
		contractAddr:     common.HexToAddress(contractAddress),
		contractABI:      parsedABI,
		privateKeyString: privateKeyHex,
		defaultTimeout:   30 * time.Second,
	}, nil
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
	signatureBytes := hexutil.MustDecode(signature)

	data, err := s.contractABI.Pack("verifyProof", tokenAddr, walletAddr, signatureBytes)
	if err != nil {
		return false, fmt.Errorf("failed to pack parameters for verifyProof: %v", err)
	}

	tx := types.NewTransaction(auth.Nonce.Uint64(), s.contractAddr, auth.Value, auth.GasLimit, auth.GasPrice, data)

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKeyECDSA)
	if err != nil {
		return false, fmt.Errorf("failed to sign transaction: %v", err)
	}

	err = s.client.SendTransaction(ctx, signedTx)
	if err != nil {
		return false, fmt.Errorf("failed to send transaction: %v", err)
	}

	receipt, err := bind.WaitMined(ctx, s.client, signedTx)
	if err != nil {
		return false, fmt.Errorf("failed to mine transaction: %v", err)
	}

	if receipt.Status == types.ReceiptStatusFailed {
		callErr := checkTxFailureReason(s.client, fromAddress, signedTx, receipt.BlockNumber)
		log.Printf("On-chain transaction %s failed. Block: %s. Revert Reason: %v", signedTx.Hash().Hex(), receipt.BlockNumber.String(), callErr)
		return false, fmt.Errorf("transaction failed on-chain (receipt status 0)")
	}

	return true, nil
}

// Helper function to try and get a revert reason
func checkTxFailureReason(client *ethclient.Client, from common.Address, tx *types.Transaction, blockNumber *big.Int) error {
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

// IsReserveWallet is a method of blockchainServiceImpl.
func (s *blockchainServiceImpl) IsReserveWallet(token, wallet string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.defaultTimeout)
	defer cancel()

	data, err := s.contractABI.Pack("isReserveWallet",
		common.HexToAddress(token),
		common.HexToAddress(wallet),
	)
	if err != nil {
		return false, fmt.Errorf("failed to pack data for isReserveWallet: %v", err)
	}

	msg := ethereum.CallMsg{
		To:   &s.contractAddr,
		Data: data,
	}

	output, err := s.client.CallContract(ctx, msg, nil)
	if err != nil {
		return false, fmt.Errorf("failed to call contract for isReserveWallet: %v", err)
	}

	var result bool
	if err := s.contractABI.UnpackIntoInterface(&result, "isReserveWallet", output); err != nil {
		return false, fmt.Errorf("failed to unpack isReserveWallet result: %v", err)
	}

	return result, nil
}

// GetReserveDetails is a method of blockchainServiceImpl.
func (s *blockchainServiceImpl) GetReserveDetails(token, wallet string) (*ReserveDetailsOutput, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.defaultTimeout)
	defer cancel()

	tokenAddr := common.HexToAddress(token)
	walletAddr := common.HexToAddress(wallet)

	data, err := s.contractABI.Pack("getReserveDetails", tokenAddr, walletAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to pack data for getReserveDetails: %v", err)
	}

	msg := ethereum.CallMsg{
		To:   &s.contractAddr,
		Data: data,
	}

	output, err := s.client.CallContract(ctx, msg, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call contract for getReserveDetails: %v", err)
	}

	var method abi.Method
	var ok bool
	method, ok = s.contractABI.Methods["getReserveDetails"]
	if !ok {
		return nil, fmt.Errorf("method getReserveDetails not found in ABI")
	}

	if len(method.Outputs) == 0 {
		return nil, fmt.Errorf("no outputs defined for getReserveDetails in ABI")
	}

	var reserveDetails ReserveDetailsOutput
	err = s.contractABI.UnpackIntoInterface(&reserveDetails, "getReserveDetails", output)
	if err == nil {
		return &reserveDetails, nil
	}

	var resultUnpacked []interface{}
	var unpackValuesErr error
	resultUnpacked, unpackValuesErr = method.Outputs.UnpackValues(output)
	if unpackValuesErr != nil {
		log.Printf("ERROR: UnpackIntoInterface failed (%v) AND UnpackValues also failed for getReserveDetails: %v. Raw output: %s", err, unpackValuesErr, hexutil.Encode(output))
		return nil, fmt.Errorf("failed to unpack getReserveDetails (tried two methods): primary error: %v, fallback error: %v", err, unpackValuesErr)
	}

	if len(resultUnpacked) == 0 {
		return nil, fmt.Errorf("UnpackValues returned an empty slice for getReserveDetails")
	}

	dataValue := resultUnpacked[0]

	var jsonData []byte
	var jsonErr error
	jsonData, jsonErr = json.Marshal(dataValue)
	if jsonErr != nil {
		return nil, fmt.Errorf("failed to marshal anonymous struct from UnpackValues to JSON for getReserveDetails: %v. Struct: %+v", jsonErr, dataValue)
	}

	var targetStruct ReserveDetailsOutput
	jsonErr = json.Unmarshal(jsonData, &targetStruct)
	if jsonErr != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON into ReserveDetailsOutput for getReserveDetails: %v. JSON: %s", jsonErr, string(jsonData))
	}

	if targetStruct.Balance == nil {
		targetStruct.Balance = big.NewInt(0)
	}
	if targetStruct.LastVerified == nil {
		targetStruct.LastVerified = big.NewInt(0)
	}

	return &targetStruct, nil
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
		return false, fmt.Errorf("payload mismatch")
	}

	// 2. Get the signer's address from the signature
	sig := common.FromHex(signature)
	if len(sig) != 65 {
		return false, fmt.Errorf("invalid signature length")
	}

	// Construct the message hash as done in the contract
	messageHash := ethcrypto.Keccak256(expectedPayload)

	// Construct the Ethereum signed message hash
	ethSignedMessageHash := ethcrypto.Keccak256(
		[]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(messageHash))),
		messageHash,
	)

	// Recover the signer's public key
	pubKey, err := ethcrypto.Ecrecover(ethSignedMessageHash, sig)
	if err != nil {
		return false, fmt.Errorf("failed to recover public key: %v", err)
	}

	// Convert public key to address
	recoveredAddr := common.BytesToAddress(ethcrypto.Keccak256(pubKey[1:])[12:])

	// Compare with the wallet address
	return recoveredAddr == walletAddr, nil
}
