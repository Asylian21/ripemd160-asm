# ripemd160mb

[Go Reference](https://pkg.go.dev/github.com/Asylian21/ripemd160-asm)
[CI](https://github.com/Asylian21/ripemd160-asm/actions/workflows/ci.yml)
[Go Report Card](https://goreportcard.com/report/github.com/Asylian21/ripemd160-asm)
[License: MIT](LICENSE)
[Free & Open Source](LICENSE)

`ripemd160mb` is a high-performance RIPEMD-160 implementation for Go, built for
Bitcoin-style HASH160 pipelines that need to process many independent 32-byte
SHA-256 digests at once.

The library combines a portable pure-Go scalar implementation with a real
Go-assembly SIMD backend. On arm64, runtime dispatch selects a 4-lane NEON
kernel, with a shorter SHA3 instruction sequence when the CPU supports it.
The fixed-32-byte scalar path is also unrolled; all `Hash32` paths allocate
nothing. The local Apple M5 Pro measurements are recorded below.

This is free and open-source software released under the permissive
MIT license.

```go
src := make([]byte, n*32) // n SHA-256 digests, contiguous
dst := make([]byte, n*ripemd160mb.Size)

ripemd160mb.Hash32(dst, src, n)
fmt.Println(ripemd160mb.Backend(), ripemd160mb.Lanes())
```

## Why This Exists

Bitcoin addresses and scripts frequently rely on HASH160:

```text
HASH160(x) = RIPEMD160(SHA256(x))
```

Most Go code computes RIPEMD-160 one message at a time. That is simple, but it
leaves throughput on the table when the workload is naturally batched: public
key analysis, wallet tooling, address derivation experiments, indexers, test
generators, benchmark suites, and cryptography education.

`ripemd160mb` focuses on that exact hot path: many fixed 32-byte inputs,
contiguous memory, no per-message allocations, runtime backend reporting, and
correctness checked against the independent `golang.org/x/crypto/ripemd160`
oracle.

It is a performance library, not a shortcut around Bitcoin security. Faster
hashing makes experiments and benchmarks better; it does not make brute force a
business model.

## Highlights

- **Batched RIPEMD-160 for Go** through `Hash32(dst, src, n)`.
- **Bitcoin HASH160 ready** for `RIPEMD160(SHA256(x))` pipelines.
- **arm64 NEON/SHA3 SIMD backends** with 4 parallel lanes and runtime dispatch.
- **Portable scalar fallback** for every supported Go architecture.
- **Zero allocations** on the `Hash32` hot path.
- **Concurrent-safe API** with no package-level mutable hashing state.
- **Reference-compatible** `New`, `Sum`, and `HashEach` APIs for general
RIPEMD-160 use.
- **Differential, property, race, fuzz, and coverage tests** enforced in CI.
- **Free and open source** under the permissive MIT license.

## Install

```sh
go get github.com/Asylian21/ripemd160-asm@v0.2.0
```

Import the root package for RIPEMD-160:

```go
import "github.com/Asylian21/ripemd160-asm"
```

Import the optional HASH160 helper when you want SHA-256 plus RIPEMD-160 in one
convenience call:

```go
import "github.com/Asylian21/ripemd160-asm/hash160"
```

To build against a local checkout before a release tag is available, add a
temporary `replace` directive to the consumer's `go.mod`:

```go
require github.com/Asylian21/ripemd160-asm v0.0.0

replace github.com/Asylian21/ripemd160-asm => /path/to/RIPEMD-160-Assembly
```

## Quick Start

Use `Hash32` when you already have `n` SHA-256 digests laid out as one
contiguous byte slice:

```go
package main

import (
	"fmt"

	"github.com/Asylian21/ripemd160-asm"
)

func main() {
	const n = 1024

	src := make([]byte, n*32) // msg i: src[i*32:(i+1)*32]
	dst := make([]byte, n*ripemd160mb.Size)

	// Fill src with 32-byte SHA-256 digests, then hash the batch.
	ripemd160mb.Hash32(dst, src, n)

	fmt.Printf("backend=%s lanes=%d first=%x\n",
		ripemd160mb.Backend(),
		ripemd160mb.Lanes(),
		dst[:ripemd160mb.Size],
	)
}
```

`Hash32` panics on programmer errors (`n < 0`, short input, or short output).
For `n == 0`, it is a no-op and accepts nil buffers.

## HASH160 Helper

The `hash160` subpackage computes Bitcoin-style
`RIPEMD160(SHA256(message))` for batches of fixed-width inputs, such as
33-byte compressed public keys:

```go
import (
	"github.com/Asylian21/ripemd160-asm"
	"github.com/Asylian21/ripemd160-asm/hash160"
)

const (
	n     = 1000
	width = 33 // compressed public keys
)

src := make([]byte, n*width)
dst := make([]byte, n*ripemd160mb.Size)

hash160.Hash160_32(dst, src, n, width)
```

`Hash160_32` is intentionally a convenience wrapper. It allocates one
intermediate `n*32` buffer and uses the standard library `crypto/sha256`. For
maximum throughput, compute SHA-256 into a reusable `n*32` buffer, then call
`ripemd160mb.Hash32` directly.

See `[examples/hash160](examples/hash160/main.go)` and the package examples for
runnable versions of both patterns.

## API

```go
const (
	Size      = 20
	BlockSize = 64
)

func Hash32(dst, src []byte, n int)
func HashEach(dst [][Size]byte, src [][]byte)
func New() hash.Hash
func Sum(p []byte) [Size]byte
func Lanes() int
func Backend() string
```

`Hash32` is the fast path. It reads `n` contiguous 32-byte messages
(`src[i*32:(i+1)*32]`) and writes `n` contiguous 20-byte digests
(`dst[i*Size:(i+1)*Size]`). It allocates nothing, keeps no state between calls,
and is safe for concurrent use from many goroutines.

`HashEach`, `New`, and `Sum` cover general RIPEMD-160 usage for arbitrary
message lengths and are byte-for-byte compatible with
`golang.org/x/crypto/ripemd160`.

## Backends

Runtime dispatch selects the fastest implemented backend once during package
initialization. `Backend()` always reports the kernel that actually executes:


| Backend     | GOARCH  | Lanes | Status                                  |
| ----------- | ------- | ----- | --------------------------------------- |
| `neon-sha3` | `arm64` | 4     | default when FEAT_SHA3 is available      |
| `neon`      | `arm64` | 4     | default on arm64 without FEAT_SHA3       |
| `scalar`    | all     | 1     | unrolled fixed-width portable fallback  |


Set `GORIPEMD160MB_FORCE=scalar`, `neon`, or `neon-sha3` to pin a backend.
Unknown or unsupported values fall back to scalar. An explicit SHA3 request
cannot bypass the CPU capability check. Darwin uses the positive
`hw.optional.arm.FEAT_SHA3` sysctl result; other arm64 platforms use `x/sys/cpu`.

amd64 SIMD kernels (SSE2, AVX2, AVX-512) are planned but not yet implemented,
so amd64 currently runs scalar. The NEON generator in
`[internal/neongen](internal/neongen)` is the reference template for adding new
kernels, and every new backend must be verified bit-for-bit against the scalar
oracle before it is wired into dispatch.

## Performance

Local Apple M5 Pro measurements, 2026-10-04:

```text
GOMAXPROCS=1
n = 2048
Go 1.22.5, darwin/arm64
6 samples per case, 200ms benchmark duration
```


| Backend   | Median ns/op | hashes/s   | allocs/op |
| --------- | ------------ | ---------- | --------- |
| scalar    | 238300       | 8,594,000  | 0         |
| neon      | 91630        | 22,350,000 | 0         |
| neon-sha3 | 82780        | 24,740,000 | 0         |


These results apply to this M5 Pro and Go 1.22.5. On Go 1.27.1 the same
`neon-sha3` case, now hashing eight messages per iteration, measures 41.85M
hashes/s at n=2048 (67.41% above the four-lane kernel, `p=0.008`, five
samples). `Lanes()` remains 4. The three accepted four-lane changes
are shorter NEON boolean sequences, SHA3 boolean instructions with adjusted
round constants, and independent preparation of the message/additive terms.
The scalar tail uses constant rotations and omits the fixed padding's zero
words. Before/after samples and `benchstat` comparisons are in
[benchmarks/2026-10-04](benchmarks/2026-10-04/README.md).

Reproduce and compare results with:

```sh
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

The full methodology, profiling workflow, and release criteria live in
`[PERFORMANCE.md](PERFORMANCE.md)`.

## Correctness and Security Posture

This repository treats performance claims as secondary to correctness:

- Known vectors, the million-`a` vector, differential tests, and fuzzing compare
against `golang.org/x/crypto/ripemd160`.
- Every implemented backend must pass the same contract tests, including lane
boundaries and scalar tail handling.
- `Hash32` is tested for zero allocations, bounds safety, and concurrent use.
- CI runs `go test`, forced-scalar tests, race tests, vet, static analysis,
coverage checks, cross-builds, and generated-assembly drift checks.

RIPEMD-160 remains a legacy hash used by Bitcoin HASH160 and compatibility
workflows. For new protocol design, prefer modern primitives selected for that
protocol's threat model. This package is for correct RIPEMD-160 compatibility
and batched performance, not for inventing new cryptographic constructions.

## Testing

```sh
go test ./...                                   # all packages, native backend
GORIPEMD160MB_FORCE=scalar go test ./...        # force the scalar oracle
go test -race ./...                             # data-race detector
GORIPEMD160MB_FORCE=neon-sha3 go test ./...       # SHA3 when supported
```

Coverage of the importable library packages (root plus `hash160`):

```sh
go test -covermode=atomic -coverprofile=coverage.out ./ ./hash160
go tool cover -func=coverage.out | tail -1
go tool cover -html=coverage.out
```

CI enforces a 95% statement-coverage floor on these two packages; both are
currently at 100%.

Fuzzing:

```sh
go test -run '^$' -fuzz '^FuzzSum$'        -fuzztime=30s .
go test -run '^$' -fuzz '^FuzzHash32$'     -fuzztime=30s .
go test -run '^$' -fuzz '^FuzzHash160_32$' -fuzztime=30s ./hash160
```

Contribution guidelines, generator rules, and the pull request checklist are in
`[CONTRIBUTING.md](CONTRIBUTING.md)`. The precise API contract is documented in
`[SPEC.md](SPEC.md)`.

## GitHub SEO

Recommended repository description:

```text
Free and open-source high-performance batched RIPEMD-160 and Bitcoin HASH160 for Go, with arm64 NEON SIMD, zero-allocation Hash32, fuzz-tested correctness, and scalar fallback.
```

Recommended GitHub topics:

```text
ripemd160, ripemd-160, hash160, bitcoin, bitcoin-address, cryptography, go, golang, open-source, foss, mit-license, simd, neon, arm64, assembly, hashing, benchmark, performance, fuzz-testing
```

Search keywords naturally covered by this repository:

```text
RIPEMD-160 Go implementation, RIPEMD160 Golang, Bitcoin HASH160 Go, batched RIPEMD-160, arm64 NEON hash, Go assembly SIMD, zero allocation hashing, Bitcoin public key hash, RIPEMD-160 benchmark, Hash160 benchmark
```

## Roadmap

- Add amd64 SIMD backends only when real SSE2, AVX2, or AVX-512 kernels beat
scalar and pass the same bit-for-bit verification suite.
- Keep `Hash32` zero-allocation and stable for high-throughput callers.
- Improve benchmark coverage across more CPUs and Go releases.
- Expand documentation around real-world HASH160 pipeline design.

## Support This Project ₿

If this project helped you understand Bitcoin security, benchmark Go code, or
explain why brute force is not a business model, you can support continued
research here:

Bitcoin donation address:

```text
bc1q9c5mmx9d3ajevjrvvw9yf52jclsre8x86qhnak
```

Every satoshi helps fund more experiments, better documentation, and fewer
hand-wavy claims about cryptography.

## License

`ripemd160mb` is free and open-source software under the MIT license.
You may use, copy, modify, and redistribute it under the terms in
[`LICENSE`](LICENSE).
