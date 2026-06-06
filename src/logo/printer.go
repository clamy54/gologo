package logo

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// capacite du backend a sauver l'ecran entier (champ + texte) en PNG, pour COPIE
// cherchee sur i.Out, absente en mode sans ecran (no-op)
type ScreenSaver interface {
	SaveScreenPNG(path string) error
}

// sous-dossier "imprimante" ou COPIE depose ses cliches (PAGE_1.PNG, PAGE_2.PNG...)
const printerDirName = "PRINTER"

// reconnait un cliche PAGE_<n>.PNG (n >= 0), nom deja en MAJ
var pageRe = regexp.MustCompile(`^PAGE_(\d+)\.PNG$`)

// nom du prochain cliche : PAGE_<max+1>.PNG, max etant le plus grand deja present
// (0 si aucun -> PAGE_1.PNG). les noms hors motif sont ignores
func nextPageName(existing []string) string {
	max := 0
	for _, name := range existing {
		m := pageRe.FindStringSubmatch(strings.ToUpper(name))
		if m == nil {
			continue
		}
		if n, err := strconv.Atoi(m[1]); err == nil && n > max {
			max = n
		}
	}
	return fmt.Sprintf("PAGE_%d.PNG", max+1)
}

func (i *Interp) registerPrinter() {
	// COPIE : "imprime" l'ecran en sauvant un PNG numerote dans le sous-dossier
	// PRINTER (la version moderne de la copie vers l'imprimante thermique du MO5).
	// en headless, ne fait rien
	i.register(cmd(0, func(in *Interp, a []Value) error {
		saver, ok := in.Out.(ScreenSaver)
		if !ok {
			return nil
		}
		dir, err := in.ensureWorkDir()
		if err != nil {
			return err
		}
		printerDir := filepath.Join(dir, printerDirName)
		if err := os.MkdirAll(printerDir, 0o755); err != nil {
			return fmt.Errorf("ECRITURE IMPOSSIBLE")
		}
		var names []string
		if entries, err := os.ReadDir(printerDir); err == nil {
			for _, e := range entries {
				if !e.IsDir() {
					names = append(names, e.Name())
				}
			}
		}
		path := filepath.Join(printerDir, nextPageName(names))
		if err := saver.SaveScreenPNG(path); err != nil {
			return fmt.Errorf("ECRITURE IMPOSSIBLE")
		}
		return nil
	}), "COPIE")
}
