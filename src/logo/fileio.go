package logo

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

// E/S fichier facon FMSLogo : pas de handle passe a chaque lecture, mais des
// "flux courants". on ouvre un fichier, on en fait le flux de lecture (ou
// d'ecriture) courant, puis les recepteurs habituels (LISCAR/LISMOT/LL/LISLIGNE)
// lisent dedans, et ECRIS/MONTRE/TAPE ecrivent dedans.
//
// choix d'implementation : on travaille en direct sur *os.File, sans bufio. la
// position vue par Logo est donc l'offset reel du fichier - aucune subtilite de
// buffer a invalider, et OUVREMAJ partage naturellement une seule position.

type fileMode int

const (
	modeRead fileMode = iota
	modeWrite
	modeAppend
	modeUpdate
)

func (m fileMode) readable() bool { return m == modeRead || m == modeUpdate }
func (m fileMode) writable() bool { return m == modeWrite || m == modeAppend || m == modeUpdate }

// un fichier ouvert : son nom logique, son chemin reel, le descripteur et le mode
type openFile struct {
	name string
	path string
	f    *os.File
	mode fileMode
}

// l'etat des E/S fichier d'un interpreteur : les fichiers ouverts et les deux flux
// courants (nil = clavier pour la lecture, console pour l'ecriture)
type fileIO struct {
	open     map[string]*openFile
	curRead  *openFile
	curWrite *openFile
}

func (i *Interp) fileSet() *fileIO {
	if i.fio == nil {
		i.fio = &fileIO{open: map[string]*openFile{}}
	}
	return i.fio
}

// le flux de lecture courant, ou nil (clavier). consulte par LISCAR/LISMOT/LL
func (i *Interp) curReadStream() *openFile {
	if i.fio == nil {
		return nil
	}
	return i.fio.curRead
}

