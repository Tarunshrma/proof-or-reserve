package api

import (
	"math/big"
	"net/http"

	"github.com/Tarunshrma/proof-or-reserve/internal/config"
	"github.com/Tarunshrma/proof-or-reserve/internal/storage"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	cfg            *config.Config
	store          *storage.JSONStorage
	signatureStore *storage.SignatureStorage
	ethClient      *ethclient.Client
}

type VerifySignatureRequest struct {
	Token  string `json:"token" binding:"required"`
	Wallet string `json:"wallet" binding:"required"`
}

func NewHandler(cfg *config.Config, store *storage.JSONStorage, signatureStore *storage.SignatureStorage) *Handler {
	client, err := ethclient.Dial(cfg.EthereumRPC)
	if err != nil {
		panic(err)
	}

	return &Handler{
		cfg:            cfg,
		store:          store,
		signatureStore: signatureStore,
		ethClient:      client,
	}
}

func (h *Handler) GetReserveBalance(c *gin.Context) {
	token := c.Param("token")
	wallet := c.Param("wallet")

	// Validate addresses
	if !common.IsHexAddress(token) || !common.IsHexAddress(wallet) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid address format"})
		return
	}

	// Get balance from contract
	balance, err := h.getBalance(token, wallet)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":   token,
		"wallet":  wallet,
		"balance": balance.String(),
	})
}

func (h *Handler) GetSignature(c *gin.Context) {
	token := c.Param("token")
	wallet := c.Param("wallet")

	// Validate addresses
	if !common.IsHexAddress(token) || !common.IsHexAddress(wallet) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid address format"})
		return
	}

	record, err := h.signatureStore.Get(token, wallet)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Signature not found"})
		return
	}

	c.JSON(http.StatusOK, record)
}

func (h *Handler) VerifySignature(c *gin.Context) {
	var req VerifySignatureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate addresses
	if !common.IsHexAddress(req.Token) || !common.IsHexAddress(req.Wallet) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid address format"})
		return
	}

	// Get stored signature
	record, err := h.signatureStore.Get(req.Token, req.Wallet)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Signature not found"})
		return
	}

	// Store verification record
	verificationRecord := &storage.VerificationRecord{
		Token:     req.Token,
		Wallet:    req.Wallet,
		Signature: record.Signature,
		Timestamp: record.GeneratedAt,
	}

	if err := h.store.StoreVerification(verificationRecord); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store verification"})
		return
	}

	c.JSON(http.StatusOK, record)
}

func (h *Handler) GetLastVerified(c *gin.Context) {
	token := c.Param("token")
	wallet := c.Param("wallet")

	// Validate addresses
	if !common.IsHexAddress(token) || !common.IsHexAddress(wallet) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid address format"})
		return
	}

	record, err := h.store.GetLastVerification(token, wallet)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if record == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No verification record found"})
		return
	}

	c.JSON(http.StatusOK, record)
}

// Helper functions to interact with the smart contract
func (h *Handler) getBalance(token, wallet string) (*big.Int, error) {
	// Implement balance checking using the contract
	return big.NewInt(0), nil
}

func (h *Handler) verifySignature(token, wallet, balance, signature string) error {
	// Implement signature verification using the contract
	return nil
}
