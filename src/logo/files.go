package logo

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// fichiers .GLG dans le dossier de travail (i.workDir)
// FORMATE/LECTEUR/FLECTEUR sont des no-op : compat avec l'ancien materiel

const glgExt = ".GLG"

// fixe le dossier de travail .GLG (main ou tests). vide => dossier courant
func (i *Interp) SetWorkDir(dir string) { i.workDir = dir }

// rend i.workDir s'il est fixe, sinon le dossier "Logo" sous le parent par defaut
// (home sous Linux/macOS, "Documents" localise sous Windows, cf defaultLogoParent
// par plateforme dans workdir_*.go). a defaut, repli sur "Logo" dans le courant
func (i *Interp) workDirPath() string {
	if i.workDir != "" {
		return i.workDir
	}
	if base := defaultLogoParent(); base != "" {
		return filepath.Join(base, "Logo")
	}
	return "Logo"
}

// cree le dossier de travail au besoin et verifie qu'il est accessible
// appele avant toute operation fichier
func (i *Interp) ensureWorkDir() (string, error) {
	dir := i.workDirPath()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return dir, fmt.Errorf("ECRITURE IMPOSSIBLE")
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return dir, fmt.Errorf("ECRITURE IMPOSSIBLE")
	}
	return dir, nil
}

// cree/verifie le dossier et rend le chemin complet du fichier (base nettoyee + .GLG)
func (i *Interp) resolvePath(name string) (string, error) {
	dir, err := i.ensureWorkDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, fileBase(name)), nil
}

// ramene un nom a la forme standard : base seule (anti-traversee de dossier),
// MAJ, extension ext forcee (remplace celle fournie)
func fileBaseExt(name, ext string) string {
	name = filepath.Base(strings.ToUpper(strings.TrimSpace(name)))
	if e := filepath.Ext(name); e != "" {
		name = strings.TrimSuffix(name, e)
	}
	if name == "" || name == "." || name == ".." || name == string(filepath.Separator) {
		name = "SANSNOM"
	}
	return name + ext
}

// nom de fichier .GLG standard (espace de travail Logo)
func fileBase(name string) string { return fileBaseExt(name, glgExt) }

func hasGlgExt(name string) bool { return strings.HasSuffix(strings.ToUpper(name), glgExt) }

// capacite du backend a sauver le champ graphique en PNG (SAUVEPNG)
// cherchee sur i.Out, absente en mode sans ecran (no-op)
type ImageSaver interface {
	SaveFieldPNG(path string) error
}

// pose une question oui/non au clavier (O/N en FR, Y/N en EN)
// sans clavier (headless/tests), rend true : on fonce
func (i *Interp) confirmYesNo(question string) (bool, error) {
	if i.keyb == nil {
		return true, nil
	}
	fmt.Fprint(i.Out, question)
	r, ok := i.keyb.ReadChar()
	if !ok {
		return false, ErrInterrompu
	}
	fmt.Fprintf(i.Out, "%c\n", r)
	return r == 'O' || r == 'Y', nil
}

// demande confirmation si le fichier existe deja, rend true s'il faut ecrire
// sans clavier (headless/tests), ecrase sans poser de question
func (i *Interp) confirmOverwrite(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		return true, nil // n'existe pas : on ecrit
	}
	base := filepath.Base(path)
	q := fmt.Sprintf("LE FICHIER %s EXISTE DEJA. ECRASER ? (O/N) ", base)
	cancel := "ANNULE"
	if i.Lang() == "EN" {
		q = fmt.Sprintf("FILE %s ALREADY EXISTS. OVERWRITE? (Y/N) ", base)
		cancel = "NOT SAVED"
	}
	ok, err := i.confirmYesNo(q)
	if err != nil {
		return false, err
	}
	if !ok {
		fmt.Fprintln(i.Out, cancel)
	}
	return ok, nil
}

