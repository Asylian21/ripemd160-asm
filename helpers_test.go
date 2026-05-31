package ripemd160mb

import (
	"crypto/rand"
	"testing"

	//lint:ignore SA1019 the deprecated x/crypto/ripemd160 is intentionally used as an independent reference oracle for differential tests
	xripemd160 "golang.org/x/crypto/ripemd160"
)

// referenceSum returns RIPEMD-160(msg) computed by the independent
// golang.org/x/crypto/ripemd160 implementation.
//
// It is used as an external oracle so correctness tests do not merely compare
// the package against itself; a shared bug in our scalar path and our fast
// path would still be caught against this third-party reference.
func referenceSum(msg []byte) []byte {
	h := xripemd160.New()
	_, _ = h.Write(msg)
	return h.Sum(nil)
}

// randomBytes returns n cryptographically random bytes, failing the test on a
// reader error instead of forcing every caller to handle it.
func randomBytes(t testing.TB, n int) []byte {
	t.Helper()
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("rand.Read(%d): %v", n, err)
	}
	return b
}

// withBackend forces the named backend for the duration of fn and always
// restores the previously active backend, even if fn panics or the test fails.
// It skips the test when the backend is not available on the current CPU so the
// matrix tests stay portable across architectures.
func withBackend(t *testing.T, name string, fn func()) {
	t.Helper()
	if !backendAvailable(name) {
		t.Skipf("backend %q not available on %s", name, Backend())
	}
	restore := forceBackendForTest(name)
	defer restore()
	fn()
}

// backendAvailable reports whether name appears in the set of backends that can
// be exercised on the current CPU.
func backendAvailable(name string) bool {
	for _, n := range availableBackendsForTest() {
		if n == name {
			return true
		}
	}
	return false
}
