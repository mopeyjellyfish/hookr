package runtime

import (
	"testing"

	runtimecontract "github.com/mopeyjellyfish/hookr/runtime/contract"
	"github.com/stretchr/testify/require"
)

func TestContractIntrospectionNilAndEmpty(t *testing.T) {
	var rt *Runtime

	require.False(t, rt.SupportsMethodABI())
	_, ok := rt.PluginHandshake()
	require.False(t, ok)
	_, ok = rt.ExpectedHandshake()
	require.False(t, ok)
	require.Equal(t, uint64(0), rt.PluginCapabilities())
	require.True(t, rt.HasPluginCapabilities(0))
	require.Nil(t, rt.PluginMethodIDs())
	require.False(t, rt.HasPluginMethodID(1))
	_, ok = rt.ContractSchema()
	require.False(t, ok)
	require.False(t, rt.ContractHasMethodID(1))

	empty := &Runtime{}
	require.Nil(t, empty.PluginMethodIDs())
	require.False(t, empty.HasPluginMethodID(7))
	_, ok = empty.ContractSchema()
	require.False(t, ok)
	_, ok = empty.ExpectedHandshake()
	require.False(t, ok)
}

func TestContractIntrospectionExpectedAndPluginData(t *testing.T) {
	schema := runtimecontract.Schema{
		Name:       "example",
		SchemaHash: [runtimecontract.SchemaHashLen]byte{1},
		Methods: []runtimecontract.Method{
			{ID: 7, Name: "Echo", RequestType: "Req", ResponseType: "Resp"},
		},
	}
	handshake := runtimecontract.NewHandshake(schema.SchemaHash)
	handshake.Capabilities = 3
	rt := &Runtime{
		expectedSchema:    &schema,
		expectedHandshake: &handshake,
		pluginHandshake:   &handshake,
		pluginMethods: map[uint32]struct{}{
			7: {},
			2: {},
		},
	}

	gotSchema, ok := rt.ContractSchema()
	require.True(t, ok)
	require.Equal(t, schema.Name, gotSchema.Name)
	gotHandshake, ok := rt.ExpectedHandshake()
	require.True(t, ok)
	require.Equal(t, uint64(3), gotHandshake.Capabilities)
	require.Equal(t, []uint32{2, 7}, rt.PluginMethodIDs())
	require.True(t, rt.HasPluginMethodID(7))
	require.True(t, rt.ContractHasMethodID(7))
}
