# Hookr

<p align="center">
    <strong>Seamless WebAssembly plugins for Go — secure, type-safe, and blazingly fast.</strong>
</p>

<p align="center">
    Extend Go applications with dynamically loaded WASM modules.
</p>

---

<p align="center">
  <a href="https://github.com/mopeyjellyfish/hookr/actions/workflows/tests.yml"><img src="https://github.com/mopeyjellyfish/hookr/actions/workflows/tests.yml/badge.svg" alt="Tests"></a>
  <a href="https://github.com/mopeyjellyfish/hookr/actions/workflows/lint.yml"><img src="https://github.com/mopeyjellyfish/hookr/actions/workflows/lint.yml/badge.svg" alt="Lint"></a>
  <img alt="GitHub Release" src="https://img.shields.io/github/v/release/mopeyjellyfish/hookr">
  <a href="https://codecov.io/github/mopeyjellyfish/hookr" > 
     <img src="https://codecov.io/github/mopeyjellyfish/hookr/graph/badge.svg?token=peUgWB4joM"/> 
  </a>
</p>

## Features

- **Schema-defined contracts**: Host applications define one `Plugin` service and any number of host callback modules in FlatBuffers
- **Generated Go SDK/PDK glue**: `hookr gen` produces typed host and plugin bindings
- **Method-ID Wasm ABI**: Fast numeric dispatch suitable for tight plugin loops
- **Integrity checks by default**: Plugin files are hash-verified unless the host explicitly allows unsigned artifacts
- **TinyGo-first builds**: `hookr build` produces Wasm plugins for local development and CI
- **Bidirectional calls**: Plugins can call host-defined callbacks through the generated `PluginContext`
- **Developer tooling**: `hookr inspect`, `hookr call`, and a Bubble Tea-based `hookr tui` help validate and debug plugins without a full host app

## Table of Contents

