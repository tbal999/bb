package main

import "os"

// bb only targets Linux; on Windows (where the unit tests also run) locking is skipped.
func lockFile(f *os.File, exclusive bool) error { return nil }
