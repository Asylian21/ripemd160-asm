# Performance

This document describes how to measure `ripemd160mb`, how to interpret the
numbers, and the acceptance criteria that gate a performance release.

The arm64 NEON backend currently delivers ~3x the throughput of scalar on Apple
M3 (about 13.2M vs 4.3M hashes/s at `n = 2048`, `GOMAXPROCS=1`) with zero
allocations, and it is verified bit-for-bit against the `golang.org/x/crypto`
oracle. amd64 SIMD is not yet implemented and runs scalar; the criteria below
apply to any new vector backend before it ships.

## What to measure

The hot path is `Hash32`. The benchmarks report four quantities per case:

- `ns/op` — wall-clock time for one `Hash32(dst, src, n)` call.
- `MB/s` — input throughput, via `b.SetBytes(n*32)`.
- `hashes/s` — messages hashed per second (the metric that matters for Hash160
  pipelines), via a custom `b.ReportMetric`.
- `allocs/op` — must be `0` for `Hash32`; a regression here is a correctness
  bug in the zero-alloc contract, not just a slowdown.

Batch sizes deliberately include the lane boundaries (`lanes-1`, `lanes`,
`lanes+1`) so that regressions in either the vectorized body or the scalar tail
are visible, plus larger batches (`64`, `1024`, `2048`) that amortize call
overhead.

## Running benchmarks

```sh
# All Hash32 benchmarks, all available backends, with allocation stats.
go test -run '^$' -bench '^BenchmarkHash32$' -benchmem ./

# Just the hash160 pipeline.
go test -run '^$' -bench '^BenchmarkHash160_32$' -benchmem ./hash160

# A single backend, comparable across runs (pin GOMAXPROCS and force a backend).
GOMAXPROCS=1 GORIPEMD160MB_FORCE=scalar \
	go test -run '^$' -bench '^BenchmarkHash32$' -benchmem -count=10 ./ \
	| tee scalar.txt
GOMAXPROCS=1 GORIPEMD160MB_FORCE=neon \
	go test -run '^$' -bench '^BenchmarkHash32$' -benchmem -count=10 ./ \
	| tee neon.txt
```

Use `-count=10` (or more) and a quiet machine so the noise is small enough for
`benchstat` to draw conclusions.

## Comparing with benchstat

```sh
go install golang.org/x/perf/cmd/benchstat@latest

# Old vs new code on the same backend.
benchstat old.txt new.txt

# Scalar vs a vector backend (rename the sub-benchmarks so they line up).
benchstat scalar.txt neon.txt
```

A change is only meaningful when `benchstat` reports it outside the noise band
(it prints `~` when the delta is not statistically significant).

## Profiling

```sh
# CPU profile of the hot path.
go test -run '^$' -bench '^BenchmarkHash32$' -cpuprofile=cpu.out ./
go tool pprof -top cpu.out
go tool pprof -http=:0 cpu.out        # interactive flame graph

# Memory profile (expect ~zero allocations on the Hash32 path).
go test -run '^$' -bench '^BenchmarkHash32$' -memprofile=mem.out ./
go tool pprof -alloc_space -top mem.out
```

For the scalar path, the `compress` function in
[`scalar.go`](scalar.go) dominates; for the vector backends, look at register
pressure and the per-lane load/store sequences emitted by the generators.

## Acceptance criteria for a vector backend

A vector backend is considered ready to ship as a default when, for the same
GOARCH and a documented reference CPU:

1. It remains bit-for-bit correct (`go test ./...` and the fuzz targets pass on
   that backend).
2. It is zero-allocation (`allocs/op == 0`) on the `Hash32` path.
3. `benchstat` shows a statistically significant `hashes/s` improvement over the
   scalar backend at `n = 2048` (the large-batch, steady-state case), with no
   regression at the small/`lanes`-sized cases.

The arm64 NEON backend meets all three on Apple M3 and is the default on arm64.
A backend that does not meet criterion 3 must not be wired as a default; it may
still be kept behind `GORIPEMD160MB_FORCE` for development, but `Backend()` must
never report a SIMD name for a kernel that is actually the scalar fallback.

## Recording results

When you capture a new baseline, update the smoke-benchmark table in
[README.md](README.md) with the GOARCH, CPU model, Go version, and the relevant
`Hash32` rows so the documented numbers stay reproducible.
