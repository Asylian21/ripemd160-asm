package hash160_test

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"testing"

	"github.com/Asylian21/ripemd160-asm"
	"github.com/Asylian21/ripemd160-asm/hash160"
)

// manualHash160 computes RIPEMD160(SHA256(msg)) using the standard library and
// the package's own single-message Sum, serving as the oracle the batched
// Hash160_32 path is validated against.
func manualHash160(msg []byte) [ripemd160mb.Size]byte {
	sha := sha256.Sum256(msg)
	return ripemd160mb.Sum(sha[:])
}

func randomBytes(t testing.TB, n int) []byte {
	t.Helper()
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("rand.Read(%d): %v", n, err)
	}
	return b
}

// TestHash160_32Correctness checks the batched pipeline against the oracle for a
// matrix of message widths and counts, covering single messages, lane-aligned
// batches, and counts with a scalar tail.
func TestHash160_32Correctness(t *testing.T) {
	widths := []int{1, 20, 32, 33, 65, 128}
	counts := []int{1, 2, 3, 4, 5, 8, 16, 17, 64, 100}
	for _, width := range widths {
		for _, n := range counts {
			src := randomBytes(t, n*width)
			dst := make([]byte, n*ripemd160mb.Size)
			hash160.Hash160_32(dst, src, n, width)
			for i := 0; i < n; i++ {
				want := manualHash160(src[i*width : (i+1)*width])
				got := dst[i*ripemd160mb.Size : (i+1)*ripemd160mb.Size]
				if !bytes.Equal(got, want[:]) {
					t.Fatalf("width=%d n=%d msg=%d:\n got  %x\n want %x", width, n, i, got, want)
				}
			}
		}
	}
}

// TestHash160_32CompressedPubkey uses the canonical Bitcoin use case: 33-byte
// compressed public keys hashed to 20-byte HASH160 addresses.
func TestHash160_32CompressedPubkey(t *testing.T) {
	const (
		n     = 4
		width = 33
	)
	src := randomBytes(t, n*width)
	for i := 0; i < n; i++ {
		src[i*width] = 0x02 | byte(i&1) // valid compressed-key prefix
	}
	dst := make([]byte, n*ripemd160mb.Size)
	hash160.Hash160_32(dst, src, n, width)
	for i := 0; i < n; i++ {
		want := manualHash160(src[i*width : (i+1)*width])
		if !bytes.Equal(dst[i*ripemd160mb.Size:(i+1)*ripemd160mb.Size], want[:]) {
			t.Fatalf("pubkey %d mismatch", i)
		}
	}
}

// TestHash160_32Zero ensures a zero count is a no-op that tolerates nil buffers
// and leaves an existing destination untouched.
func TestHash160_32Zero(t *testing.T) {
	hash160.Hash160_32(nil, nil, 0, 32)

	dst := []byte{0xAA, 0xBB, 0xCC}
	hash160.Hash160_32(dst, nil, 0, 32)
	if !bytes.Equal(dst, []byte{0xAA, 0xBB, 0xCC}) {
		t.Fatalf("Hash160_32(n=0) modified dst: %x", dst)
	}
}

// TestHash160_32WidthZero documents the degenerate but valid case where each
// message is empty: every digest is RIPEMD160(SHA256("")).
func TestHash160_32WidthZero(t *testing.T) {
	const n = 3
	dst := make([]byte, n*ripemd160mb.Size)
	hash160.Hash160_32(dst, nil, n, 0)
	want := manualHash160(nil)
	for i := 0; i < n; i++ {
		if !bytes.Equal(dst[i*ripemd160mb.Size:(i+1)*ripemd160mb.Size], want[:]) {
			t.Fatalf("width=0 msg %d = %x, want %x", i, dst[i*ripemd160mb.Size:(i+1)*ripemd160mb.Size], want)
		}
	}
}

// TestHash160_32RespectsBounds verifies the function writes exactly
// n*ripemd160mb.Size bytes and does not disturb trailing sentinel bytes.
func TestHash160_32RespectsBounds(t *testing.T) {
	const (
		n     = 5
		width = 32
		guard = 8
	)
	src := randomBytes(t, n*width)
	dst := make([]byte, n*ripemd160mb.Size+guard)
	for i := range dst {
		dst[i] = 0x7E
	}
	hash160.Hash160_32(dst[:n*ripemd160mb.Size], src, n, width)
	for i := n * ripemd160mb.Size; i < len(dst); i++ {
		if dst[i] != 0x7E {
			t.Fatalf("wrote past dst[:%d] at index %d", n*ripemd160mb.Size, i)
		}
	}
}

// TestHash160_32Panics exercises every documented precondition.
func TestHash160_32Panics(t *testing.T) {
	cases := []struct {
		name string
		fn   func()
	}{
		{"negative-count", func() { hash160.Hash160_32(nil, nil, -1, 32) }},
		{"negative-width", func() { hash160.Hash160_32(nil, nil, 1, -1) }},
		{"short-src", func() { hash160.Hash160_32(make([]byte, ripemd160mb.Size), make([]byte, 10), 1, 32) }},
		{"short-dst", func() { hash160.Hash160_32(make([]byte, ripemd160mb.Size-1), make([]byte, 32), 1, 32) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatalf("expected panic for %s", tc.name)
				}
			}()
			tc.fn()
		})
	}
}

// FuzzHash160_32 derives a width and message count from arbitrary fuzz input
// and asserts every digest matches the oracle.
func FuzzHash160_32(f *testing.F) {
	f.Add([]byte("abc"), uint8(32))
	f.Add(bytes.Repeat([]byte{0x02}, 99), uint8(33))
	f.Add([]byte{}, uint8(1))
	f.Fuzz(func(t *testing.T, data []byte, w uint8) {
		width := int(w%96) + 1 // 1..96, always positive
		n := len(data) / width
		if n == 0 {
			return
		}
		src := data[:n*width]
		dst := make([]byte, n*ripemd160mb.Size)
		hash160.Hash160_32(dst, src, n, width)
		for i := 0; i < n; i++ {
			want := manualHash160(src[i*width : (i+1)*width])
			if !bytes.Equal(dst[i*ripemd160mb.Size:(i+1)*ripemd160mb.Size], want[:]) {
				t.Fatalf("width=%d n=%d msg=%d mismatch", width, n, i)
			}
		}
	})
}
