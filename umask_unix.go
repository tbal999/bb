//go:build !windows
// +build !windows

package main

import "syscall"

func setUmask(mask int) int { return syscall.Umask(mask) }
