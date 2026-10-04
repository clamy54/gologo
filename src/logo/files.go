package logo

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// fichiers .GLG dans le dossier de travail (i.workDir)
// FORMATE/LECTEUR/FLECTEUR sont des no-op : compat avec l'ancien materiel

const glgExt = ".GLG"

// taille maximale d'un source .GLG relu d'un coup (RAMENE, CHARGE et leurs ...EX) :
// tres au-dela de tout programme reel, mais un fichier enorme pris pour du Logo
// ne mange pas toute la memoire
const maxSourceBytes = 16 << 20

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

// ouvre le dossier de travail comme RACINE des E/S fichier. tous les acces passent
// par elle : os.Root refuse ce qui sort du dossier, y compris par un lien
// symbolique ou une jonction places dedans (un controle du nom seul ne voit pas les
// liens, et verifier avant d'ouvrir laisserait le temps de changer le lien). les
// liens qui restent a l'interieur du dossier sont suivis. a refermer par l'appelant
func (i *Interp) openWorkRoot() (*os.Root, error) {
	dir, err := i.ensureWorkDir()
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, errEcritureImpossible
	}
	return root, nil
}

// noms de peripheriques reserves par Windows : NUL.TXT ou COM1 ne designent pas un
// fichier, quelle que soit l'extension
var reservedNames = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true, "CONIN$": true, "CONOUT$": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true,
	"COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true,
	"LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

// un nom de fichier (sans dossier) acceptable, tel qu'il sera cree : le controle
// porte sur le nom une fois mis en forme (blancs de debut et de fin retires ; pour
// un .GLG, majuscules et extension remplacee), pas sur le texte tape. "TRAIL." donne
// donc TRAIL.GLG, alors qu'un fichier de donnees "x." est refuse.
// on applique partout les regles de la
// plateforme la plus stricte, pour qu'un programme se comporte pareil sur toutes :
// pas de caractere de controle ni de < > : " | ? * (le ':' ouvrirait un flux NTFS
// ou un volume), pas de point ni d'espace final (Windows les retire en silence :
// deux noms pour un fichier), pas de nom de peripherique
func validFileName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	for _, r := range name {
		if r < 0x20 || strings.ContainsRune(`<>:"|?*\/`, r) {
			return false
		}
	}
	if last := name[len(name)-1]; last == '.' || last == ' ' {
		return false
	}
	stem := name
	if k := strings.IndexByte(stem, '.'); k >= 0 {
		stem = stem[:k]
	}
	return !reservedNames[strings.ToUpper(strings.TrimRight(stem, " "))]
}

// nom (relatif au dossier de travail) du fichier .GLG designe par name
func glgName(name string) (string, error) { return workName(name, glgExt) }

// nom du fichier d'extension ext designe par name, verifie
func workName(name, ext string) (string, error) {
	base := fileBaseExt(name, ext)
	if !validFileName(base) {
		return "", errNomInvalide
	}
	return base, nil
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

// lit en entier le fichier name de root, a condition que ce soit un fichier
// ordinaire d'au plus max octets. un tube nomme ou un peripherique bloquerait la
// lecture sans fin : on les refuse (avant d'ouvrir, puis sur le fichier ouvert)
func readRegular(root *os.Root, name string, max int64) ([]byte, error) {
	info, err := root.Stat(name)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, errIntrouvable
		}
		return nil, errLectureImpossible
	}
	if !info.Mode().IsRegular() {
		return nil, errLectureImpossible
	}
	if info.Size() > max {
		return nil, errTropGros
	}
	// ouverture non bloquante : si le fichier a ete remplace par un tube nomme
	// entre le controle et ici, on recupere quand meme la main pour le voir
	f, err := root.OpenFile(name, os.O_RDONLY|openNonBlock, 0)
	if err != nil {
		return nil, errLectureImpossible
	}
	defer f.Close()
	if fi, err := f.Stat(); err != nil || !fi.Mode().IsRegular() {
		return nil, errLectureImpossible
	}
	data, err := io.ReadAll(io.LimitReader(f, max+1))
	if err != nil {
		return nil, errLectureImpossible
	}
	if int64(len(data)) > max {
		return nil, errTropGros // a grossi entre le controle et la lecture
	}
	return data, nil
}

// lit un source .GLG du dossier root. toute erreur de lecture rend LECTURE
// IMPOSSIBLE (message historique de RAMENE/CHARGE), sauf le fichier trop gros
func readSource(root *os.Root, name string) ([]byte, error) {
	data, err := readRegular(root, name, maxSourceBytes)
	if err != nil && err != errTropGros {
		return nil, errLectureImpossible
	}
	return data, err
}

