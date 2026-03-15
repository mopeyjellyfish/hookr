# FlatBuffers Benchmarks

Date: March 9, 2026
Machine: Apple M4 Pro, darwin/arm64

## Commands

```bash
go test ./testdata/contracts/tickloop -run '^$' -bench BenchmarkTickLoopTick -benchmem -count=1
go test ./testdata/contracts/tickloop -run '^$' -bench BenchmarkTickLoopWarmup -benchmem -count=1
go test ./testdata/contracts/urlbalancer -run '^$' -bench BenchmarkURLBalancerBalance -benchmem -count=1
```

## Before Optimization

Initial baseline before the generated-code pooling and callback-path changes:

| Benchmark | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| `BenchmarkTickLoopTick` | 4046 | 560 | 18 |
| `BenchmarkTickLoopWarmup` | 1472 | 296 | 9 |

## After Optimization

Current results after:

- pooled FlatBuffers builders in generated glue
- borrowed request encoding on generated caller paths
- callback-style runtime / PDK response handling to avoid extra transient
  `[]byte` allocations on the fast path

| Benchmark | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| `BenchmarkTickLoopTick` | 2090 | 136 | 7 |
| `BenchmarkTickLoopWarmup` | 820 | 40 | 3 |
| `BenchmarkURLBalancerBalance` | 5242 | 272 | 12 |

## Observed Slow Paths

The main remaining costs are:

- Wasm engine call overhead inside wazero / wazevo
- object API decode (`UnPack`) for response structs
- host callback response encode/decode on the callback-heavy paths

The biggest avoidable allocator in the first pass was FlatBuffers builder churn,
which is now removed from the hot caller path.
