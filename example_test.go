package ripemd160mb_test

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"

	"github.com/Asylian21/ripemd160-asm"
)

// ExampleSum hashes a single message with the one-shot helper.
func ExampleSum() {
	sum := ripemd160mb.Sum([]byte("abc"))
	fmt.Printf("%x\n", sum)
	// Output: 8eb208f7e05d987a9b044a8e98c6b087f15a0bfc
}

// ExampleNew uses the streaming hash.Hash interface, which accepts arbitrary
// messages written in any number of chunks.
func ExampleNew() {
	h := ripemd160mb.New()
	io.WriteString(h, "ab")
	io.WriteString(h, "c")
	fmt.Printf("%x\n", h.Sum(nil))
	// Output: 8eb208f7e05d987a9b044a8e98c6b087f15a0bfc
}

// ExampleHash32 shows the multi-buffer fast path: several 32-byte messages are
// laid out contiguously and hashed in a single call. Each 20-byte output lane
// equals the single-message Sum of the matching input slice.
func ExampleHash32() {
	const n = 3
	src := make([]byte, n*32)
	for i := range src {
		src[i] = byte(i)
	}

	dst := make([]byte, n*ripemd160mb.Size)
	ripemd160mb.Hash32(dst, src, n)

	consistent := true
	for i := 0; i < n; i++ {
		want := ripemd160mb.Sum(src[i*32 : (i+1)*32])
		if !bytes.Equal(dst[i*ripemd160mb.Size:(i+1)*ripemd160mb.Size], want[:]) {
			consistent = false
		}
	}
	fmt.Println(consistent)
	// Output: true
}

// Example_hash160HotPath shows the recommended high-throughput HASH160 pattern,
// the one a Bitcoin-style brute-force pipeline should use: compute the SHA-256
// digests yourself (here with crypto/sha256, but typically a vectorized SHA-256)
// into a single reusable n*32 buffer, then call Hash32 once. Hash32 allocates
// nothing and is safe to call from many worker goroutines, so the only work per
// batch is the hashing itself.
func Example_hash160HotPath() {
	const (
		n     = 8
		width = 33 // compressed public keys
	)
	keys := make([]byte, n*width)
	for i := range keys {
		keys[i] = byte(i)
	}

	// Reused across batches in a real worker; shown once here.
	digests := make([]byte, n*32) // SHA-256 outputs, contiguous
	out := make([]byte, n*ripemd160mb.Size)

	for i := 0; i < n; i++ {
		sum := sha256.Sum256(keys[i*width : (i+1)*width])
		copy(digests[i*32:], sum[:])
	}
	ripemd160mb.Hash32(out, digests, n)

	ok := true
	for i := 0; i < n; i++ {
		sum := sha256.Sum256(keys[i*width : (i+1)*width])
		want := ripemd160mb.Sum(sum[:])
		if !bytes.Equal(out[i*ripemd160mb.Size:(i+1)*ripemd160mb.Size], want[:]) {
			ok = false
		}
	}
	fmt.Println(ok)
	// Output: true
}

// ExampleBackend reports which kernel is active and how many messages it hashes
// per call. A diagnostic banner like this lets a caller confirm at startup
// whether it is running the SIMD or the scalar path.
func ExampleBackend() {
	// In real code: log.Printf("ripemd160mb backend=%s lanes=%d", ...)
	_ = ripemd160mb.Backend() // e.g. "neon" on arm64, "scalar" elsewhere
	lanes := ripemd160mb.Lanes()
	fmt.Println(lanes >= 1)
	// Output: true
}
