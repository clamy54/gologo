//go:build windows

package logo

import (
	"os"

	"golang.org/x/sys/windows/registry"
)

// parent du dossier "Logo" sous Windows -> le "Documents" de l'utilisateur. ce
// dossier est localise ("Mes Documents" en FR) et souvent redirige (OneDrive), et
// aucune variable d'env ne le donne de facon fiable. on lit donc le chemin resolu
// dans le registre ("Shell Folders" / "Personal"). repli sur le home si ca rate
func defaultLogoParent() string {
	if docs := windowsDocuments(); docs != "" {
		return docs
	}
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}
	return ""
}

func windowsDocuments() string {
	k, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Explorer\Shell Folders`,
		registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer k.Close()
	v, _, err := k.GetStringValue("Personal")
	if err != nil {
		return ""
	}
	return v // "Shell Folders" donne deja un chemin absolu resolu
}
