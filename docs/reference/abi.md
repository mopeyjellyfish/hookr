# ABI Reference

Hookr uses a method-ID based Wasm ABI.

## Core Calls

- Host -> plugin: `method_id:uint32 + payload_ptr + payload_len`
- Plugin -> host: `method_id:uint32 + payload_ptr + payload_len`

## Handshake Exports

Plugins expose:

- `__hookr_abi_version`
- `__hookr_schema_hash`
- `__hookr_capabilities`
- `__hookr_methods`

Hosts validate:

1. exact ABI major match
2. exact ABI minor match
3. schema hash equality
4. required capability bits
5. required method support, plus optional-method availability when schema metadata is provided

## Runtime Notes

- Hookr uses guest linear memory for transport.
- One boundary copy per direction is expected.
- Generated code handles encode/decode and method dispatch wiring.

## Full Spec

- [Hookr ABI Spec](../abi.md)
