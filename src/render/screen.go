// Package render : l'interface graphique Gio (Wayland/X11/D3D/Metal). implemente
// turtle.Canvas et io.Writer. la scene est composee dans une image memoire
// (image.RGBA), Gio la recopie a l'ecran
package render

import (
	"beroot.com/logo/logo"
	"beroot.com/logo/turtle"
	"errors"
	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
	"image"
	"image/color"
	"image/draw"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// dimensions logiques. Gio met l'image composee a l'echelle de la fenetre en
// gardant le ratio
const (
	ScreenW    = 1280
	ScreenH    = 800
	fieldScale = 0.6 // 1 pas Logo -> 0,6 px ; champ 1600x1000 -> 960x600 px
	fieldW     = 960 // 1600 * fieldScale
	fieldH     = 600 // 1000 * fieldScale
	fieldX     = (ScreenW - fieldW) / 2
	fieldY     = 16
	textScale  = 2
	lineH      = 13*textScale + 6
	margin     = 16
	textCols   = 80 // grille texte adressable (FCURS/ME) : 80 colonnes (remplit la largeur)
	textRows   = 25 // ... x 25 lignes
)

// ecran d'accueil, au demarrage et apres RAZ (bilingue), suivi d'une ligne vide
// avant l'invite du REPL
const banner = "GoLogo v" + logo.Version + "\n" +
	"(C)2024-2026 Cyril LAMY\n" +
	"Press F1 for HELP or type BYE to quit\n" +
	"F1 pour l'aide, QUITTE pour sortir.\n\n"

// les codes couleur (0..15) vers du RGBA moderne
var palette = [16]color.RGBA{
	{0, 0, 0, 255}, {220, 0, 0, 255}, {0, 200, 0, 255}, {230, 230, 0, 255},
	{40, 60, 230, 255}, {220, 0, 220, 255}, {0, 230, 230, 255}, {255, 255, 255, 255},
	{128, 128, 128, 255}, {255, 120, 120, 255}, {140, 255, 140, 255}, {255, 255, 160, 255},
	{150, 170, 255, 255}, {255, 150, 255, 255}, {170, 255, 255, 255}, {255, 165, 0, 255},
}

func rgba(c turtle.Color) color.RGBA {
	if c.IsRGB() { // couleur RGB directe (FCC [ r v b ])
		r, g, b := c.Components()
		return color.RGBA{uint8(r), uint8(g), uint8(b), 255}
	}
	i := int(c)
	if i < 0 {
		i = -i - 1 // code negatif (couleur de fond)
	}
	return palette[((i%16)+16)%16]
}

// couleur de texte par defaut (gris clair). pour le REPL (modifiable via FCT) et
// le navigateur d'aide
var textDefault = color.RGBA{210, 210, 210, 255}

// bitmap de la tortue (cap 0 = pointe en haut), transcrit de tortue.png : triangle
// plein qui s'elargit, une ligne creuse (les "pieds"), puis la base
var turtleSprite = []string{
	"...#...",
	"..###..",
	"..###..",
	".#####.",
	".#####.",
	".#####.",
	"#.....#",
	"#######",
}

// tortue detaillee (SPRITE 1), cap 0 = tete en haut : tete a yeux, 4 nageoires,
// carapace a motif, petite queue. symetrique, 21 cellules de large
var turtleSprite1 = []string{
	"........#####........",
	".......#######.......",
	".......#..#..#.......",
	".......#######.......",
	"........#####........",
	"..##....#####....##..",
	".####..#######..####.",
	".#####.#######.#####.",
	".####.#########.####.",
	"..##..#########..##..",
	".....###########.....",
	".....###.###.###.....",
	".....##.#####.##.....",
	".....##.#####.##.....",
	".....###.###.###.....",
	".....###########.....",
	"..##..#########..##..",
	".####.#########.####.",
	".#####.#######.#####.",
	".####...#####...####.",
	"..##......#......##..",
	".....................",
	".....................",
}

// voiture vue de dessus (SPRITE 2), cap 0 = avant en haut : capot, pare-brise,
// retros, habitacle, coffre. symetrique, 19 cellules
var turtleSprite2 = []string{
	".......#####.......",
	".....#########.....",
	".....#.......#.....",
	"....#.........#....",
	"....#.........#....",
	"....#.........#....",
	"....#.#######.#....",
	"....#.#######.#....",
	"....#.........#....",
	"..###.........###..",
	"..###.........###..",
	"....#.#######.#....",
	"....#.#.....#.#....",
	"....#.#.....#.#....",
	"....#.#.....#.#....",
	"....#.#######.#....",
	"....#.........#....",
	"....#.#######.#....",
	"....#.#######.#....",
	"....#.........#....",
	"....#.........#....",
	"....#.........#....",
	".....#.......#.....",
	".....#########.....",
	"......#.....#......",
	"......#######......",
	".......#...#.......",
}

// un bitmap de tortue, son pas (pix = espacement des cellules) et la taille du carre
// dessine pour chaque cellule
type turtleShape struct {
	bm     []string
	pix    float64
	square int
}

// formes indexees par SPRITE : 0 triangle, 1 tortue detaillee, 2 voiture
var turtleShapes = []turtleShape{
	{turtleSprite, 2.5, 3},
	{turtleSprite1, 1.5, 2},
	// Voiture (27 lignes) : pix reduit a 1.1 pour que sa hauteur (~31 px) tienne
	// dans la boite de collision (collisionHalf 26 pas = ~31 px), la forme tournant
	// avec le cap. A 1.6 elle faisait ~44 px et debordait : deux voitures
	// pouvaient se chevaucher sans que COLLISION? le voie.
	{turtleSprite2, 1.1, 2},
}

// taille de rendu d'une forme 16x16 de DEFSPRITE : ~32 px de cote, comme les formes integrees
const (
	userSpritePix    = 2.0
	userSpriteSquare = 2
)

// execute une ligne Logo (branche apres coup pour eviter un cycle d'import)
type Runner func(src string) error

// compose la scene et gere le REPL. une tache de fond separee execute Logo ; le
// verrou mu protege l'etat partage avec la composition
type Screen struct {
	mu       sync.Mutex
	gfx      []gfxOp // journal graphique chronologique unique (traits/remplissages/etiquettes)
	clearGen uint64
	bg       turtle.Color
	border   turtle.Color   // couleur du bord (FCB), defaut noir
	turtles  []turtle.State // une fiche par tortue (multi-tortue) ; vide tant qu'aucune notif
	graphics bool

	userShapes map[int]turtleShape // formes 16x16 definies par DEFSPRITE (index >= 3)

	// grille texte adressable 80x25 (FCURS/ME), a la place d'un journal qui defile.
	// grid = caracteres, gridFg = couleur par cellule, curseur = ou on ecrit
	grid       [textRows][textCols]rune
	gridFg     [textRows][textCols]color.RGBA
	curRow     int  // ligne du curseur texte (0..24)
	curCol     int  // colonne du curseur texte (0..79)
	meLines    int  // ME : nb de lignes texte visibles (1..25), defaut 25
	meTextOnly bool // ME 25 : plein texte (cache le champ graphique)

	textCol color.RGBA // couleur du texte (FCT), defaut gris clair
	textBg  color.RGBA // fond de la zone texte (FCFT), defaut noir

	input    string // saisie courante (boucle d'evenements seule)
	inputCol int    // position du curseur dans `input` (en runes)
	run      Runner
	complete func(string) []string // noms de commandes pour l'autocompletion (TAB)
	startup  string
	started  bool

	history []string // 100 dernieres commandes (REPL ; fleches haut/bas)
	histIdx int      // position de navigation (== len(history) : nouvelle ligne)

	cmdCh     chan string
	running   atomic.Bool
	quit      atomic.Bool
	interrupt func()
	win       *app.Window

	// regle les modes modaux (editeur, aide, clavier) : la tache de fond remplit les
	// champs PUIS passe le flag atomique a true ; apres ca, ils sont a l'interface

	// editeur ED (modal plein ecran). edActive bascule le rendu et le routage
	// clavier ; la tache de fond attend sur edDone le temps de l'edition
	edActive     atomic.Bool
	edLines      []string // lignes editees (propriete de l'interface graphique)
	edRow, edCol int      // curseur / tete de selection (col en runes)
	edScroll     int      // 1re ligne visible
	edYank       string   // tampon CTRL+R/CTRL+B
	edDone       chan editResult

	// selection (Shift+deplacement) : ancre fixe, tete = curseur. surlignee tant que
	// edSel est vrai ; Ctrl+C copie, un deplacement ou une edition sans Shift la vide
	edSel                    bool
	edAnchorRow, edAnchorCol int

	// undo (Ctrl+Z) : pile de sauvegardes prises avant chaque modif ; une rafale de
	// frappes compte pour une seule etape (edTyping)
	edUndo   []edSnap
	edTyping bool

	// recherche dans l'editeur (Ctrl+F), facon page d'aide : champ dans la barre du
	// bas, Entree va a la 1re occurrence, Ctrl+N/Ctrl+P naviguent sans boucler. la
	// correspondance courante est en jaune ; edSearchRow/Col = son debut
	edSearchTyping bool
	edSearchInput  string
	edSearchRow    int
	edSearchCol    int

	// navigateur d'aide AIDE (plein ecran). pgActive bascule rendu et routage ; la
	// tache de fond attend sur pgDone
	pgActive   atomic.Bool
	pgNames    []string            // grille des commandes (mode liste)
	pgDetails  map[string][]string // nom canonique -> lignes de detail
	pgDetail   string              // "" = liste ; sinon nom dont on affiche le detail
	pgSel      int                 // index selectionne dans la grille
	pgScroll   int                 // defilement en mode detail
	pgLang     string              // langue de l'interface ("FR"/"EN")
	pgSwitch   logo.HelpSwitch     // bascule de langue a chaud (Ctrl+L)
	pgOverlay  bool                // aide ouverte en superposition (F1) : aucune tache n'attend pgDone
	pgExtended bool                // aide complete (Shift+F1 / AIDE) vs debutant (F1 = commandes d'origine)
	pgDone     chan struct{}

	// preference qui survit a la fermeture de l'aide : Shift+F1 bascule debutant <->
	// complet, et F1 rouvre dans ce mode-la (au lieu de toujours retomber en debutant)
	helpExtended bool

	// recherche dans l'aide generale (Ctrl+F ou '/') : saisie dans la barre du bas,
	// puis surbrillance jaune des commandes dont la fiche contient la phrase
	pgSearchTyping bool            // saisie en cours (la barre du bas devient un champ)
	pgSearchInput  string          // texte tape
	pgSearchHits   map[string]bool // commandes a surligner (page contenant la phrase)

	// lecture clavier (LISCAR/LL/TOUCHE?). la tache de fond attend sur kbResult ; l'interface route les frappes.
	// kbBuf = frappes en avance (sous verrou mu, max 256) : lu par TOUCHE?, consomme par LISCAR
	kbActive   atomic.Bool // une lecture clavier (LISCAR/LL) est en cours
	kbWantLine bool        // true = LL (ligne affichee), false = LISCAR (un car) ; propriete de l'interface
	kbInput    string      // ligne LL en cours de saisie (propriete de l'interface pendant la lecture)
	kbInputCol int
	kbResult   chan kbRes
	kbBuf      []rune // tampon clavier (protege par le verrou mu, max 256)

	// souris (POSOPT/CONTACT?), mise a jour par les events pointeur (verrou mu)
	mouseLX, mouseLY float64 // derniere position en coords Logo
	mouseInField     bool    // le pointeur est-il dans le champ ?
	mouseBtn         bool    // un bouton est-il enfonce ?

	// manettes emulees au clavier (MANETTE/BOUTON?) : fleches + barre d'espace
	joyUp, joyDown, joyLeft, joyRight, joyFire bool
	// instant du dernier relache de chaque touche : anti-rebond contre un Press
	// d'auto-repetition Gio qui arrive juste apres le Release (sinon touche collee)
	joyRel map[string]time.Time

	// afficheur de sortie texte (CATALOGUE...), facon "more" plein ecran defilable
	txtActive atomic.Bool
	txtTitle  string
	txtLines  []string
	txtScroll int
	txtDone   chan struct{}

	// fourni par main : ouvre l'aide hors primitive AIDE (F1) et donne la langue
	// courante pour traduire l'affichage (barre de statut de l'editeur)
	pgOpen    func(extended bool) (names []string, details map[string][]string, lang string, switchLang logo.HelpSwitch)
	pgResolve func(word string) (string, bool) // mot -> nom de fiche (F1 contextuel editeur)
	getLang   func() string
	translate func(src string, toEN bool) string // editeur Ctrl+T : traduit FR<->EN
	edEN      bool                               // sens courant du Ctrl+T (faux = prochaine bascule vers l'anglais)
	errText   func(error) string                 // localise un message d'erreur (FR/EN) pour l'affichage

	// capture plein ecran (COPIE) : la tache de fond depose un canal de reponse ; la
	// boucle d'evenements, juste apres compose(), y renvoie une copie de frame
	// (s.frame appartient au thread UI). voir SaveScreenPNG / draw
	captureCh      chan chan *image.RGBA
	captureWaiting atomic.Int32 // COPIE en attente : force une frame meme pendant un SCENE gele

	// composition cote processeur
	frame       *image.RGBA // image plein ecran recopiee a l'ecran a chaque frame
	fieldImg    *image.RGBA // traits dessines (fond transparent), coordonnees locales du champ
	baked       int
	bakedGen    uint64
	frozen      atomic.Int32 // SCENE : profondeur de gel ; >0 = ne presente pas de frame partielle
	lastWin     image.Point  // derniere taille de fenetre vue (detection de resize)
	resizeUntil time.Time    // redraws de rattrapage jusqu'a cet instant apres un resize
}

// cree l'ecran
func New() *Screen {
	s := &Screen{
		bg:        turtle.Bleu,
		textCol:   textDefault,
		textBg:    color.RGBA{0, 0, 0, 255},
		meLines:   textRows, // par defaut, pleine hauteur de texte
		cmdCh:     make(chan string, 1),
		edDone:    make(chan editResult, 1),
		kbResult:  make(chan kbRes, 1),
		pgDone:    make(chan struct{}, 1),
		txtDone:   make(chan struct{}, 1),
		captureCh: make(chan chan *image.RGBA, 1), // cap 1 : l'envoi ne bloque pas (cf. captureFrame)
		frame:     image.NewRGBA(image.Rect(0, 0, ScreenW, ScreenH)),
		fieldImg:  image.NewRGBA(image.Rect(0, 0, fieldW, fieldH)),
		joyRel:    make(map[string]time.Time),
	}
	gridClear(&s.grid, &s.gridFg, &s.curRow, &s.curCol)
	gridWrite(&s.grid, &s.gridFg, &s.curRow, &s.curCol, banner, s.textCol)
	return s
}

func (s *Screen) SetRunner(r Runner)    { s.run = r }
func (s *Screen) SetInterrupt(f func()) { s.interrupt = f }
func (s *Screen) SetStartup(src string) { s.startup = src }

// branche la liste des noms de commandes pour la completion (TAB)
func (s *Screen) SetCompleter(f func(string) []string) { s.complete = f }

// branche l'ouverture de l'aide hors primitive AIDE (F1 editeur) : f rend les
// donnees, la langue et la fonction de bascule (logo.Interp.HelpOpen)
func (s *Screen) SetHelpOpener(f func(extended bool) ([]string, map[string][]string, string, logo.HelpSwitch)) {
	s.pgOpen = f
}

// branche la resolution mot -> nom de fiche, pour le F1 contextuel de l'editeur
// (logo.Interp.HelpName)
func (s *Screen) SetHelpResolver(f func(string) (string, bool)) {
	s.pgResolve = f
}

// branche l'acces a la langue courante de l'interpreteur
func (s *Screen) SetLangFunc(f func() string) { s.getLang = f }

// branche la traduction FR<->EN du programme edite (Ctrl+T)
func (s *Screen) SetTranslator(f func(src string, toEN bool) string) { s.translate = f }

// branche la localisation des messages d'erreur (FR/EN)
func (s *Screen) SetErrorText(f func(error) string) { s.errText = f }

// demande un redraw (depuis n'importe quelle tache). pendant un SCENE (frozen > 0)
// on s'abstient : l'ecran garde la derniere image complete jusqu'au bout du bloc
// (cf BeginFrame/EndFrame)
func (s *Screen) invalidate() {
	// pendant un SCENE on s'abstient, SAUF si une COPIE attend une frame : sans ca
	// le thread Logo resterait bloque sur captureCh (interblocage SCENE [ ... COPIE ... ])
	if s.frozen.Load() > 0 && s.captureWaiting.Load() == 0 {
		return
	}
	if s.win != nil {
		s.win.Invalidate()
	}
}

// logo.frameBuffer (primitive SCENE) : un double tampon "logique" anti-clignotement.
// entre les deux, invalidate() est neutralise et compose() n'affiche pas l'etat
// partiel (NETTOIE + redessin) ; l'ecran garde la derniere image complete. au dernier
// EndFrame, une seule recomposition : la nouvelle image apparait d'un coup
func (s *Screen) BeginFrame() { s.frozen.Add(1) }

func (s *Screen) EndFrame() {
	if s.frozen.Add(-1) <= 0 {
		s.frozen.Store(0) // borne a 0 si un EndFrame est depareille
		s.invalidate()
	}
}

// --- turtle.Canvas ---

func (s *Screen) DrawSegment(seg turtle.Segment) {
	s.mu.Lock()
	s.gfx = append(s.gfx, gfxOp{kind: gfxSeg, seg: seg})
	s.mu.Unlock()
	s.invalidate()
}

func (s *Screen) Clear() {
	s.mu.Lock()
	s.gfx = nil
	s.clearGen++
	s.mu.Unlock()
	s.invalidate()
}

func (s *Screen) SetBackground(c turtle.Color) {
	s.mu.Lock()
	s.bg = c
	s.mu.Unlock()
	s.invalidate()
}

// --- logo.screenControl (FCB/FCT/FCFT/VT) ---

func (s *Screen) SetBorder(code int) {
	s.mu.Lock()
	s.border = turtle.Color(code)
	s.mu.Unlock()
	s.invalidate()
}

func (s *Screen) SetTextColor(code int) {
	s.mu.Lock()
	s.textCol = rgba(turtle.Color(code))
	s.mu.Unlock()
	s.invalidate()
}

func (s *Screen) SetTextBg(code int) {
	s.mu.Lock()
	s.textBg = rgba(turtle.Color(code))
	s.mu.Unlock()
	s.invalidate()
}

func (s *Screen) ClearText() {
	s.mu.Lock()
	gridClear(&s.grid, &s.gridFg, &s.curRow, &s.curCol)
	s.mu.Unlock()
	s.invalidate()
}

// logo.displayResetter (RAZ) : remet l'affichage a l'etat de demarrage (champ cache,
// couleurs par defaut, banniere)
func (s *Screen) ResetScreen() {
	s.mu.Lock()
	s.gfx = nil
	s.clearGen++
	s.userShapes = nil // RAZ efface les formes DEFSPRITE (VE les conserve)
	s.bg = turtle.Bleu
	s.border = 0
	s.graphics = false
	s.meLines, s.meTextOnly = textRows, false
	s.textCol, s.textBg = textDefault, color.RGBA{0, 0, 0, 255}
	gridClear(&s.grid, &s.gridFg, &s.curRow, &s.curCol)
	gridWrite(&s.grid, &s.gridFg, &s.curRow, &s.curCol, banner, s.textCol)
	s.mu.Unlock()
	s.invalidate()
}

func (s *Screen) ShowGraphics() {
	s.mu.Lock()
	s.graphics = true
	s.mu.Unlock()
	s.invalidate()
}

// DEFSPRITE (logo.screenControl) : enregistre une forme 16x16 de l'utilisateur,
// indexee par n (>= 3 ; rows n'est pas mute apres coup)
func (s *Screen) DefineSprite(n int, rows []string) {
	s.mu.Lock()
	if s.userShapes == nil {
		s.userShapes = make(map[int]turtleShape)
	}
	s.userShapes[n] = turtleShape{bm: rows, pix: userSpritePix, square: userSpriteSquare}
	s.mu.Unlock()
	s.invalidate()
}

func (s *Screen) TurtleMoved(st turtle.State) {
	s.mu.Lock()
	s.turtles = []turtle.State{st}
	s.mu.Unlock()
	s.invalidate()
}

// turtle.MultiCanvas : recoit l'etat de toutes les tortues, qu'on garde pour dessiner
// chaque sprite visible. current n'influe pas sur le dessin (toutes les tortues
// visibles sont rendues pareil)
func (s *Screen) TurtlesMoved(states []turtle.State, current int) {
	s.mu.Lock()
	s.turtles = append(s.turtles[:0:0], states...) // copie defensive
	s.mu.Unlock()
	s.invalidate()
}

// --- grille texte 80x25 (FCURS/ME) ---

// borne v dans [lo, hi]
func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// --- io.Writer (sortie texte ECRIS/TAPE/erreurs) ---

func (s *Screen) Write(p []byte) (int, error) {
	s.mu.Lock()
	gridWrite(&s.grid, &s.gridFg, &s.curRow, &s.curCol, string(p), s.textCol)
	s.mu.Unlock()
	s.invalidate()
	return len(p), nil
}

// ecrit une ligne (avec retour) dans la grille texte
func (s *Screen) Print(line string) {
	s.mu.Lock()
	gridWrite(&s.grid, &s.gridFg, &s.curRow, &s.curCol, line+"\n", s.textCol)
	s.mu.Unlock()
	s.invalidate()
}

// reveille toute lecture bloquante (clavier/editeur/aide/pager) pour que la tache de
// fond ne reste pas coincee a la fermeture, a attendre pour l'eternite
func (s *Screen) wakeBlockers() {
	s.closeKb(kbRes{})
	s.closeEdit(editResult{})
	s.closeHelp()
	s.closePage()
}

// la boucle d'evenements Gio (appelee depuis main, sur sa tache principale)
func (s *Screen) Run(w *app.Window) error {
	s.win = w
	var ops op.Ops
	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			s.wakeBlockers()
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			if !s.started {
				s.started = true
				go s.worker()
				if s.startup != "" {
					s.Print("?" + s.startup)
					s.dispatch(s.startup)
				}
			}
			s.handleInput(gtx)
			// Verifie APRES handleInput : un Ctrl+Q (ou la primitive QUITTE) tape
			// dans cette frame doit fermer tout de suite (sinon, au repos, aucune
			// frame ne suit). On rend la main a main() qui fait os.Exit(0) : sortie
			// fiable sur toutes les plateformes. (system.ActionClose ne ferme PAS la
			// fenetre plein ecran sur macOS -> ecran noir sans fin de process.)
			if s.quit.Load() {
				s.wakeBlockers()
				return nil
			}
			s.draw(gtx)
			e.Frame(gtx.Ops)
		}
	}
}

