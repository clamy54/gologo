package logo

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"

	"beroot.com/logo/turtle"
)

// renvoyee par QUITTE pour demander l'arret propre de l'appli (le backend
// graphique l'intercepte)
var ErrQuitter = errors.New("QUITTE")

// l'utilisateur a coupe (touche PAUSE)
var ErrInterrompu = errors.New("INTERROMPU !")

// pile d'appels pleine (recursion). erreur de ressource, pas rattachee a une
// procedure (donc pas de "DANS ...")
var errPlusDePlace = errors.New("PLUS DE PLACE")

// erreur de conversion ; invoke y accroche le nom de la primitive pour faire
// "<PRIMITIVE> N'AIME PAS <objet>"
type badData struct{ obj string }

func (e *badData) Error() string { return "N'AIME PAS " + e.obj }

// rattache une erreur a la proc ou elle s'est produite : "<erreur> DANS <proc>".
// pose une seule fois, tout au fond ; au-dessus on la laisse passer
type procErr struct {
	inner error
	proc  string
}

func (e *procErr) Error() string { return e.inner.Error() + " DANS " + e.proc }

// garde la sentinelle interne joignable (errors.Is traverse le "DANS")
func (e *procErr) Unwrap() error { return e.inner }

// l'interpreteur Logo : pilote une tortue et tient les tables des primitives,
// des procedures utilisateur et des variables
type Interp struct {
	Turtle              *turtle.Turtle
	Out                 io.Writer
	prims               map[string]*primitive
	procs               map[string]*userProc
	vars                map[string]Value       // variables globales
	plists              map[string][]propEntry // listes de proprietes (PLIST/PPROP...), cle = nom en MAJUSCULES
	frames              []map[string]Value     // pile des contextes locaux (appels de procs)
	repStack            []int                  // compteurs de boucle REPETE (COMPTEUR/REPCOUNT)
	testResult, testSet bool                   // resultat du dernier TESTE (SIVRAI/SIFAUX)
	brk                 atomic.Bool            // drapeau partage : interruption demandee (PAUSE)
	editor              Editor                 // ouvre l'editeur ED
	edBuf               string                 // contenu garde pour un ED sans argument
	help                Helper                 // navigateur d'aide AIDE
	keyb                Keyboard               // clavier LISCAR/LL/TOUCHE?
	mouse               Mouse                  // souris POSOPT/CONTACT?
	sound               Sound                  // son JOUE
	joy                 Joystick               // manettes MANETTE/BOUTON?

	// reglages musicaux (OCTAVE/DUREE/TEMPO/TIMBRE)
	musOctave, musDuree, musTempo, musTimbre int
	spriteDefs                               map[int][]string // formes 16x16 de DEFSPRITE (numero >= turtle.SpriteCount)
	pager                                    Pager            // pagination de la sortie texte (CATALOGUE...)
	lang                                     string           // langue courante : "FR" (defaut) ou "EN"
	langMu                                   sync.Mutex       // protege lang (le worker ecrit, l'UI lit)
	workDir                                  string           // dossier des .GLG (vide => ~/Logo)
	examplesDir                              string           // dossier des exemples (vide => defaut plateforme) ; commandes ...EX
}

// bascule la langue (Ctrl+L dans le navigateur) ; rend les donnees regenerees et
// la traduction de current dans la nouvelle langue (ou "")
type HelpSwitch func(lang, current string) (names []string, details map[string][]string, newCurrent string)

// ouvre le navigateur d'aide plein ecran. names = grille, details = fiches,
// start = "" pour la liste ou un nom canonique, switchLang sur Ctrl+L
type Helper func(names []string, details map[string][]string, start, lang string, switchLang HelpSwitch)

func (i *Interp) SetHelper(h Helper) { i.help = h }

// pagine un texte multi-lignes, facon "| more". nil en headless
type Pager func(title string, lines []string)

func (i *Interp) SetPager(p Pager) { i.pager = p }

// envoie un texte multi-lignes au pager si branche, sinon l'imprime ligne a
// ligne sur Out (headless / tests)
func (i *Interp) showPaged(title string, lines []string) {
	if i.pager != nil {
		i.pager(title, lines)
		return
	}
	for _, l := range lines {
		fmt.Fprintln(i.Out, l)
	}
}

