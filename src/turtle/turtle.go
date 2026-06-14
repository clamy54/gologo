// Package turtle : l'etat de la tortue Logo, separe du rendu (cf Canvas).
// Le backend reel (Gio, ou un canvas bidon pour les tests) implemente Canvas.
package turtle

import (
	"errors"
	"fmt"
	"math"
	"sync"
)

// quand la tortue tente de sortir d'un champ CLOS (message exact du manuel)
var ErrSortir = errors.New("LA TORTUE VA SORTIR")

// code couleur de la palette (0..15). le passage code -> RGB est l'affaire du Canvas
type Color int

// palette, cf manuel p.35. 8..15 = couleurs etendues
const (
	Noir Color = iota
	Rouge
	Vert
	Jaune
	Bleu
	Magenta
	Cyan
	Blanc
	Gris
	RougePale
	VertPale
	JaunePale
	BleuPale
	MagentaPale
	CyanPale
	Orange
)

// bit 24 leve = couleur RGB directe, pas un index de palette.
// les index 0-15 et les codes negatifs (gomme) restent en dessous
const rgbFlag = 1 << 24

// RGB : couleur depuis trois composantes 0-255, pour FCC [ r v b ]
func RGB(r, g, b int) Color { return Color(rgbFlag | clamp8(r)<<16 | clamp8(g)<<8 | clamp8(b)) }

func (c Color) IsRGB() bool { return int(c) >= rgbFlag }

// composantes 0-255 d'une couleur RGB directe
func (c Color) Components() (r, g, b int) {
	v := int(c)
	return (v >> 16) & 255, (v >> 8) & 255, v & 255
}

func clamp8(n int) int {
	if n < 0 {
		return 0
	}
	if n > 255 {
		return 255
	}
	return n
}

// bornes du champ en mode CLOS, en pas Logo. repere centre 1600x1000
// (largeur -800..800, hauteur -500..500 ; ratio 1,6 ; pixels carres).
// AV 1600 depuis le bord gauche tombe pile sur le bord droit
const (
	MinX, MaxX = -800.0, 800.0
	MinY, MaxY = -500.0, 500.0
)

type FieldMode int

const (
	Clos    FieldMode = iota // borne au bord (defaut)
	Enroule                  // ressort au bord oppose
	Fenetre                  // champ etendu, ecran = fenetre
)

// etat d'une tortue, passe au Canvas pour dessiner le sprite.
// coords en pas Logo, origine au centre, X a droite, Y vers le haut
type State struct {
	X, Y    float64
	Heading float64 // cap : 0 = haut (Nord), horaire positif
	PenDown bool
	Pen     Color
	PenSize int  // epaisseur du trait en px (FIXETAILLECRAYON), def 2
	Visible bool // MT / CT
	Shape   int  // forme (SPRITE) : 0 triangle, 1 tortue, 2 voiture
}

// nombre de formes integrees (SPRITE 0/1/2). doit coller a render.turtleShapes.
// au-dela, ce sont les formes definies par l'utilisateur (DEFSPRITE)
const SpriteCount = 3

// borne le numero de forme. SPRITE et DEFSPRITE prennent 0..255 (donc 253 definissables)
const MaxSprites = 256

// comportement d'une anim auto (ANIME)
type AnimMode int

const (
	Once     AnimMode = iota // va a la cible puis stop
	Loop                     // va a la cible, saute au depart, recommence
	PingPong                 // aller-retour sans fin
)

// cadence par defaut du moteur d'anim (CADENCE)
const AnimDefaultFPS = 30

// mouvement auto d'une tortue (ANIME), inactif par defaut.
// sx,sy = depart ; tx,ty = cible ; speed = pas Logo par cran
type motionState struct {
	active bool
	sx, sy float64
	tx, ty float64
	speed  float64
	mode   AnimMode
}

// un trait a tracer, en coords Logo
type Segment struct {
	X0, Y0, X1, Y1 float64
	Pen            Color
	Width          int // epaisseur px (0 => defaut du backend)
}

