//go:build wasip1

package contract

import "github.com/mopeyjellyfish/hookr/pdk"

// PublishContractHandshake publishes schema hash, capabilities, and implemented
// method IDs for generated Hookr plugins.
func PublishContractHandshake(schemaHash [SchemaHashLen]byte, capabilities uint64, methodIDs []uint32) {
	pdk.SetABISchemaHash(schemaHash)
	pdk.SetABICapabilities(capabilities)
	pdk.SetABIMethods(methodIDs)
}
