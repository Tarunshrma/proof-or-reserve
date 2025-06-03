package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
)

// blockchainServiceImpl is the concrete implementation of the BlockchainService interface.
type blockchainServiceImpl struct {
	client         *ethclient.Client
	contractAddr   common.Address
	contractABI    abi.ABI
	defaultTimeout time.Duration
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
func NewBlockchainService(rpcURL string, contractAddress string, abiFilePath string) (BlockchainService, error) {
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

// VerifySignature is a method of blockchainServiceImpl.
func (s *blockchainServiceImpl) VerifySignature(token, wallet, signature string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.defaultTimeout)
	defer cancel()

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

	isActive, err := s.IsReserveWallet(token, wallet)
	if err != nil {
		return false, fmt.Errorf("failed to check reserve status: %v", err)
	}
	fmt.Printf("Is Reserve Active: %v\n", isActive)

	if !isActive {
		return false, fmt.Errorf("reserve not active")
	}

	tokenAddr := common.HexToAddress(token)
	walletAddr := common.HexToAddress(wallet)
	signatureBytes := hexutil.MustDecode(signature)

	data, err := s.contractABI.Pack("verifyProof", tokenAddr, walletAddr, signatureBytes)
	if err != nil {
		return false, fmt.Errorf("failed to pack parameters: %v", err)
	}

	fmt.Printf("\nContract Call Data:\n")
	fmt.Printf("Method: verifyProof\n")
	fmt.Printf("Packed Data: %s\n", hexutil.Encode(data))

	callResult, err := s.client.CallContract(ctx, ethereum.CallMsg{
		To:   &s.contractAddr,
		Data: data,
	}, nil)
	if err != nil {
		return false, fmt.Errorf("contract call failed: %v", err)
	}

	fmt.Printf("Contract Call Result: %s\n", hexutil.Encode(callResult))

	var success bool
	if err := s.contractABI.UnpackIntoInterface(&success, "verifyProof", callResult); err != nil {
		fmt.Printf("Error unpacking verifyProof result: %v. Raw result: %s\n", err, hexutil.Encode(callResult))
		return false, fmt.Errorf("failed to unpack verifyProof result: %v", err)
	}

	fmt.Printf("Verification Result: %v\n", success)
	fmt.Printf("=== End Debug ===\n\n")

	return success, nil
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
		fmt.Println("DIAGNOSTIC: Successfully unpacked with contractABI.UnpackIntoInterface into struct")
		return &reserveDetails, nil
	} else {
		fmt.Printf("DIAGNOSTIC: contractABI.UnpackIntoInterface into struct failed: %v. Raw: %s\n", err, hexutil.Encode(output))
	}

	fmt.Println("DIAGNOSTIC: Falling back to method.Outputs.UnpackValues")
	var resultUnpacked []interface{}
	var unpackValuesErr error
	resultUnpacked, unpackValuesErr = method.Outputs.UnpackValues(output)
	if unpackValuesErr != nil {
		fmt.Printf("DIAGNOSTIC: method.Outputs.UnpackValues failed: %v. Raw: %s\n", unpackValuesErr, hexutil.Encode(output))
		return nil, fmt.Errorf("failed to unpack getReserveDetails using UnpackValues (after UnpackIntoInterface also failed): %v", unpackValuesErr)
	}

	fmt.Printf("DIAGNOSTIC: method.Outputs.UnpackValues successful. Result: %+v\n", resultUnpacked)

	if len(resultUnpacked) == 0 {
		return nil, fmt.Errorf("UnpackValues returned an empty slice, expected at least one element (the struct)")
	}

	dataValue := resultUnpacked[0]
	fmt.Printf("DIAGNOSTIC: Concrete type of resultUnpacked[0] is: %T\n", dataValue)

	if _, ok := dataValue.(ReserveDetailsOutput); ok {
		fmt.Println("DIAGNOSTIC: Successfully type-asserted resultUnpacked[0] to ReserveDetailsOutput (unexpected)")
		castedStruct := dataValue.(ReserveDetailsOutput)
		if castedStruct.Balance == nil {
			castedStruct.Balance = big.NewInt(0)
		}
		if castedStruct.LastVerified == nil {
			castedStruct.LastVerified = big.NewInt(0)
		}
		return &castedStruct, nil
	}

	fmt.Println("DIAGNOSTIC: Attempting conversion of anonymous struct to ReserveDetailsOutput via JSON.")
	var jsonData []byte
	var jsonErr error
	jsonData, jsonErr = json.Marshal(dataValue)
	if jsonErr != nil {
		return nil, fmt.Errorf("failed to marshal anonymous struct from UnpackValues to JSON: %v. Struct: %+v", jsonErr, dataValue)
	}

	fmt.Printf("DIAGNOSTIC: Successfully marshalled anonymous struct to JSON: %s\n", string(jsonData))

	var targetStruct ReserveDetailsOutput
	jsonErr = json.Unmarshal(jsonData, &targetStruct)
	if jsonErr != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON into ReserveDetailsOutput: %v. JSON: %s", jsonErr, string(jsonData))
	}

	if targetStruct.Balance == nil {
		targetStruct.Balance = big.NewInt(0)
	}
	if targetStruct.LastVerified == nil {
		targetStruct.LastVerified = big.NewInt(0)
	}

	fmt.Println("DIAGNOSTIC: Successfully converted anonymous struct to ReserveDetailsOutput via JSON.")
	return &targetStruct, nil
}
