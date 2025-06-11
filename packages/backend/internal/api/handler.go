package api

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/Tarunshrma/proof-or-reserve/internal/config"
	"github.com/Tarunshrma/proof-or-reserve/internal/service"
	"github.com/Tarunshrma/proof-or-reserve/internal/storage"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	config         *config.Config
	store          *storage.JSONStorage
	signatureStore *storage.SignatureStorage
	sigService     service.SignatureService
	blockchain     service.BlockchainService
}

func NewHandler(cfg *config.Config, store *storage.JSONStorage, sigStore *storage.SignatureStorage, blockchainSvc service.BlockchainService, sigSvc service.SignatureService) *Handler {
	return &Handler{
		config:         cfg,
		store:          store,
		signatureStore: sigStore,
		sigService:     sigSvc,
		blockchain:     blockchainSvc,
	}
}

// GetSignature returns the stored signature for a token/wallet pair
func (h *Handler) GetSignature(c *gin.Context) {
	token := c.Param("token")
	wallet := c.Param("wallet")

	if !common.IsHexAddress(token) || !common.IsHexAddress(wallet) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token or wallet address"})
		return
	}

	// First check if this is a valid reserve wallet
	isReserve, err := h.blockchain.IsReserveWallet(token, wallet)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check reserve status: " + err.Error()})
		return
	}

	if !isReserve {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Not a configured reserve wallet"})
		return
	}

	record, err := h.sigService.GetSignature(token, wallet)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if record == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No signature found for this token/wallet pair"})
		return
	}

	// Check if signature is still valid
	isValid, validUntil, err := h.sigService.IsSignatureValid(token, wallet)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check signature validity: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":       token,
		"wallet":      wallet,
		"signature":   record.Signature,
		"validUntil":  validUntil,
		"isValid":     isValid,
		"generatedAt": record.GeneratedAt,
	})
}

// VerifySignature is the old endpoint. We are creating a new one /initiate-onchain-verification
// that handles signature generation internally before calling the contract.
// This old endpoint expected signature in request. For now, let's comment it out or decide if it's still needed.
/*
func (h *Handler) VerifySignature(c *gin.Context) {
	var req struct {
		Token     string `json:"token" binding:"required"`
		Wallet    string `json:"wallet" binding:"required"`
		Signature string `json:"signature" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !common.IsHexAddress(req.Token) || !common.IsHexAddress(req.Wallet) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token or wallet address"})
		return
	}

	// Verify proof on-chain
	success, err := h.blockchain.VerifySignature(req.Token, req.Wallet, req.Signature)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify proof: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":     req.Token,
		"wallet":    req.Wallet,
		"isValid":   success,
		"signature": req.Signature,
	})
}
*/

// GetReserveDetails handles fetching comprehensive reserve details from the blockchain.
func (h *Handler) GetReserveDetails(c *gin.Context) {
	token := c.Param("token")
	wallet := c.Param("wallet")

	if !common.IsHexAddress(token) || !common.IsHexAddress(wallet) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token or wallet address"})
		return
	}

	details, err := h.blockchain.GetReserveDetails(token, wallet)
	if err != nil {
		// Check for specific error types if needed, e.g., contract not found vs. other errors
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get reserve details: " + err.Error()})
		return
	}

	// Convert big.Int to string for JSON response to ensure readability and compatibility
	c.JSON(http.StatusOK, gin.H{
		"tokenAddress":  token,
		"walletAddress": wallet,
		"isConfigured":  details.IsConfigured,
		"name":          details.Name,
		"symbol":        details.Symbol,
		"balance":       details.Balance.String(),      // Convert big.Int to string
		"lastVerified":  details.LastVerified.String(), // Convert big.Int to string
	})
}

// InitiateOnchainVerificationRequest defines the expected JSON body for the verification request.
type InitiateOnchainVerificationRequest struct {
	Token  string `json:"token" binding:"required"`
	Wallet string `json:"wallet" binding:"required"`
}

// InitiateOnchainVerification handles verifying a stored signature on-chain
func (h *Handler) InitiateOnchainVerification(c *gin.Context) {
	var req InitiateOnchainVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	if !common.IsHexAddress(req.Token) || !common.IsHexAddress(req.Wallet) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token or wallet address"})
		return
	}

	// 1. Ensure the wallet is a configured reserve wallet before proceeding
	isReserve, err := h.blockchain.IsReserveWallet(req.Token, req.Wallet)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check reserve status: " + err.Error()})
		return
	}
	if !isReserve {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Wallet is not a configured reserve for the token"})
		return
	}

	// 2. Get the stored signature
	sigRecord, err := h.sigService.GetSignature(req.Token, req.Wallet)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve signature: " + err.Error()})
		return
	}
	if sigRecord == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No signature found. Please submit a signature first."})
		return
	}

	// 3. Check if signature is still valid
	isValid, validUntil, err := h.sigService.IsSignatureValid(req.Token, req.Wallet)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check signature validity: " + err.Error()})
		return
	}
	if !isValid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Signature has expired. Please submit a new signature."})
		return
	}

	// 4. Call VerifySignature on the blockchain with the stored signature
	onChainSuccess, err := h.blockchain.VerifySignature(req.Token, req.Wallet, sigRecord.Signature)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "On-chain verification failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    onChainSuccess,
		"token":      req.Token,
		"wallet":     req.Wallet,
		"signature":  sigRecord.Signature,
		"validUntil": validUntil,
	})
}

