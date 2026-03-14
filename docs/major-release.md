# Major Release Migration Note

This document belongs to the Diataxis Explanation section:
[Explanation Index](./explanation/index.md).

Hookr now has one public plugin system model: schema-defined FlatBuffers
contracts with generated SDK/PDK glue.

Supported public surface:

- FlatBuffers-defined contracts
- generated host SDK and plugin PDK glue
- `hookr gen`
- `hookr build`
- `hookr inspect`
- `hookr bench`

Migration guidance:

- treat this as a major-version breaking change in release/versioning metadata
- remove public references to parallel/alternate plugin API tracks
- remove deprecated byte/string/msgpack public APIs from release notes and docs

Application model remains the same: the consuming host project defines its own
`Plugin` and `Host` services in FlatBuffers, and Hookr provides runtime,
validation, and generation around that contract.
