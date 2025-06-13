package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Tarunshrma/proof-or-reserve/internal/config"
	"github.com/Tarunshrma/proof-or-reserve/internal/storage"
	"github.com/Tarunshrma/proof-or-reserve/internal/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mock BlockchainService ---

type MockBlockchainService struct {
	mock.Mock
}

func (m *MockBlockchainService) GetReserveDetails(token, wallet string) (*types.ReserveDetailsOutput, error) {
	args := m.Called(token, wallet)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.ReserveDetailsOutput), args.Error(1)
}

func (m *MockBlockchainService) IsReserveWallet(token, wallet string) (bool, error) {
	args := m.Called(token, wallet)
	return args.Bool(0), args.Error(1)
}

func (m *MockBlockchainService) VerifyProof(token, wallet string, validUntil uint64, signature string) (bool, error) {
	args := m.Called(token, wallet, validUntil, signature)
	return args.Bool(0), args.Error(1)
}

func (m *MockBlockchainService) GetReserveBalance(token, wallet string) (*big.Int, error) {
	args := m.Called(token, wallet)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*big.Int), args.Error(1)
}

func (m *MockBlockchainService) GetChainID() (*big.Int, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*big.Int), args.Error(1)
}

func (m *MockBlockchainService) GetContractAddress() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockBlockchainService) SetPrivateKey(privateKeyHex string) error {
	args := m.Called(privateKeyHex)
	return args.Error(0)
}

// --- Mock SignatureService ---

type MockSignatureService struct {
	mock.Mock
}

func (m *MockSignatureService) StoreSignature(token, wallet, signature string, validUntil uint64) error {
	args := m.Called(token, wallet, signature, validUntil)
	return args.Error(0)
}

func (m *MockSignatureService) GetSignature(token, wallet string) (*storage.SignatureRecord, error) {
	args := m.Called(token, wallet)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.SignatureRecord), args.Error(1)
}

func (m *MockSignatureService) IsSignatureValid(token, wallet string) (bool, uint64, error) {
	args := m.Called(token, wallet)
	return args.Bool(0), args.Get(1).(uint64), args.Error(2)
}

// --- Test Setup ---

func setupTestRouter(mockBlockchain *MockBlockchainService, mockSigService *MockSignatureService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	testHandler := &Handler{
		config:         &config.Config{},
		store:          nil, // Not needed for these tests
		signatureStore: nil, // Not needed for these tests
		sigService:     mockSigService,
		blockchain:     mockBlockchain,
	}

	router.GET("/reserve-details/:token/:wallet", testHandler.GetReserveDetails)
	router.POST("/submit-signature", testHandler.SubmitSignature)

	return router
}

// --- Test Cases ---

func TestGetReserveDetails_Success(t *testing.T) {
	// Arrange
	mockBlockchain := new(MockBlockchainService)
	mockSigService := new(MockSignatureService)

	tokenAddr := "0x123"
	walletAddr := "0x456"
	expectedDetails := &types.ReserveDetailsOutput{
		IsConfigured: true,
		Name:         "Test Token",
		Symbol:       "TTT",
		Balance:      "1000",
		LastVerified: "1234567890",
	}

	// Setup expectations
	mockBlockchain.On("IsReserveWallet", tokenAddr, walletAddr).Return(true, nil)
	mockBlockchain.On("GetReserveDetails", tokenAddr, walletAddr).Return(expectedDetails, nil)

	router := setupTestRouter(mockBlockchain, mockSigService)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/reserve-details/%s/%s", tokenAddr, walletAddr), nil)
	rr := httptest.NewRecorder()

	// Act
	router.ServeHTTP(rr, req)

	// Assert
	assert.Equal(t, http.StatusOK, rr.Code)

	var response types.ReserveDetailsOutput
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, expectedDetails.IsConfigured, response.IsConfigured)
	assert.Equal(t, expectedDetails.Name, response.Name)
	assert.Equal(t, expectedDetails.Symbol, response.Symbol)
	assert.Equal(t, expectedDetails.Balance, response.Balance)
	assert.Equal(t, expectedDetails.LastVerified, response.LastVerified)

	mockBlockchain.AssertExpectations(t)
}

func TestGetReserveDetails_BlockchainError(t *testing.T) {
	// Arrange
	mockBlockchain := new(MockBlockchainService)
	mockSigService := new(MockSignatureService)

	tokenAddr := "0x123"
	walletAddr := "0x456"
	expectedError := errors.New("blockchain communication failed")

	// Setup expectations
	mockBlockchain.On("IsReserveWallet", tokenAddr, walletAddr).Return(true, nil)
	mockBlockchain.On("GetReserveDetails", tokenAddr, walletAddr).Return(nil, expectedError)

	router := setupTestRouter(mockBlockchain, mockSigService)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/reserve-details/%s/%s", tokenAddr, walletAddr), nil)
	rr := httptest.NewRecorder()

	// Act
	router.ServeHTTP(rr, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, rr.Code)

	var errorResponse map[string]string
	err := json.Unmarshal(rr.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Contains(t, errorResponse["error"], "Failed to get reserve details: "+expectedError.Error())

	mockBlockchain.AssertExpectations(t)
}

func TestGetReserveDetails_InvalidAddresses(t *testing.T) {
	mockBlockchain := new(MockBlockchainService)
	mockSigService := new(MockSignatureService)
	router := setupTestRouter(mockBlockchain, mockSigService)

	tests := []struct {
		name       string
		tokenAddr  string
		walletAddr string
	}{
		{"invalid token address", "0x123invalid", "0xValidWalletAddress00000000000000000000"},
		{"invalid wallet address", "0xValidTokenAddress00000000000000000000", "0x456invalid"},
		{"non-hex token address", "notAHex", "0xValidWalletAddress00000000000000000000"},
		{"non-hex wallet address", "0xValidTokenAddress00000000000000000000", "notAHexEither"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", fmt.Sprintf("/reserve-details/%s/%s", tt.tokenAddr, tt.walletAddr), nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusBadRequest, rr.Code)
			var errorResponse map[string]string
			err := json.Unmarshal(rr.Body.Bytes(), &errorResponse)
			assert.NoError(t, err)
			assert.Equal(t, "Invalid token or wallet address", errorResponse["error"])
		})
	}
}

// TODO: Add tests for InitiateOnchainVerification
// TODO: Add tests for GetReserveBalance
// TODO: Add tests for GetSignature
