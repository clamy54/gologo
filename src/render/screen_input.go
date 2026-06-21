package render

import (
	"beroot.com/logo/turtle"
	"gioui.org/io/clipboard"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/transfer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"io"
	"strings"
	"time"
)

// resultat d'une lecture clavier. LISCAR remplit ch, LL remplit line. ok faux si
// l'attente a ete interrompue (Ctrl+C) ou coupee par QUITTE
type kbRes struct {
	line string
	ch   rune
	ok   bool
}

// LISCAR (logo.Keyboard) : rend un caractere tape. vide d'abord les frappes en
// avance (coherent avec TOUCHE?), sinon met la tache de fond en attente
func (s *Screen) ReadChar() (rune, bool) {
	s.mu.Lock()
	if len(s.kbBuf) > 0 {
		r := s.kbBuf[0]
		s.kbBuf = s.kbBuf[1:]
		s.mu.Unlock()
		return r, true
	}
	s.mu.Unlock()
	s.kbWantLine = false
	s.kbActive.Store(true)
	s.invalidate()
	res := <-s.kbResult
	s.invalidate()
	return res.ch, res.ok
}

// LL (logo.Keyboard) : affiche une invite, laisse saisir une ligne (visible a
// l'ecran, modifiable) et la rend sur Entree. met la tache de fond en attente
func (s *Screen) ReadLine() (string, bool) {
	s.kbInput, s.kbInputCol = "", 0
	s.kbWantLine = true
	s.kbActive.Store(true)
	s.invalidate()
	res := <-s.kbResult
	s.invalidate()
	return res.line, res.ok
}

// TOUCHE? (logo.Keyboard) : une frappe attend-elle dans le tampon ? non bloquant
func (s *Screen) KeyPending() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.kbBuf) > 0
}

// termine une lecture clavier, une seule fois (ne fait rien si c'est deja fait ; non bloquant)
func (s *Screen) closeKb(res kbRes) {
	if s.kbActive.CompareAndSwap(true, false) {
		select {
		case s.kbResult <- res:
		default:
		}
	}
}

// empile des frappes dans le tampon clavier (borne, pour TOUCHE?/LISCAR) pendant
// qu'un programme tourne sans lecture active
func (s *Screen) kbPush(rs ...rune) {
	s.mu.Lock()
	if len(s.kbBuf)+len(rs) <= 256 {
		s.kbBuf = append(s.kbBuf, rs...)
	}
	s.mu.Unlock()
}

// met a jour position et etat souris depuis un evenement
func (s *Screen) handlePointer(e pointer.Event) {
	lx, ly, in := s.windowToLogo(e.Position.X, e.Position.Y)
	s.mu.Lock()
	s.mouseLX, s.mouseLY, s.mouseInField = lx, ly, in
	switch e.Kind {
	case pointer.Press:
		s.mouseBtn = true
	case pointer.Release:
		s.mouseBtn = false
	case pointer.Leave, pointer.Cancel:
		s.mouseInField, s.mouseBtn = false, false
	}
	s.mu.Unlock()
}

// pixels fenetre vers coords Logo, et dit si le pointeur est dans le champ.
// l'inverse de la mise a l'echelle de draw + logoToField
func (s *Screen) windowToLogo(px, py float32) (float64, float64, bool) {
	win := s.lastWin
	if win.X == 0 || win.Y == 0 {
		return 0, 0, false
	}
	sc := float32(win.X) / ScreenW
	if v := float32(win.Y) / ScreenH; v < sc {
		sc = v
	}
	fx := (px - (float32(win.X)-sc*ScreenW)/2) / sc // coords frame (ScreenW x ScreenH)
	fy := (py - (float32(win.Y)-sc*ScreenH)/2) / sc
	lx := turtle.MinX + (float64(fx)-fieldX)/fieldScale
	ly := turtle.MaxY - (float64(fy)-fieldY)/fieldScale
	in := fx >= fieldX && fx < fieldX+fieldW && fy >= fieldY && fy < fieldY+fieldH
	return lx, ly, in
}

// logo.Mouse : position Logo, et si le pointeur est dans le champ
func (s *Screen) MousePos() (float64, float64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mouseLX, s.mouseLY, s.mouseInField
}

// logo.Mouse : un bouton est-il presse ?
func (s *Screen) MouseDown() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mouseBtn
}

// un Press qui suit un Release de moins de ce delai, c'est une auto-repetition Gio
// en retard (course Stop/wakeup) : on l'ignore, sinon la touche reste "collee" a
// vrai apres qu'on l'a lachee, comme un fantome
const joyRepeatGuard = 90 * time.Millisecond

