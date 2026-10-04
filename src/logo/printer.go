package logo

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// capacite du backend a sauver l'ecran entier (champ + texte) en PNG, pour COPIE
// cherchee sur i.Out, absente en mode sans ecran (no-op)
type ScreenSaver interface {
	SaveScreenPNG(w io.Writer) error
}

// sous-dossier "imprimante" ou COPIE depose ses cliches (PAGE_1.PNG, PAGE_2.PNG...)
const printerDirName = "PRINTER"

// reconnait un cliche PAGE_<n>.PNG (n >= 0), nom deja en MAJ
var pageRe = regexp.MustCompile(`^PAGE_(\d+)\.PNG$`)

// plus grand numero de cliche pris en compte : au-dela (fichier nomme a la main),
// le numero suivant deborderait
const maxPageNum = 999_999_999

// numero du prochain cliche : max+1, max etant le plus grand numero deja present
// (0 si aucun -> PAGE_1.PNG). les noms hors motif ou demesures sont ignores
func nextPageNum(existing []string) int {
	max := 0
	for _, name := range existing {
		m := pageRe.FindStringSubmatch(strings.ToUpper(name))
		if m == nil {
			continue
		}
		if n, err := strconv.Atoi(m[1]); err == nil && n > max && n < maxPageNum {
			max = n
		}
	}
	return max + 1
}

func pageName(n int) string { return fmt.Sprintf("PAGE_%d.PNG", n) }

func (i *Interp) registerPrinter() {
	// COPIE : "imprime" l'ecran en sauvant un PNG numerote dans le sous-dossier
	// PRINTER (la version moderne de la copie vers l'imprimante thermique du MO5).
	// en headless, ne fait rien
	i.register(cmd(0, func(in *Interp, a []Value) error {
		saver, ok := in.Out.(ScreenSaver)
		if !ok {
			return nil
		}
		root, err := in.openWorkRoot()
		if err != nil {
			return err
		}
		defer root.Close()
		if err := root.MkdirAll(printerDirName, 0o755); err != nil {
			return errEcritureImpossible
		}
		var names []string
		if dir, err := root.Open(printerDirName); err == nil {
			if entries, err := dir.ReadDir(-1); err == nil {
				for _, e := range entries {
					if !e.IsDir() {
						names = append(names, e.Name())
					}
				}
			}
			dir.Close()
		}
		// le numero est RESERVE par une creation exclusive : deux GoLogo lances en
		// meme temps ne peuvent pas choisir le meme et s'ecraser l'un l'autre. si
		// le nom vient d'etre pris, on essaie le suivant
		n := nextPageNum(names)
		var f *os.File
		var path string
		for tries := 0; ; tries++ {
			path = filepath.Join(printerDirName, pageName(n+tries))
			f, err = root.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
			if err == nil {
				break
			}
			if !errors.Is(err, fs.ErrExist) || tries >= 1000 {
				return errEcritureImpossible
			}
		}
		err = saver.SaveScreenPNG(f)
		if cerr := f.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			root.Remove(path) // pas de cliche a moitie ecrit
			return errEcritureImpossible
		}
		return nil
	}), "COPIE")
}