// envoie une ligne a la tache de fond (running passe a true pour bloquer la saisie)
func (s *Screen) dispatch(line string) {
	if s.run == nil {
		return
	}
	s.mu.Lock()
	s.kbBuf = nil // vide le tampon clavier avant chaque nouveau programme
	s.mu.Unlock()
	s.clearJoystick() // une touche manette restee collee ne contamine pas le programme suivant
	s.running.Store(true)
	select {
	case s.cmdCh <- line:
	default:
		s.running.Store(false)
	}
}

// tache de fond qui execute les lignes Logo. un programme long, voire increvable,
// ne bloque pas l'interface et reste tuable par Ctrl+C
func (s *Screen) worker() {
	for line := range s.cmdCh {
		if err := s.run(line); err != nil {
			if errors.Is(err, logo.ErrQuitter) {
				s.quit.Store(true)
				s.invalidate()
				return
			}
			msg := err.Error() // dont "INTERROMPU !" sur Ctrl+C
			if s.errText != nil {
				msg = s.errText(err) // localise FR/EN selon la langue courante
			}
			s.Print(msg)
		}
		s.running.Store(false)
		s.invalidate()
	}
}

// nature d'une operation du journal graphique
type gfxKind uint8

const (
	gfxSeg   gfxKind = iota // trait (turtle.Segment)
	gfxFill                 // remplissage (REMPLIS) a partir d'un point
	gfxLabel                // etiquette (ETIQUETTE) : texte a un point
)

