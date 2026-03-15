# Run Benchmarks

## Goal

Run Hookr benchmark suites for runtime call-path performance.

## Steps

1. Run default benchmark package:

```bash
hookr bench
```

2. Run a specific benchmark:

```bash
hookr bench \
  --package ./testdata/contracts/tickloop \
  --bench BenchmarkTickLoopTick \
  --count 3
```

3. Run runtime micro-benchmarks:

```bash
go test ./runtime -run '^$' -bench BenchmarkInvokeMethodBytes -benchmem -count=1
```

## Related

- [Reference: CLI](../reference/cli.md)
- [Reference: Benchmark Snapshots](../reference/benchmarks.md)
- [Explanation: Performance Model](../explanation/performance-model.md)