func (i *Interp) registerFiles() {
	// SAUVE mot liste : ecrit dans mot les proc/var de liste, en source re-executable
	// (nom nu = proc, "nom = var, :nom = liste recursive, CONTENU = tout)
	i.register(cmd(2, func(in *Interp, a []Value) error {
		name, err := toWord(a[0])
		if err != nil {
			return err
		}
		path, err := in.resolvePath(name)
		if err != nil {
			return err
		}
		ok, err := in.confirmOverwrite(path)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		procs, vars := in.collectSave(a[1])
		src := in.workspaceSource(procs, vars)
		if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
			return fmt.Errorf("ECRITURE IMPOSSIBLE")
		}
		return nil
	}), "SAUVE")

	// SAUVEPNG mot : sauve le champ graphique dans mot.PNG (meme dossier que SAUVE)
	// en mode sans ecran, ne fait rien
	i.register(cmd(1, func(in *Interp, a []Value) error {
		name, err := toWord(a[0])
		if err != nil {
			return err
		}
		saver, ok := in.Out.(ImageSaver)
		if !ok {
			return nil
		}
		dir, err := in.ensureWorkDir() // dossier pret avant d'ecrire
		if err != nil {
			return err
		}
		path := filepath.Join(dir, fileBaseExt(name, ".PNG"))
		proceed, err := in.confirmOverwrite(path)
		if err != nil {
			return err
		}
		if !proceed {
			return nil
		}
		if err := saver.SaveFieldPNG(path); err != nil {
			return fmt.Errorf("ECRITURE IMPOSSIBLE")
		}
		return nil
	}), "SAUVEPNG")

	// SAUVED mot : sauve le contenu courant de l'editeur dans mot
	i.register(cmd(1, func(in *Interp, a []Value) error {
		name, err := toWord(a[0])
		if err != nil {
			return err
		}
		path, err := in.resolvePath(name)
		if err != nil {
			return err
		}
		ok, err := in.confirmOverwrite(path)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		if err := os.WriteFile(path, []byte(in.edBuf), 0o644); err != nil {
			return fmt.Errorf("ECRITURE IMPOSSIBLE")
		}
		return nil
	}), "SAUVED")

	// RAMENE mot : relit mot et l'ajoute a l'espace de travail (definit les procs,
	// execute les DONNE, affiche "VOUS VENEZ DE DEFINIR ...")
	i.register(cmd(1, func(in *Interp, a []Value) error {
		name, err := toWord(a[0])
		if err != nil {
			return err
		}
		path, err := in.resolvePath(name)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("LECTURE IMPOSSIBLE")
		}
		in.edBuf = string(data) // ED nu rouvrira ce source (editable)
		return in.defineFromEditor(string(data))
	}), "RAMENE")

	// CHARGE mot : met mot dans l'editeur SANS l'interpreter (ED nu rouvre,
	// a valider par Ctrl+S)
	i.register(cmd(1, func(in *Interp, a []Value) error {
		name, err := toWord(a[0])
		if err != nil {
			return err
		}
		path, err := in.resolvePath(name)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("LECTURE IMPOSSIBLE")
		}
		in.edBuf = string(data)
		return nil
	}), "CHARGE")

	// CATALOGUE : liste les .GLG du dossier (nom + taille) avec un recap
	// sortie paginee si elle deborde de l'ecran
	i.register(cmd(0, func(in *Interp, a []Value) error {
		dir, err := in.ensureWorkDir()
		if err != nil {
			return err
		}
		lines, err := catalogLines(in.Lang(), dir, emptyDirMsg(in.Lang()))
		if err != nil {
			return err
		}
		in.showPaged("CATALOGUE", lines)
		return nil
	}), "CATALOGUE")

	i.registerExamples()

	// DETRUIS mot : supprime le fichier. pas de corbeille, pas de remords
	i.register(cmd(1, func(in *Interp, a []Value) error {
		name, err := toWord(a[0])
		if err != nil {
			return err
		}
		path, err := in.resolvePath(name)
		if err != nil {
			return err
		}
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("LECTURE IMPOSSIBLE")
		}
		return nil
	}), "DETRUIS")

	// compat disquette/cassette : sans effet sur une machine moderne
	i.register(cmd(1, func(in *Interp, a []Value) error { return nil }), "FORMATE")  // formater
	i.register(cmd(1, func(in *Interp, a []Value) error { return nil }), "FLECTEUR") // choisir le lecteur
	i.register(&primitive{arity: 0, reporter: true, fn: func(in *Interp, a []Value) (Value, error) {
		return NumberValue(1), nil // lecteur courant (sans effet)
	}}, "LECTEUR")
}

// analyse les args de SAUVE et rend les noms de procs et de variables (tries, sans
// doublons). :nom deroule recursivement la variable-liste du meme nom
func (i *Interp) collectSave(v Value) (procs, vars []string) {
	procSet := map[string]bool{}
	varSet := map[string]bool{}
	var walk func(items []Datum)
	walk = func(items []Datum) {
		for _, d := range items {
			name := strings.ToUpper(d.Text)
			if name == "" {
				continue
			}
			if d.Kind == DVarRef { // :nom -> variable-liste de noms, recursif
				if val, ok := i.vars[name]; ok {
					if val.Kind == KList {
						walk(val.List)
					} else {
						varSet[name] = true
					}
				}
				continue
			}
			if _, ok := i.procs[name]; ok {
				procSet[name] = true
			}
			if _, ok := i.vars[name]; ok {
				varSet[name] = true
			}
		}
	}
	if v.Kind == KList {
		walk(v.List)
	}
	return sortedKeys(procSet), sortedKeys(varSet)
}

// source Logo re-executable : d'abord les procedures (POUR...FIN), puis les
// variables (DONNE)
func (i *Interp) workspaceSource(procs, vars []string) string {
	var b strings.Builder
	for _, n := range procs {
		if p := i.procs[n]; p != nil {
			b.WriteString(p.sourceText())
			b.WriteByte('\n')
		}
	}
	for _, n := range vars {
		if val, ok := i.vars[n]; ok {
			fmt.Fprintf(&b, "DONNE \"%s %s\n", n, valueSource(val))
		}
	}
	return b.String()
}

// sing si n == 1, plur sinon (accord FR/EN)
func plural(n int64, sing, plur string) string {
	if n == 1 {
		return sing
	}
	return plur
}

// unite de taille (octet/octets, byte/bytes) accordee selon n
func unitLabel(lang string, n int64) string {
	if lang == "EN" {
		return plural(n, "byte", "bytes")
	}
	return plural(n, "octet", "octets")
}

// message "dossier vide" dans la langue courante
func emptyDirMsg(lang string) string {
	if lang == "EN" {
		return "WORKING DIRECTORY EMPTY"
	}
	return "DOSSIER DE TRAVAIL VIDE"
}

// ligne recap "X fichiers pour Y octets"
func catalogSummary(lang string, count int, total int64) string {
	if lang == "EN" {
		return fmt.Sprintf("%d %s, %d %s", count, plural(int64(count), "file", "files"), total, unitLabel(lang, total))
	}
	return fmt.Sprintf("%d %s pour %d %s", count, plural(int64(count), "fichier", "fichiers"), total, unitLabel(lang, total))
}

// les cles d'un ensemble, triees
func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