// une operation du journal graphique. traits, remplissages et etiquettes partagent
// le meme flux, donc le rendu suit l'ordre reel des commandes (un REMPLIS ne voit
// que les traits d'avant)
type gfxOp struct {
	kind gfxKind
	seg  turtle.Segment // gfxSeg
	x, y float64        // gfxFill / gfxLabel : coords Logo
	col  turtle.Color   // gfxFill / gfxLabel
	text string         // gfxLabel
}

// logo.Filler (REMPLIS) : remplit la zone qui contient le point
func (s *Screen) Fill(x, y float64, c turtle.Color) {
	s.mu.Lock()
	s.gfx = append(s.gfx, gfxOp{kind: gfxFill, x: x, y: y, col: c})
	s.mu.Unlock()
	s.invalidate()
}

// logo.Labeler (ETIQUETTE) : ecrit un texte dans le champ
func (s *Screen) Label(x, y float64, text string, c turtle.Color) {
	s.mu.Lock()
	s.gfx = append(s.gfx, gfxOp{kind: gfxLabel, x: x, y: y, col: c, text: text})
	s.mu.Unlock()
	s.invalidate()
}

// remplit la zone connexe de meme couleur que le pixel de depart (4-connexite, sans
// lissage), arretee par les traits deja dessines
func floodFill(img *image.RGBA, sx, sy int, col color.RGBA) {
	b := img.Bounds()
	if sx < b.Min.X || sx >= b.Max.X || sy < b.Min.Y || sy >= b.Max.Y {
		return
	}
	target := img.RGBAAt(sx, sy)
	if target == col {
		return
	}
	stack := []image.Point{{X: sx, Y: sy}}
	for len(stack) > 0 {
		p := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if p.X < b.Min.X || p.X >= b.Max.X || p.Y < b.Min.Y || p.Y >= b.Max.Y {
			continue
		}
		if img.RGBAAt(p.X, p.Y) != target {
			continue
		}
		img.SetRGBA(p.X, p.Y, col)
		// on n'empile un voisin que s'il est encore de la couleur cible : evite
		// d'empiler chaque pixel jusqu'a 4 fois (pile bien plus petite)
		if p.X+1 < b.Max.X && img.RGBAAt(p.X+1, p.Y) == target {
			stack = append(stack, image.Point{X: p.X + 1, Y: p.Y})
		}
		if p.X-1 >= b.Min.X && img.RGBAAt(p.X-1, p.Y) == target {
			stack = append(stack, image.Point{X: p.X - 1, Y: p.Y})
		}
		if p.Y+1 < b.Max.Y && img.RGBAAt(p.X, p.Y+1) == target {
			stack = append(stack, image.Point{X: p.X, Y: p.Y + 1})
		}
		if p.Y-1 >= b.Min.Y && img.RGBAAt(p.X, p.Y-1) == target {
			stack = append(stack, image.Point{X: p.X, Y: p.Y - 1})
		}
	}
}

