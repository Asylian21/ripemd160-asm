package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	"github.com/davidzita/ripemd160mb"
	"github.com/davidzita/ripemd160mb/hash160"
)

func main() {
	const n = 2
	const width = 33

	pubkeys := make([]byte, n*width)
	pubkeys[0] = 0x02
	pubkeys[width] = 0x03

	out := make([]byte, n*ripemd160mb.Size)
	hash160.Hash160_32(out, pubkeys, n, width)

	for i := 0; i < n; i++ {
		sha := sha256.Sum256(pubkeys[i*width : (i+1)*width])
		check := ripemd160mb.Sum(sha[:])
		got := out[i*ripemd160mb.Size : (i+1)*ripemd160mb.Size]
		fmt.Printf("%x %t\n", got, bytes.Equal(got, check[:]))
	}
}
