//go:build !windows

package logo

import "os"

// parent du dossier "Logo" hors Windows (Linux/macOS) -> le home
// "" si introuvable, le reste se debrouille
func defaultLogoParent() string {
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}
	return ""
}