// --- operations d'edition (sur runes, ligne courante = s.edRow) ---

// compose la scene dans frame puis la recopie a l'ecran, a l'echelle de la fenetre
// (ratio garde, centree)
func (s *Screen) draw(gtx layout.Context) {
	s.compose()
	// Sert une eventuelle capture plein ecran (COPIE) : frame vient d'etre composee
	// et appartient a ce thread, on en renvoie une copie a la tache de fond.
	select {
	case resp := <-s.captureCh:
		resp <- cloneRGBA(s.frame)
	default:
	}
	win := gtx.Constraints.Max
	// Resize Wayland/NVIDIA : la surface graphique est stable ~600 ms apres un configure ;
	// on programme des redraws via op.InvalidateCmd (Invalidate ignore dans une frame).
	if win != s.lastWin {
		s.lastWin = win
		s.resizeUntil = gtx.Now.Add(600 * time.Millisecond)
	}
	if gtx.Now.Before(s.resizeUntil) {
		// Cadence espacee (100 ms) : couvre la stabilisation de la surface du pilote
		// sur 600 ms avec peu de redraws, sans faire clignoter la decoration.
		gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(100 * time.Millisecond)})
	}
	// Un programme tourne (jeu, animation...) : on garde la boucle d'evenements en
	// vie a la cadence ecran, pour que chaque image redessinee s'affiche tout de
	// suite. Sur Wayland, un win.Invalidate() isole depuis la tache de fond ne
	// reveille pas toujours une boucle endormie : une animation espacee par des
	// ATTENDS (le de qui tourne) reste alors figee jusqu'au prochain evenement
	// clavier. Ce battement de coeur "in-frame", lui, est fiable. Au repos (pas de
	// programme en cours), on ne programme rien : cpu quasi nul.
	if s.running.Load() {
		gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(16 * time.Millisecond)})
	}
	// Fond noir sur toute la fenetre (evite les bords blancs sur les cotes).
	paint.FillShape(gtx.Ops, color.NRGBA{A: 255}, clip.Rect{Max: win}.Op())
	sc := float32(win.X) / ScreenW
	if v := float32(win.Y) / ScreenH; v < sc {
		sc = v
	}
	offx := (float32(win.X) - sc*ScreenW) / 2
	offy := (float32(win.Y) - sc*ScreenH) / 2
	defer op.Affine(f32.Affine2D{}.Scale(f32.Pt(0, 0), f32.Pt(sc, sc)).Offset(f32.Pt(offx, offy))).Push(gtx.Ops).Pop()
	defer clip.Rect{Max: image.Pt(ScreenW, ScreenH)}.Push(gtx.Ops).Pop()
	paint.NewImageOp(s.frame).Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
}