// ferme tous les fichiers ouverts et oublie les flux. appele a RAZ et a QUITTE.
// les ecritures ne sont pas bufferisees, mais Close peut quand meme remonter une
// erreur d'ecriture differee (disque plein, etc.) : on rend la premiere rencontree
func (i *Interp) closeAllFiles() error {
	if i.fio == nil {
		return nil
	}
	var firstErr error
	for _, of := range i.fio.open {
		if err := of.f.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	i.fio = nil
	return firstErr
}

// CloseFiles : ferme proprement les fichiers ouverts (a appeler a la sortie du
// programme hote). rend la premiere erreur de Close, ou nil s'il n'y en a aucun
func (i *Interp) CloseFiles() error { return i.closeAllFiles() }

// messages d'erreur maison (traduits en EN a l'affichage, cf localize.go)
var (
	errIntrouvable        = fmt.Errorf("FICHIER INTROUVABLE")
	errDejaOuvert         = fmt.Errorf("FICHIER DEJA OUVERT")
	errNonOuvert          = fmt.Errorf("FICHIER NON OUVERT")
	errFichierOuvert      = fmt.Errorf("FICHIER OUVERT")
	errPositionInvalide   = fmt.Errorf("POSITION INVALIDE")
	errNomInvalide        = fmt.Errorf("NOM DE FICHIER INVALIDE")
	errMauvaisMode        = fmt.Errorf("MAUVAIS MODE D'OUVERTURE")
	errLectureImpossible  = fmt.Errorf("LECTURE IMPOSSIBLE")
	errEcritureImpossible = fmt.Errorf("ECRITURE IMPOSSIBLE")
)

// resout un nom de fichier de DONNEES : confine au dossier de travail (rejet des
// chemins absolus et de la traversee ".."), mais qui PRESERVE le nom, l'extension
// et la casse (contrairement a resolvePath qui force MAJUSCULES + .GLG). rend le
// nom logique (cle du registre) et le chemin reel
func (i *Interp) resolveDataPath(name string) (string, string, error) {
	name = strings.TrimSpace(name)
	if name == "" || filepath.IsAbs(name) {
		return "", "", errNomInvalide
	}
	clean := filepath.Clean(filepath.FromSlash(name))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || filepath.IsAbs(clean) {
		return "", "", errNomInvalide
	}
	dir, err := i.ensureWorkDir()
	if err != nil {
		return "", "", err
	}
	return clean, filepath.Join(dir, clean), nil
}

// un argument entier >= 0 (refuse 1.5 et les negatifs). rend un badData sinon, pour
// le message maison "<PRIMITIVE> N'AIME PAS <valeur>" (cf invoke). pour LISCARS et
// les positions, ou un demi-octet n'aurait aucun sens
func wholeArg(v Value) (int64, error) {
	n, err := toNumber(v)
	if err != nil {
		return 0, err
	}
	if n < 0 || float64(int64(n)) != n {
		return 0, &badData{v.String()}
	}
	return int64(n), nil
}

// intArg : un argument entier de signe quelconque (taille, indice, position).
// passe par l'entier EXACT (asIntOperand) et non par float64 : refuse une valeur
// non entiere (2.9) ET ne tronque/arrondit jamais un grand entier exact (un KInt
// > 2^53, ou un float64 deja imprecis, est rejete plutot qu'accepte de travers).
// les controles de plage restent a la charge de l'appelant. badData -> message
// maison "<PRIMITIVE> N'AIME PAS <valeur>".
func intArg(v Value) (int, error) {
	b, ok := asIntOperand(v)
	if !ok || !b.IsInt64() {
		return 0, &badData{v.String()}
	}
	n := b.Int64()
	if int64(int(n)) != n { // ne tient pas dans un int (plateforme 32 bits)
		return 0, &badData{v.String()}
	}
	return int(n), nil
}

// --- lecture/ecriture bas niveau sur le descripteur (en runes, pas en octets) ---

// nombre d'octets de la rune d'apres son 1er octet (UTF-8)
func runeLen(b byte) int {
	switch {
	case b < 0x80:
		return 1
	case b < 0xC0:
		return 1 // octet de continuation isole : traite comme 1 (deviendra RuneError)
	case b < 0xE0:
		return 2
	case b < 0xF0:
		return 3
	default:
		return 4
	}
}

// lit une rune. eof=true (rune 0) s'il n'y a plus rien
func (of *openFile) readRune() (rune, bool, error) {
	var buf [4]byte
	n, err := of.f.Read(buf[:1])
	if n == 0 {
		if err == nil || err == io.EOF {
			return 0, true, nil
		}
		return 0, false, err
	}
	size := runeLen(buf[0])
	total := 1
	for total < size { // complete la rune (s'arrete si le fichier se termine)
		m, _ := of.f.Read(buf[total : total+1])
		if m == 0 {
			break
		}
		total++
	}
	r, _ := utf8.DecodeRune(buf[:total])
	return r, false, nil
}

// lit une ligne (sans le saut final, \r\n tolere). eof=true si rien a lire.
// une derniere ligne sans saut compte comme une ligne (comportement FMSLogo)
func (of *openFile) readLine() (string, bool, error) {
	var b strings.Builder
	var one [1]byte
	got := false
	for {
		n, err := of.f.Read(one[:])
		if n == 0 {
			if err != nil && err != io.EOF {
				return "", false, err
			}
			if !got {
				return "", true, nil
			}
			return b.String(), false, nil
		}
		got = true
		if one[0] == '\n' {
			return strings.TrimSuffix(b.String(), "\r"), false, nil
		}
		b.WriteByte(one[0])
	}
}

// lit n runes et les rend comme un mot. eof=true si rien a lire
func (of *openFile) readChars(n int) (string, bool, error) {
	if n <= 0 {
		return "", false, nil
	}
	var b strings.Builder
	for k := 0; k < n; k++ {
		r, eof, err := of.readRune()
		if err != nil {
			return "", false, err
		}
		if eof {
			if k == 0 {
				return "", true, nil
			}
			break
		}
		b.WriteRune(r)
	}
	return b.String(), false, nil
}

// fin de fichier ? teste sans rien consommer (lit 1 octet puis recule)
func (of *openFile) atEOF() (bool, error) {
	var one [1]byte
	n, err := of.f.Read(one[:])
	if n == 0 {
		if err == nil || err == io.EOF {
			return true, nil
		}
		return false, err
	}
	if _, err := of.f.Seek(-1, io.SeekCurrent); err != nil {
		return false, err
	}
	return false, nil
}

func (of *openFile) pos() (int64, error) { return of.f.Seek(0, io.SeekCurrent) }
func (of *openFile) size() (int64, error) {
	info, err := of.f.Stat()
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// deplace le pointeur. refuse une position hors fichier ou tombant au milieu d'une
// rune UTF-8 (POSITION INVALIDE) : on ne lit jamais un demi-caractere
func (of *openFile) setPos(p int64) error {
	if p < 0 {
		return errPositionInvalide
	}
	sz, err := of.size()
	if err != nil {
		return errLectureImpossible
	}
	if p > sz {
		return errPositionInvalide
	}
	if p < sz {
		if _, err := of.f.Seek(p, io.SeekStart); err != nil {
			return errPositionInvalide
		}
		var one [1]byte
		if n, _ := of.f.Read(one[:]); n == 1 && one[0]&0xC0 == 0x80 {
			return errPositionInvalide // octet de continuation : milieu de caractere
		}
	}
	if _, err := of.f.Seek(p, io.SeekStart); err != nil {
		return errPositionInvalide
	}
	return nil
}

// --- les primitives ---

func (i *Interp) registerFileIO() {
	op := func(arity int, fn func(*Interp, []Value) (Value, error), names ...string) {
		i.register(&primitive{arity: arity, reporter: true, fn: fn}, names...)
	}

	// ouverture : un fichier ne peut etre ouvert qu'une fois a la fois
	i.register(cmd(1, func(in *Interp, a []Value) error { return in.openData(a[0], modeRead) }), "OUVRELECTURE")
	i.register(cmd(1, func(in *Interp, a []Value) error { return in.openData(a[0], modeWrite) }), "OUVREECRITURE")
	i.register(cmd(1, func(in *Interp, a []Value) error { return in.openData(a[0], modeAppend) }), "OUVREAJOUT")
	i.register(cmd(1, func(in *Interp, a []Value) error { return in.openData(a[0], modeUpdate) }), "OUVREMAJ")

	// FIXEFINLIGNE "LF | "CRLF : fin de ligne ecrite dans les fichiers (defaut LF).
	// la lecture accepte toujours les deux. utile pour produire un fichier facon Windows
	i.register(cmd(1, func(in *Interp, a []Value) error {
		switch strings.ToUpper(a[0].String()) {
		case "LF":
			in.eolCRLF = false
		case "CRLF":
			in.eolCRLF = true
		default:
			return fmt.Errorf("FIXEFINLIGNE VEUT \"LF OU \"CRLF")
		}
		return nil
	}), "FIXEFINLIGNE", "SETEOL")

	// FINLIGNE : rend "LF ou "CRLF, le reglage courant
	op(0, func(in *Interp, a []Value) (Value, error) {
		if in.eolCRLF {
			return WordValue("CRLF"), nil
		}
		return WordValue("LF"), nil
	}, "FINLIGNE", "EOL")

	// FERME nomfichier : ferme le fichier (et le retire des flux courants)
	i.register(cmd(1, func(in *Interp, a []Value) error {
		of, _, err := in.findOpen(a[0])
		if err != nil {
			return err
		}
		return in.closeFile(of)
	}), "FERME")

	// FERMETOUT : ferme tous les fichiers ouverts
	i.register(cmd(0, func(in *Interp, a []Value) error {
		fs := in.fileSet()
		var firstErr error
		for _, of := range fs.open {
			if err := of.f.Close(); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		fs.open = map[string]*openFile{}
		fs.curRead, fs.curWrite = nil, nil
		return firstErr
	}), "FERMETOUT")

	// TOUSOUVERTS : liste des noms de fichiers actuellement ouverts
	op(0, func(in *Interp, a []Value) (Value, error) {
		fs := in.fileSet()
		names := make([]string, 0, len(fs.open))
		for n := range fs.open {
			names = append(names, n)
		}
		sort.Strings(names)
		out := make([]Datum, len(names))
		for k, n := range names {
			out[k] = Datum{Kind: DWord, Text: n}
		}
		return ListValue(out), nil
	}, "TOUSOUVERTS")

	// FIXELECTURE nomfichier / [ ] : choisit le flux de lecture courant (liste vide = clavier)
	i.register(cmd(1, func(in *Interp, a []Value) error { return in.setCurrent(a[0], false) }), "FIXELECTURE")
	// FIXEECRITURE nomfichier / [ ] : choisit le flux d'ecriture courant (liste vide = console)
	i.register(cmd(1, func(in *Interp, a []Value) error { return in.setCurrent(a[0], true) }), "FIXEECRITURE")

	// FLUXLECTURE / FLUXECRITURE : nom du flux courant ([ ] si clavier/console)
	op(0, func(in *Interp, a []Value) (Value, error) { return streamName(in.fileSet().curRead), nil }, "FLUXLECTURE")
	op(0, func(in *Interp, a []Value) (Value, error) { return streamName(in.fileSet().curWrite), nil }, "FLUXECRITURE")

	// FINFICHIER? : VRAI s'il n'y a plus rien a lire dans le flux courant (FAUX au clavier)
	op(0, func(in *Interp, a []Value) (Value, error) {
		of := in.curReadStream()
		if of == nil {
			return BoolValue(false), nil
		}
		eof, err := of.atEOF()
		if err != nil {
			return Value{}, errLectureImpossible
		}
		return BoolValue(eof), nil
	}, "FINFICHIER?")

	// POSLECTURE / POSECRITURE : position d'octet courante (-1 si clavier/console)
	op(0, func(in *Interp, a []Value) (Value, error) { return streamPos(in.fileSet().curRead) }, "POSLECTURE")
	op(0, func(in *Interp, a []Value) (Value, error) { return streamPos(in.fileSet().curWrite) }, "POSECRITURE")

	// FIXEPOSLECTURE / FIXEPOSECRITURE octet : deplace le pointeur du flux courant
	i.register(cmd(1, func(in *Interp, a []Value) error { return in.setStreamPos(in.fileSet().curRead, a[0]) }), "FIXEPOSLECTURE")
	i.register(cmd(1, func(in *Interp, a []Value) error { return in.setStreamPos(in.fileSet().curWrite, a[0]) }), "FIXEPOSECRITURE")

	// LISLIGNE : lit une ligne brute comme un mot (LE recepteur pour parcourir un
	// fichier de donnees ligne par ligne). [ ] en fin de fichier
	op(0, func(in *Interp, a []Value) (Value, error) {
		of := in.curReadStream()
		if of == nil { // pas de flux : on lit le clavier comme LISMOT
			if in.keyb == nil {
				return Value{}, fmt.Errorf("CLAVIER INDISPONIBLE")
			}
			line, ok := in.keyb.ReadLine()
			if !ok {
				return Value{}, ErrInterrompu
			}
			return WordValue(line), nil
		}
		line, eof, err := of.readLine()
		if err != nil {
			return Value{}, errLectureImpossible
		}
		if eof {
			return ListValue(nil), nil
		}
		return WordValue(line), nil
	}, "LISLIGNE")

	// LISCARS n : lit n caracteres et les rend comme un mot. [ ] en fin de fichier.
	// n doit etre un entier >= 0 (pas de demi-caractere)
	op(1, func(in *Interp, a []Value) (Value, error) {
		n, err := wholeArg(a[0])
		if err != nil {
			return Value{}, err
		}
		of := in.curReadStream()
		if of == nil { // au clavier : lit n caracteres un a un
			if in.keyb == nil {
				return Value{}, fmt.Errorf("CLAVIER INDISPONIBLE")
			}
			var b strings.Builder
			for k := 0; k < int(n); k++ {
				r, ok := in.keyb.ReadChar()
				if !ok {
					return Value{}, ErrInterrompu
				}
				b.WriteRune(r)
			}
			return WordValue(b.String()), nil
		}
		w, eof, err := of.readChars(int(n))
		if err != nil {
			return Value{}, errLectureImpossible
		}
		if eof {
			return ListValue(nil), nil
		}
		return WordValue(w), nil
	}, "LISCARS")

	// FICHIERVERSTABLEAU nomfichier : charge tout un fichier TEXTE d'un coup et rend
	// un tableau dont chaque case est une ligne (1re ligne -> case 1). COMPTE donne
	// le nombre de lignes, pratique pour iterer. refuse un fichier binaire
	op(1, func(in *Interp, a []Value) (Value, error) {
		name, err := toWord(a[0])
		if err != nil {
			return Value{}, err
		}
		_, path, err := in.resolveDataPath(name)
		if err != nil {
			return Value{}, err
		}
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				return Value{}, errIntrouvable
			}
			return Value{}, errLectureImpossible
		}
		if info.IsDir() {
			return Value{}, errLectureImpossible
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return Value{}, errLectureImpossible
		}
		if !isTextData(data) {
			return Value{}, errBinaire // un binaire ferait n'importe quoi en "lignes"
		}
		lines := splitLines(string(data))
		items := make([]Value, len(lines))
		for k, l := range lines {
			items[k] = WordValue(l)
		}
		return ArrayValue(&Array{Items: items, Origin: 1}), nil
	}, "FICHIERVERSTABLEAU")
}

var errBinaire = fmt.Errorf("FICHIER BINAIRE")

// un contenu est "texte" s'il n'a pas d'octet nul et est de l'UTF-8 valide.
// heuristique simple mais qui ecarte les vrais binaires (images, .exe...)
func isTextData(b []byte) bool {
	if bytes.IndexByte(b, 0) >= 0 {
		return false
	}
	return utf8.Valid(b)
}

// decoupe un texte en lignes (\r\n tolere). un saut final ne cree pas une derniere
// ligne vide ; un fichier vide donne zero ligne
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, "\n")
	if n := len(parts); n > 0 && parts[n-1] == "" {
		parts = parts[:n-1]
	}
	for i := range parts {
		parts[i] = strings.TrimSuffix(parts[i], "\r")
	}
	return parts
}

