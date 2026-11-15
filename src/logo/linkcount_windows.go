//go:build windows

package logo

import (
	"os"
	"syscall"
)

// nombre de liens physiques du fichier ouvert, 0 si inconnu
func linkCount(f *os.File) uint64 {
	var info syscall.ByHandleFileInformation
	if err := syscall.GetFileInformationByHandle(syscall.Handle(f.Fd()), &info); err != nil {
		return 0
	}
	return uint64(info.NumberOfLinks)
}