// dessine toute la scene dans s.frame (image CPU)
func (s *Screen) compose() {
	if s.frozen.Load() > 0 {
		return // SCENE en cours : on conserve la derniere image complete (pas de frame partielle)
	}
	if s.pgActive.Load() {
		s.composeHelp()
		return
	}
	if s.txtActive.Load() {
		s.composePager()
		return
	}
	if s.edActive.Load() {
		s.composeEditor()
		return
	}
	s.mu.Lock()
	graphics := s.graphics && !s.meTextOnly // ME 25 : plein texte, champ cache
	bg := s.bg
	border := s.border
	tsts := append([]turtle.State(nil), s.turtles...) // copie sous verrou
	var ushapes map[int]turtleShape                   // snapshot des formes DEFSPRITE
	if len(s.userShapes) > 0 {
		ushapes = make(map[int]turtleShape, len(s.userShapes))
		for k, v := range s.userShapes {
			ushapes[k] = v
		}
	}
	textCol, textBg := s.textCol, s.textBg
	clearGen := s.clearGen
	// Reinit du dessin si clearGen a change (VE/NETTOIE) ou incoherence du compteur.
	reset := clearGen != s.bakedGen || s.baked > len(s.gfx)
	from := s.baked
	if reset {
		from = 0
	}
	gfxCount := len(s.gfx)
	// Copie de la queue non encore dessinee sous verrou : la tache de fond peut append dans
	// s.gfx en meme temps ; on travaille sur notre propre tranche.
	newGfx := append([]gfxOp(nil), s.gfx[from:gfxCount]...)
	// Copie de la grille texte + curseur (tableaux : copie par valeur).
	grid, gridFg := s.grid, s.gridFg
	curRow, curCol, meLines := s.curRow, s.curCol, s.meLines
	s.mu.Unlock()

	if reset {
		clearImage(s.fieldImg)
		s.baked = 0
		s.bakedGen = clearGen
	}
	// Rejoue la tranche du journal dans l'ordre : traits, remplissages, etiquettes
	// melanges chronologiquement (sur le fil interface).
	bakeGfx(s.fieldImg, newGfx)
	s.baked = gfxCount

	// Frame : fond noir.
	draw.Draw(s.frame, s.frame.Bounds(), uniBlack, image.Point{}, draw.Src)
	textTop := margin
	if graphics {
		const bw = 6 // epaisseur du bord
		bord := image.Rect(fieldX-bw, fieldY-bw, fieldX+fieldW+bw, fieldY+fieldH+bw)
		draw.Draw(s.frame, bord, uniform(rgba(border)), image.Point{}, draw.Src) // bord FCB
		fr := image.Rect(fieldX, fieldY, fieldX+fieldW, fieldY+fieldH)
		draw.Draw(s.frame, fr, uniform(rgba(bg)), image.Point{}, draw.Src) // fond champ
		draw.Draw(s.frame, fr, s.fieldImg, image.Point{}, draw.Over)       // traits
		for _, tst := range tsts {
			if tst.Visible {
				s.drawTurtle(tst, ushapes)
			}
		}
		textTop = fieldY + fieldH + margin
	}
	// Fond de la zone texte (FCFT) si different du noir deja pose.
	if textBg != (color.RGBA{0, 0, 0, 255}) {
		fillRect(s.frame, 0, textTop-4, ScreenW, ScreenH-(textTop-4), textBg)
	}
	s.drawGrid(textTop, grid, gridFg, curRow, curCol, meLines, textCol)
}