- [Installation](#installation)
- [What Hookr Is For](#what-hookr-is-for)
- [Quick Start](#quick-start)
- [Plugin Trust Model](#plugin-trust-model)
- [Project Structure](#project-structure)
- [Language Roadmap](#language-roadmap)

## Installation

Install the CLI:

```bash
go install github.com/mopeyjellyfish/hookr/cmd/hookr@latest
```

Your host application will usually import the generated contract package, which
pulls in Hookr as a normal Go module dependency.

### Prerequisites

- Go 1.24 or higher
- TinyGo 0.30.0 or higher (for building plugins)

## What Hookr Is For

Hookr is for applications that want plugins with:

- typed request and response contracts
- host-to-plugin and plugin-to-host calls
- good performance for both infrequent calls and tight loops
- a small integration surface for host and plugin authors

Typical fits:

- game logic or simulation plugins
- text, validation, or routing plugins
- application-defined extension points where the host owns the contract

Hookr ships comprehensive Diataxis documentation under [`docs/`](./docs/):

- tutorials: [`docs/tutorials/`](./docs/tutorials/)
- how-to guides: [`docs/how-to/`](./docs/how-to/)
- reference: [`docs/reference/`](./docs/reference/)
- explanation: [`docs/explanation/`](./docs/explanation/)

If you want one place to start, use [`docs/index.md`](./docs/index.md).

## Quick Start

The smallest Hookr flow is:

```bash
hookr gen \
  --schema ./testdata/contracts/textfilter/textfilter.fbs \
  --out ./testdata/contracts/textfilter/gen \
  --package textfilterhookr

hookr build \
  --plugin ./testdata/contracts/textfilter/plugin \
  --out ./testdata/contracts/textfilter/bin/textfilter.wasm

hookr inspect \
  --schema ./testdata/contracts/textfilter/textfilter.fbs \
  --plugin ./testdata/contracts/textfilter/bin/textfilter.wasm \
  --allow-unsigned
```

The consuming application owns the contract. Hookr owns the ABI, transport,
validation, code generation, and host/plugin glue.

Minimal host usage in Go:

```go
package main

import (
	"context"
	"fmt"
	"log"

	hookr "github.com/mopeyjellyfish/hookr/runtime"
	textfilter "github.com/mopeyjellyfish/hookr/testdata/contracts/textfilter/gen/textfilterhookr"
)

func main() {
	ctx := context.Background()

	plugin, err := textfilter.Open(ctx, textfilter.Config{
		PluginPath: "./testdata/contracts/textfilter/bin/textfilter.wasm",
		FileOptions: []hookr.FileOption{
			hookr.WithAllowUnsigned(),
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer plugin.Close(ctx)

	info, err := plugin.GetInfo(ctx, &textfilter.EmptyT{})
	if err != nil {
		log.Fatal(err)
	}
	resp, err := plugin.Filter(ctx, &textfilter.FilterRequestT{
		Input:        "this platform has bad words",
		BlockedTerms: []string{"bad"},
		Replacement:  "[filtered]",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s %s => %s\n", info.Name, info.Version, resp.Output)
}
```

`PluginPath` is the plugin artifact path for the generated contract. Hosts can
swap in different `.wasm` files there as long as they were built against the
same schema and generated package.

For local development, the generated SDK can also watch that plugin artifact
and reload it automatically:

```go
plugin, err := textfilter.Open(ctx, textfilter.Config{
	PluginPath: "./testdata/contracts/textfilter/bin/textfilter.wasm",
	FileOptions: []hookr.FileOption{
		hookr.WithAllowUnsigned(),
	},
	Reload: &textfilter.ReloadConfig{
		OnReload: func(ctx context.Context, next *textfilter.Runtime, event hookr.ReloadEvent) error {
			_, err := next.GetInfo(ctx, &textfilter.EmptyT{})
			return err
		},
	},
})
```

While Hookr reloads, it pauses new calls, loads and validates the replacement
plugin, runs `OnReload`, and only then swaps the runtime. If reload fails, the
existing runtime stays active.

The generated package name is also your choice. In examples, aliasing the
generated import to the contract name usually reads better than repeating the
`...hookr` suffix in every type.

If your contract defines host callbacks, the generated SDK groups them by
service. Here is a complete working host for the `urlbalancer` example:

```go
package main

import (
	"context"
	"log"

	hookr "github.com/mopeyjellyfish/hookr/runtime"
	urlbalancer "github.com/mopeyjellyfish/hookr/testdata/contracts/urlbalancer/gen/urlbalancerhookr"
)

type host struct{}

func (host) Int(_ context.Context, req *urlbalancer.RngIntRequestT) (*urlbalancer.RngIntResponseT, error) {
	if req == nil || req.Max <= req.Min {
		return &urlbalancer.RngIntResponseT{Value: req.Min}, nil
	}
	return &urlbalancer.RngIntResponseT{Value: req.Min + ((req.Max - req.Min) / 2)}, nil
}

func (host) Float(_ context.Context, _ *urlbalancer.RngFloatRequestT) (*urlbalancer.RngFloatResponseT, error) {
	return &urlbalancer.RngFloatResponseT{Value: 0.5}, nil
}

func main() {
	ctx := context.Background()

	plugin, err := urlbalancer.Open(ctx, urlbalancer.Config{
		PluginPath: "./plugin.wasm",
		Host: urlbalancer.Host{
			Rng: host{},
		},
		FileOptions: []hookr.FileOption{
			hookr.WithAllowUnsigned(),
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer plugin.Close(ctx)

	resp, err := plugin.Balance(ctx, &urlbalancer.BalanceRequestT{
		Url:   "https://example.com/api",
		Nodes: []string{"node-a", "node-b", "node-c"},
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("selected %s (valid=%v rng=%.2f)", resp.SelectedNode, resp.Valid, resp.RngFloat)
}
```

### Host Callbacks

Host callbacks are defined by the host application in the FlatBuffers contract,
not by Hookr. The common pattern is:

1. Define callback modules as normal `rpc_service`s alongside `Plugin`.
2. Run `hookr gen`.
3. Implement the generated per-module interfaces in your Go host.
4. Pass that implementation into `Config.Host` when opening the plugin.
5. Call those callbacks from plugin code through `PluginContext`.

Minimal end-to-end shape:

```fbs
rpc_service Plugin {
  Balance(BalanceRequest):BalanceResponse;
}

rpc_service Rng {
  Int(RngIntRequest):RngIntResponse;
}
```

```go
type host struct{}

func (host) Int(ctx context.Context, req *urlbalancer.RngIntRequestT) (*urlbalancer.RngIntResponseT, error) {
	return &urlbalancer.RngIntResponseT{Value: req.Min}, nil
}

plugin, err := urlbalancer.Open(ctx, urlbalancer.Config{
	PluginPath: "./plugin.wasm",
	Host: urlbalancer.Host{
		Rng: host{},
	},
	FileOptions: []hookr.FileOption{
		hookr.WithAllowUnsigned(),
	},
})
```

```go
func (plugin) Balance(ctx *urlbalancer.PluginContext, req *urlbalancer.BalanceRequestT) (*urlbalancer.BalanceResponseT, error) {
	rng, err := ctx.Rng.Int(&urlbalancer.RngIntRequestT{Min: 0, Max: int32(len(req.Nodes) - 1)})
	if err != nil {
		return nil, err
	}
	return &urlbalancer.BalanceResponseT{SelectedNode: req.Nodes[rng.Value]}, nil
}
```

That is the whole registration story: the schema declares the callbacks, the
host implements the generated module interfaces, and Hookr wires the rest.

### Live Reload

Generated host SDKs also expose a `Reload` field in `Config` for local
development:

```go
plugin, err := urlbalancer.Open(ctx, urlbalancer.Config{
	PluginPath: "./plugin.wasm",
	Host: urlbalancer.Host{
		Rng: host{},
	},
	FileOptions: []hookr.FileOption{
		hookr.WithAllowUnsigned(),
	},
	Reload: &urlbalancer.ReloadConfig{
		OnReload: func(ctx context.Context, next *urlbalancer.Runtime, event hookr.ReloadEvent) error {
			_, err := next.GetInfo(ctx, &urlbalancer.EmptyT{})
			return err
		},
	},
})
```

Use `OnReload` to warm caches, rehydrate host-side state, or run a typed sanity
check against the replacement plugin before Hookr resumes traffic.

For plugin development, Hookr can validate and call plugins directly from the
CLI. For example, this calls the `urlbalancer` plugin with a host callback
fixture instead of a handwritten host app:

```bash
hookr call \
  --schema ./testdata/contracts/urlbalancer/urlbalancer.fbs \
  --plugin ./testdata/contracts/urlbalancer/bin/urlbalancer.wasm \
  --allow-unsigned \
  --method Balance \
  --input ./testdata/contracts/urlbalancer/requests/balance.json \
  --host-fixture ./testdata/contracts/urlbalancer/fixtures/host.json
```

And for interactive exploration:

```bash
hookr tui \
  --schema ./testdata/contracts/textfilter/textfilter.fbs \
  --plugin ./testdata/contracts/textfilter/bin/textfilter.wasm \
  --allow-unsigned
```

The TUI now:

- pre-fills requests from the schema
- shows the active schema, Wasm, method, and loop timings in a top bar
- reloads the plugin when the Wasm file changes on disk
- supports single-key actions for call, loop, reset, and editor workflows
- keeps the request read-only in the UI and opens your default editor for edits
- shows loop timing stats and runtime debug metadata while you work
- keeps the key shortcuts visible at the bottom of the screen

What this gives you:

- typed host calls into Wasm plugins
- optional typed callbacks from plugin back to host
- schema validation before first call
- a fast path suitable for high-frequency calls like game ticks

The host application decides what the contract is. Hookr only owns the Wasm ABI, handshake, transport, code generation, and host/plugin glue.

For architecture details, see:
- [`docs/plugin-system.md`](./docs/plugin-system.md)
- [`docs/abi.md`](./docs/abi.md)
- [`docs/explanation/architecture.md`](./docs/explanation/architecture.md)

For runnable examples, see:
- [`testdata/contracts/urlbalancer`](./testdata/contracts/urlbalancer)
- [`testdata/contracts/textfilter`](./testdata/contracts/textfilter)
- [`testdata/contracts/tickloop`](./testdata/contracts/tickloop)

Recommended reading order:

1. [`docs/tutorials/textfilter.md`](./docs/tutorials/textfilter.md)
2. [`docs/tutorials/urlbalancer.md`](./docs/tutorials/urlbalancer.md)
3. [`docs/reference/cli.md`](./docs/reference/cli.md)

## Plugin Trust Model

Hookr now requires explicit trust for local unsigned Wasm artifacts. Production
hosts should load signed or hash-pinned plugins; local fixtures and tutorials
can opt in to unsigned development artifacts when needed.

Hash-pinned example:

```go
plugin, err := runtime.New(ctx,
    runtime.WithFile("./plugin.wasm",
        runtime.WithHash("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"),
        runtime.WithHasher(runtime.Sha256Hasher{}),
    ),
)
```

Local development artifact example:

```go
plugin, err := runtime.New(ctx,
	runtime.WithFile("./plugin.wasm", runtime.WithAllowUnsigned()),
)
```

## Project Structure

- `cmd/hookr/`: installable CLI
- `runtime/`: host-side Wasm runtime
- `pdk/`: low-level plugin runtime support used by generated bindings
- `internal/codegen/`: `hookr gen` orchestration and generated glue templates
- `testdata/contracts/`: end-to-end fixture contracts and examples
- `docs/`: Diataxis documentation site source

## Language Roadmap

Hookr currently supports the following languages for plugin development:

| Language       | Support Level | Notes                                    |
|----------------|---------------|------------------------------------------|
| Go             | Full          | Using TinyGo compiler for WASM modules   |
| Rust           | Planned       | Coming in future releases                |
| Zig            | Planned       | Coming in future releases                |
| AssemblyScript | Planned       | Coming in future releases                |
| C/C++          | Planned       | Coming in future releases                |
