//go:build !windows

package main

import (
	"os"
	"syscall"
)

// lockFile takes an advisory lock on f until it is closed.
func lockFile(f *os.File, exclusive bool) error {
	how := syscall.LOCK_SH
	if exclusive {
		how = syscall.LOCK_EX
	}
	return syscall.Flock(int(f.Fd()), how)
}
