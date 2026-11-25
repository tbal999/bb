package main

// Windows has no umask. bb only targets Linux; this file just lets the tests build on Windows.
func setUmask(mask int) int { return 0 }
