# Performance

This document describes how to measure `ripemd160mb`, how to interpret the
numbers, and the acceptance criteria that gate a performance release.

The local M5 Pro measurement reaches 41.85M hashes/s with `neon-sha3` at
`n = 2048`, Go 1.27.1, and `GOMAXPROCS=1`, with zero allocations. Batches of
eight or more messages run two independent four-lane groups; `Lanes()` stays 4
and a remainder of four messages still uses the four-lane kernel. Against that
four-lane kernel on the same machine, n=2048 throughput rose 67.41%
(`p=0.008`, five 250 ms samples) and n=4 fell 1.06% (`p=0.016`) because of the
Go dispatch wrapper. The earlier four-lane figure on Go 1.22.5 was 24.74M
hashes/s, with the base NEON kernel at 22.35M. The independent
`golang.org/x/crypto` oracle validates both, including bit patterns, unaligned
buffers, and scalar tails.
These throughput measurements cover the M5 Pro only; cross-compilation on
other architectures verifies buildability, not their runtime speed.
amd64 SIMD is not yet implemented and runs scalar.

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

# A single backend. The benchmark enumerates every available backend itself,
# so the sub-benchmark regex is necessary even when FORCE is set.
GOMAXPROCS=1 GORIPEMD160MB_FORCE=scalar \
	go test -run '^$' -bench '^BenchmarkHash32/scalar/' -benchmem -count=10 ./ \
	| tee scalar.txt
GOMAXPROCS=1 GORIPEMD160MB_FORCE=neon \
	go test -run '^$' -bench '^BenchmarkHash32/neon/' -benchmem -count=10 ./ \
	| tee neon.txt
GOMAXPROCS=1 GORIPEMD160MB_FORCE=neon-sha3 \
	go test -run '^$' -bench '^BenchmarkHash32/neon-sha3/' -benchmem -count=10 ./ \
	| tee neon-sha3.txt
```

Use `-count=10` (or more) and a quiet machine so the noise is small enough for
`benchstat` to draw conclusions.

## Comparing with benchstat

```sh
go install golang.org/x/perf/cmd/benchstat@latest

# Old vs new code on the same backend.
benchstat old.txt new.txt

# Scalar vs a vector backend requires normalizing their different backend
# names first, or using benchstat projections.
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

For fixed-width scalar hashing, the generated `sum32` function dominates.
The general streaming API uses `compress` in [scalar.go](scalar.go). For vector
kernels, inspect dependency chains, register pressure, and generated loads and
stores. Preadding the message and round constant to `a` allows those operations
to overlap with the independent boolean calculation of `f(b,c,d)`.

## Acceptance criteria for a vector backend

A vector backend is considered ready to ship as a default when, for the same
GOARCH and a documented reference CPU:

1. It remains bit-for-bit correct (`go test ./...` and the fuzz targets pass on
   that backend).
2. It is zero-allocation (`allocs/op == 0`) on the `Hash32` path.
3. `benchstat` shows a statistically significant `hashes/s` improvement over the
   scalar backend at `n = 2048` (the large-batch, steady-state case), with no
   regression at the small/`lanes`-sized cases.

The base NEON and SHA3 kernels satisfy these criteria on the measured M5 Pro.
SHA3 is selected only when the CPU probe succeeds; base NEON remains available
on every arm64 CPU.
A backend that does not meet criterion 3 must not be wired as a default; it may
still be kept behind `GORIPEMD160MB_FORCE` for development, but `Backend()` must
never report a SIMD name for a kernel that is actually the scalar fallback.

## Recording results

When you capture a new baseline, update the smoke-benchmark table in
[README.md](README.md) with the GOARCH, CPU model, Go version, and the relevant
`Hash32` rows so the documented numbers stay reproducible.
