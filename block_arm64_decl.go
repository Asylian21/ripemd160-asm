//go:build arm64

package ripemd160mb

//go:noescape
func hash32NEON(dst, src []byte, n int)

// hash32NEONSHA3 requires FEAT_SHA3. Dispatch verifies CPU support before
// selecting it, including when the backend is requested explicitly.
//
//go:noescape
func hash32NEONSHA3(dst, src []byte, n int)

//go:noescape
func hash32NEON8(dst, src []byte, n int)

// hash32NEON8SHA3 requires FEAT_SHA3, same as hash32NEONSHA3.
//
//go:noescape
func hash32NEON8SHA3(dst, src []byte, n int)
