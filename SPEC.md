# ripemd160mb Specification

This repository implements a pure-Go plus Go-assembler multi-buffer RIPEMD-160
library. The primary API is `Hash32`, which hashes many independent 32-byte
messages from a contiguous input buffer into contiguous 20-byte digests.

## Public API

```go
const (
	Size      = 20 // RIPEMD-160 digest size in bytes
	BlockSize = 64 // RIPEMD-160 internal block size in bytes
)

func Hash32(dst, src []byte, n int)
func HashEach(dst [][Size]byte, src [][]byte)
func New() hash.Hash
func Sum(p []byte) [Size]byte
func Lanes() int
func Backend() string
```

The `hash160` subpackage exposes:

```go
func Hash160_32(dst, src []byte, n, width int) // RIPEMD160(SHA256(x)) batched
```

## Contract

### Hash32

- Reads `n` messages of exactly 32 bytes: message `i` is `src[i*32:(i+1)*32]`.
- Writes `n` digests of exactly `Size` bytes: digest `i` is
  `dst[i*Size:(i+1)*Size]`.
- Requires `len(src) >= n*32` and `len(dst) >= n*Size`.
- Reads at most `n*32` source bytes and writes exactly `n*Size` destination
  bytes; it never touches memory beyond those ranges.
- Allocates nothing, keeps no state between calls, and is safe for concurrent
  use by multiple goroutines.
- For all inputs, lane `i` equals `Sum(src[i*32:(i+1)*32])`. The scalar backend
  is the reference; every other backend must match it bit-for-bit.

### HashEach / New / Sum

- `Sum(p)` and `New()` implement standard RIPEMD-160 for messages of any length
  and are byte-for-byte compatible with `golang.org/x/crypto/ripemd160`.
- `HashEach` sets `dst[i] = Sum(src[i])` and requires `len(dst) == len(src)`.

### Panics

The following are programming errors and panic rather than returning an error:

- `Hash32`: `n < 0`, `len(src) < n*32`, or `len(dst) < n*Size`.
- `HashEach`: `len(dst) != len(src)`.
- `Hash160_32`: `n < 0`, `width < 0`, `len(src) < n*width`, or
  `len(dst) < n*Size`.

A count of `n == 0` is always a valid no-op and tolerates nil buffers.

## Backends

Implemented backends:

- `scalar`: pure-Go fallback and correctness oracle (1 lane). Default off arm64.
- `neon`: arm64 4-lane Advanced SIMD backend. Default on arm64.

amd64 SIMD kernels (SSE2/AVX2/AVX-512) are planned but not yet implemented;
amd64 currently runs the scalar backend. A backend is only advertised — and only
selectable — once it has a real kernel verified bit-for-bit against the scalar
oracle, so `Backend()` never reports a SIMD name while secretly running scalar.

The active backend is chosen once at package initialization.
`GORIPEMD160MB_FORCE` may be set to `scalar` or `neon` to pin a backend.
Selection rules:

- empty string or `auto`: choose the fastest backend implemented for the
  current architecture.
- an implemented backend name: use it.
- an unknown name, or a backend not implemented for the current architecture
  (for example `neon` on amd64, or any amd64 SIMD name): fall back to `scalar`.
  Selection never panics.

`Backend()` returns the active backend name and `Lanes()` returns its lane
count (always `1` for scalar, `4` for neon). The two always agree, and the
reported backend is always the kernel that actually executes.

## Quality bar

"Well tested" for this repository means all of the following hold in CI:

- Correctness is validated against the independent `golang.org/x/crypto`
  reference via known vectors, the million-`a` vector, differential tests, and
  fuzzing (`FuzzSum`, `FuzzHash32`, `FuzzHash160_32`).
- Every available backend is exercised through the same correctness suite using
  the forced-backend hooks, including the lane/tail boundary counts.
- `Hash32` is asserted to be zero-allocation and to respect its buffer bounds.
- Tests pass under the race detector and under forced-scalar execution.
- Statement coverage of the importable library packages (root + `hash160`)
  stays at or above 95%; it is currently 100%.
- `gofmt`, `go vet`, and `staticcheck` are clean, and `go generate` produces no
  diff.

The NEON code generator (`internal/neongen`) is validated by the `go generate` +
clean-tree check rather than by statement coverage.