// mise en page de l'editeur ED : plein ecran bleu, bord cyan, texte cyan, pas d'invite
const (
	edBorder  = 8                // epaisseur du cadre cyan
	edPad     = 16               // marge texte sous le cadre
	edTextX   = edBorder + edPad // origine X du texte
	edTextY   = edBorder + edPad // origine Y de la 1re ligne
	edStatusH = lineH            // barre de raccourcis (video inverse) en bas
	edRows    = (ScreenH - 2*(edBorder+edPad) - edStatusH) / lineH
	edCols    = (ScreenW - 2*edTextX) / charW // largeur de l'editeur en colonnes
)

// cache d'uniformes de couleur : un *image.Uniform est immuable une fois cree, donc
// partageable entre threads sans risque. evite d'en allouer un a chaque trait, fond
// ou etiquette (appeles en boucle a chaque frame, grosse pression GC sinon)
var (
	uniMu    sync.Mutex
	uniCache = map[color.RGBA]*image.Uniform{}
	uniBlack = image.NewUniform(color.RGBA{0, 0, 0, 255})
)

func uniform(c color.RGBA) *image.Uniform {
	uniMu.Lock()
	defer uniMu.Unlock()
	if u := uniCache[c]; u != nil {
		return u
	}
	u := image.NewUniform(c)
	if len(uniCache) < 4096 { // borne le cache : au-dela on alloue sans memoriser
		uniCache[c] = u
	}
	return u
}

