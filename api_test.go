package ripemd160mb

import (
	"bytes"
	"testing"
)

// TestHashEachMatchesSum checks the convenience HashEach helper against the
// single-message oracle for a spread of message lengths, including the empty
// and nil cases.
func TestHashEachMatchesSum(t *testing.T) {
	src := [][]byte{
		nil,
		{},
		[]byte("a"),
		[]byte("abc"),
		randomBytes(t, 32),
		randomBytes(t, 55),
		randomBytes(t, 64),
		randomBytes(t, 65),
		bytes.Repeat([]byte{0x42}, 4096),
	}
	dst := make([][Size]byte, len(src))
	HashEach(dst, src)
	for i := range src {
		want := Sum(src[i])
		if dst[i] != want {
			t.Fatalf("HashEach[%d] = %x, want %x", i, dst[i], want)
		}
	}
}

// TestHashEachLengthMismatchPanics covers the documented precondition that the
// destination and source slices must have equal length.
func TestHashEachLengthMismatchPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on len(dst) != len(src)")
		}
	}()
	HashEach(make([][Size]byte, 1), make([][]byte, 2))
}

// TestHashEachEmpty confirms hashing zero messages is a valid no-op.
func TestHashEachEmpty(t *testing.T) {
	HashEach(nil, nil)
	HashEach([][Size]byte{}, [][]byte{})
}

// TestStreamingMatchesReference exercises the hash.Hash implementation with a
// variety of write chunk sizes to cover the internal block-buffering paths
// (partial block carry-over, full blocks, and tail bytes), comparing the result
// to the x/crypto reference.
func TestStreamingMatchesReference(t *testing.T) {
	message := randomBytes(t, 1000)
	chunkSizes := []int{1, 3, 7, 31, 32, 63, 64, 65, 127, 128, 999, 1000}
	for _, chunk := range chunkSizes {
		h := New()
		for off := 0; off < len(message); off += chunk {
			end := off + chunk
			if end > len(message) {
				end = len(message)
			}
			n, err := h.Write(message[off:end])
			if err != nil || n != end-off {
				t.Fatalf("chunk=%d Write returned (%d, %v)", chunk, n, err)
			}
		}
		got := h.Sum(nil)
		want := referenceSum(message)
		if !bytes.Equal(got, want) {
			t.Fatalf("chunk=%d: got %x, want %x", chunk, got, want)
		}
	}
}

// TestSumIsRepeatable verifies that hash.Hash.Sum does not mutate the digest
// state: calling it twice yields the same value, and writing more data
// afterwards continues from the prior state rather than a corrupted one.
func TestSumIsRepeatable(t *testing.T) {
	h := New()
	_, _ = h.Write([]byte("abc"))

	first := h.Sum(nil)
	second := h.Sum(nil)
	if !bytes.Equal(first, second) {
		t.Fatalf("Sum not repeatable: %x vs %x", first, second)
	}

	_, _ = h.Write([]byte("def"))
	combined := h.Sum(nil)
	if !bytes.Equal(combined, referenceSum([]byte("abcdef"))) {
		t.Fatalf("writing after Sum produced %x, want hash of \"abcdef\"", combined)
	}
}

// TestSumAppendsToPrefix documents that Sum appends the digest to its argument
// rather than overwriting it, matching the standard hash.Hash contract.
func TestSumAppendsToPrefix(t *testing.T) {
	h := New()
	_, _ = h.Write([]byte("abc"))
	prefix := []byte("digest:")
	out := h.Sum(prefix)
	if !bytes.HasPrefix(out, prefix) {
		t.Fatalf("Sum did not preserve prefix: %q", out)
	}
	if got := out[len(prefix):]; !bytes.Equal(got, referenceSum([]byte("abc"))) {
		t.Fatalf("appended digest = %x", got)
	}
}

// TestResetReusesDigest confirms a hash can be reset and reused to produce the
// same result as a freshly constructed one.
func TestResetReusesDigest(t *testing.T) {
	h := New()
	_, _ = h.Write([]byte("garbage that should be discarded"))
	h.Reset()
	_, _ = h.Write([]byte("abc"))
	if got := h.Sum(nil); !bytes.Equal(got, referenceSum([]byte("abc"))) {
		t.Fatalf("after Reset got %x, want hash of \"abc\"", got)
	}
}

// TestHashMetadata pins the advertised digest and block sizes, both through the
// package constants and the hash.Hash methods.
func TestHashMetadata(t *testing.T) {
	if Size != 20 {
		t.Fatalf("Size = %d, want 20", Size)
	}
	if BlockSize != 64 {
		t.Fatalf("BlockSize = %d, want 64", BlockSize)
	}
	h := New()
	if got := h.Size(); got != Size {
		t.Fatalf("h.Size() = %d, want %d", got, Size)
	}
	if got := h.BlockSize(); got != BlockSize {
		t.Fatalf("h.BlockSize() = %d, want %d", got, BlockSize)
	}
}
