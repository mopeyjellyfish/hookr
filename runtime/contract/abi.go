package contract

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
)

const (
	// ABIVersionMajor identifies the current incompatible ABI line.
	ABIVersionMajor uint16 = 2
	// ABIVersionMinor identifies backward-compatible additions within ABI v2.
	ABIVersionMinor uint16 = 0
	// SchemaHashLen is the expected byte length of a contract schema hash.
	SchemaHashLen = 32
)

var (
	ErrIncompatibleABIMajor = errors.New("incompatible ABI major version")
	ErrSchemaHashMismatch   = errors.New("schema hash mismatch")
	ErrCapabilityMismatch   = errors.New("required capability not provided")
)

// Handshake is exchanged by host and plugin before contract calls.
type Handshake struct {
	ABIMajor   uint16
	ABIMinor   uint16
	SchemaHash [SchemaHashLen]byte
	// Capabilities is a bitmask of optional ABI v2 features supported/required.
	// Host-side handshakes treat this value as a required bitmask.
	Capabilities uint64
}

// NewHandshake creates a handshake for the current runtime ABI constants.
func NewHandshake(schemaHash [SchemaHashLen]byte) Handshake {
	return Handshake{
		ABIMajor:   ABIVersionMajor,
		ABIMinor:   ABIVersionMinor,
		SchemaHash: schemaHash,
	}
}

// SchemaHashHex returns a lowercase hex string for diagnostics and transport.
func (h Handshake) SchemaHashHex() string {
	return hex.EncodeToString(h.SchemaHash[:])
}

// ParseSchemaHashHex parses a schema hash from lowercase/uppercase hex text.
func ParseSchemaHashHex(s string) ([SchemaHashLen]byte, error) {
	var out [SchemaHashLen]byte
	raw, err := hex.DecodeString(s)
	if err != nil {
		return out, fmt.Errorf("decode schema hash: %w", err)
	}
	if len(raw) != SchemaHashLen {
		return out, fmt.Errorf("invalid schema hash length: got %d want %d", len(raw), SchemaHashLen)
	}
	copy(out[:], raw)
	return out, nil
}

// ValidateHandshake checks host/plugin compatibility rules for ABI v2.
func ValidateHandshake(host, plugin Handshake) error {
	if host.ABIMajor != plugin.ABIMajor {
		return fmt.Errorf(
			"%w: host=%d plugin=%d",
			ErrIncompatibleABIMajor,
			host.ABIMajor,
			plugin.ABIMajor,
		)
	}
	if !bytes.Equal(host.SchemaHash[:], plugin.SchemaHash[:]) {
		return fmt.Errorf("%w: host=%s plugin=%s", ErrSchemaHashMismatch, host.SchemaHashHex(), plugin.SchemaHashHex())
	}
	missing := host.Capabilities &^ plugin.Capabilities
	if missing != 0 {
		return fmt.Errorf("%w: missing_bits=0x%x", ErrCapabilityMismatch, missing)
	}
	return nil
}
