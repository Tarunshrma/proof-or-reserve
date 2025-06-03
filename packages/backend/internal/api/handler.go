package api

import (
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
		// Generate new signature if none exists
		if err := h.sigService.GenerateSignature(token, wallet); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate signature: " + err.Error()})
			return
		}
		record, _ = h.sigService.GetSignature(token, wallet)
	}

	c.JSON(http.StatusOK, record)
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

// InitiateOnchainVerification handles generating a signature and calling verifyProof on the smart contract.
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

	// 2. Generate and store the signature. The signature is generated by the backend's private key.
	if err := h.sigService.GenerateSignature(req.Token, req.Wallet); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate signature: " + err.Error()})
		return
	}

	// 3. Retrieve the generated signature
	sigRecord, err := h.sigService.GetSignature(req.Token, req.Wallet)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve generated signature: " + err.Error()})
		return
	}
	if sigRecord == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Signature record not found after generation"})
		return
	}

	// 4. Call VerifySignature on the blockchain with the generated signature
	onChainSuccess, err := h.blockchain.VerifySignature(req.Token, req.Wallet, sigRecord.Signature)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "On-chain verification failed: " + err.Error()})
		return
	}

	// 5. Optionally, retrieve updated details after verification to include in response
	// For simplicity, we'll just return the verification status and signature.
	// The frontend can re-fetch details if needed.
	c.JSON(http.StatusOK, gin.H{
		"token":                 req.Token,
		"wallet":                req.Wallet,
		"verificationInitiated": true,
		"onChainSuccess":        onChainSuccess,
		"signatureUsed":         sigRecord.Signature,
		"message":               "On-chain verification attempted.",
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
