# Repository Guidelines

## Project Structure & Module Organization

- `cmd/hookr/`: CLI entrypoint.
- `runtime/`: core host runtime, live reload, trust handling, and Wasm dispatch.
- `pdk/`: plugin-side runtime surface used by generated plugins.
- `internal/`: code generation, contract loading, CLI commands, build tooling, inspect/call/TUI helpers.
- `docs/`: VitePress documentation site.
- `testdata/`: fixture contracts and standalone Wasm test modules.
- `.github/workflows/`: CI, release, and docs deployment workflows.

## Agent Discovery & Context

- Start with `docs/agent-index.md`. It maps tasks to canonical docs and source files.
- Published machine-oriented docs live at `/agent-index`, `/llms.txt`, `/llms-full.txt`, and `/agent-index.json`.
- For schema and API questions, read `docs/reference/contracts.md`, `docs/reference/generated-go-api.md`, and `docs/reference/cli.md`.
- For runtime internals, start with `runtime/runtime.go`, `runtime/live.go`, `runtime/file.go`, and `internal/codegen/flatbuffers.go`.
- For working examples, use `testdata/contracts/textfilter`, `testdata/contracts/urlbalancer`, and `testdata/contracts/tickloop`.

## Build, Test, and Development Commands

- `make build`: build the `hookr` CLI into `bin/hookr`.
- `make test`: run the runtime-focused test suite with race detection and coverage via `gotestsum`.
- `make test/ff`: fail-fast variant for faster local loops.
- `make lint`: run `golangci-lint` on `./runtime/...`.
- `make docs/build`: build the VitePress site.
- `make docs/serve`: run the docs site locally.
- `make hooks/install`: install the repo-local pre-commit hook.

Examples:

```bash
make build
make test
go run ./cmd/hookr version
```

## Coding Style & Naming Conventions

- Follow standard Go formatting: run `go fmt ./...`.
- Keep edits ASCII unless a file already requires Unicode.
- Use clear, package-appropriate names; prefer `PluginPath` over ambiguous names like `Path`.
- Keep Hookr terminology generic: one configured plugin `rpc_service`; every other `rpc_service` is a host callback module.
- Keep runtime and codegen APIs generic to Hookr. Avoid app-specific naming bleed-through.
- Use `apply_patch` for manual file edits when working through agents.

## Testing Guidelines

- Tests use Go’s standard `testing` package.
- Name tests `TestXxx` and benchmarks `BenchmarkXxx`.
- Prefer cheap unit tests over expensive Wasm compilation paths when covering parser/error logic.
- Keep runtime coverage healthy; `make test` is the canonical local and CI path.
- For docs or tooling changes, also run `make docs/build`.

## Commit & Pull Request Guidelines

- Use Conventional Commits: `feat:`, `fix:`, `docs:`, `test:`, `refactor:`, `style:`, `chore:`.
- Keep subjects short and imperative, for example: `feat: add live reloading for plugin runtimes`.
- PRs should include a short summary, key validation commands, and note any behavior changes or migration impact.
- For UI/docs changes, include screenshots only when they add value.

## Security & Configuration Tips

- Default trust is hash-pinned plugins; use unsigned plugin loading only for local development.
- Install hooks with `make hooks/install` so lint, tests, and docs run before commits.
- If you change docs structure or command behavior, update both the human docs and the agent retrieval artifacts in `docs/public/`.
