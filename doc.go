// Package ripemd160mb computes RIPEMD-160 hashes, with a multi-buffer
// fixed-32-byte fast path designed for Bitcoin-style Hash160 pipelines.
//
// The package offers two layers of API:
//
//   - General purpose: [New] returns a streaming [hash.Hash], and [Sum] hashes a
//     single message of any length. Both are byte-for-byte compatible with
//     golang.org/x/crypto/ripemd160.
//   - Fast path: [Hash32] hashes many independent 32-byte messages in one call,
//     and [HashEach] is a convenience wrapper for a slice of arbitrary messages.
//
// # Hash32 buffer layout
//
// Hash32 reads n messages of exactly 32 bytes from a single contiguous source
// buffer and writes n digests of exactly [Size] (20) bytes to a single
// contiguous destination buffer. Message i occupies src[i*32:(i+1)*32] and its
// digest occupies dst[i*Size:(i+1)*Size]:
//
//	src: | msg 0 (32B) | msg 1 (32B) | ... | msg n-1 (32B) |
//	dst: | dig 0 (20B) | dig 1 (20B) | ... | dig n-1 (20B) |
//
// Hash32 does not allocate, holds no state between calls, and is safe for
// concurrent use by multiple goroutines.
//
// # Panics
//
// Hash32 panics if n is negative, if len(src) < n*32, or if len(dst) < n*Size.
// HashEach panics if len(dst) != len(src). These conditions indicate a caller
// bug rather than a runtime error, so they are not reported via error values.
//
// # Backend selection
//
// At package initialization the implementation selects the fastest backend
// available in this build for the current architecture. The scalar backend is a
// pure-Go implementation that also serves as the correctness oracle for the
// vector backends. [Backend] reports the active backend name and [Lanes]
// reports how many messages it processes in parallel.
//
// Two backends are implemented today: a hand-tuned 4-lane arm64 NEON kernel
// (the default on arm64, several times faster than scalar) and the portable
// scalar fallback (the default everywhere else). amd64 SIMD kernels are planned
// but not yet implemented, so amd64 currently runs the scalar backend.
//
// The environment variable GORIPEMD160MB_FORCE may be set to scalar or neon to
// pin a specific backend. An unknown value, or a backend not implemented for
// the current architecture, falls back to scalar rather than failing; the value
// reported by [Backend] always names the kernel that actually runs.
package ripemd160mb
