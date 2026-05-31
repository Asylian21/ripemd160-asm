package hash160_test

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	"github.com/davidzita/ripemd160mb"
	"github.com/davidzita/ripemd160mb/hash160"
)

// Example_hash160_32 computes Bitcoin-style HASH160 = RIPEMD160(SHA256(x)) for
// a batch of fixed-width public keys and verifies the result against the
// equivalent step-by-step computation.
//
// It is declared as a package-level example because Go's example-naming rules
// reserve the underscore in ExampleX_y for an optional suffix, which collides
// with the Hash160_32 function name.
func Example_hash160_32() {
	const (
		n     = 2
		width = 33 // compressed public keys
	)
	pubkeys := make([]byte, n*width)
	pubkeys[0] = 0x02
	pubkeys[width] = 0x03

	dst := make([]byte, n*ripemd160mb.Size)
	hash160.Hash160_32(dst, pubkeys, n, width)

	match := true
	for i := 0; i < n; i++ {
		sha := sha256.Sum256(pubkeys[i*width : (i+1)*width])
		want := ripemd160mb.Sum(sha[:])
		if !bytes.Equal(dst[i*ripemd160mb.Size:(i+1)*ripemd160mb.Size], want[:]) {
			match = false
		}
	}
	fmt.Println(match)
	// Output: true
}
