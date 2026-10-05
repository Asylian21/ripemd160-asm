//go:build arm64

package ripemd160mb

import "testing"

func TestARM64SHA3Dispatch(t *testing.T) {
	if got := arm64BestBackend(false); got.name != "neon" {
		t.Fatalf("unsupported SHA3 selected %q, want neon", got.name)
	}
	if _, ok := arm64VectorBackend("neon-sha3", false); ok {
		t.Fatal("forcing SHA3 succeeded on an unsupported CPU")
	}
	if got := arm64BestBackend(true); got.name != "neon-sha3" {
		t.Fatalf("supported SHA3 selected %q, want neon-sha3", got.name)
	}
	if got, ok := arm64VectorBackend("neon-sha3", true); !ok || got.name != "neon-sha3" {
		t.Fatalf("supported SHA3 could not be selected: %+v, %v", got, ok)
	}
}