// contrat de rendu. implemente par le canvas d'enregistrement (tests) et le backend Gio
type Canvas interface {
	DrawSegment(seg Segment) // trace un trait (crayon baisse)
	Clear()                  // efface le champ
	SetBackground(c Color)   // couleur de fond (FCFG)
	// appele a la premiere primitive graphique : l'ecran graphique doit
	// alors apparaitre (cf manuel p.29)
	ShowGraphics()
	// signale un changement d'etat pour rafraichir le sprite
	TurtleMoved(s State)
}

// implemente par les backends qui savent afficher plusieurs tortues : ils
// recoivent toutes les fiches d'un coup, pas seulement la tortue active.
//
// attention : le slice states appartient a la Turtle et peut bouger apres
// l'appel. qui veut le garder doit le copier, et ne jamais muter les State recus.
type MultiCanvas interface {
	TurtlesMoved(states []State, current int)
}

// borne le nombre de tortues (FIXETORTUE/SETTURTLE) : 0 a 1023
const MaxTurtles = 1024

// Turtle : le controleur geometrique, sans rien de graphique.
// gere une ou plusieurs tortues. states = une fiche par tortue (la n0 existe
// toujours), current = la tortue active sur laquelle agissent AV, TD, FCC...
// Le mode de champ, le fond et l'echelle sont GLOBAUX (ecran partage).
//
// on prend toujours turtle.mu avant le canvas, jamais l'inverse.
type Turtle struct {
	mu             sync.Mutex
	states         []State // une fiche par tortue ; states[0] toujours la
	current        int     // tortue active (FIXETORTUE / SETTURTLE)
	canvas         Canvas
	graphics       bool      // l'ecran graphique est-il deja apparu ?
	bg             Color     // fond courant (FCFG / CF)
	scaleH, scaleV float64   // echelles H/V en % (FECH/ECH), def 100
	flood          Color     // couleur de remplissage (FCR) si floodSet
	floodSet       bool      // FCR posee ? sinon REMPLIS prend la couleur du crayon (MO5)
	field          FieldMode // mode du champ GLOBAL, partage par toutes les tortues

	motions  []motionState // anim par tortue ; meme longueur que states
	fps      int           // cadence du moteur d'anim (CADENCE)
	animWake func()        // reveille le driver temps reel (nil en test)
}

func (t *Turtle) cur() *State { return &t.states[t.current] }

// etat initial d'une tortue (VE) : centre, cap Nord, crayon baisse cyan, visible
func defaultState() State {
	return State{X: 0, Y: 0, Heading: 0, PenDown: true, Pen: Cyan, PenSize: 2, Visible: true}
}

// FECH ; Scale rend l'inverse (ECH), en %
func (t *Turtle) SetScale(h, v float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.scaleH, t.scaleV = h, v
}

func (t *Turtle) Scale() (h, v float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.scaleH, t.scaleV
}

func (t *Turtle) Background() Color {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.bg
}

// cree une tortue a l'etat initial (VE). l'ecran graphique n'apparait qu'a la
// premiere primitive graphique
func New(canvas Canvas) *Turtle {
	return &Turtle{
		states:  []State{defaultState()},
		current: 0,
		canvas:  canvas,
		bg:      Bleu,
		scaleH:  100,
		scaleV:  100,
		field:   Clos,
		motions: []motionState{{}},
		fps:     AnimDefaultFPS,
	}
}

// copie de l'etat de la tortue active
func (t *Turtle) State() State {
	t.mu.Lock()
	defer t.mu.Unlock()
	return *t.cur()
}

// selectionne la tortue active (FIXETORTUE / SETTURTLE), en creant au besoin
// les tortues manquantes (etat par defaut). numerotation a partir de 0
func (t *Turtle) SetTurtle(n int) error {
	if n < 0 {
		return fmt.Errorf("NUMERO DE TORTUE INVALIDE : %d", n)
	}
	if n >= MaxTurtles {
		return fmt.Errorf("TROP DE TORTUES (MAX %d)", MaxTurtles)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.activate()
	for len(t.states) <= n {
		t.states = append(t.states, defaultState())
	}
	t.ensureMotions()
	t.current = n
	t.notify()
	return nil
}

// numero de la tortue active (TORTUE / TURTLE)
func (t *Turtle) CurrentTurtle() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.current
}