// ouvre un fichier de donnees dans le mode demande, en verifiant toutes les
// preconditions (deja ouvert, introuvable, dossier, droits...)
func (i *Interp) openData(nameVal Value, mode fileMode) error {
	name, err := toWord(nameVal)
	if err != nil {
		return err
	}
	key, path, err := i.resolveDataPath(name)
	if err != nil {
		return err
	}
	fs := i.fileSet()
	if _, ok := fs.open[key]; ok {
		return errDejaOuvert
	}
	var f *os.File
	switch mode {
	case modeRead:
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				return errIntrouvable
			}
			return errLectureImpossible
		}
		if info.IsDir() {
			return errLectureImpossible
		}
		f, err = os.Open(path)
		if err != nil {
			return errLectureImpossible
		}
	case modeWrite:
		// ecrase sans confirmer (fidelite FMSLogo) : pas de confirmOverwrite ici
		f, err = os.Create(path)
		if err != nil {
			return errEcritureImpossible
		}
	case modeAppend:
		// pas d'O_APPEND : sinon le noyau force chaque ecriture en fin de fichier,
		// meme apres FIXEPOSECRITURE. on ouvre en ecriture et on se place en fin a
		// l'ouverture (les premieres ecritures ajoutent), tout en laissant le
		// pointeur d'ecriture reellement deplacable.
		f, err = os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return errEcritureImpossible
		}
		if _, err = f.Seek(0, io.SeekEnd); err != nil {
			f.Close()
			return errEcritureImpossible
		}
	case modeUpdate:
		f, err = os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
		if err != nil {
			return errEcritureImpossible
		}
	}
	fs.open[key] = &openFile{name: key, path: path, f: f, mode: mode}
	return nil
}

