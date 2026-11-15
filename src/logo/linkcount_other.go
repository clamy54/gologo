//go:build !unix && !windows

package logo

import "os"

// pas de liens physiques a craindre
func linkCount(f *os.File) uint64 { return 1 }
