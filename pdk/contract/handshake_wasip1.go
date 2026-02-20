//go:build wasip1

package contract

import "github.com/mopeyjellyfish/hookr/pdk"

// EnableHandshake publishes the schema hash to Hookr ABI v2 handshake exports.
func EnableHandshake(schemaHash [SchemaHashLen]byte) {
	pdk.SetABIV2SchemaHash(schemaHash)
	pdk.SetABIV2Capabilities(0)
}

// EnableHandshakeWithCapabilities publishes schema hash + capability bits for ABI v2.
func EnableHandshakeWithCapabilities(schemaHash [SchemaHashLen]byte, capabilities uint64) {
	pdk.SetABIV2SchemaHash(schemaHash)
	pdk.SetABIV2Capabilities(capabilities)
}