// retrouve un fichier ouvert par son nom (erreur FICHIER NON OUVERT sinon)
func (i *Interp) findOpen(nameVal Value) (*openFile, string, error) {
	name, err := toWord(nameVal)
	if err != nil {
		return nil, "", err
	}
	key, _, err := i.resolveDataPath(name)
	if err != nil {
		return nil, "", err
	}
	of, ok := i.fileSet().open[key]
	if !ok {
		return nil, key, errNonOuvert
	}
	return of, key, nil
}

// ferme un fichier et le retire des flux courants au besoin ; rend l'erreur de Close
func (i *Interp) closeFile(of *openFile) error {
	err := of.f.Close()
	fs := i.fileSet()
	delete(fs.open, of.name)
	if fs.curRead == of {
		fs.curRead = nil
	}
	if fs.curWrite == of {
		fs.curWrite = nil
	}
	return err
}

// FIXELECTURE/FIXEECRITURE : pose le flux courant (write=true pour l'ecriture).
// une liste vide [ ] revient au clavier / a la console
func (i *Interp) setCurrent(arg Value, write bool) error {
	fs := i.fileSet()
	if arg.Kind == KList && len(arg.List) == 0 {
		if write {
			fs.curWrite = nil
		} else {
			fs.curRead = nil
		}
		return nil
	}
	of, _, err := i.findOpen(arg)
	if err != nil {
		return err
	}
	if write {
		if !of.mode.writable() {
			return errMauvaisMode
		}
		fs.curWrite = of
	} else {
		if !of.mode.readable() {
			return errMauvaisMode
		}
		fs.curRead = of
	}
	return nil
}

