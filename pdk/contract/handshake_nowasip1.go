//go:build !wasip1

package contract

// EnableHandshake is a no-op outside wasip1 builds.
func EnableHandshake(schemaHash [SchemaHashLen]byte) {}
