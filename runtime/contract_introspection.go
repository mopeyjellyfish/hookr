package runtime

import runtimecontract "github.com/mopeyjellyfish/hookr/runtime/contract"

// SupportsStringABI reports whether the loaded plugin exports __plugin_call.
func (e *Runtime) SupportsStringABI() bool {
	return e != nil && e.pluginCall != nil
}

// SupportsMethodABI reports whether the loaded plugin exports __plugin_call_v2.
func (e *Runtime) SupportsMethodABI() bool {
	return e != nil && e.pluginCallV2 != nil
}

// PluginHandshake returns the plugin-reported ABI v2 handshake if available.
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

// RequiredCapabilities returns the effective capability requirements mask.
func (e *Runtime) RequiredCapabilities() uint64 {
	if e == nil {
		return 0
	}
	required := e.requiredCapabilities
	if e.expectedHandshake != nil {
		required |= e.expectedHandshake.Capabilities
	}
	return required
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