// langue courante "FR"/"EN", pour localiser l'UI du backend (cf barre de statut
// de l'editeur). lecture verrouillee : l'UI la consulte de son cote
func (i *Interp) Lang() string {
	i.langMu.Lock()
	defer i.langMu.Unlock()
	return i.lang
}

// change la langue (ecriture verrouillee)
func (i *Interp) setLang(l string) {
	i.langMu.Lock()
	i.lang = l
	i.langMu.Unlock()
}

// ouvre l'editeur plein ecran. valider=false = on a annule (Ctrl+C). nil en headless
type Editor func(initial string) (texte string, valider bool)

func (i *Interp) SetEditor(e Editor) { i.editor = e }

// lit le clavier (LISCAR/LL/TOUCHE?). nil en headless.
// ok==false = Ctrl+C : la primitive renvoie alors ErrInterrompu
type Keyboard interface {
	ReadChar() (r rune, ok bool)
	ReadLine() (line string, ok bool)
	KeyPending() bool
}

func (i *Interp) SetKeyboard(k Keyboard) { i.keyb = k }

// position et etat de la souris pour POSOPT/CONTACT?. MousePos rend les coords
// Logo et si le pointeur est dans le champ ; MouseDown si un bouton est presse
type Mouse interface {
	MousePos() (x, y float64, inField bool)
	MouseDown() bool
}

func (i *Interp) SetMouse(m Mouse) { i.mouse = m }

// joue une note (backend audio) pour JOUE. Tone bloque le temps de la note ;
// freq <= 0 = silence ; timbre 0-255 choisit la forme d'onde. nil = pas de son
type Sound interface {
	Tone(freqHz float64, ms int, timbre int)
}

func (i *Interp) SetSound(s Sound) { i.sound = s }

// manettes emulees au clavier (MANETTE/BOUTON?). JoyDir rend 0-8 (0 = repos,
// 1 = haut, horaire) ; JoyButton si le tir est presse. le clavier fait la manette 0
type Joystick interface {
	JoyDir(n int) int
	JoyButton(n int) bool
}

func (i *Interp) SetJoystick(j Joystick) { i.joy = j }

// Filler / Labeler : trucs que le backend graphique sait faire, cherches sur i.Out.
// REMPLIS peint la zone sous la tortue ; ETIQUETTE ecrit du texte dans le champ.
// absents sans ecran (la primitive ne fait alors rien)
type Filler interface {
	Fill(x, y float64, c turtle.Color)
}
type Labeler interface {
	Label(x, y float64, text string, c turtle.Color)
}

// demande l'arret du programme (Ctrl+C / PAUSE). appelable depuis plusieurs
// taches ; pris en compte aux points de controle de exec. coupe aussi les anims
// en cours, histoire de ne pas laisser des sprites gigoter tout seuls
func (i *Interp) Interrupt() {
	i.brk.Store(true)
	if i.Turtle != nil {
		i.Turtle.StopAllMotion()
	}
}

// une primitive native : soit fn (arite fixe), soit special (forme speciale qui
// lit elle-meme ses arguments via le curseur)
type primitive struct {
	name     string
	arity    int  // nb d'arguments hors parentheses
	reporter bool // rend une valeur (operation)
	variadic bool // arite variable entre parentheses
	fn       func(i *Interp, args []Value) (Value, error)
	special  func(e *eval) (Value, error)
}

// une procedure utilisateur (POUR ... FIN)
type userProc struct {
	name   string
	params []string
	body   []Datum
	text   string // source brute (saisie editeur) ; vide => on la reconstruit
}

// texte source de la proc pour l'editeur : la saisie brute si on l'a, sinon on
// rebatit un "POUR nom :p ... <corps> FIN"
func (p *userProc) sourceText() string {
	if p.text != "" {
		return p.text
	}
	var b strings.Builder
	b.WriteString("POUR ")
	b.WriteString(p.name)
	for _, par := range p.params {
		b.WriteString(" :")
		b.WriteString(par)
	}
	b.WriteByte('\n')
	parts := make([]string, len(p.body))
	for i, d := range p.body {
		parts[i] = d.String()
	}
	b.WriteString(strings.Join(parts, " "))
	b.WriteString("\nFIN")
	return b.String()
}

