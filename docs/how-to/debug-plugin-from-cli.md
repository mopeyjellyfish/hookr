# Debug A Plugin From The CLI

## Goal

Validate and invoke a plugin without writing a Go host first.

## Inspect The Plugin

```bash
hookr inspect \
  --schema ./testdata/contracts/textfilter/textfilter.fbs \
  --wasm ./testdata/contracts/textfilter/bin/textfilter.wasm \
  --allow-unsigned
```

That shows:

- the contract schema hash
- the plugin-reported schema hash
- ABI version
- required and optional plugin methods
- implemented methods reported by the plugin

## Call A Plugin Method

Create a request file:

```json
{
  "input": "Bad inputs and bad habits",
  "blocked_terms": ["bad", "habits"],
  "replacement": "[filtered]",
  "case_sensitive": false,
  "max_replacements": 2
}
```

Call the plugin:

```bash
hookr call \
  --schema ./testdata/contracts/textfilter/textfilter.fbs \
  --wasm ./testdata/contracts/textfilter/bin/textfilter.wasm \
  --allow-unsigned \
  --method Filter \
  --input ./request.json
```

Hookr converts the JSON request to FlatBuffers, invokes the plugin, then
converts the response back to JSON.

Progress lines are written to stderr, so stdout stays safe to pipe into tools
like `jq`.

## Stub Host Callbacks

If the contract defines a `Host` service, provide a fixture file for callback
responses.

Example host fixture:

```json
{
  "RngInt": { "response": { "value": 1 } },
  "RngFloat": { "response": { "value": 0.5 } }
}
```

Example request:

```json
{
  "url": "https://example.com/api?q=1",
  "nodes": ["node-a", "node-b", "node-c"]
}
```

Example call:

```bash
hookr call \
  --schema ./testdata/contracts/urlbalancer/urlbalancer.fbs \
  --wasm ./testdata/contracts/urlbalancer/bin/urlbalancer.wasm \
  --allow-unsigned \
  --method Balance \
  --input ./balance.json \
  --host-fixture ./host.json
```

## Use The Interactive TUI

For exploratory work, use the Bubble Tea terminal UI:

```bash
hookr tui \
  --schema ./testdata/contracts/textfilter/textfilter.fbs \
  --wasm ./testdata/contracts/textfilter/bin/textfilter.wasm \
  --allow-unsigned
```

The TUI lets you:

- browse plugin methods with single-key shortcuts
- start from a schema-derived JSON request template
- see the active schema, Wasm, method, and loop timings in a top bar
- edit requests in your default editor instead of typing directly into the UI
- run one call or a tight call loop
- automatically reload the plugin when the Wasm file changes on disk
- watch loop timing stats and runtime debug metadata
- keep the shortcut legend visible at the bottom of the screen

Useful shortcuts:

- `j` / `k`: move between methods
- `e` or `o`: open the request in your default editor
- `c`: call the selected method once
- `l`: start or stop a tight call loop
- `r`: reset the request from the schema template
- `p`: pretty-format the current request JSON
- `q`: quit

## Related

- [How-To: Inspect A Contract Or Plugin](./inspect-contract.md)
- [Reference: CLI](../reference/cli.md)
