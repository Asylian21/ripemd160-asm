//go:build arm64 && !darwin

package ripemd160mb

import "golang.org/x/sys/cpu"

func sha3Available() bool { return cpu.ARM64.HasSHA3 }