// signal de controle de flux (STOP / RENDS / LOGO), trimballe comme une erreur
// pour casser l'execution en cours
type ctrl struct {
	kind ctrlKind
	val  Value
}

type ctrlKind int

const (
	ctlStop   ctrlKind = iota // STOP : fin de la procedure (commande)
	ctlOutput                 // RENDS : fin + une valeur (operation)
	ctlLogo                   // LOGO : arret total, retour tout en haut
)

func (c *ctrl) Error() string { return "ctrl" }

// signal interne d'appel en position terminale. au lieu d'empiler un appel Go de
// plus, callProc l'attrape et reutilise le contexte courant (trampoline) : la
// recursion terminale ne mange donc pas de pile. ne sort jamais d'un corps de proc
type tailCall struct {
	proc   *userProc
	args   []Value
	output bool // true = RENDS <proc> (on attend une valeur) ; false = appel commande nu (rien ne doit remonter)
}

func (t *tailCall) Error() string { return "tailcall" }

// cree un interpreteur relie a une tortue et a une sortie texte
func New(t *turtle.Turtle, out io.Writer) *Interp {
	i := &Interp{
		Turtle: t,
		Out:    out,
		prims:  map[string]*primitive{},
		procs:  map[string]*userProc{},
		vars:   map[string]Value{},
		plists: map[string][]propEntry{},
		lang:   "FR",
		// workDir vide => ~/Logo par defaut, resolu paresseusement (voir files.go)
		musOctave: octaveDefaut, musDuree: dureeDefaut, musTempo: tempoDefaut, musTimbre: timbreDefaut,
	}
	i.registerBuiltins()
	i.registerOperations()
	i.registerWorkspace()
	i.registerScreenExt()
	i.registerFiles()          // SAUVE / RAMENE / CHARGE / SAUVED / CATALOGUE / DETRUIS
	i.registerPrinter()        // COPIE (copie d'ecran PNG numerotee)
	i.registerPlist()          // DONNEPROP / PROP / EFPROP / LISTEPROP (listes de proprietes)
	i.registerDefine()         // DEFINIS / TEXTE (procedures comme donnees)
	i.registerIODevices()      // clavier / souris / manettes (LISCAR/SOURISPOS/MANETTE...)
	i.registerMusic()          // OCTAVE / DUREE / TEMPO / TIMBRE / JOUE
	i.registerIterators()      // POURCHAQUE / APPLIQUE / FILTRE / REDUIS
	i.registerControl2()       // PIEGE / LANCE / TESTE / SIVRAI / SIFAUX
	i.registerHardwareCompat() // peripheriques d'origine en no-op
	i.registerLang()           // FRANCAIS / ENGLISH
	i.registerMultiTurtle()    // FIXETORTUE / TORTUE / NBTORTUES / DEMANDE / DISTORTUE
	i.registerSprites()        // DEFSPRITE (formes 16x16 redefinissables)
	i.registerAnimate()        // ANIME / STOPANIME / CADENCE (animation automatique)
	i.registerEnglishAliases() // alias anglais (lus depuis helpData)
	i.registerXLogoAliases()   // alias XLogo (noms longs francophones) ; n'ecrase rien
	i.registerFMSLogoAliases() // alias FMSLogo (noms anglais Windows) ; n'ecrase rien

	return i
}

func (i *Interp) register(p *primitive, names ...string) {
	for _, n := range names {
		i.prims[strings.ToUpper(n)] = p
	}
}

// lit puis execute une source Logo complete
func (i *Interp) RunString(src string) error {
	i.brk.Store(false) // arme la detection d'interruption pour ce programme
	data, err := Read(src)
	if err != nil {
		return err
	}
	err = i.runSeq(data)
	if c, ok := err.(*ctrl); ok {
		_ = c // tout en haut, les signaux de controle sont avales
		return nil
	}
	return err
}

