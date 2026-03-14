package contract

import shared "github.com/mopeyjellyfish/hookr/internal/contractspec"

const (
	ABIVersionMajor = shared.ABIVersionMajor
	ABIVersionMinor = shared.ABIVersionMinor
	SchemaHashLen   = shared.SchemaHashLen
)

var (
	ErrIncompatibleABIMajor = shared.ErrIncompatibleABIMajor
	ErrIncompatibleABIMinor = shared.ErrIncompatibleABIMinor
	ErrSchemaHashMismatch   = shared.ErrSchemaHashMismatch
	ErrCapabilityMismatch   = shared.ErrCapabilityMismatch
)

type Handshake = shared.Handshake

func NewHandshake(schemaHash [SchemaHashLen]byte) Handshake {
	return shared.NewHandshake(schemaHash)
}

func ParseSchemaHashHex(s string) ([SchemaHashLen]byte, error) {
	return shared.ParseSchemaHashHex(s)
}

func ValidateHandshake(host, plugin Handshake) error {
	return shared.ValidateHandshake(host, plugin)
}