// remplit le rectangle (x,y,w,h) avec col dans img
func fillRect(img *image.RGBA, x, y, w, h int, col color.RGBA) {
	draw.Draw(img, image.Rect(x, y, x+w, y+h), uniform(col), image.Point{}, draw.Src)
}

// dessine la tortue dans frame (petits carres pivotes selon le cap), selon sa forme
// (Shape : 0/1/2 integrees, >=3 par DEFSPRITE)
func (s *Screen) drawTurtle(st turtle.State, userShapes map[int]turtleShape) {
	sh := turtleShapes[0]
	if st.Shape >= 0 && st.Shape < len(turtleShapes) {
		sh = turtleShapes[st.Shape] // forme integree 0/1/2
	} else if us, ok := userShapes[st.Shape]; ok {
		sh = us // forme definie par DEFSPRITE (sinon repli triangle)
	}
	fx, fy := logoToField(st.X, st.Y)
	cx, cy := float64(fieldX)+fx, float64(fieldY)+fy
	hd := st.Heading * math.Pi / 180
	sin, cos := math.Sin(hd), math.Cos(hd)
	col := rgba(st.Pen)
	ccol := float64(len(sh.bm[0])-1) / 2 // colonne centrale (bitmap de largeur uniforme)
	crow := float64(len(sh.bm)-1) / 2
	for row, line := range sh.bm {
		for c, ch := range line {
			if ch != '#' {
				continue
			}
			r := (float64(c) - ccol) * sh.pix
			u := (crow - float64(row)) * sh.pix
			px := cx + r*cos + u*sin
			py := cy + r*sin - u*cos
			fillSquare(s.frame, int(px), int(py), sh.square, col)
		}
	}
}

// coupe une ligne en bouts d'au plus cols caracteres (coupure brutale, sans egard
// pour les espaces)
func wrapLine(s string, cols int) []string {
	r := []rune(s)
	if len(r) == 0 {
		return []string{""}
	}
	var out []string
	for len(r) > cols {
		out = append(out, string(r[:cols]))
		r = r[cols:]
	}
	return append(out, string(r))
}

// --- helpers de rendu CPU ---

// coordonnees Logo vers pixel local du champ (en flottant, pour la precision)
func logoToField(x, y float64) (float64, float64) {
	return (x - turtle.MinX) * fieldScale, (turtle.MaxY - y) * fieldScale
}

func clearImage(img *image.RGBA) {
	for i := range img.Pix {
		img.Pix[i] = 0
	}
}

