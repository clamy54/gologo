package logo

import (
	"fmt"
	"os"
	"path/filepath"
)

// commandes ...EX : CATALOGUEEX / CHARGEEX / RAMENEEX. memes effets que CATALOGUE /
// CHARGE / RAMENE mais sur le dossier "examples" livre avec l'install (lecture seule),
// dont l'emplacement depend de la plateforme (cf examples_*.go)

// fixe le dossier des exemples (tests, ou override depuis le main)
// vide => defaut plateforme (ou GOLOG_EXAMPLES s'il est defini)
func (i *Interp) SetExamplesDir(dir string) { i.examplesDir = dir }

// dossier des exemples : i.examplesDir s'il est fixe, sinon GOLOG_EXAMPLES (pratique
// pour les paquets exotiques : AppImage, Flatpak...), sinon le defaut plateforme
func (i *Interp) examplesDirPath() string {
	if i.examplesDir != "" {
		return i.examplesDir
	}
	if env := os.Getenv("GOLOG_EXAMPLES"); env != "" {
		return env
	}
	return platformExamplesDir()
}

// verifie que le dossier des exemples existe et est bien un dossier. au contraire
// d'ensureWorkDir il ne le cree pas (livre en lecture seule). erreur sinon
func (i *Interp) ensureExamplesDir() (string, error) {
	dir := i.examplesDirPath()
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return dir, fmt.Errorf("PAS DE DOSSIER D'EXEMPLES")
	}
	return dir, nil
}

// verifie le dossier puis lit le .GLG nomme et rend son contenu
// erreur LECTURE IMPOSSIBLE s'il manque, est illisible ou est un dossier
func (i *Interp) readExampleFile(name string) ([]byte, error) {
	dir, err := i.ensureExamplesDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, fileBase(name))
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		return nil, fmt.Errorf("LECTURE IMPOSSIBLE")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("LECTURE IMPOSSIBLE")
	}
	return data, nil
}

// lignes d'affichage des .GLG d'un dossier, au format de CATALOGUE (nom + taille +
// recap). emptyMsg si aucun .GLG, erreur LECTURE IMPOSSIBLE si le dossier resiste.
// partage entre CATALOGUE (dossier de travail) et CATALOGUEEX (exemples)
func catalogLines(lang, dir, emptyMsg string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("LECTURE IMPOSSIBLE")
	}
	var lines []string
	var count int
	var total int64
	for _, e := range entries {
		if e.IsDir() || !hasGlgExt(e.Name()) {
			continue // .GLG uniquement
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		base := e.Name()[:len(e.Name())-len(glgExt)] // sans l'extension
		lines = append(lines, fmt.Sprintf("%-16s %d %s", base, info.Size(), unitLabel(lang, info.Size())))
		count++
		total += info.Size()
	}
	if count == 0 {
		return []string{emptyMsg}, nil
	}
	return append(lines, "", catalogSummary(lang, count, total)), nil
}

// message "dossier d'exemples vide" dans la langue courante
func examplesEmptyMsg(lang string) string {
	if lang == "EN" {
		return "EXAMPLES DIRECTORY EMPTY"
	}
	return "DOSSIER D'EXEMPLES VIDE"
}

// CATALOGUEEX / CHARGEEX / RAMENEEX (les noms anglais arrivent via registerEnglishAliases)
func (i *Interp) registerExamples() {
	// CATALOGUEEX : liste les .GLG du dossier des exemples (meme affichage que
	// CATALOGUE). verifie d'abord que le dossier existe
	i.register(cmd(0, func(in *Interp, a []Value) error {
		dir, err := in.ensureExamplesDir()
		if err != nil {
			return err
		}
		lines, err := catalogLines(in.Lang(), dir, examplesEmptyMsg(in.Lang()))
		if err != nil {
			return err
		}
		in.showPaged("CATALOGUEEX", lines)
		return nil
	}), "CATALOGUEEX")

	// RAMENEEX mot : comme RAMENE mais lit dans le dossier des exemples
	i.register(cmd(1, func(in *Interp, a []Value) error {
		name, err := toWord(a[0])
		if err != nil {
			return err
		}
		data, err := in.readExampleFile(name)
		if err != nil {
			return err
		}
		in.edBuf = string(data) // ED nu rouvrira ce source (editable)
		return in.defineFromEditor(string(data))
	}), "RAMENEEX")

	// CHARGEEX mot : comme CHARGE mais lit dans le dossier des exemples (sans interpreter)
	i.register(cmd(1, func(in *Interp, a []Value) error {
		name, err := toWord(a[0])
		if err != nil {
			return err
		}
		data, err := in.readExampleFile(name)
		if err != nil {
			return err
		}
		in.edBuf = string(data)
		return nil
	}), "CHARGEEX")
}
