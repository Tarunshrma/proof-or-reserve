package types

import (
	"github.com/Tarunshrma/proof-or-reserve/internal/storage"
)

// SignatureService defines the interface for signature operations
type SignatureService interface {
	StoreSignature(token, wallet, signature string, validUntil uint64) error
	GetSignature(token, wallet string) (*storage.SignatureRecord, error)
	IsSignatureValid(token, wallet string) (bool, uint64, error)
	SaveSignature(record *storage.SignatureRecord) error
}