// execute une suite d'instructions sans recursion terminale (tout appel de proc
// empile un contexte). cas general : programme, corps de boucle, EXEC...
func (i *Interp) runSeq(data []Datum) error {
	return (&eval{i: i, data: data}).exec()
}

// comme runSeq mais si tail est vrai, la derniere instruction est terminale (son
// resultat devient celui de la proc englobante). pour le corps des procs et les
// branches terminales de SI, pour que callProc deroule la TCO
func (i *Interp) runSeqTail(data []Datum, tail bool) error {
	return (&eval{i: i, data: data, tailOK: tail}).exec()
}

// deroule les instructions du curseur jusqu'au bout
func (e *eval) exec() error {
	for e.pos < len(e.data) {
		if e.i.brk.Load() { // point de controle d'interruption (PAUSE)
			return ErrInterrompu
		}
		if err := e.statement(); err != nil {
			return err
		}
	}
	return nil
}

// variables : on cherche dans les contextes locaux, puis dans les globales

func (i *Interp) lookupVar(name string) (Value, bool) {
	name = strings.ToUpper(name)
	for k := len(i.frames) - 1; k >= 0; k-- {
		if v, ok := i.frames[k][name]; ok {
			return v, true
		}
	}
	v, ok := i.vars[name]
	return v, ok
}

func (i *Interp) setVar(name string, v Value) {
	name = strings.ToUpper(name)
	for k := len(i.frames) - 1; k >= 0; k-- {
		if _, ok := i.frames[k][name]; ok {
			i.frames[k][name] = v
			return
		}
	}
	i.vars[name] = v // cree ou met a jour une globale
}

// le backend sait-il remettre l'affichage dans l'etat de demarrage ? (cherche sur
// i.Out ; absent sans ecran)
type displayResetter interface {
	ResetScreen()
}

// RAZ : remet l'interpreteur et l'affichage a neuf. procedures et variables
// effacees, parametres par defaut, ecran reinitialise
func (i *Interp) resetAll() {
	i.procs = map[string]*userProc{}
	i.vars = map[string]Value{}
	i.plists = map[string][]propEntry{}
	i.spriteDefs = nil // RAZ efface aussi les sprites definis (VE les conserve)
	i.frames = nil
	i.repStack = nil
	i.edBuf = ""
	i.musOctave, i.musDuree, i.musTempo, i.musTimbre = octaveDefaut, dureeDefaut, tempoDefaut, timbreDefaut
	i.setLang("FR")
	i.Turtle.Reset() // tortue recentree + graphique efface + fond bleu
	if r, ok := i.Out.(displayResetter); ok {
		r.ResetScreen() // texte (banniere), champ cache, couleurs par defaut
	}
	// RAZ repasse en mode texte : Reset() a "active" la tortue mais l'ecran est
	// masque. on remet le drapeau a faux pour que la prochaine primitive graphique
	// (MT, AV...) rouvre bien le champ
	i.Turtle.ResetGraphicsShown()
}

// LOCAL : cree une variable locale dans le contexte courant (la proc en cours).
// tout en haut, ca cree une globale
func (i *Interp) declareLocal(name string) {
	name = strings.ToUpper(name)
	if n := len(i.frames); n > 0 {
		i.frames[n-1][name] = WordValue("")
	} else {
		i.vars[name] = WordValue("")
	}
}

// curseur d'evaluation sur un flux de Datum
type eval struct {
	i        *Interp
	data     []Datum
	pos      int
	group    bool // groupe ( ... ) : la tete peut avoir une arite variable
	tailOK   bool // la derniere instruction de la suite est en position terminale
	headTail bool // la tete courante est candidate a la TCO (lu par les formes speciales)
}

func (e *eval) atEnd() bool { return e.pos >= len(e.data) }

func (e *eval) next() Datum {
	d := e.data[e.pos]
	e.pos++
	return d
}

