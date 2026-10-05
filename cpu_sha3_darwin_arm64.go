//go:build darwin && arm64

package ripemd160mb

import "syscall"

// x/sys v0.21 does not probe Darwin's optional ARM64 features. The positive
// sysctl check protects the SHA3 instructions on older or unknown CPUs.
func sha3Available() bool {
	feature, err := syscall.SysctlUint32("hw.optional.arm.FEAT_SHA3")
	return err == nil && feature == 1
}
