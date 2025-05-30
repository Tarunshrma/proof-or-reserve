package api

import (
	"encoding/hex"
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
	sigService     *service.SignatureService
	blockchain     *service.BlockchainService
}

func NewHandler(cfg *config.Config, store *storage.JSONStorage, sigStore *storage.SignatureStorage) *Handler {
	sigService, err := service.NewSignatureService(cfg.PrivateKey, sigStore)
	if err != nil {
		panic(err)
	}

	blockchain, err := service.NewBlockchainService(cfg.EthereumRPC, cfg.ContractAddress)
	if err != nil {
		panic(err)
	}

	return &Handler{
		config:         cfg,
		store:          store,
		signatureStore: sigStore,
		sigService:     sigService,
		blockchain:     blockchain,
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

// VerifySignature verifies the stored signature against the smart contract
func (h *Handler) VerifySignature(c *gin.Context) {
	var req struct {
		Token  string `json:"token" binding:"required"`
		Wallet string `json:"wallet" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !common.IsHexAddress(req.Token) || !common.IsHexAddress(req.Wallet) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token or wallet address"})
		return
	}

	// Get stored signature
	record, err := h.sigService.GetSignature(req.Token, req.Wallet)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if record == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Signature not found"})
		return
	}

	// Convert hex signature to bytes
	sigBytes, err := hex.DecodeString(record.Signature[2:]) // Remove "0x" prefix
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid signature format"})
		return
	}

	// Verify proof on-chain
	success, err := h.blockchain.VerifyProof(req.Token, req.Wallet, sigBytes, record.ValidUntil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify proof: " + err.Error()})
		return
	}

	// Get on-chain validity status
	isValid, validUntil, err := h.blockchain.IsProofValid(req.Token, req.Wallet)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check proof validity: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      req.Token,
		"wallet":     req.Wallet,
		"isValid":    isValid && success,
		"validUntil": validUntil,
		"signature":  record.Signature,
	})
}

// GetReserveBalance returns the current balance of a reserve wallet
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

// GetLastVerified returns the last verification timestamp from the blockchain
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
