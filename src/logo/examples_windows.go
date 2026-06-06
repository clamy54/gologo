//go:build windows

package logo

import (
	"os"
	"path/filepath"
)

// sous Windows, "examples" est installe a cote de l'executable (chemin reel selon
// le dossier d'install choisi dans Inno Setup). repli sur "examples" si l'executable
// est introuvable
func platformExamplesDir() string {
	if exe, err := os.Executable(); err == nil {
		return filepath.Join(filepath.Dir(exe), "examples")
	}
	return "examples"
}