// demi-taille (pas Logo) de la boite englobante d'une forme, pour COLLISION?.
// calee sur l'empreinte affichee : sinon une boite trop grande declenche une
// collision alors que les sprites ne se touchent meme pas. le triangle est petit
func collisionHalf(shape int) float64 {
	if shape == 0 {
		return 15 // triangle : ~17 px de cote
	}
	return 26 // tortue / voiture / sprites 16x16 : ~32 px
}

// etat de la tortue n (false si elle n'existe pas)
func (t *Turtle) StateOf(n int) (State, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.stateOfLocked(n)
}

func (t *Turtle) stateOfLocked(n int) (State, bool) {
	if n < 0 || n >= len(t.states) {
		return State{}, false
	}
	return t.states[n], true
}

// vrai si a et b existent, sont visibles et se chevauchent. boites AABB calees
// sur chaque sprite : deux carres se touchent si la distance des centres est
// sous la somme des demi-tailles
func (t *Turtle) Collide(a, b int) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	sa, oka := t.stateOfLocked(a)
	sb, okb := t.stateOfLocked(b)
	if !oka || !okb || !sa.Visible || !sb.Visible {
		return false
	}
	reach := collisionHalf(sa.Shape) + collisionHalf(sb.Shape)
	return math.Abs(sa.X-sb.X) < reach && math.Abs(sa.Y-sb.Y) < reach
}

// nombre de tortues (NBTORTUES / TURTLES)
func (t *Turtle) TurtleCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.states)
}

// tue toutes les tortues sauf la n0 et la reselectionne (DISTORTUE)
func (t *Turtle) KillExtraTurtles() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.states = t.states[:1]
	t.motions = t.motions[:1]
	t.current = 0
	t.notify()
}

func (t *Turtle) GraphicsShown() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.graphics
}

// remet le drapeau "ecran graphique apparu" a faux, pour que la prochaine
// primitive graphique relance ShowGraphics. utilise par RAZ qui repasse en mode
// texte : sans ca, activate() ne ferait plus rien et le champ ne reviendrait jamais
func (t *Turtle) ResetGraphicsShown() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.graphics = false
}

// fait apparaitre l'ecran graphique au premier appel d'une primitive graphique
// (AV, RE, TD, TG, FCAP, FPOS, CT, MT, ORIGINE, FCFG, FEN, ENR, CLOS...)
func (t *Turtle) activate() {
	if !t.graphics {
		t.graphics = true
		if t.canvas != nil {
			t.canvas.ShowGraphics()
		}
	}
}

func (t *Turtle) notify() {
	if t.canvas == nil {
		return
	}
	// un backend multi-tortue recoit toutes les fiches d'un coup ; sinon on se
	// rabat sur la tortue active (compat Canvas simple)
	if mc, ok := t.canvas.(MultiCanvas); ok {
		mc.TurtlesMoved(t.states, t.current)
		return
	}
	t.canvas.TurtleMoved(*t.cur())
}

// avance de d pas selon le cap (cap 0 = haut, horaire).
// CLOS : refus si hors champ (ErrSortir) ; ENR : position enroulee
func (t *Turtle) move(d float64) error {
	t.activate()
	rad := t.cur().Heading * math.Pi / 180
	nx := t.cur().X + d*math.Sin(rad)*t.scaleH/100
	ny := t.cur().Y + d*math.Cos(rad)*t.scaleV/100
	return t.goTo(nx, ny)
}

// amene la tortue en (nx,ny) selon le mode de champ, en tracant au besoin
func (t *Turtle) goTo(nx, ny float64) error {
	switch t.field {
	case Clos:
		if nx < MinX || nx > MaxX || ny < MinY || ny > MaxY {
			return ErrSortir
		}
	case Enroule:
		// trajet coupe a chaque bord et repris au bord oppose, pas un seul
		// trait qui traverse tout l'ecran
		t.goToWrapped(nx, ny)
		t.notify()
		return nil
	case Fenetre:
		// champ etendu : on laisse passer
	}
	if t.cur().PenDown && t.canvas != nil {
		t.canvas.DrawSegment(Segment{X0: t.cur().X, Y0: t.cur().Y, X1: nx, Y1: ny, Pen: t.cur().Pen, Width: t.cur().PenSize})
	}
	t.cur().X, t.cur().Y = nx, ny
	t.notify()
	return nil
}

