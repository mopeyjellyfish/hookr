package contractspec

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHandshakeValidation(t *testing.T) {
	hash := [SchemaHashLen]byte{1, 2, 3}
	host := NewHandshake(hash)
	plugin := NewHandshake(hash)
	require.NoError(t, ValidateHandshake(host, plugin))

	plugin.ABIMajor++
	require.ErrorIs(t, ValidateHandshake(host, plugin), ErrIncompatibleABIMajor)

	plugin = NewHandshake(hash)
	plugin.ABIMinor++
	require.ErrorIs(t, ValidateHandshake(host, plugin), ErrIncompatibleABIMinor)

	plugin = NewHandshake(hash)
	plugin.SchemaHash[0] ^= 0xFF
	require.ErrorIs(t, ValidateHandshake(host, plugin), ErrSchemaHashMismatch)

	plugin = NewHandshake(hash)
	host.Capabilities = 1 << 2
	require.ErrorIs(t, ValidateHandshake(host, plugin), ErrCapabilityMismatch)
}

func TestParseSchemaHashHex(t *testing.T) {
	hashHex := "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	hash, err := ParseSchemaHashHex(hashHex)
	require.NoError(t, err)
	require.Equal(t, hashHex, NewHandshake(hash).SchemaHashHex())
}

func TestSchemaValidationAndLookup(t *testing.T) {
	schema := Schema{
		Name:       "example",
		SchemaHash: [SchemaHashLen]byte{1},
		Methods: []Method{
			{ID: 10, Name: "One", RequestType: "Req", ResponseType: "Resp"},
		},
	}
	require.NoError(t, schema.Validate())
	require.True(t, schema.HasMethodID(10))
	require.False(t, schema.HasMethodID(11))

	schema.Methods = append(schema.Methods, Method{ID: 10, Name: "Two", RequestType: "Req", ResponseType: "Resp"})
	require.ErrorIs(t, schema.Validate(), ErrMethodIDDuplicate)
}
