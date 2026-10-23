//go:build arm64

package ripemd160mb

// Advanced SIMD (NEON) is mandatory on arm64. CPUs advertising FEAT_SHA3 use
// the extended NEON kernel; all other arm64 CPUs use the base NEON kernel.

func bestBackend() backend { return arm64BestBackend(sha3Available()) }

func arm64BestBackend(sha3 bool) backend {
	if sha3 {
		return neonSHA3Backend()
	}
	return neonBackend()
}

func neonBackend() backend {
	return backend{name: "neon", lanes: 4, hash32: hash32NEONWide}
}

func neonSHA3Backend() backend {
	return backend{name: "neon-sha3", lanes: 4, hash32: hash32NEONSHA3Wide}
}

// The reported lane count stays 4: a multiple of 4 but not of 8 is finished by
// the four-message kernel, and 1..3 messages stay on the scalar tail. Batches
// of 8 or more run two independent four-lane groups per iteration.
// Hash32 has validated the buffers; the kernels access exactly n messages.
// Concrete calls avoid function-value dispatch and aligned calls need no
// additional slice bounds checks.
func hash32NEONWide(dst, src []byte, n int) {
	n8 := n &^ 7
	if n8 == n {
		hash32NEON8(dst, src, n)
		return
	}
	if n8 == 0 {
		hash32NEON(dst, src, n)
		return
	}
	hash32NEON8(dst, src, n8)
	hash32NEON(dst[n8*Size:], src[n8*32:], n-n8)
}

func hash32NEONSHA3Wide(dst, src []byte, n int) {
	n8 := n &^ 7
	if n8 == n {
		hash32NEON8SHA3(dst, src, n)
		return
	}
	if n8 == 0 {
		hash32NEONSHA3(dst, src, n)
		return
	}
	hash32NEON8SHA3(dst, src, n8)
	hash32NEONSHA3(dst[n8*Size:], src[n8*32:], n-n8)
}

// vectorBackend only exposes kernels that are safe on the running CPU. An
// explicit "neon-sha3" request cannot bypass the capability check.
func vectorBackend(name string) (backend, bool) {
	return arm64VectorBackend(name, sha3Available())
}

func arm64VectorBackend(name string, sha3 bool) (backend, bool) {
	if name == "neon" {
		return neonBackend(), true
	}
	if name == "neon-sha3" && sha3 {
		return neonSHA3Backend(), true
	}
	return backend{}, false
}
