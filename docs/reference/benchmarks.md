# Benchmark Snapshots

Published benchmark result snapshots:

- [ABI Results (2026-02-19)](../benchmarks/abi-results-2026-02-19.md)
- [FlatBuffers Results (2026-03-09)](../benchmarks/flatbuffers-2026-03-09.md)

Primary benchmark fixture packages:

- `./testdata/contracts/tickloop`
- `./testdata/contracts/urlbalancer`

Runtime micro-benchmarks:

- `./runtime` benchmark suite (`BenchmarkInvokeMethodBytes*`)

Use [`hookr bench`](./cli.md) for fixture benchmark runs and `go test -bench`
for deeper package-level profiling.
