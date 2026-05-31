package ripemd160mb

import "runtime"

func forceBackendForTest(name string) (restore func()) {
	prev := active
	active = selectBackend(name)
	return func() { active = prev }
}

// availableBackendsForTest lists the backends that are actually implemented and
// runnable on the current build, so the matrix tests only exercise real
// kernels. Scalar is always present; neon is present on arm64.
func availableBackendsForTest() []string {
	names := []string{"scalar"}
	if runtime.GOARCH == "arm64" {
		names = append(names, "neon")
	}
	return names
}