// trace en mode ENR (tore) : a chaque bord franchi le trait est coupe et reprend
// en face. la tortue finit a la cible enroulee
func (t *Turtle) goToWrapped(tx, ty float64) {
	const w = MaxX - MinX // 1600
	const h = MaxY - MinY // 1000
	cx, cy := t.cur().X, t.cur().Y
	vx, vy := tx-cx, ty-cy
	rem := 1.0        // fraction du deplacement restant
	for range 10000 { // garde-fou : un AV 1e9 ne doit pas nous faire tourner jusqu'a la nuit des temps
		// distance parametrique (s dans [0,1]) jusqu'au prochain bord, par axe
		sx := math.Inf(1)
		if vx > 0 {
			sx = (MaxX - cx) / vx
		} else if vx < 0 {
			sx = (MinX - cx) / vx
		}
		sy := math.Inf(1)
		if vy > 0 {
			sy = (MaxY - cy) / vy
		} else if vy < 0 {
			sy = (MinY - cy) / vy
		}
		s := math.Min(rem, math.Min(sx, sy))
		ex, ey := cx+s*vx, cy+s*vy
		if t.cur().PenDown && t.canvas != nil {
			t.canvas.DrawSegment(Segment{X0: cx, Y0: cy, X1: ex, Y1: ey, Pen: t.cur().Pen, Width: t.cur().PenSize})
		}
		if s >= rem { // cible atteinte sans franchir d'autre bord
			cx, cy = ex, ey
			break
		}
		rem -= s
		// enroule la/les coord(s) qui ont touche un bord, puis on repart
		if s == sx {
			if vx > 0 {
				ex -= w
			} else {
				ex += w
			}
		}
		if s == sy {
			if vy > 0 {
				ey -= h
			} else {
				ey += h
			}
		}
		cx, cy = ex, ey
	}
	// la position enroulee finale ne depend que de la cible nette : si le garde-fou
	// a coupe le trace (deplacement enorme), cx,cy serait partiel, pas tx,ty
	t.cur().X, t.cur().Y = wrap(tx, ty)
}

// AV. coupe une anim en cours sur la tortue active
func (t *Turtle) Forward(d float64) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cancelMotionLocked(t.current)
	return t.move(d)
}

// FIXETAILLECRAYON, mini 1 px
func (t *Turtle) SetPenSize(n int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if n < 1 {
		n = 1
	}
	t.cur().PenSize = n
}

// RE. coupe une anim en cours
func (t *Turtle) Back(d float64) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cancelMotionLocked(t.current)
	return t.move(-d)
}

// enroule (x,y) dans le champ (mode ENR)
func wrap(x, y float64) (float64, float64) {
	w := MaxX - MinX // 1600
	h := MaxY - MinY // 1000
	x = math.Mod(x-MinX, w)
	if x < 0 {
		x += w
	}
	y = math.Mod(y-MinY, h)
	if y < 0 {
		y += h
	}
	return x + MinX, y + MinY
}

// TD (droite / horaire)
func (t *Turtle) Right(a float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cancelMotionLocked(t.current)
	t.activate()
	t.cur().Heading = normalizeAngle(t.cur().Heading + a)
	t.notify()
}

// TG (gauche)
func (t *Turtle) Left(a float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cancelMotionLocked(t.current)
	t.activate()
	t.cur().Heading = normalizeAngle(t.cur().Heading - a)
	t.notify()
}

// cap absolu (FCAP)
func (t *Turtle) SetHeading(a float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cancelMotionLocked(t.current)
	t.activate()
	t.cur().Heading = normalizeAngle(a)
	t.notify()
}

// FPOS : va en (x,y) en tracant si crayon baisse, sans toucher au cap.
// coupe une anim en cours
func (t *Turtle) SetPos(x, y float64) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cancelMotionLocked(t.current)
	t.activate()
	return t.goTo(x, y)
}

// LC : leve le crayon. n'allume pas l'ecran graphique
func (t *Turtle) PenUp() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cur().PenDown = false
}