// ecrit le fichier name de root sans jamais abimer celui qui existe : on ecrit a
// cote, dans un fichier temporaire du meme dossier, et on ne le renomme par-dessus
// qu'une fois l'ecriture reussie et fermee. un disque plein ou une coupure laisse
// donc l'ancien fichier intact, au lieu d'un fichier vide ou tronque.
//
// le nouveau fichier REMPLACE l'ancien : il en reprend les droits d'acces (un
// fichier prive le reste), mais pas le proprietaire ni les autres attributs
func writeAtomic(root *os.Root, name string, write func(w io.Writer) error) error {
	var rnd [6]byte
	if _, err := rand.Read(rnd[:]); err != nil {
		return err
	}
	// droits du fichier remplace. tant qu'on ecrit, le temporaire n'est lisible
	// que par nous ; un fichier nouveau prend les droits habituels (umask applique)
	create, keep := os.FileMode(0o644), os.FileMode(0)
	if info, err := root.Stat(name); err == nil && info.Mode().IsRegular() {
		create, keep = 0o600, info.Mode().Perm()
	}
	tmp := filepath.Join(filepath.Dir(name), ".gologo-"+hex.EncodeToString(rnd[:])+".tmp")
	f, err := root.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, create)
	if err != nil {
		return err
	}
	err = write(f)
	if err == nil && keep != 0 && runtime.GOOS != "windows" {
		err = f.Chmod(keep)
	}
	if err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = root.Rename(tmp, name)
	}
	if err != nil {
		root.Remove(tmp)
	}
	return err
}

// capacite du backend a sauver le champ graphique en PNG (SAUVEPNG)
// cherchee sur i.Out, absente en mode sans ecran (no-op)
type ImageSaver interface {
	SaveFieldPNG(w io.Writer) error
}

// pose une question oui/non au clavier (O/N en FR, Y/N en EN)
// sans clavier (headless/tests), rend true : on fonce
func (i *Interp) confirmYesNo(question string) (bool, error) {
	if i.keyb == nil {
		return true, nil
	}
	// une question qui ne s'affiche pas (sortie cassee) ne se pose pas : on
	// n'attend pas la reponse a une question que personne ne voit
	if _, err := fmt.Fprint(i.Out, question); err != nil {
		return false, errEcritureImpossible
	}
	r, ok := i.keyb.ReadChar()
	if !ok {
		return false, ErrInterrompu
	}
	if _, err := fmt.Fprintf(i.Out, "%c\n", r); err != nil {
		return false, errEcritureImpossible
	}
	return r == 'O' || r == 'Y', nil
}

// demande confirmation si le fichier existe deja, rend true s'il faut ecrire
// sans clavier (headless/tests), ecrase sans poser de question
func (i *Interp) confirmOverwrite(root *os.Root, name string) (bool, error) {
	if _, err := root.Stat(name); err != nil {
		return true, nil // n'existe pas : on ecrit
	}
	base := filepath.Base(name)
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
		if err := i.printLine(cancel); err != nil {
			return false, err
		}
	}
	return ok, nil
}

// sauve un fichier du dossier de travail : nom verifie, refus si c'est un flux de
// donnees ouvert, confirmation s'il existe, puis ecriture sans risque pour l'ancien
func (i *Interp) saveWorkFile(name, ext string, write func(w io.Writer) error) error {
	rel, err := workName(name, ext)
	if err != nil {
		return err
	}
	root, err := i.openWorkRoot()
	if err != nil {
		return err
	}
	defer root.Close()
	if i.streamOn(root, rel) != nil {
		return errFichierOuvert // on n'ecrase pas un fichier en cours de lecture/ecriture
	}
	ok, err := i.confirmOverwrite(root, rel)
	if err != nil || !ok {
		return err
	}
	if err := writeAtomic(root, rel, write); err != nil {
		return errEcritureImpossible
	}
	return nil
}

// lit le source .GLG name du dossier de travail
func (i *Interp) loadWorkFile(name string) ([]byte, error) {
	rel, err := glgName(name)
	if err != nil {
		return nil, err
	}
	root, err := i.openWorkRoot()
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return readSource(root, rel)
}