// FIXEPOSLECTURE/FIXEPOSECRITURE : deplace le pointeur du flux donne
func (i *Interp) setStreamPos(of *openFile, posVal Value) error {
	if of == nil {
		return errNonOuvert
	}
	p, err := wholeArg(posVal) // une position d'octet : un entier, pas 1.5
	if err != nil {
		return err
	}
	return of.setPos(p)
}

// nom du flux pour FLUXLECTURE/FLUXECRITURE ([ ] si clavier/console)
func streamName(of *openFile) Value {
	if of == nil {
		return ListValue(nil)
	}
	return WordValue(of.name)
}

// position pour POSLECTURE/POSECRITURE (-1 si clavier/console)
func streamPos(of *openFile) (Value, error) {
	if of == nil {
		return NumberValue(-1), nil
	}
	p, err := of.pos()
	if err != nil {
		return Value{}, errLectureImpossible
	}
	return NumberValue(float64(p)), nil
}

// ecrit du texte sur le flux d'ecriture courant s'il y en a un, sinon sur la
// console. utilise par ECRIS/MONTRE/TAPE
func (i *Interp) writeText(s string) error {
	if i.fio != nil && i.fio.curWrite != nil {
		// en mode CRLF (FIXEFINLIGNE "CRLF) on traduit les sauts de ligne a
		// l'ecriture fichier ; la console reste en LF, elle n'en a pas besoin
		if i.eolCRLF {
			s = strings.ReplaceAll(s, "\n", "\r\n")
		}
		if _, err := i.fio.curWrite.f.WriteString(s); err != nil {
			return errEcritureImpossible
		}
		return nil
	}
	fmt.Fprint(i.Out, s)
	return nil
}
