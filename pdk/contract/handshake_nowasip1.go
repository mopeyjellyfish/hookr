//go:build !wasip1

package contract

// PublishContractHandshake is a no-op outside wasip1 builds.
func PublishContractHandshake(schemaHash [SchemaHashLen]byte, capabilities uint64, methodIDs []uint32) {}
