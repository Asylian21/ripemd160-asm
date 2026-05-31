package ripemd160mb

import (
	"crypto/rand"
	"fmt"
	"testing"
)

// benchCounts returns batch sizes spanning the lane/tail boundaries (one short
// of a lane group, exactly aligned, and one over) plus a few larger batches, so
// regressions in either the vector body or the scalar tail are visible.
func benchCounts(lanes int) []int {
	ns := []int{1}
	if lanes > 1 {
		ns = append(ns, lanes-1, lanes, lanes+1)
	}
	return append(ns, 64, 1024, 2048)
}

func BenchmarkHash32(b *testing.B) {
	backends := availableBackendsForTest()
	for _, backendName := range backends {
		restore := forceBackendForTest(backendName)
		lanes := Lanes()
		restore()

		for _, n := range benchCounts(lanes) {
			b.Run(fmt.Sprintf("%s/n=%d", backendName, n), func(b *testing.B) {
				restore := forceBackendForTest(backendName)
				defer restore()
				src := make([]byte, n*32)
				dst := make([]byte, n*Size)
				if _, err := rand.Read(src); err != nil {
					b.Fatal(err)
				}
				b.SetBytes(int64(n * 32))
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					Hash32(dst, src, n)
				}
				b.ReportMetric(float64(n*b.N)/b.Elapsed().Seconds(), "hashes/s")
			})
		}
	}
}

func BenchmarkHashEach(b *testing.B) {
	for _, n := range []int{1, 64, 1024} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			src := make([][]byte, n)
			for i := range src {
				msg := make([]byte, 32)
				if _, err := rand.Read(msg); err != nil {
					b.Fatal(err)
				}
				src[i] = msg
			}
			dst := make([][Size]byte, n)
			b.SetBytes(int64(n * 32))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				HashEach(dst, src)
			}
			b.ReportMetric(float64(n*b.N)/b.Elapsed().Seconds(), "hashes/s")
		})
	}
}
