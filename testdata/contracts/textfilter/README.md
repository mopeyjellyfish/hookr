# TextFilter Contract

`textfilter` is a minimal Hookr fixture for a host-defined plugin API with
no host callbacks.

It is intentionally small so it can serve as the easiest tutorial-style
contract in the repository.

## Contract Summary

Plugin methods:

- `GetInfo`
- `Filter`

Host callbacks:

- none

`Filter` accepts:

- input text,
- a list of blocked terms,
- replacement behavior options.

The plugin is expected to:

1. apply the filter rules,
2. return the transformed output,
3. report whether changes were made and how many replacements occurred.

## Why This Fixture Exists

This fixture validates the simplest generated Hookr flow: host -> plugin typed
calls with structured payloads and no plugin -> host callback wiring.
