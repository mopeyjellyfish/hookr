# TickLoop Contract

`tickloop` is a benchmark-oriented Hookr fixture for tight-loop plugin
execution.

It stays consumer-defined and generic while modeling a repeated tick/update
style call pattern.

## Contract Summary

Plugin methods:

- `GetInfo`
- `Tick`
- `Warmup` (optional)

Host callbacks:

- `RngInt`

`Tick` accepts:

- tick index and delta time,
- lightweight loop state metadata,
- event count information.

The plugin is expected to:

1. process one tick quickly,
2. optionally use host RNG for jitter/bucket selection,
3. return next-state metadata and control flags.

## Why This Fixture Exists

This fixture is used to exercise and benchmark hot-path host -> plugin calls
while still validating plugin -> host callbacks in a loop-friendly contract.
