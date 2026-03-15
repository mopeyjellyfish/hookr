# Contract Model Reference

Hookr contracts are defined by the host application in FlatBuffers.

## Default Service Names

- `Plugin`: methods the host calls on the plugin
- `Host`: callbacks the plugin calls on the host

These are configurable in CLI flags.

## Optional Methods

Plugin methods are required by default.
Optional methods are marked with a FlatBuffers RPC attribute, by default:
`hookr_optional`.

Example:

```fbs
attribute "hookr_optional";

rpc_service Plugin {
  Warmup(Empty):Empty (hookr_optional);
}
```

## Method IDs

Hookr derives stable method IDs from service + method identity during contract
loading.

## Contract Identity

Hookr computes a canonical schema hash from normalized contract metadata (from
`.bfbs` reflection data), not from raw schema text formatting.

## Full Design Context

- [Plugin System Direction](../plugin-system.md)
- [Implementation Plan](../implementation-plan.md)
