package contract

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateHandshake(t *testing.T) {
	hash := [SchemaHashLen]byte{1, 2, 3}
	host := NewHandshake(hash)
	plugin := NewHandshake(hash)

	err := ValidateHandshake(host, plugin)
	require.NoError(t, err)

	plugin.ABIMajor = host.ABIMajor + 1
	err = ValidateHandshake(host, plugin)
	require.ErrorIs(t, err, ErrIncompatibleABIMajor)

	plugin = NewHandshake(hash)
	plugin.ABIMinor = host.ABIMinor + 1
	err = ValidateHandshake(host, plugin)
	require.ErrorIs(t, err, ErrIncompatibleABIMinor)

	plugin = NewHandshake(hash)
	plugin.SchemaHash[0] = 99
	err = ValidateHandshake(host, plugin)
	require.ErrorIs(t, err, ErrSchemaHashMismatch)

	host = NewHandshake(hash)
	host.Capabilities = 1 << 3
	plugin = NewHandshake(hash)
	err = ValidateHandshake(host, plugin)
	require.ErrorIs(t, err, ErrCapabilityMismatch)

	plugin.Capabilities = host.Capabilities
	err = ValidateHandshake(host, plugin)
	require.NoError(t, err)
}

func TestParseSchemaHashHex(t *testing.T) {
	hashHex := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	hash, err := ParseSchemaHashHex(hashHex)
	require.NoError(t, err)
	require.Equal(t, hashHex, NewHandshake(hash).SchemaHashHex())
}

func TestSchemaValidate(t *testing.T) {
	hash := [SchemaHashLen]byte{1}
	schema := Schema{
		Name:       "example",
		SchemaHash: hash,
		Methods: []Method{
			{ID: 1, Name: "Echo", RequestType: "EchoReq", ResponseType: "EchoResp"},
		},
	}
	require.NoError(t, schema.Validate())

	schema.Methods = append(schema.Methods, Method{
		ID:           1,
		Name:         "EchoAgain",
		RequestType:  "EchoReq",
		ResponseType: "EchoResp",
	})
	err := schema.Validate()
	require.ErrorIs(t, err, ErrMethodIDDuplicate)
}

func TestSchemaHasMethodID(t *testing.T) {
	schema := Schema{
		Name:       "example",
		SchemaHash: [SchemaHashLen]byte{1},
		Methods: []Method{
			{ID: 11, Name: "One", RequestType: "Req", ResponseType: "Resp"},
		},
	}
	require.True(t, schema.HasMethodID(11))
	require.False(t, schema.HasMethodID(12))
}

func TestRegistry(t *testing.T) {
	reg, err := NewRegistry(PluginMethod{
		ID:   10,
		Name: "Hello",
		Handler: func(payload []byte) ([]byte, error) {
			if len(payload) == 0 {
				return nil, errors.New("empty")
			}
			return append([]byte("hello "), payload...), nil
		},
	})
	require.NoError(t, err)

	data, err := reg.Call(10, []byte("david"))
	require.NoError(t, err)
	require.Equal(t, "hello david", string(data))

	_, err = reg.Call(404, nil)
	require.ErrorIs(t, err, ErrMethodNotFound)

	id, ok := reg.MethodID("Hello")
	require.True(t, ok)
	require.Equal(t, MethodID(10), id)
}
