package runtime

import (
	"slices"

	runtimecontract "github.com/mopeyjellyfish/hookr/runtime/contract"
)

// SupportsMethodABI reports whether the loaded plugin exports __plugin_call.
func (e *Runtime) SupportsMethodABI() bool {
	return e != nil && e.pluginCall != nil
}

// PluginHandshake returns the plugin-reported handshake if available.
func (e *Runtime) PluginHandshake() (runtimecontract.Handshake, bool) {
	if e == nil || e.pluginHandshake == nil {
		return runtimecontract.Handshake{}, false
	}
	return *e.pluginHandshake, true
}

// ExpectedHandshake returns the configured host-side handshake requirements.
func (e *Runtime) ExpectedHandshake() (runtimecontract.Handshake, bool) {
	if e == nil || e.expectedHandshake == nil {
		return runtimecontract.Handshake{}, false
	}
	return *e.expectedHandshake, true
}

// PluginCapabilities returns plugin-reported capability bits (zero when unavailable).
func (e *Runtime) PluginCapabilities() uint64 {
	if e == nil || e.pluginHandshake == nil {
		return 0
	}
	return e.pluginHandshake.Capabilities
}

// HasPluginCapabilities reports whether plugin capabilities include all bits in mask.
func (e *Runtime) HasPluginCapabilities(mask uint64) bool {
	return e.PluginCapabilities()&mask == mask
}

// PluginMethodIDs returns the method IDs the plugin reported during startup introspection.
func (e *Runtime) PluginMethodIDs() []uint32 {
	if e == nil || len(e.pluginMethods) == 0 {
		return nil
	}
	out := make([]uint32, 0, len(e.pluginMethods))
	for methodID := range e.pluginMethods {
		out = append(out, methodID)
	}
	slices.Sort(out)
	return out
}

// HasPluginMethodID reports whether the plugin declared support for methodID.
func (e *Runtime) HasPluginMethodID(methodID uint32) bool {
	if e == nil || len(e.pluginMethods) == 0 {
		return false
	}
	_, ok := e.pluginMethods[methodID]
	return ok
}

// ContractSchema returns the configured expected schema metadata, when provided.
func (e *Runtime) ContractSchema() (runtimecontract.Schema, bool) {
	if e == nil || e.expectedSchema == nil {
		return runtimecontract.Schema{}, false
	}
	return *e.expectedSchema, true
}

// ContractHasMethodID reports whether the configured schema contains methodID.
func (e *Runtime) ContractHasMethodID(methodID uint32) bool {
	if e == nil || e.expectedSchema == nil {
		return false
	}
	return e.expectedSchema.HasMethodID(runtimecontract.MethodID(methodID))
}
