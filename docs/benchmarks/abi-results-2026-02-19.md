# Runtime Method-ID Benchmark Snapshot (2026-02-19)

Environment:

- `goos=darwin`
- `goarch=arm64`
- `cpu=Apple M4 Pro`
- package: `github.com/mopeyjellyfish/hookr/runtime`

## Commands

```bash
GOCACHE=$(pwd)/.cache/go-build go test -bench 'InvokeMethodBytes(Vowel|Echo)' -benchmem -count=3 ./runtime
GOCACHE=$(pwd)/.cache/go-build go test -bench DispatchBy -benchmem -count=3 ./runtime
```

## Snapshot

- `BenchmarkInvokeMethodBytesVowel/Vowel`: about `706.8 ns/op`, `160 B/op`, `3 allocs/op`
- `BenchmarkInvokeMethodBytesEcho/Echo`: about `667.6 ns/op`, `448 B/op`, `6 allocs/op`

## Dispatch Microbenchmarks

- `BenchmarkDispatchByName`: about `5.79 ns/op`, `0 allocs/op`
- `BenchmarkDispatchByMethodID` (map): about `2.74 ns/op`, `0 allocs/op`

## Interpretation

This snapshot is a runtime-focused microbenchmark, not an end-to-end fixture
benchmark. It is useful for tracking dispatch and invoke overhead inside the
core Hookr runtime.

For schema-driven FlatBuffers fixture benchmarks, use:

- [FlatBuffers Results 2026-03-09](./flatbuffers-2026-03-09.md)
- [`./testdata/contracts/tickloop`](https://github.com/mopeyjellyfish/hookr/tree/main/testdata/contracts/tickloop)
- [`./testdata/contracts/urlbalancer`](https://github.com/mopeyjellyfish/hookr/tree/main/testdata/contracts/urlbalancer)