// GetReserveBalance returns the current balance of a reserve wallet
// This is now covered by GetReserveDetails, but keeping it if other parts of the system use it.
func (h *Handler) GetReserveBalance(c *gin.Context) {
	token := c.Param("token")
	wallet := c.Param("wallet")

	if !common.IsHexAddress(token) || !common.IsHexAddress(wallet) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token or wallet address"})
		return
	}

	// Check if this is a valid reserve wallet
	isReserve, err := h.blockchain.IsReserveWallet(token, wallet)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check reserve status: " + err.Error()})
		return
	}

	if !isReserve {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Not a configured reserve wallet"})
		return
	}

	// Get balance from blockchain
	balance, err := h.blockchain.GetReserveBalance(token, wallet)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get balance: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":   token,
		"wallet":  wallet,
		"balance": balance.String(),
	})
}

// GetLastVerified is removed as blockchain service method was removed.
// It's now part of GetReserveDetails.
/*
func (h *Handler) GetLastVerified(c *gin.Context) {
	token := c.Param("token")
	wallet := c.Param("wallet")

	if !common.IsHexAddress(token) || !common.IsHexAddress(wallet) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token or wallet address"})
		return
	}

	// Check if this is a valid reserve wallet
	isReserve, err := h.blockchain.IsReserveWallet(token, wallet)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check reserve status: " + err.Error()})
		return
	}

	if !isReserve {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Not a configured reserve wallet"})
		return
	}

	// Get last verified timestamp from blockchain
	lastVerified, err := h.blockchain.GetLastVerified(token, wallet)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get last verified timestamp: " + err.Error()})
		return
	}

	// Get stored signature for additional info
	record, _ := h.sigService.GetSignature(token, wallet)
	var generatedAt interface{} = nil
	if record != nil {
		generatedAt = record.GeneratedAt
	}

	c.JSON(http.StatusOK, gin.H{
		"token":        token,
		"wallet":       wallet,
		"lastVerified": lastVerified,
		"generatedAt":  generatedAt,
	})
}
*/

// AssetConfigJSON defines the structure for asset configurations read from the backend JSON file.
// This matches the structure that was previously in the frontend's assets.ts
// and the new reserves_config.json file.
type AssetConfigJSON struct {
	ID            string `json:"id"`
	DisplayName   string `json:"displayName"`
	TokenAddress  string `json:"tokenAddress"`
	WalletAddress string `json:"walletAddress"`
	LogoURL       string `json:"logoUrl,omitempty"`
}

// GetConfiguredAssets serves the list of statically configured assets from a JSON file.
func (h *Handler) GetConfiguredAssets(c *gin.Context) {
	filePath := h.config.AssetsConfigPath

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		// Log the error for backend visibility
		log.Printf("ERROR: Failed to read assets configuration from %s: %v", filePath, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read assets configuration", "details": err.Error()}) // It can be useful to pass err.Error() in details for admins/devs
		return
	}

	var assets []AssetConfigJSON
	err = json.Unmarshal(data, &assets)
	if err != nil {
		// Log the error for backend visibility
		log.Printf("ERROR: Failed to parse assets configuration from %s: %v", filePath, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse assets configuration", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, assets)
}

// GetContractConfig returns the contract address and chain ID
func (h *Handler) GetContractConfig(c *gin.Context) {
	chainID, err := h.blockchain.GetChainID()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get chain ID"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"address": h.blockchain.GetContractAddress(),
		"chainId": chainID,
	})
}

// SubmitSignature handles signature submission from the frontend
type SubmitSignatureRequest struct {
	Token      string `json:"token" binding:"required"`
	Wallet     string `json:"wallet" binding:"required"`
	Signature  string `json:"signature" binding:"required"`
	Payload    string `json:"payload" binding:"required"`
	ValidUntil uint64 `json:"validUntil" binding:"required"`
}

func (h *Handler) SubmitSignature(c *gin.Context) {
	var req SubmitSignatureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	// Log signature details
	log.Printf("Received signature: %s\n", req.Signature)
	log.Printf("Signature length: %d chars\n", len(req.Signature))
	log.Printf("Signature bytes length: %d bytes\n", len(common.FromHex(req.Signature)))

	if !common.IsHexAddress(req.Token) || !common.IsHexAddress(req.Wallet) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token or wallet address"})
		return
	}

	// Verify that this is a valid reserve wallet
	isReserve, err := h.blockchain.IsReserveWallet(req.Token, req.Wallet)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check reserve status: " + err.Error()})
		return
	}
	if !isReserve {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Not a configured reserve wallet"})
		return
	}

	// Verify the signature is valid before storing
	isValid, err := h.blockchain.VerifySignatureOffchain(req.Token, req.Wallet, req.Signature, req.Payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify signature: " + err.Error()})
		return
	}
	if !isValid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid signature"})
		return
	}

	// Store the signature
	if err := h.sigService.StoreSignature(req.Token, req.Wallet, req.Signature, req.ValidUntil); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store signature: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Signature stored successfully",
		"token":      req.Token,
		"wallet":     req.Wallet,
		"validUntil": req.ValidUntil,
	})
}
