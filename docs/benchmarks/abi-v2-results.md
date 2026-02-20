# ABI v2 Benchmark Snapshot (2026-02-19)

Environment:

- `goos=darwin`
- `goarch=arm64`
- `cpu=Apple M4 Pro`
- command package: `github.com/mopeyjellyfish/hookr/runtime`

## Commands

```bash
GOCACHE=$(pwd)/.cache/go-build go test -bench 'Invoke(Bytes(Vowel|$)|V2Bytes(Vowel|Echo|EchoDirect)|MsgP)' -benchmem -count=3 ./runtime
GOCACHE=$(pwd)/.cache/go-build go test -bench DispatchBy -benchmem -count=3 ./runtime
```

## Baseline (Before Latest Optimization Pass)

Legacy bytes path:

- `BenchmarkInvokeBytesVowel/Vowel`: ~`1052.3 ns/op`, `216 B/op`, `5 allocs/op`
- `BenchmarkInvokeBytes/Echo`: ~`1086.0 ns/op`, `520 B/op`, `9 allocs/op`

ABI v2 method-ID path:

- `BenchmarkInvokeV2BytesVowel/Vowel`: ~`751.9 ns/op`, `208 B/op`, `4 allocs/op`
- `BenchmarkInvokeV2BytesEcho/Echo`: ~`722.7 ns/op`, `496 B/op`, `7 allocs/op`

## Current (After Optimization Pass)

Changes included in this pass:

1. Removed `context.WithValue` invoke path from runtime hot call loop.
2. Added PDK scratch buffer reuse for plugin request payload/op buffers.
3. Added generated direct dispatch helpers (switch-based) for runtime and plugin.
4. Removed invoke-path closure wrappers in runtime call setup.

Legacy bytes path (improved by runtime hot-path changes):

- `BenchmarkInvokeBytesVowel/Vowel`: ~`961.9 ns/op`, `168 B/op`, `4 allocs/op`
- `BenchmarkInvokeBytes/Echo`: ~`1027.7 ns/op`, `472 B/op`, `8 allocs/op`

ABI v2 method-ID path:

- `BenchmarkInvokeV2BytesVowel/Vowel`: ~`706.8 ns/op`, `160 B/op`, `3 allocs/op`
- `BenchmarkInvokeV2BytesEcho/Echo`: ~`667.6 ns/op`, `448 B/op`, `6 allocs/op`
- `BenchmarkInvokeV2BytesEchoDirect/Echo`: ~`658.4 ns/op`, `448 B/op`, `6 allocs/op`

End-to-end improvements vs legacy (current run):

- Vowel path: `~26.5%` faster, `-8 B/op`, `-1 alloc/op`
- Echo path: `~35.0%` faster, `-24 B/op`, `-2 alloc/op`

Additional improvement vs prior v2 snapshot:

- V2 Vowel: ~`3.9%` faster (`735.5 -> 706.8 ns/op`), `0 B/op`, `0 alloc/op`
- V2 Echo: ~`3.0%` faster (`688.5 -> 667.6 ns/op`), `0 B/op`, `0 alloc/op`
- V2 EchoDirect: ~`7.4%` faster (`711.4 -> 658.4 ns/op`), `0 B/op`, `0 alloc/op`

MessagePack reference (unchanged path):

- `BenchmarkInvokeMsgP/Echo`: ~`2800.7 ns/op`, `736 B/op`, `13 allocs/op`

## Dispatch Microbench

- `BenchmarkDispatchByName`: ~`5.79 ns/op`, `0 allocs/op`
- `BenchmarkDispatchByMethodID` (map): ~`2.74 ns/op`, `0 allocs/op`
- `BenchmarkDispatchByMethodIDDirect` (switch): ~`1.53 ns/op`, `0 allocs/op`

Improvement:

- Method-ID map dispatch: ~`52.7%` lower latency (~`2.11x` faster) vs string dispatch.
- Method-ID direct dispatch: ~`73.6%` lower latency (~`3.78x` faster) vs string dispatch.

## Notes

- You will see `Host error: planned failure` lines in benchmark output because this package includes tests/bench setup paths that intentionally exercise error behavior.
- The ABI v2 numbers use the new wasm fixture at `testdata/simplev2/bin/simplev2.wasm`.