// execute une instruction : un symbole (commande) + ses arguments. une operation
// laissee seule (qui rend une valeur) declenche "QUE FAIRE DE"
func (e *eval) statement() error {
	d := e.data[e.pos]
	// un groupe tape seul, ex (AV 50 TD 90), est execute. group=true pour que la
	// tete variadique marche, ex (ECRIS a b c) ; sans effet sinon
	if d.Kind == DGroup && startsWithCommand(e.i, d.List) {
		e.pos++
		_, err := e.i.evalCode(d.List, true)
		return err
	}
	if d.Kind != DSymbol {
		return fmt.Errorf("QUE FAIRE DE %s", d.String())
	}
	e.pos++
	// la tete est candidate a la TCO si la suite l'autorise ; invoke0 confirmera
	// (seulement si plus rien ne suit l'appel)
	v, err := e.invoke(d.Text, e.tailOK)
	if err != nil {
		return err
	}
	if v.HasValue() {
		return fmt.Errorf("QUE FAIRE DE %s", v.String())
	}
	return nil
}

// appelle une primitive ou une proc ; colle le nom sur un badData pour faire
// "<PRIMITIVE> N'AIME PAS <objet>"
func (e *eval) invoke(name string, tail bool) (Value, error) {
	up := strings.ToUpper(name)
	v, err := e.invoke0(up, tail)
	var bd *badData
	if errors.As(err, &bd) {
		return Value{}, fmt.Errorf("%s N'AIME PAS %s", up, bd.obj)
	}
	return v, err
}

// appelle une primitive ou une proc nommee (up deja en majuscules). tail = cet
// appel pourrait etre terminal : si la cible est une proc utilisateur et que rien
// ne suit ses arguments, on renvoie un tailCall (deroule par callProc) plutot que
// d'empiler un appel
func (e *eval) invoke0(up string, tail bool) (Value, error) {
	// le drapeau "groupe" ne vaut que pour cet appel (la tete) ; les arguments
	// suivants se lisent en nombre fixe
	group := e.group
	e.group = false
	if p := e.i.prims[up]; p != nil {
		if p.special != nil {
			e.headTail = tail // lu par formeSi / formeRends
			return p.special(e)
		}
		if p.variadic && group { // arite variable ( OP a b c ... )
			args, err := e.readAllArgs(up)
			if err != nil {
				return Value{}, err
			}
			return p.fn(e.i, args)
		}
		args, err := e.readArgs(up, p.arity)
		if err != nil {
			return Value{}, err
		}
		return p.fn(e.i, args)
	}
	if proc := e.i.procs[up]; proc != nil {
		args, err := e.readArgs(up, len(proc.params))
		if err != nil {
			return Value{}, err
		}
		// position terminale : tete d'instruction et plus rien apres les arguments
		if tail && e.atEnd() {
			return Value{}, &tailCall{proc: proc, args: args}
		}
		return e.i.callProc(proc, args)
	}
	return Value{}, fmt.Errorf("COMMENT FAIRE %s", up)
}

// lit tous les arguments restants : ( OP a b c ... ) gobe tout le contenu du groupe
func (e *eval) readAllArgs(name string) ([]Value, error) {
	var args []Value
	for !e.atEnd() {
		v, err := e.expr(0)
		if err != nil {
			return nil, err
		}
		if !v.HasValue() {
			return nil, fmt.Errorf("PAS ASSEZ DE DONNEES POUR %s", name)
		}
		args = append(args, v)
	}
	return args, nil
}

// lit n arguments, chacun une expression qui rend un objet Logo
func (e *eval) readArgs(name string, n int) ([]Value, error) {
	args := make([]Value, n)
	for k := 0; k < n; k++ {
		if e.atEnd() {
			return nil, fmt.Errorf("PAS ASSEZ DE DONNEES POUR %s", name)
		}
		v, err := e.expr(0)
		if err != nil {
			return nil, err
		}
		if !v.HasValue() {
			return nil, fmt.Errorf("PAS ASSEZ DE DONNEES POUR %s", name)
		}
		args[k] = v
	}
	return args, nil
}

// plafond de recursion des procedures. au-dela on rend "PLUS DE PLACE", plutot
// que de laisser Go faire deborder la pile et mourir sans un mot
const maxCallDepth = 5000

