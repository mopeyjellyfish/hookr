# Inspect A Contract Or Plugin

## Goal

Print the normalized Hookr view of a FlatBuffers schema, and optionally validate
it against a real plugin module.

## Steps

1. Run inspect:

```bash
hookr inspect --schema ./contract.fbs --package mycontracthookr
```

2. Optional include paths:

```bash
hookr inspect \
  --schema ./schemas/contract.fbs \
  --package mycontracthookr \
  --include ./schemas \
  --include ./third_party
```

3. Optional service/attribute overrides:

```bash
hookr inspect \
  --schema ./contract.fbs \
  --package mycontracthookr \
  --plugin-service EnginePlugin \
  --host-service EngineHost \
  --optional-attribute hookr_optional
```

4. Inspect a built plugin against the schema:

```bash
hookr inspect \
  --schema ./contract.fbs \
  --wasm ./plugin.wasm
```

5. If the contract defines a `Host` service and the plugin needs callbacks
during startup, provide a host fixture:

```bash
hookr inspect \
  --schema ./contract.fbs \
  --wasm ./plugin.wasm \
  --host-fixture ./fixtures/host.json
```

## Output Includes

- contract name
- schema hash
- plugin service methods and IDs
- host service methods and IDs
- required vs optional plugin method flags
- plugin ABI version and schema hash
- plugin-reported method inventory
- contract versus plugin method match status

## Related

- [Reference: Contract Model](../reference/contracts.md)
- [How-To: Debug A Plugin From The CLI](./debug-plugin-from-cli.md)
