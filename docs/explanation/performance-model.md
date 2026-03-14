# Performance Model

Hookr performance is primarily shaped by Wasm boundary crossings and
serialization behavior.

## Core Model

- method-ID dispatch avoids string lookup overhead
- generated switch dispatch avoids map lookups on hot paths
- transport uses Wasm linear memory and bounded copy steps
- FlatBuffers object API keeps contract types ergonomic for host/plugin code

## Practical Tradeoff

Hookr targets:

- low allocation overhead
- predictable method-call cost
- tight-loop friendliness for high update rates

Hookr does not claim true shared-memory zero-copy between host and guest, since
Wasm host and guest pointers are separate memory domains.

## Benchmarking Strategy

- use fixture contracts (`tickloop`, `urlbalancer`) for realistic workloads
- track ns/op, B/op, allocs/op
- keep benchmark snapshots versioned in docs

Related:

- [How To Run Benchmarks](../how-to/run-benchmarks.md)
- [Benchmark Snapshots](../reference/benchmarks.md)
