package ripemd160mb

import (
	"fmt"
	"hash"
)

const (
	// Size is the size, in bytes, of a RIPEMD-160 digest.
	Size = 20

	// BlockSize is RIPEMD-160's internal block size in bytes.
	BlockSize = 64
)

type hash32Func func(dst, src []byte, n int)

type backend struct {
	name   string
	lanes  int
	hash32 hash32Func
}

var active = backend{
	name:   "scalar",
	lanes:  1,
	hash32: scalarHash32,
}

// Hash32 hashes n independent 32-byte messages from src into dst.
//
// The buffers are contiguous: message i is src[i*32:(i+1)*32] and its digest is
// written to dst[i*Size:(i+1)*Size]. src must contain at least n*32 bytes and
// dst at least n*Size bytes. Hash32 does not allocate and is safe for
// concurrent use by multiple goroutines.
//
// Hash32 panics if n is negative, if len(src) < n*32, or if len(dst) < n*Size.
// A count of zero is a no-op and tolerates nil buffers.
func Hash32(dst, src []byte, n int) {
	if n < 0 {
		panic("ripemd160mb: negative message count")
	}
	needSrc := n * 32
	needDst := n * Size
	if len(src) < needSrc {
		panic(fmt.Sprintf("ripemd160mb: src too short for %d 32-byte messages", n))
	}
	if len(dst) < needDst {
		panic(fmt.Sprintf("ripemd160mb: dst too short for %d RIPEMD-160 digests", n))
	}
	if n == 0 {
		return
	}

	b := active
	if b.lanes <= 1 {
		scalarHash32(dst[:needDst], src[:needSrc], n)
		return
	}

	vecN := n - n%b.lanes
	if vecN > 0 {
		b.hash32(dst[:vecN*Size], src[:vecN*32], vecN)
	}
	if vecN != n {
		scalarHash32(dst[vecN*Size:needDst], src[vecN*32:needSrc], n-vecN)
	}
}

// HashEach hashes each source message independently into the corresponding dst
// element, where dst[i] = Sum(src[i]). Unlike Hash32, the messages may have
// arbitrary lengths.
//
// HashEach panics if len(dst) != len(src).
func HashEach(dst [][Size]byte, src [][]byte) {
	if len(dst) != len(src) {
		panic("ripemd160mb: len(dst) must equal len(src)")
	}
	for i := range src {
		dst[i] = Sum(src[i])
	}
}

// Sum returns the RIPEMD-160 digest of p.
func Sum(p []byte) [Size]byte {
	var d digest
	d.Reset()
	_, _ = d.Write(p)
	var out [Size]byte
	d.checkSum(out[:])
	return out
}

// New returns a new RIPEMD-160 hash.Hash.
func New() hash.Hash {
	d := new(digest)
	d.Reset()
	return d
}

// Lanes returns the number of messages processed in parallel by the active
// backend. It returns 1 for the scalar backend.
func Lanes() int { return active.lanes }

// Backend returns the active backend name.
func Backend() string { return active.name }