// execute une proc utilisateur, parametres ranges en variables locales.
// trampoline : un appel terminal (tailCall) ne reempile pas d'appel Go ; on
// remplace proc/args et on reboucle sur place. la TCO tourne donc a pile constante
func (i *Interp) callProc(proc *userProc, args []Value) (Value, error) {
	// le trampoline aplatit la chaine d'appels terminaux ; on retient le dernier
	// appel "commande" et le dernier "RENDS" pour rejouer, sur la valeur finale, le
	// controle de type qu'aurait fait l'execution sans TCO :
	//   - une commande terminale ne doit PAS remonter de valeur (QUE FAIRE DE)
	//   - un RENDS <proc> terminal DOIT en remonter une (PAS ASSEZ DE DONNEES)
	var needNone, needValue bool
	var noneCaller, valueCaller string
	for {
		if len(i.frames) >= maxCallDepth {
			return Value{}, errPlusDePlace
		}
		frame := make(map[string]Value, len(proc.params))
		for k, p := range proc.params {
			frame[p] = args[k]
		}
		i.frames = append(i.frames, frame)
		err := i.runSeqTail(proc.body, true)
		i.frames = i.frames[:len(i.frames)-1]

		if tc, ok := err.(*tailCall); ok {
			// proc vient de faire un appel terminal : on note ce qu'il exige de la
			// valeur finale, puis on reboucle sur la cible (arguments deja evalues
			// dans le contexte qu'on quitte)
			if tc.output {
				needValue, valueCaller = true, proc.name
			} else {
				needNone, noneCaller = true, proc.name
			}
			proc, args = tc.proc, tc.args
			continue
		}
		v, ferr := i.finishProc(proc, err)
		if ferr != nil {
			return v, ferr
		}
		if v.HasValue() && needNone {
			return Value{}, &procErr{inner: fmt.Errorf("QUE FAIRE DE %s", v.String()), proc: noneCaller}
		}
		if !v.HasValue() && needValue {
			return Value{}, &procErr{inner: fmt.Errorf("PAS ASSEZ DE DONNEES POUR RENDS"), proc: valueCaller}
		}
		return v, nil
	}
}

// traduit le resultat d'un corps de proc (signal de controle ou erreur) en valeur
// de retour pour callProc
func (i *Interp) finishProc(proc *userProc, err error) (Value, error) {
	if c, ok := err.(*ctrl); ok {
		switch c.kind {
		case ctlStop:
			return None, nil
		case ctlOutput:
			return c.val, nil
		case ctlLogo:
			return None, c // on laisse remonter jusqu'en haut
		}
	}
	if err != nil {
		// rattache l'erreur a cette proc ("... DANS nom"), sauf interruption, QUITTE
		// ou erreur deja rattachee plus bas (on garde la proc la plus interne)
		if errors.Is(err, ErrInterrompu) || errors.Is(err, ErrQuitter) || errors.Is(err, errPlusDePlace) {
			return Value{}, err
		}
		if _, ok := err.(*procErr); ok {
			return Value{}, err
		}
		return Value{}, &procErr{inner: err, proc: proc.name}
	}
	return None, nil
}

// priorite d'un operateur infixe (0 = pas un operateur)
func precedence(op string) int {
	switch op {
	case "=", "<", ">", "<=", ">=", "<>":
		return 1
	case "+", "-":
		return 2
	case "*", "/":
		return 3
	}
	return 0
}

// evalue une expression a operateurs infixes (du moins au plus prioritaire)
func (e *eval) expr(minPrec int) (Value, error) {
	left, err := e.primary()
	if err != nil {
		return Value{}, err
	}
	return e.exprFrom(left, minPrec)
}

// continue une expression infixe a partir d'un terme de gauche deja evalue. ca
// laisse RENDS reperer un appel terminal avant d'enchainer un eventuel operateur,
// sans reevaluer le terme
func (e *eval) exprFrom(left Value, minPrec int) (Value, error) {
	for !e.atEnd() {
		d := e.data[e.pos]
		if d.Kind != DOp || d.Unary {
			break // un '-' unaire n'est jamais un operateur binaire ici
		}
		prec := precedence(d.Text)
		if prec == 0 || prec < minPrec {
			break
		}
		e.pos++
		right, err := e.expr(prec + 1)
		if err != nil {
			return Value{}, err
		}
		left, err = applyOp(d.Text, left, right)
		if err != nil {
			return Value{}, err
		}
	}
	return left, nil
}

