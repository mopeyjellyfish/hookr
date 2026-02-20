package contract

import (
	"errors"
	"fmt"
)

var (
	ErrContractNameEmpty    = errors.New("contract name cannot be empty")
	ErrContractHashEmpty    = errors.New("contract schema hash cannot be zero")
	ErrMethodIDDuplicate    = errors.New("method id is duplicated")
	ErrMethodNameDuplicate  = errors.New("method name is duplicated")
	ErrMethodNameEmpty      = errors.New("method name cannot be empty")
	ErrMethodRequestMissing = errors.New("method request type cannot be empty")
	ErrMethodReplyMissing   = errors.New("method response type cannot be empty")
)

type MethodID uint32

// Method describes one callable operation in a generated contract.
type Method struct {
	ID           MethodID
	Name         string
	RequestType  string
	ResponseType string
}

// Schema is generated from a user-defined contract file (for example, a .capnp).
type Schema struct {
	Name       string
	SchemaHash [SchemaHashLen]byte
	// Capabilities is a schema-defined capability bitmask used during handshake checks.
	Capabilities uint64
	Methods      []Method
}

// Validate ensures generated metadata is internally consistent before wiring calls.
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

// HasMethodID reports whether this schema contains the given method ID.
func (s Schema) HasMethodID(id MethodID) bool {
	for _, m := range s.Methods {
		if m.ID == id {
			return true
		}
	}
	return false
}
