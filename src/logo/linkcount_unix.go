//go:build unix

package logo

import (
	"os"
	"syscall"
)

// nombre de liens physiques du fichier ouvert, 0 si inconnu. passe par Stat et non
// par Fd, qui repasserait le descripteur en mode bloquant
func linkCount(f *os.File) uint64 {
	fi, err := f.Stat()
	if err != nil {
		return 0
	}
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		return uint64(st.Nlink)
	}
	return 0
}
