package ripemd160mb

import (
	"bytes"
	"testing"
)

// laneCounts returns a representative set of message counts for the active
// backend, focused on the boundaries where Hash32 switches between the
// vectorized body and the scalar tail: zero, a single message, the values
// straddling one and two full lane groups, and a few larger batches.
func laneCounts(lanes int) []int {
	set := map[int]struct{}{
		0: {}, 1: {}, 2: {},
		64: {}, 100: {}, 257: {},
	}
	for _, base := range []int{lanes, 2 * lanes, 3 * lanes} {
		set[base] = struct{}{}
		if base-1 >= 0 {
			set[base-1] = struct{}{}
		}
		set[base+1] = struct{}{}
	}
	out := make([]int, 0, len(set))
	for n := range set {
		out = append(out, n)
	}
	return out
}

// TestHash32MatchesReference is the primary correctness gate for the fast path:
// for every available backend and a range of batch sizes spanning the lane/tail
// boundaries, each 20-byte output lane must equal the digest produced by the
// independent x/crypto reference for the matching 32-byte input slice.
func TestHash32MatchesReference(t *testing.T) {
	for _, name := range availableBackendsForTest() {
		t.Run(name, func(t *testing.T) {
			withBackend(t, name, func() {
				for _, n := range laneCounts(Lanes()) {
					src := randomBytes(t, n*32)
					dst := make([]byte, n*Size)
					Hash32(dst, src, n)
					for i := 0; i < n; i++ {
						got := dst[i*Size : (i+1)*Size]
						want := referenceSum(src[i*32 : (i+1)*32])
						if !bytes.Equal(got, want) {
							t.Fatalf("backend=%s n=%d lane=%d:\n got  %x\n want %x", name, n, i, got, want)
						}
					}
				}
			})
		})
	}
}

// TestHash32AgreesWithSum verifies that the batched fast path and the
// single-message Sum entry point are interchangeable for 32-byte inputs across
// every backend, which is the invariant Hash160 pipelines rely on.
func TestHash32AgreesWithSum(t *testing.T) {
	src := randomBytes(t, 40*32)
	for _, name := range availableBackendsForTest() {
		withBackend(t, name, func() {
			dst := make([]byte, 40*Size)
			Hash32(dst, src, 40)
			for i := 0; i < 40; i++ {
				want := Sum(src[i*32 : (i+1)*32])
				if !bytes.Equal(dst[i*Size:(i+1)*Size], want[:]) {
					t.Fatalf("backend=%s lane=%d: Hash32 disagrees with Sum", name, i)
				}
			}
		})
	}
}

// TestHash32Zero documents that a zero count is a no-op that tolerates nil
// buffers and never reads or writes memory.
func TestHash32Zero(t *testing.T) {
	Hash32(nil, nil, 0)

	dst := []byte{0x11, 0x22, 0x33}
	Hash32(dst, nil, 0)
	if !bytes.Equal(dst, []byte{0x11, 0x22, 0x33}) {
		t.Fatalf("Hash32(n=0) modified dst: %x", dst)
	}
}

// TestHash32RespectsBounds guarantees Hash32 writes exactly n*Size bytes and
// reads exactly n*32 bytes, leaving any surrounding sentinel bytes untouched
// even when the caller passes larger backing arrays.
func TestHash32RespectsBounds(t *testing.T) {
	const n = 9
	const guard = 16

	src := make([]byte, n*32+guard)
	copy(src, randomBytes(t, n*32))
	for i := n * 32; i < len(src); i++ {
		src[i] = 0xAB // poison: reading these would corrupt the result
	}

	dst := make([]byte, n*Size+guard)
	for i := range dst {
		dst[i] = 0xCD
	}

	Hash32(dst[:n*Size], src[:n*32], n)

	for i := n * Size; i < len(dst); i++ {
		if dst[i] != 0xCD {
			t.Fatalf("Hash32 wrote past dst[:%d] at index %d", n*Size, i)
		}
	}
	for i := 0; i < n; i++ {
		want := referenceSum(src[i*32 : (i+1)*32])
		if !bytes.Equal(dst[i*Size:(i+1)*Size], want) {
			t.Fatalf("lane %d incorrect", i)
		}
	}
}

// TestHash32Deterministic confirms repeated calls on identical input yield
// byte-identical output (no hidden state between invocations).
func TestHash32Deterministic(t *testing.T) {
	src := randomBytes(t, 12*32)
	first := make([]byte, 12*Size)
	second := make([]byte, 12*Size)
	Hash32(first, src, 12)
	Hash32(second, src, 12)
	if !bytes.Equal(first, second) {
		t.Fatal("Hash32 produced different output for identical input")
	}
}

// FuzzHash32 explores arbitrary inputs: the body is split into 32-byte
// messages, hashed through every available backend, and each lane is compared
// against the independent reference implementation.
func FuzzHash32(f *testing.F) {
	for _, seed := range [][]byte{
		nil,
		make([]byte, 32),
		bytes.Repeat([]byte{0xFF}, 32),
		bytes.Repeat([]byte("seed"), 40),
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		n := len(data) / 32
		if n == 0 {
			return
		}
		src := data[:n*32]
		for _, name := range availableBackendsForTest() {
			func() {
				restore := forceBackendForTest(name)
				defer restore()
				dst := make([]byte, n*Size)
				Hash32(dst, src, n)
				for i := 0; i < n; i++ {
					want := referenceSum(src[i*32 : (i+1)*32])
					if !bytes.Equal(dst[i*Size:(i+1)*Size], want) {
						t.Fatalf("backend=%s n=%d lane=%d mismatch", name, n, i)
					}
				}
			}()
		}
	})
}
