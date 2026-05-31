//go:build !arm64

package ripemd160mb

// On architectures without a vector kernel in this build, the portable scalar
// implementation is the only backend. amd64 SIMD kernels (SSE2/AVX2/AVX-512)
// are planned but not yet implemented; the arm64 NEON generator in
// internal/neongen is the reference template for adding them.

func bestBackend() backend { return scalarBackend() }

// vectorBackend reports no implemented vector backends on these architectures,
// so any named request falls back to scalar.
func vectorBackend(string) (backend, bool) { return backend{}, false }
