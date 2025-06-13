package types

// SubmitSignatureRequest represents the request body for signature submission
type SubmitSignatureRequest struct {
	Token      string `json:"token" binding:"required"`
	Wallet     string `json:"wallet" binding:"required"`
	Signature  string `json:"signature" binding:"required"`
	ValidUntil uint64 `json:"validUntil" binding:"required"`
}

// VerificationResponse represents the response for signature verification
type VerificationResponse struct {
	Success    bool   `json:"success"`
	Token      string `json:"token"`
	Wallet     string `json:"wallet"`
	Signature  string `json:"signature"`
	ValidUntil uint64 `json:"validUntil"`
}
