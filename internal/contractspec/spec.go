package contractspec

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
)

const (
	ABIVersionMajor uint16 = 2
	ABIVersionMinor uint16 = 0
	SchemaHashLen          = 32
)

var (
	ErrIncompatibleABIMajor = errors.New("incompatible ABI major version")
	ErrIncompatibleABIMinor = errors.New("incompatible ABI minor version")
	ErrSchemaHashMismatch   = errors.New("schema hash mismatch")
	ErrCapabilityMismatch   = errors.New("required capability not provided")

	ErrContractNameEmpty    = errors.New("contract name cannot be empty")
	ErrContractHashEmpty    = errors.New("contract schema hash cannot be zero")
	ErrMethodIDDuplicate    = errors.New("method id is duplicated")
	ErrMethodNameDuplicate  = errors.New("method name is duplicated")
	ErrMethodNameEmpty      = errors.New("method name cannot be empty")
	ErrMethodRequestMissing = errors.New("method request type cannot be empty")
	ErrMethodReplyMissing   = errors.New("method response type cannot be empty")
)

type Handshake struct {
	ABIMajor     uint16
	ABIMinor     uint16
	SchemaHash   [SchemaHashLen]byte
	Capabilities uint64
}

func NewHandshake(schemaHash [SchemaHashLen]byte) Handshake {
	return Handshake{
		ABIMajor:   ABIVersionMajor,
		ABIMinor:   ABIVersionMinor,
		SchemaHash: schemaHash,
	}
}

func (h Handshake) SchemaHashHex() string {
	return hex.EncodeToString(h.SchemaHash[:])
}

func ParseSchemaHashHex(s string) ([SchemaHashLen]byte, error) {
	var out [SchemaHashLen]byte
	raw, err := hex.DecodeString(s)
	if err != nil {
		return out, fmt.Errorf("decode schema hash: %w", err)
	}
	if len(raw) != SchemaHashLen {
		return out, fmt.Errorf(
			"invalid schema hash length: got %d want %d",
			len(raw),
			SchemaHashLen,
		)
	}
	copy(out[:], raw)
	return out, nil
}

func ValidateHandshake(host, plugin Handshake) error {
	if host.ABIMajor != plugin.ABIMajor {
		return fmt.Errorf(
			"%w: host=%d plugin=%d",
			ErrIncompatibleABIMajor,
			host.ABIMajor,
			plugin.ABIMajor,
		)
	}
	if host.ABIMinor != plugin.ABIMinor {
		return fmt.Errorf(
			"%w: host=%d plugin=%d",
			ErrIncompatibleABIMinor,
			host.ABIMinor,
			plugin.ABIMinor,
		)
	}
	if !bytes.Equal(host.SchemaHash[:], plugin.SchemaHash[:]) {
		return fmt.Errorf(
			"%w: host=%s plugin=%s",
			ErrSchemaHashMismatch,
			host.SchemaHashHex(),
			plugin.SchemaHashHex(),
		)
	}
	missing := host.Capabilities &^ plugin.Capabilities
	if missing != 0 {
		return fmt.Errorf("%w: missing_bits=0x%x", ErrCapabilityMismatch, missing)
	}
	return nil
}

type MethodID uint32

type Method struct {
	ID           MethodID
	Name         string
	RequestType  string
	ResponseType string
	Optional     bool
}

type Schema struct {
	Name         string
	SchemaHash   [SchemaHashLen]byte
	Capabilities uint64
	Methods      []Method
}

func (s Schema) Validate() error {
	if s.Name == "" {
		return ErrContractNameEmpty
	}
	var zero [SchemaHashLen]byte
	if s.SchemaHash == zero {
		return ErrContractHashEmpty
	}
	seenIDs := make(map[MethodID]struct{}, len(s.Methods))
	seenNames := make(map[string]struct{}, len(s.Methods))
	for _, m := range s.Methods {
		if m.Name == "" {
			return ErrMethodNameEmpty
		}
		if m.RequestType == "" {
			return fmt.Errorf("%w (%s)", ErrMethodRequestMissing, m.Name)
		}
		if m.ResponseType == "" {
			return fmt.Errorf("%w (%s)", ErrMethodReplyMissing, m.Name)
		}
		if _, ok := seenIDs[m.ID]; ok {
			return fmt.Errorf("%w (%d)", ErrMethodIDDuplicate, m.ID)
		}
		seenIDs[m.ID] = struct{}{}
		if _, ok := seenNames[m.Name]; ok {
			return fmt.Errorf("%w (%s)", ErrMethodNameDuplicate, m.Name)
		}
		seenNames[m.Name] = struct{}{}
	}
	return nil
}

func (s Schema) HasMethodID(id MethodID) bool {
	for _, m := range s.Methods {
		if m.ID == id {
			return true
		}
	}
	return false
}