// suit l'etat des touches qui font la manette (fleches + espace) sans consommer
// l'evenement, pour que le REPL garde ses fleches
func (s *Screen) trackJoystick(e key.Event) {
	var st *bool
	switch e.Name {
	case key.NameUpArrow:
		st = &s.joyUp
	case key.NameDownArrow:
		st = &s.joyDown
	case key.NameLeftArrow:
		st = &s.joyLeft
	case key.NameRightArrow:
		st = &s.joyRight
	case key.NameSpace:
		st = &s.joyFire
	default:
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if e.State == key.Press {
		if time.Since(s.joyRel[string(e.Name)]) < joyRepeatGuard {
			return // Press d'auto-repetition arrive juste apres le Release : on l'ignore
		}
		*st = true
	} else {
		*st = false
		s.joyRel[string(e.Name)] = time.Now()
	}
}

// remet toutes les touches manette a "lache". appele au lancement d'un programme :
// si une touche etait restee collee, START repart propre
func (s *Screen) clearJoystick() {
	s.mu.Lock()
	s.joyUp, s.joyDown, s.joyLeft, s.joyRight, s.joyFire = false, false, false, false, false
	s.mu.Unlock()
}

// MANETTE (logo.Joystick) : direction 0-8 (0 = repos, 1 = haut, puis horaire). le
// clavier fait la manette 0 ; les autres dorment
func (s *Screen) JoyDir(n int) int {
	if n != 0 {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	vx, vy := 0, 0
	if s.joyRight {
		vx++
	}
	if s.joyLeft {
		vx--
	}
	if s.joyUp {
		vy++
	}
	if s.joyDown {
		vy--
	}
	return joyCode(vx, vy)
}

// transforme un deplacement (vx : +1 droite / -1 gauche ; vy : +1 haut / -1 bas)
// en code manette 0-8 (0 = repos, 1 = haut, puis horaire)
func joyCode(vx, vy int) int {
	switch {
	case vx == 0 && vy == 1:
		return 1
	case vx == 1 && vy == 1:
		return 2
	case vx == 1 && vy == 0:
		return 3
	case vx == 1 && vy == -1:
		return 4
	case vx == 0 && vy == -1:
		return 5
	case vx == -1 && vy == -1:
		return 6
	case vx == -1 && vy == 0:
		return 7
	case vx == -1 && vy == 1:
		return 8
	}
	return 0
}

// BOUTON? (logo.Joystick) : tir (barre d'espace) de la manette n. le clavier fait la 0
func (s *Screen) JoyButton(n int) bool {
	if n != 0 {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.joyFire
}

// lit le clavier et route vers l'editeur, le REPL ou les afficheurs. Ctrl+C
// interrompt, Ctrl+Q quitte a tout moment ; la frappe est bloquee pendant l'execution
func (s *Screen) handleInput(gtx layout.Context) {
	tag := s
	event.Op(gtx.Ops, tag) // enregistre le tag comme cible d'evenements clavier
	key.InputHintOp{Tag: tag, Hint: key.HintText}.Add(gtx.Ops)
	gtx.Execute(key.FocusCmd{Tag: tag})
	// zone pointeur plein ecran : suit la position et les clics
	ptr := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
	event.Op(gtx.Ops, tag)
	ptr.Pop()
	for {
		ev, ok := gtx.Event(
			pointer.Filter{Target: tag, Kinds: pointer.Move | pointer.Drag | pointer.Press | pointer.Release | pointer.Leave | pointer.Cancel},
			key.FocusFilter{Target: tag}, // EditEvent (texte tape) + focus
			key.Filter{Focus: tag, Name: key.NameReturn},
			key.Filter{Focus: tag, Name: key.NameEnter}, // Entree du pave numerique
			key.Filter{Focus: tag, Name: key.NameDeleteBackward},
			key.Filter{Focus: tag, Name: key.NameDeleteForward}, // EFF (suppr avant)
			// Deplacements : Shift optionnel (selection dans l'editeur).
			key.Filter{Focus: tag, Name: key.NameLeftArrow, Optional: key.ModShift},
			key.Filter{Focus: tag, Name: key.NameRightArrow, Optional: key.ModShift},
			key.Filter{Focus: tag, Name: key.NameUpArrow, Optional: key.ModShift},
			key.Filter{Focus: tag, Name: key.NameDownArrow, Optional: key.ModShift},
			key.Filter{Focus: tag, Name: key.NameHome, Optional: key.ModShift},
			key.Filter{Focus: tag, Name: key.NameEnd, Optional: key.ModShift},
			key.Filter{Focus: tag, Name: key.NamePageUp, Optional: key.ModShift},
			key.Filter{Focus: tag, Name: key.NamePageDown, Optional: key.ModShift},
			key.Filter{Focus: tag, Name: "C", Required: cmdMod},
			// Ctrl+C reste l'interruption sur TOUTES les plateformes (convention
			// terminal), meme sur Mac ou cmdMod = Command. la Cmd+C copie, Ctrl+C arrete
			key.Filter{Focus: tag, Name: "C", Required: key.ModCtrl},
			key.Filter{Focus: tag, Name: "Q", Required: cmdMod},              // QUITTER (Cmd+Q sur Mac, Ctrl+Q ailleurs)
			key.Filter{Focus: tag, Name: "S", Required: cmdMod},              // sauver+quitter (editeur)
			key.Filter{Focus: tag, Name: "D", Required: cmdMod},              // suppr. a gauche
			key.Filter{Focus: tag, Name: "R", Required: cmdMod},              // tue fin de ligne
			key.Filter{Focus: tag, Name: "K", Required: cmdMod},              // idem (raccourci connu)
			key.Filter{Focus: tag, Name: "B", Required: cmdMod},              // reinsere
			key.Filter{Focus: tag, Name: "V", Required: cmdMod},              // coller
			key.Filter{Focus: tag, Name: "X", Required: cmdMod},              // quitter l'editeur sans sauver
			key.Filter{Focus: tag, Name: "Z", Required: cmdMod},              // annuler (editeur)
			key.Filter{Focus: tag, Name: "A", Required: cmdMod},              // selectionner tout (editeur)
			key.Filter{Focus: tag, Name: "T", Required: cmdMod},              // traduire FR<->EN (editeur)
			key.Filter{Focus: tag, Name: "L", Required: cmdMod},              // basculer FR/EN (aide)
			key.Filter{Focus: tag, Name: "E", Required: cmdMod},              // ouvrir l'editeur (REPL, ligne vide)
			key.Filter{Focus: tag, Name: "I", Required: cmdMod},              // inserer la commande (aide)
			key.Filter{Focus: tag, Name: "F", Required: cmdMod},              // rechercher (aide)
			key.Filter{Focus: tag, Name: "N", Required: cmdMod},              // resultat de recherche suivant (aide)
			key.Filter{Focus: tag, Name: "P", Required: cmdMod},              // resultat de recherche precedent (aide)
			key.Filter{Focus: tag, Name: key.NameTab},                        // autocompletion (REPL)
			key.Filter{Focus: tag, Name: key.NameSpace},                      // bouton de tir manette (BOUTON?)
			key.Filter{Focus: tag, Name: key.NameF1, Optional: key.ModShift}, // aide generale (F1 = origine, Shift+F1 = tout)
			key.Filter{Focus: tag, Name: key.NameEscape},                     // quitter le pager
			transfer.TargetFilter{Target: tag, Type: "application/text"},
		)
		if !ok {
			break
		}
		if pe, ok := ev.(pointer.Event); ok { // souris : suivi position + clic
			s.handlePointer(pe)
			continue
		}
		if ke, ok := ev.(key.Event); ok { // suit l'etat manette (ne consomme pas l'evenement)
			s.trackJoystick(ke)
		}
		if s.pgActive.Load() { // navigateur d'aide : capte tout le clavier
			s.helpKey(ev)
			continue
		}
		if s.txtActive.Load() { // afficheur texte (CATALOGUE...) : capte tout le clavier
			s.pagerKey(ev)
			continue
		}
		if s.kbActive.Load() { // lecture clavier (LISCAR/LL) : capte tout le clavier
			s.kbKey(ev)
			continue
		}
		// F1 : ouvre l'aide par-dessus. dans l'editeur, si le mot sous le curseur a
		// une fiche, on l'ouvre direct ; sinon (et au REPL) l'aide generale
		if e, ok := ev.(key.Event); ok && e.State == key.Press && e.Name == key.NameF1 {
			if e.Modifiers.Contain(key.ModShift) { // Shift+F1 : bascule debutant <-> complet
				s.helpExtended = !s.helpExtended
			}
			if s.edActive.Load() && s.pgResolve != nil {
				if word := s.edWordAtCursor(); word != "" {
					if name, ok := s.pgResolve(word); ok {
						// aide directe sur un mot : on ouvre la vue complete. toutes
						// les fiches y figurent sous leur nom canonique (celui que
						// rend pgResolve) ; en vue debutant la fiche serait indexee
						// sous un autre nom (alias long) ou absente (commande
						// d'extension), d'ou une page vide.
						s.openHelpAt(name, true)
						continue
					}
				}
			}
			s.openHelp(s.helpExtended)
			continue
		}
		// presse-papier (a besoin de gtx) : traite ici, hors editorKey/replKey
		switch e := ev.(type) {
		case transfer.DataEvent: // reponse a un coller : on n'accepte que du texte
			r := e.Open()
			b, _ := io.ReadAll(r)
			r.Close()
			s.paste(string(b))
			continue
		case key.Event:
			if e.State == key.Press && e.Modifiers.Contain(cmdMod) {
				switch e.Name {
				case "V": // coller (REPL + editeur)
					gtx.Execute(clipboard.ReadCmd{Tag: tag}) // livre via DataEvent
					continue
				case "C": // copier : selection, ou a defaut le caractere sous le curseur
					if s.edActive.Load() {
						if txt := s.copyText(); txt != "" {
							gtx.Execute(clipboard.WriteCmd{Type: "application/text",
								Data: io.NopCloser(strings.NewReader(txt))})
						}
						continue // en editeur Ctrl+C = copier (jamais break)
					}
				}
			}
		}
		if s.edActive.Load() {
			s.editorKey(ev)
			continue
		}
		s.replKey(ev)
	}
}

// colle le presse-papier (en MAJUSCULES) au curseur : avec sauts de ligne dans
// l'editeur, en une seule ligne au REPL
func (s *Screen) paste(text string) {
	text = strings.ReplaceAll(text, "\r", "")
	text = typedText(text)
	if text == "" {
		return
	}
	if s.edActive.Load() {
		s.edBeginEdit(false)
		if s.edSel { // coller remplace la selection
			s.edDeleteSelection()
		}
		for i, part := range strings.Split(text, "\n") {
			if i > 0 {
				s.edNewline()
			}
			s.edInsert(part)
		}
		s.invalidate()
		return
	}
	if s.running.Load() {
		return
	}
	s.replInsert(strings.ReplaceAll(text, "\n", " "))
	s.invalidate()
}

// route une frappe pendant LISCAR/LL. LISCAR : le 1er caractere resout la lecture
// (Entree = CR). LL : edition de ligne a l'ecran, validee par Entree. Ctrl+C coupe, Ctrl+Q quitte
func (s *Screen) kbKey(ev event.Event) {
	switch e := ev.(type) {
	case key.EditEvent:
		txt := typedText(e.Text)
		if txt == "" {
			return
		}
		if !s.kbWantLine { // LISCAR : 1er caractere tape
			s.closeKb(kbRes{ch: []rune(txt)[0], ok: true})
			return
		}
		r := []rune(s.kbInput) // LL : insere au curseur
		s.kbInput = string(r[:s.kbInputCol]) + txt + string(r[s.kbInputCol:])
		s.kbInputCol += len([]rune(txt))
		s.invalidate()
	case key.Event:
		if e.State != key.Press {
			return
		}
		switch {
		case e.Name == "C" && e.Modifiers.Contain(key.ModCtrl): // interruption : toujours Ctrl+C
			if s.interrupt != nil {
				s.interrupt()
			}
			s.closeKb(kbRes{ok: false})
		case e.Name == "Q" && e.Modifiers.Contain(cmdMod):
			s.quit.Store(true)
			s.closeKb(kbRes{ok: false})
		case !s.kbWantLine: // LISCAR : Entree = retour-chariot ; le reste ignore
			if e.Name == key.NameReturn || e.Name == key.NameEnter {
				s.closeKb(kbRes{ch: '\r', ok: true})
			}
		case e.Name == key.NameReturn || e.Name == key.NameEnter: // LL : valide la ligne
			line := s.kbInput
			s.kbInput, s.kbInputCol = "", 0
			s.Print("?" + line) // echo de la ligne validee
			s.closeKb(kbRes{line: line, ok: true})
		case e.Name == key.NameDeleteBackward:
			if s.kbInputCol > 0 {
				r := []rune(s.kbInput)
				s.kbInput = string(r[:s.kbInputCol-1]) + string(r[s.kbInputCol:])
				s.kbInputCol--
				s.invalidate()
			}
		case e.Name == key.NameDeleteForward:
			r := []rune(s.kbInput)
			if s.kbInputCol < len(r) {
				s.kbInput = string(r[:s.kbInputCol]) + string(r[s.kbInputCol+1:])
				s.invalidate()
			}
		case e.Name == key.NameLeftArrow:
			if s.kbInputCol > 0 {
				s.kbInputCol--
				s.invalidate()
			}
		case e.Name == key.NameRightArrow:
			if s.kbInputCol < len([]rune(s.kbInput)) {
				s.kbInputCol++
				s.invalidate()
			}
		case e.Name == key.NameHome:
			s.kbInputCol = 0
			s.invalidate()
		case e.Name == key.NameEnd:
			s.kbInputCol = len([]rune(s.kbInput))
			s.invalidate()
		}
	}
}