// BC : baisse le crayon
func (t *Turtle) PenDownState() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cur().PenDown = true
}

// FCC
func (t *Turtle) SetPenColor(c Color) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cur().Pen = c
	t.notify()
}

// FCR / SETFLOODCOLOR : couleur de REMPLIS, distincte du crayon.
// compat FMSLogo (FILL se base la-dessus, pas sur le crayon)
func (t *Turtle) SetFloodColor(c Color) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.flood, t.floodSet = c, true
}

// couleur de remplissage + un bool disant si elle a ete posee.
// sinon (cas MO5) REMPLIS prend la couleur du crayon
func (t *Turtle) FloodColor() (Color, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.flood, t.floodSet
}

// FCFG (primitive graphique)
func (t *Turtle) SetBackground(c Color) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.activate()
	t.bg = c
	if t.canvas != nil {
		t.canvas.SetBackground(c)
	}
}

// MT
func (t *Turtle) Show() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.activate()
	t.cur().Visible = true
	t.notify()
}

// CT
func (t *Turtle) Hide() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.activate()
	t.cur().Visible = false
	t.notify()
}

// SPRITE : forme de la tortue active. un index hors [0,MaxSprites) est ignore
// (on garde la forme courante). la validite d'une forme DEFSPRITE est verifiee
// plus haut, par la primitive
func (t *Turtle) SetShape(n int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if n < 0 || n >= MaxSprites {
		return
	}
	t.activate()
	t.cur().Shape = n
	t.notify()
}

// mode du champ CLOS/ENR/FEN (primitive graphique)
func (t *Turtle) SetField(m FieldMode) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.activate()
	t.field = m
	t.notify()
}

// POINT : allume un point a (x,y), couleur du crayon, sans bouger la tortue
func (t *Turtle) Point(x, y float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.activate()
	if t.canvas != nil {
		t.canvas.DrawSegment(Segment{X0: x, Y0: y, X1: x, Y1: y, Pen: t.cur().Pen, Width: t.cur().PenSize})
	}
}

// ORIGINE : retour au centre, cap 0, sans toucher au crayon ni effacer
func (t *Turtle) Home() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cancelMotionLocked(t.current)
	t.activate()
	if t.cur().PenDown && t.canvas != nil {
		t.canvas.DrawSegment(Segment{X0: t.cur().X, Y0: t.cur().Y, X1: 0, Y1: 0, Pen: t.cur().Pen, Width: t.cur().PenSize})
	}
	t.cur().X, t.cur().Y, t.cur().Heading = 0, 0, 0
	t.notify()
}

// (x,y) est-il atteignable dans le champ courant ?
func (t *Turtle) CanReach(x, y float64) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.field == Clos {
		return x >= MinX && x <= MaxX && y >= MinY && y <= MaxY
	}
	return true
}

// NETTOIE : efface le champ sans toucher a la tortue ni au crayon
func (t *Turtle) Wipe() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.activate()
	if t.canvas != nil {
		t.canvas.Clear()
	}
}

// VE : remet tout a zero. champ efface, fond bleu, une tortue a l'origine cap 0
// visible, crayon baisse cyan, champ CLOS. les anims sont coupees ; la cadence reste
func (t *Turtle) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.activate()
	t.states = []State{defaultState()} // une seule tortue, etat par defaut
	t.motions = []motionState{{}}
	t.current = 0
	t.field = Clos
	t.bg = Bleu
	t.scaleH, t.scaleV = 100, 100
	t.flood, t.floodSet = 0, false // FCR oubliee : REMPLIS reprend le crayon
	if t.canvas != nil {
		t.canvas.Clear()
		t.canvas.SetBackground(Bleu)
	}
	t.notify()
}

// ramene un angle dans [0,360)
func normalizeAngle(a float64) float64 {
	a = math.Mod(a, 360)
	if a < 0 {
		a += 360
	}
	return a
}

// --- Animation auto des tortues (ANIME / STOPANIME / CADENCE) ---

// branche le callback qui reveille le moteur temps reel (fourni par l'app, une
// fois au demarrage). nil en headless/test : l'anim n'avance alors que par
// appels explicites a AnimStep
func (t *Turtle) SetAnimWake(f func()) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.animWake = f
}

