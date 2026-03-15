package contract

import (
	"context"
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
	hashHex := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
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

func TestHostRegistry(t *testing.T) {
	reg, err := NewHostRegistry(HostMethod{
		ID:   10,
		Name: "Hello",
		Handler: func(_ context.Context, payload []byte) ([]byte, error) {
			if len(payload) == 0 {
				return nil, errors.New("empty")
			}
			return append([]byte("hello "), payload...), nil
		},
	})
	require.NoError(t, err)

	data, err := reg.Call(context.Background(), 10, []byte("david"))
	require.NoError(t, err)
	require.Equal(t, "hello david", string(data))

	_, err = reg.Call(context.Background(), 404, nil)
	require.ErrorIs(t, err, ErrMethodNotFound)

	id, ok := reg.MethodID("Hello")
	require.True(t, ok)
	require.Equal(t, MethodID(10), id)
}

func TestHostRegistryErrors(t *testing.T) {
	_, err := NewHostRegistry(HostMethod{ID: 1, Name: "bad", Handler: nil})
	require.ErrorIs(t, err, ErrMethodHandlerMissing)

	_, err = NewHostRegistry(
		HostMethod{
			ID:      1,
			Name:    "one",
			Handler: func(context.Context, []byte) ([]byte, error) { return nil, nil },
		},
		HostMethod{
			ID:      1,
			Name:    "two",
			Handler: func(context.Context, []byte) ([]byte, error) { return nil, nil },
		},
	)
	require.ErrorIs(t, err, ErrMethodIDDuplicate)

	_, err = NewHostRegistry(
		HostMethod{
			ID:      1,
			Name:    "dup",
			Handler: func(context.Context, []byte) ([]byte, error) { return nil, nil },
		},
		HostMethod{
			ID:      2,
			Name:    "dup",
			Handler: func(context.Context, []byte) ([]byte, error) { return nil, nil },
		},
	)
	require.ErrorIs(t, err, ErrMethodNameDuplicate)

	var reg *HostRegistry
	_, err = reg.Call(context.Background(), 1, nil)
	require.ErrorIs(t, err, ErrMethodNotFound)

	id, ok := reg.MethodID("missing")
	require.False(t, ok)
	require.Equal(t, MethodID(0), id)
}
