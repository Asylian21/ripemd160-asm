package ripemd160mb

import (
	"bytes"
	"runtime"
	"sync"
	"testing"
)

// The tests in this file pin the contract that downstream multi-threaded
// pipelines (for example Bitcoin Hash160 brute-forcers) rely on: Hash32 is the
// hot path, it is zero-allocation on every backend, it is safe for concurrent
// use by many worker goroutines, the library transparently handles message
// counts that are not a multiple of the lane width, and the reported backend is
// always the real kernel that executes.

// TestHash32DefaultBackendIsFastestAvailable confirms package init selects the
// best implemented backend, so callers get SIMD by default where it exists
// (neon on arm64) and an honest scalar elsewhere.
func TestHash32DefaultBackendIsFastestAvailable(t *testing.T) {
	want := "scalar"
	if runtime.GOARCH == "arm64" {
		want = "neon"
	}
	if got := bestBackend().name; got != want {
		t.Fatalf("bestBackend() = %q, want %q on %s", got, want, runtime.GOARCH)
	}
	// The package-level active backend (chosen at init from the environment)
	// must also be a real, implemented kernel.
	if !backendAvailable(Backend()) {
		t.Fatalf("active backend %q is not implemented", Backend())
	}
}

// TestHash32ZeroAllocEveryBackend asserts the zero-allocation contract for each
// implemented backend, not just whichever one happens to be active, so a future
// kernel cannot silently start allocating on the hot path.
func TestHash32ZeroAllocEveryBackend(t *testing.T) {
	for _, name := range availableBackendsForTest() {
		withBackend(t, name, func() {
			const n = 96 // spans several lane groups plus a tail for every backend
			src := randomBytes(t, n*32)
			dst := make([]byte, n*Size)
			if allocs := testing.AllocsPerRun(200, func() { Hash32(dst, src, n) }); allocs != 0 {
				t.Fatalf("backend %q: Hash32 allocated %v times per run, want 0", name, allocs)
			}
		})
	}
}

// TestHash32TailHandling verifies the library itself splits a batch into the
// vectorized body and a scalar tail when n is not a multiple of the lane width:
// callers never have to align n. Every lane, body or tail, must match the
// independent reference.
func TestHash32TailHandling(t *testing.T) {
	for _, name := range availableBackendsForTest() {
		withBackend(t, name, func() {
			lanes := Lanes()
			// Counts straddling one, two and three lane groups, including the
			// pure-tail cases below a full group.
			for _, n := range []int{1, lanes - 1, lanes, lanes + 1, 2*lanes - 1, 2 * lanes, 2*lanes + 1, 3*lanes + 2} {
				if n < 1 {
					continue
				}
				src := randomBytes(t, n*32)
				dst := make([]byte, n*Size)
				Hash32(dst, src, n)
				for i := 0; i < n; i++ {
					want := referenceSum(src[i*32 : (i+1)*32])
					if !bytes.Equal(dst[i*Size:(i+1)*Size], want) {
						t.Fatalf("backend %q n=%d lane=%d mismatch", name, n, i)
					}
				}
			}
		})
	}
}

// TestHash32ConcurrentWorkers stresses the documented thread-safety guarantee:
// many goroutines call the shared, stateless Hash32 simultaneously and every
// digest must still match the reference. Run under -race this also proves the
// hot path holds no shared mutable state.
func TestHash32ConcurrentWorkers(t *testing.T) {
	const (
		workers      = 16
		iterations   = 64
		messages     = 50 // not a multiple of any lane width, to exercise tails
		messageBytes = messages * 32
	)

	// Precompute the expected digests once with the scalar oracle.
	src := randomBytes(t, messageBytes)
	want := make([]byte, messages*Size)
	for i := 0; i < messages; i++ {
		d := Sum(src[i*32 : (i+1)*32])
		copy(want[i*Size:], d[:])
	}

	var wg sync.WaitGroup
	errs := make(chan string, workers)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			dst := make([]byte, messages*Size) // each worker owns its output
			for it := 0; it < iterations; it++ {
				Hash32(dst, src, messages)
				if !bytes.Equal(dst, want) {
					errs <- "concurrent Hash32 produced an incorrect digest"
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	if msg, bad := <-errs; bad {
		t.Fatal(msg)
	}
}