func (i *Interp) registerFiles() {
	// SAUVE mot liste : ecrit dans mot les proc/var de liste, en source re-executable
	// (nom nu = proc, "nom = var, :nom = liste recursive, CONTENU = tout)
	i.register(cmd(2, func(in *Interp, a []Value) error {
		name, err := toWord(a[0])
		if err != nil {
			return err
		}
		procs, vars, err := in.collectSave(a[1])
		if err != nil {
			return err
		}
		src, err := in.workspaceSource(procs, vars)
		if err != nil {
			return err
		}
		// ce que SAUVE ecrit, RAMENE doit pouvoir le relire : memes plafonds de
		// taille et d'imbrication que le lecteur, verifies AVANT de remplacer
		// l'ancien fichier
		if len(src) > maxSourceBytes {
			return errTropGros
		}
		if _, err := Read(src); err != nil {
			return err
		}
		return in.saveWorkFile(name, glgExt, func(w io.Writer) error {
			_, err := io.WriteString(w, src)
			return err
		})
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
		return in.saveWorkFile(name, ".PNG", saver.SaveFieldPNG)
	}), "SAUVEPNG")

	// SAUVED mot : sauve le contenu courant de l'editeur dans mot
	i.register(cmd(1, func(in *Interp, a []Value) error {
		name, err := toWord(a[0])
		if err != nil {
			return err
		}
		return in.saveWorkFile(name, glgExt, func(w io.Writer) error {
			_, err := io.WriteString(w, in.edBuf)
			return err
		})
	}), "SAUVED")

	// RAMENE mot : relit mot et l'ajoute a l'espace de travail (definit les procs,
	// execute les DONNE, affiche "VOUS VENEZ DE DEFINIR ...")
	i.register(cmd(1, func(in *Interp, a []Value) error {
		name, err := toWord(a[0])
		if err != nil {
			return err
		}
		data, err := in.loadWorkFile(name)
		if err != nil {
			return err
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
		data, err := in.loadWorkFile(name)
		if err != nil {
			return err
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
		return in.showPaged("CATALOGUE", lines)
	}), "CATALOGUE")

	i.registerExamples()

	// DETRUIS mot : supprime le fichier. pas de corbeille, pas de remords.
	// refuse de supprimer un fichier ouvert (facon FMSLogo)
	i.register(cmd(1, func(in *Interp, a []Value) error {
		name, err := toWord(a[0])
		if err != nil {
			return err
		}
		rel, err := glgName(name)
		if err != nil {
			return err
		}
		root, err := in.openWorkRoot()
		if err != nil {
			return err
		}
		defer root.Close()
		if in.streamOn(root, rel) != nil {
			return errFichierOuvert
		}
		if _, err := root.Stat(rel); err != nil {
			return errIntrouvable
		}
		if err := root.Remove(rel); err != nil {
			return errEcritureImpossible
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
// doublons). un nom nu designe la procedure, "nom la variable (a defaut l'autre,
// s'il n'y a pas d'ambiguite) ; :nom deroule la variable-liste du meme nom
func (i *Interp) collectSave(v Value) (procs, vars []string, err error) {
	procSet := map[string]bool{}
	varSet := map[string]bool{}
	if v.Kind != KList {
		return nil, nil, nil
	}
	// variables-listes deja deroulees : DONNE "L [:L], ou :A -> [:B] et :B -> [:A],
	// tourneraient en rond. et les listes a derouler attendent dans une pile, pas
	// dans des appels recursifs : une longue chaine de variables ne creuse rien
	seen := map[string]bool{}
	todo := [][]Datum{v.List}
	for len(todo) > 0 {
		if i.brk.Load() {
			return nil, nil, ErrInterrompu
		}
		items := todo[len(todo)-1]
		todo = todo[:len(todo)-1]
		for _, d := range items {
			name := strings.ToUpper(d.Text)
			if name == "" {
				continue
			}
			_, isProc := i.procs[name]
			val, isVar := i.vars[name]
			switch {
			case d.Kind == DVarRef: // :nom -> variable-liste de noms
				if !isVar {
					continue
				}
				if val.Kind != KList {
					varSet[name] = true
				} else if !seen[name] {
					seen[name] = true
					todo = append(todo, val.List)
				}
			case d.Kind == DWord: // "nom : la variable
				if isVar {
					varSet[name] = true
				} else if isProc {
					procSet[name] = true
				}
			default: // nom nu : la procedure
				if isProc {
					procSet[name] = true
				} else if isVar {
					varSet[name] = true
				}
			}
		}
	}
	return sortedKeys(procSet), sortedKeys(varSet), nil
}

// source Logo re-executable : d'abord les procedures (POUR...FIN), puis les
// variables (DONNE). interruptible entre deux objets ; erreur si l'un d'eux n'a pas
// d'ecriture fidele ou depasse le plafond de taille
func (i *Interp) workspaceSource(procs, vars []string) (string, error) {
	var b strings.Builder
	add := func(s string, err error) error {
		if err != nil {
			return err
		}
		if i.brk.Load() {
			return ErrInterrompu
		}
		b.WriteString(s)
		b.WriteByte('\n')
		if b.Len() > maxSourceBytes {
			return errTropGros // inutile de batir un texte que RAMENE refusera
		}
		return nil
	}
	for _, n := range procs {
		if p := i.procs[n]; p != nil {
			if err := add(p.sourceText()); err != nil {
				return "", err
			}
		}
	}
	for _, n := range vars {
		if val, ok := i.vars[n]; ok {
			if err := add(donneSource(n, val)); err != nil {
				return "", err
			}
		}
	}
	return b.String(), nil
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
