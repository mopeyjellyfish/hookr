# Generate Glue From A Contract

## Goal

Generate FlatBuffers bindings plus Hookr SDK/PDK glue from a schema.

## Steps

1. Run `hookr gen`:

```bash
hookr gen \
  --schema ./contract.fbs \
  --out ./gen \
  --package mycontracthookr
```

2. For imported schemas, add include paths:

```bash
hookr gen \
  --schema ./schemas/contract.fbs \
  --out ./gen \
  --package mycontracthookr \
  --include ./schemas \
  --include ./third_party
```

3. For non-default service names:

```bash
hookr gen \
  --schema ./contract.fbs \
  --out ./gen \
  --package mycontracthookr \
  --plugin-service EnginePlugin \
  --host-service EngineHost
```

## Output

Generated package typically includes:

- FlatBuffers type files (`flatc` output)
- `contract_meta_gen.go`
- `host_sdk_gen.go`
- `plugin_pdk_gen.go`

## Related

- [Reference: CLI](../reference/cli.md)
- [Reference: Contract Model](../reference/contracts.md)