// CADENCE, bornee a [1,240]
func (t *Turtle) SetFPS(n int) {
	if n < 1 {
		n = 1
	}
	if n > 240 {
		n = 240
	}
	t.mu.Lock()
	t.fps = n
	t.mu.Unlock()
}

func (t *Turtle) FPS() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.fps
}

// lance (ou remplace) l'anim auto de la tortue id vers (tx,ty), a speed pas par
// cran, selon mode (ANIME). cree la tortue id au besoin, puis reveille le moteur
func (t *Turtle) SetMotion(id int, tx, ty, speed float64, mode AnimMode) error {
	if id < 0 {
		return fmt.Errorf("NUMERO DE TORTUE INVALIDE : %d", id)
	}
	if id >= MaxTurtles {
		return fmt.Errorf("TROP DE TORTUES (MAX %d)", MaxTurtles)
	}
	t.mu.Lock()
	t.activate()
	for len(t.states) <= id {
		t.states = append(t.states, defaultState())
	}
	t.ensureMotions()
	st := t.states[id]
	t.motions[id] = motionState{active: true, sx: st.X, sy: st.Y, tx: tx, ty: ty, speed: math.Abs(speed), mode: mode}
	wake := t.animWake
	t.notify() // la tortue id vient peut-etre de naitre : on l'affiche
	t.mu.Unlock()
	if wake != nil {
		wake() // hors verrou : le callback ne doit pas re-entrer dans la tortue
	}
	return nil
}

// STOPANIME n
func (t *Turtle) StopMotion(id int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cancelMotionLocked(id)
}

// STOPANIME sans argument
func (t *Turtle) StopAllMotion() {
	t.mu.Lock()
	defer t.mu.Unlock()
	for i := range t.motions {
		t.motions[i].active = false
	}
}

// avance d'un cran toutes les tortues animees, rend true tant qu'au moins une
// anim tourne. appele par le moteur temps reel a la cadence FPS, ou direct par
// les tests (deterministe)
func (t *Turtle) AnimStep() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	for i := range t.motions {
		if t.motions[i].active {
			t.stepMotion(i)
		}
	}
	return t.anyActiveLocked()
}

// garde motions a la meme longueur que states
func (t *Turtle) ensureMotions() {
	for len(t.motions) < len(t.states) {
		t.motions = append(t.motions, motionState{})
	}
	if len(t.motions) > len(t.states) {
		t.motions = t.motions[:len(t.states)]
	}
}

func (t *Turtle) anyActiveLocked() bool {
	for i := range t.motions {
		if t.motions[i].active {
			return true
		}
	}
	return false
}

// coupe l'anim de la tortue i (si elle existe)
func (t *Turtle) cancelMotionLocked(i int) {
	if i >= 0 && i < len(t.motions) {
		t.motions[i].active = false
	}
}

// avance la tortue i d'un cran vers sa cible, puis applique le mode une fois
// arrivee. reutilise goTo (donc respecte crayon et champ) en basculant la tortue
// active le temps du deplacement. coincee par un champ CLOS, elle se cogne au mur et s'arrete la
func (t *Turtle) stepMotion(i int) {
	m := &t.motions[i]
	saved := t.current
	t.current = i
	defer func() { t.current = saved }()

	st := t.states[i]
	dx, dy := m.tx-st.X, m.ty-st.Y
	dist := math.Hypot(dx, dy)
	if dist > m.speed && dist > 0 {
		if t.goTo(st.X+dx/dist*m.speed, st.Y+dy/dist*m.speed) != nil {
			m.active = false // sortie du champ CLOS : arret propre
		}
		return
	}
	// cible atteinte : on s'y pose pile, puis le mode decide la suite
	if t.goTo(m.tx, m.ty) != nil {
		m.active = false
		return
	}
	switch m.mode {
	case Once:
		m.active = false
	case Loop:
		if t.goTo(m.sx, m.sy) != nil { // retour instantane au depart
			m.active = false
		}
	case PingPong:
		m.sx, m.sy, m.tx, m.ty = m.tx, m.ty, m.sx, m.sy // on inverse depart/cible
	}
}
