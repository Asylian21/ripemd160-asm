package hash160_test

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/davidzita/ripemd160mb"
	"github.com/davidzita/ripemd160mb/hash160"
)

// BenchmarkHash160_32 measures the combined SHA-256 + RIPEMD-160 pipeline across
// representative batch sizes for the common 33-byte compressed-pubkey width.
func BenchmarkHash160_32(b *testing.B) {
	const width = 33
	for _, n := range []int{1, 64, 1024, 4096} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			src := make([]byte, n*width)
			if _, err := rand.Read(src); err != nil {
				b.Fatal(err)
			}
			dst := make([]byte, n*ripemd160mb.Size)
			b.SetBytes(int64(n * width))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				hash160.Hash160_32(dst, src, n, width)
			}
			b.ReportMetric(float64(n*b.N)/b.Elapsed().Seconds(), "hashes/s")
		})
	}
}
