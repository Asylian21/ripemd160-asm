// Package hash160 combines SHA-256 and ripemd160mb.Hash32 for Bitcoin-style
// HASH160 pipelines.
package hash160

import (
	"crypto/sha256"

	"github.com/Asylian21/ripemd160-asm"
)

// Hash160_32 computes the Bitcoin-style HASH160 of n fixed-width messages.
//
// Each source message is width bytes long and laid out contiguously, so message
// i is src[i*width:(i+1)*width]. For each one the function computes
// RIPEMD160(SHA256(message)) and writes the 20-byte result to
// dst[i*ripemd160mb.Size:(i+1)*ripemd160mb.Size].
//
// src must contain at least n*width bytes and dst at least n*ripemd160mb.Size
// bytes. Hash160_32 panics if n or width is negative, or if either buffer is
// too short; a count of zero is a no-op.
//
// Hash160_32 is a convenience wrapper, not a hot-path primitive: it allocates a
// single intermediate buffer of n*32 bytes for the SHA-256 digests on every
// call, and it computes SHA-256 with crypto/sha256. High-throughput pipelines
// should instead compute the SHA-256 digests themselves — for example with a
// vectorized SHA-256 implementation — into a reusable n*32 buffer and call
// [github.com/Asylian21/ripemd160-asm.Hash32] directly, which allocates nothing
// and is safe for concurrent use. See the package example for that pattern.
func Hash160_32(dst, src []byte, n, width int) {
	if n < 0 {
		panic("hash160: negative message count")
	}
	if width < 0 {
		panic("hash160: negative message width")
	}
	if len(src) < n*width {
		panic("hash160: src too short")
	}
	if len(dst) < n*ripemd160mb.Size {
		panic("hash160: dst too short")
	}
	if n == 0 {
		return
	}

	digests := make([]byte, n*32)
	for i := 0; i < n; i++ {
		sum := sha256.Sum256(src[i*width : (i+1)*width])
		copy(digests[i*32:(i+1)*32], sum[:])
	}
	ripemd160mb.Hash32(dst, digests, n)
}
