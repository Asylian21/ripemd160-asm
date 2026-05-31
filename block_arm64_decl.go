//go:build arm64

package ripemd160mb

//go:noescape
func hash32NEON(dst, src []byte, n int)
