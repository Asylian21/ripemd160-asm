package ripemd160mb

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"testing"

	//lint:ignore SA1019 the deprecated x/crypto/ripemd160 is intentionally used as an independent reference oracle for differential tests
	xripemd160 "golang.org/x/crypto/ripemd160"
)

func TestKnownVectors(t *testing.T) {
	tests := []struct {
		msg string
		sum string
	}{
		{"", "9c1185a5c5e9fc54612808977ee8f548b2258d31"},
		{"a", "0bdc9d2d256b3ee9daae347be6f4dc835a467ffe"},
		{"abc", "8eb208f7e05d987a9b044a8e98c6b087f15a0bfc"},
		{"message digest", "5d0689ef49d2fae572b881b123a85ffa21595f36"},
		{"abcdefghijklmnopqrstuvwxyz", "f71c27109c692c1b56bbdceb5b9d2865b3708dbc"},
		{"abcdbcdecdefdefgefghfghighijhijkijkljklmklmnlmnomnopnopq", "12a053384a9c0c88e405a06c27dcf49ada62eb2b"},
		{"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789", "b0e20b6e3116640286ed3a87a5713079b21f5189"},
	}

	for _, tc := range tests {
		want, err := hex.DecodeString(tc.sum)
		if err != nil {
			t.Fatal(err)
		}
		got := Sum([]byte(tc.msg))
		if !bytes.Equal(got[:], want) {
			t.Fatalf("Sum(%q) = %x, want %x", tc.msg, got, want)
		}

		h := New()
		_, _ = h.Write([]byte(tc.msg))
		if got := h.Sum(nil); !bytes.Equal(got, want) {
			t.Fatalf("New().Sum(%q) = %x, want %x", tc.msg, got, want)
		}
	}
}

func TestMillionA(t *testing.T) {
	msg := bytes.Repeat([]byte("a"), 1_000_000)
	want, _ := hex.DecodeString("52783243c1697bdbe16d37f97f68f08325dc1528")
	got := Sum(msg)
	if !bytes.Equal(got[:], want) {
		t.Fatalf("million-a Sum = %x, want %x", got, want)
	}
}

func TestDifferentialAgainstXCrypto(t *testing.T) {
	sizes := []int{0, 1, 2, 3, 7, 31, 32, 33, 55, 56, 57, 63, 64, 65, 127, 128, 255, 1024, 4097}
	for _, size := range sizes {
		msg := make([]byte, size)
		if _, err := rand.Read(msg); err != nil {
			t.Fatal(err)
		}
		want := xripemd160.New()
		_, _ = want.Write(msg)
		got := Sum(msg)
		if !bytes.Equal(got[:], want.Sum(nil)) {
			t.Fatalf("size %d: got %x, want %x", size, got, want.Sum(nil))
		}
	}
}

func TestHash32Backends(t *testing.T) {
	for _, name := range availableBackendsForTest() {
		t.Run(name, func(t *testing.T) {
			restore := forceBackendForTest(name)
			defer restore()
			lane := Lanes()
			for n := 0; n <= 2*lane+3; n++ {
				src := make([]byte, n*32)
				if _, err := rand.Read(src); err != nil {
					t.Fatal(err)
				}
				dst := make([]byte, n*Size)
				Hash32(dst, src, n)
				for i := 0; i < n; i++ {
					want := Sum(src[i*32 : (i+1)*32])
					if !bytes.Equal(dst[i*Size:(i+1)*Size], want[:]) {
						t.Fatalf("backend %s n=%d i=%d: got %x want %x", name, n, i, dst[i*Size:(i+1)*Size], want)
					}
				}
			}
		})
	}
}

func TestHashEach(t *testing.T) {
	src := [][]byte{
		nil,
		[]byte("a"),
		[]byte("abc"),
		bytes.Repeat([]byte{0x42}, 1000),
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

func TestHash32Allocs(t *testing.T) {
	src := make([]byte, 64*32)
	dst := make([]byte, 64*Size)
	allocs := testing.AllocsPerRun(1000, func() {
		Hash32(dst, src, 64)
	})
	if allocs != 0 {
		t.Fatalf("Hash32 allocated: %f", allocs)
	}
}

func TestHash32Panics(t *testing.T) {
	tests := []struct {
		name string
		fn   func()
	}{
		{"negative", func() { Hash32(nil, nil, -1) }},
		{"short-src", func() { Hash32(make([]byte, Size), nil, 1) }},
		{"short-dst", func() { Hash32(nil, make([]byte, 32), 1) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatalf("expected panic")
				}
			}()
			tc.fn()
		})
	}
}

func FuzzSum(f *testing.F) {
	for _, seed := range [][]byte{
		nil,
		[]byte("a"),
		[]byte("abc"),
		bytes.Repeat([]byte("a"), 100),
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, msg []byte) {
		want := xripemd160.New()
		_, _ = want.Write(msg)
		got := Sum(msg)
		if !bytes.Equal(got[:], want.Sum(nil)) {
			t.Fatalf("got %x, want %x", got, want.Sum(nil))
		}
	})
}