// evalue un terme : valeur directe, variable, liste, parenthese, moins unaire, ou
// appel d'operation
func (e *eval) primary() (Value, error) {
	if e.atEnd() {
		return Value{}, fmt.Errorf("OBJET MANQUANT")
	}
	d := e.next()
	switch d.Kind {
	case DNumber:
		return NumberValue(d.Num), nil
	case DWord:
		return WordValue(d.Text), nil
	case DVarRef:
		v, ok := e.i.lookupVar(d.Text)
		if !ok {
			return Value{}, fmt.Errorf("PAS DE CHOSE DONNEE A %s", strings.ToUpper(d.Text))
		}
		return v, nil
	case DList:
		return ListValue(d.List), nil
	case DGroup:
		// (prim-commande ...) en position de valeur : traite comme une liste ;
		// sinon on evalue et on rend la valeur
		if startsWithPrimCommand(e.i, d.List) {
			return ListValue(d.List), nil
		}
		return e.i.evalGroupValue(d.List)
	case DOp:
		if d.Text == "-" { // moins unaire
			v, err := e.primary()
			if err != nil {
				return Value{}, err
			}
			n, err := toNumber(v)
			if err != nil {
				return Value{}, err
			}
			return NumberValue(-n), nil
		}
		return Value{}, fmt.Errorf("OPERATEUR INATTENDU %s", d.Text)
	case DSymbol:
		// appel en position de valeur : le resultat va etre reutilise (operateur,
		// argument...), donc jamais terminal ici (RENDS gere son cas a part)
		return e.invoke(d.Text, false)
	}
	return Value{}, fmt.Errorf("OBJET INATTENDU")
}

// name designe-t-il une operation (qui rend une valeur) ? les procs utilisateur
// comptent comme des commandes par defaut
func (i *Interp) isOperationName(name string) bool {
	if p := i.prims[strings.ToUpper(name)]; p != nil {
		return p.reporter
	}
	return false
}

// startsWithPrimCommand : data[0] est une primitive-commande (pas une operation).
// startsWithCommand juste apres : commande, primitive ou proc utilisateur
func startsWithPrimCommand(i *Interp, data []Datum) bool {
	if len(data) == 0 {
		return false
	}
	d := data[0]
	if d.Kind != DSymbol {
		return false
	}
	p := i.prims[strings.ToUpper(d.Text)]
	return p != nil && !p.reporter
}

func startsWithCommand(i *Interp, data []Datum) bool {
	if len(data) == 0 {
		return false
	}
	d := data[0]
	if d.Kind != DSymbol {
		return false // nombre, mot, :var, liste, operateur => expression
	}
	return !i.isOperationName(d.Text)
}

// evalue ( ... ) en position de valeur : la tete recoit tous les arguments, les
// eventuelles commandes qui suivent sont executees. jamais traite comme une liste
func (i *Interp) evalGroupValue(data []Datum) (Value, error) {
	if len(data) == 0 {
		return None, nil
	}
	ev := &eval{i: i, data: data, group: true}
	v, err := ev.expr(0)
	if err != nil {
		return Value{}, err
	}
	for !ev.atEnd() { // eventuelles commandes apres l'operation
		if err := ev.statement(); err != nil {
			return Value{}, err
		}
	}
	return v, nil
}

// execute data : si ca commence par une operation ou une valeur, on evalue et on
// rend le resultat ; sinon on execute une suite d'instructions et on rend None
func (i *Interp) evalCode(data []Datum, group bool) (Value, error) {
	if len(data) == 0 {
		return None, nil
	}
	ev := &eval{i: i, data: data, group: group}
	if startsWithCommand(i, data) {
		for !ev.atEnd() {
			if err := ev.statement(); err != nil {
				return Value{}, err
			}
		}
		return None, nil
	}
	v, err := ev.expr(0)
	if err != nil {
		return Value{}, err
	}
	for !ev.atEnd() { // eventuelles commandes apres l'operation
		if err := ev.statement(); err != nil {
			return Value{}, err
		}
	}
	return v, nil
}
