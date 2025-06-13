package api

// SubmitSignatureRequest defines the expected JSON body for signature submission
type SubmitSignatureRequest struct {
	Token      string `json:"token" binding:"required"`
	Wallet     string `json:"wallet" binding:"required"`
	Signature  string `json:"signature" binding:"required"`
	ValidUntil uint64 `json:"validUntil" binding:"required"`
}

// SignatureResponse defines the response structure for signature endpoints
type SignatureResponse struct {
	Token       string `json:"token"`
	Wallet      string `json:"wallet"`
	Signature   string `json:"signature"`
	ValidUntil  uint64 `json:"validUntil"`
	IsValid     bool   `json:"isValid"`
	GeneratedAt string `json:"generatedAt,omitempty"`
}

// VerificationResponse defines the response structure for verification endpoints
type VerificationResponse struct {
	Success    bool   `json:"success"`
	Token      string `json:"token"`
	Wallet     string `json:"wallet"`
	Signature  string `json:"signature"`
	ValidUntil uint64 `json:"validUntil"`
}
