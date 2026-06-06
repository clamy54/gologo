//go:build darwin

package logo

import (
	"os"
	"path/filepath"
)

// sous macOS, "examples" vit dans les ressources du bundle
// (GoLogo.app/Contents/Resources/examples). l'executable est dans Contents/MacOS,
// donc examples = <dossier executable>/../Resources/examples. repli sur "examples"
// si l'executable est introuvable
func platformExamplesDir() string {
	if exe, err := os.Executable(); err == nil {
		return filepath.Join(filepath.Dir(exe), "..", "Resources", "examples")
	}
	return "examples"
}