// epaisseur du trait (px champ). pas d'anti-aliasing : on ecrit le pixel direct
const lineWidth = 2.0

// epaisseur d'un trait en px (lineWidth par defaut si non precise)
func segWidth(w int) float64 {
	if w <= 0 {
		return lineWidth
	}
	return float64(w)
}

// trace un segment epais SANS lissage (pixels ecrits direct). invariant gomme : un
// trait lisse laisserait une frange ineffacable ; la distance geometrique garantit
// un effacement au pixel pres
func drawSeg(img *image.RGBA, x0, y0, x1, y1 float64, col color.RGBA, width float64) {
	hw := width / 2
	minX := int(math.Floor(math.Min(x0, x1) - hw - 1))
	maxX := int(math.Ceil(math.Max(x0, x1) + hw + 1))
	minY := int(math.Floor(math.Min(y0, y1) - hw - 1))
	maxY := int(math.Ceil(math.Max(y0, y1) + hw + 1))
	b := img.Bounds() // borne a l'image (un trait au bord ne doit pas paniquer)
	if minX < b.Min.X {
		minX = b.Min.X
	}
	if minY < b.Min.Y {
		minY = b.Min.Y
	}
	if maxX > b.Max.X {
		maxX = b.Max.X
	}
	if maxY > b.Max.Y {
		maxY = b.Max.Y
	}
	dx, dy := x1-x0, y1-y0
	length2 := dx*dx + dy*dy
	hw2 := hw * hw
	for py := minY; py < maxY; py++ {
		for px := minX; px < maxX; px++ {
			fx, fy := float64(px)+0.5, float64(py)+0.5 // centre du pixel
			var t float64                              // projection sur le segment, bornee [0,1]
			if length2 > 1e-12 {
				t = ((fx-x0)*dx + (fy-y0)*dy) / length2
				if t < 0 {
					t = 0
				} else if t > 1 {
					t = 1
				}
			}
			ex, ey := fx-(x0+t*dx), fy-(y0+t*dy)
			if ex*ex+ey*ey <= hw2 {
				img.SetRGBA(px, py, col)
			}
		}
	}
}

func fillSquare(img *image.RGBA, cx, cy, size int, col color.RGBA) {
	h := size / 2
	for yy := cy - h; yy < cy-h+size; yy++ {
		for xx := cx - h; xx < cx-h+size; xx++ {
			img.SetRGBA(xx, yy, col)
		}
	}
}

// largeur d'une cellule de caractere a l'echelle (basicfont avance de 7px)
const charW = 7 * textScale

// affiche un texte gris clair (aide) avec basicfont 7x13, x textScale
func drawString2x(dst *image.RGBA, x, y int, s string) {
	drawText2x(dst, x, y, s, textDefault)
}

// table accents courants -> ASCII (basicfont 7x13 ne couvre que 0x20-0x7E)
var accentMap = map[rune]rune{
	'à': 'a', 'â': 'a', 'ä': 'a', 'á': 'a',
	'è': 'e', 'é': 'e', 'ê': 'e', 'ë': 'e',
	'ì': 'i', 'î': 'i', 'ï': 'i', 'í': 'i',
	'ò': 'o', 'ô': 'o', 'ö': 'o', 'ó': 'o',
	'ù': 'u', 'û': 'u', 'ü': 'u', 'ú': 'u',
	'ç': 'c', 'ñ': 'n',
	'À': 'A', 'Â': 'A', 'Ä': 'A',
	'È': 'E', 'É': 'E', 'Ê': 'E', 'Ë': 'E',
	'Î': 'I', 'Ï': 'I', 'Ô': 'O', 'Ö': 'O', 'Û': 'U', 'Ü': 'U', 'Ç': 'C',
	'’': '\'', '‘': '\'', '·': '-', '«': '"', '»': '"', '…': '.', '—': '-', '–': '-',
}

// met une chaine en ASCII affichable (accent -> lettre de base, inconnu -> '?'),
// pour eviter les glyphes vides de basicfont
func deaccent(s string) string {
	ascii := true
	for _, r := range s {
		if r < 0x20 || r > 0x7e {
			ascii = false
			break
		}
	}
	if ascii {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r >= 0x20 && r <= 0x7e:
			b.WriteRune(r)
		case accentMap[r] != 0:
			b.WriteRune(accentMap[r])
		default:
			b.WriteByte('?')
		}
	}
	return b.String()
}

// rend un texte dans la couleur col, basicfont 7x13 x textScale
func drawText2x(dst *image.RGBA, x, y int, s string, col color.RGBA) {
	if s == "" {
		return
	}
	s = deaccent(s) // basicfont = ASCII seul : convertit les accents
	w := len([]rune(s)) * 7
	tmp := image.NewRGBA(image.Rect(0, 0, w, 13))
	d := &font.Drawer{Dst: tmp, Src: uniform(col), Face: basicfont.Face7x13, Dot: fixed.P(0, 10)}
	d.DrawString(s)
	for ty := 0; ty < 13; ty++ {
		for tx := 0; tx < w; tx++ {
			if tmp.RGBAAt(tx, ty).A == 0 {
				continue
			}
			for dy := 0; dy < textScale; dy++ {
				for dx := 0; dx < textScale; dx++ {
					dst.SetRGBA(x+tx*textScale+dx, y+ty*textScale+dy, col)
				}
			}
		}
	}
}
