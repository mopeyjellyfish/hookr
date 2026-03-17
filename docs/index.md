# Hookr Documentation

Hookr is a schema-defined WebAssembly plugin system for Go applications.

This documentation uses the Diataxis framework:

- [Tutorials](./tutorials/index.md): guided learning paths
- [How-to Guides](./how-to/index.md): task-focused procedures
- [Reference](./reference/index.md): precise technical lookup
- [Explanation](./explanation/index.md): architecture and design rationale
- [Agent Index](./agent-index.md): compact retrieval map for LLMs and tools

## Quick Start

1. Install the CLI:

```bash
go install github.com/mopeyjellyfish/hookr/cmd/hookr@latest
```

2. Generate contract glue from a FlatBuffers schema:

```bash
hookr gen --schema ./contract.fbs --out ./gen --package myhookr
```

3. Build your plugin to Wasm:

```bash
hookr build --plugin ./plugin --out ./bin/plugin.wasm
```

4. Load and call from your host via generated SDK.

For a full end-to-end walkthrough, start with:
[Build A Host And Plugin With FlatBuffers](./tutorials/urlbalancer.md).

## Agent Retrieval

Hookr also publishes explicit agent-oriented artifacts:

- [`./agent-index.md`](./agent-index.md): canonical retrieval map
- `/llms.txt`: short machine-oriented overview
- `/llms-full.txt`: dense single-file project summary
- `/agent-index.json`: structured machine-readable manifest
