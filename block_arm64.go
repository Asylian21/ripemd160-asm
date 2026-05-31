//go:build arm64

package ripemd160mb

// On arm64, Advanced SIMD (NEON) is mandatory per the ARMv8-A architecture, so
// the NEON backend is always available and is the default.

func bestBackend() backend { return neonBackend() }

func neonBackend() backend {
	return backend{name: "neon", lanes: 4, hash32: hash32NEON}
}

// vectorBackend reports the named vector backend implemented on this
// architecture. Only "neon" is implemented on arm64.
func vectorBackend(name string) (backend, bool) {
	if name == "neon" {
		return neonBackend(), true
	}
	return backend{}, false
}
