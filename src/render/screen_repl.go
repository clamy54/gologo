package render

import (
	"gioui.org/io/event"
	"gioui.org/io/key"
	"strings"
)

// touche dans le REPL normal. le curseur (inputCol) bouge dans la ligne avec les
// fleches gauche/droite, sans deborder
func (s *Screen) replKey(ev event.Event) {
	switch e := ev.(type) {
	case key.EditEvent:
		if e.Text == "\t" { // Tab gere comme touche (completion), pas insere
			return
		}
		if !s.running.Load() {
			s.replInsert(typedText(e.Text)) // majuscules + exposants reecrits
		} else {
			s.kbPush([]rune(typedText(e.Text))...) // tampon clavier (TOUCHE?/LISCAR)
		}
	case key.Event:
		if e.State != key.Press {
			return
		}
		r := []rune(s.input)
		switch {
		case e.Name == key.NameTab && !s.running.Load():
			s.autocomplete()
		case e.Name == "C" && e.Modifiers.Contain(key.ModCtrl): // interruption : toujours Ctrl+C
			if s.interrupt != nil {
				s.interrupt()
			}
		case e.Name == "Q" && e.Modifiers.Contain(cmdMod):
			s.quit.Store(true)
		case e.Name == "E" && e.Modifiers.Contain(cmdMod):
			// ligne vide : Ctrl+E revient a taper "ED" puis Entree (ouvre l'editeur)
			if !s.running.Load() && strings.TrimSpace(s.input) == "" {
				s.Print("?ED")
				s.dispatch("ED")
			}
		case e.Name == "K" && e.Modifiers.Contain(cmdMod):
			// Ctrl+K : efface du curseur jusqu'au bout de la ligne (comme dans l'editeur)
			if !s.running.Load() {
				s.input = string(r[:s.inputCol])
			}
		case s.running.Load():
			if e.Name == key.NameReturn || e.Name == key.NameEnter {
				s.kbPush('\r') // tampon clavier : Entree -> retour-chariot
			}
		case e.Name == key.NameReturn || e.Name == key.NameEnter:
			line := s.input
			s.setInput("")
			s.Print("?" + line)
			if strings.TrimSpace(line) != "" {
				s.histAppend(line)
				s.dispatch(line)
			}
		case e.Name == key.NameDeleteBackward:
			if s.inputCol > 0 {
				s.input = string(r[:s.inputCol-1]) + string(r[s.inputCol:])
				s.inputCol--
			}
		case e.Name == key.NameDeleteForward: // EFF : efface sous le curseur (avant)
			if s.inputCol < len(r) {
				s.input = string(r[:s.inputCol]) + string(r[s.inputCol+1:])
			}
		case e.Name == key.NameLeftArrow:
			if s.inputCol > 0 {
				s.inputCol--
			}
		case e.Name == key.NameRightArrow:
			if s.inputCol < len(r) {
				s.inputCol++
			}
		case e.Name == key.NameHome:
			s.inputCol = 0
		case e.Name == key.NameEnd:
			s.inputCol = len(r)
		case e.Name == key.NameUpArrow: // historique : commande precedente
			if s.histIdx > 0 {
				s.histIdx--
				s.setInput(s.history[s.histIdx])
			}
		case e.Name == key.NameDownArrow: // historique : commande suivante
			if s.histIdx < len(s.history)-1 {
				s.histIdx++
				s.setInput(s.history[s.histIdx])
			} else {
				s.histIdx = len(s.history)
				s.setInput("")
			}
		}
	}
}

// sur AZERTY, le 2 en exposant est une touche directe que notre fonte ne sait pas
// dessiner. on en fait la touche puissance : elle insere "^", a toi de taper
// l'exposant (x puis ^ puis 7). le 3 en exposant, plus rare, reste "^3"
var exposantClavier = strings.NewReplacer("²", "^", "³", "^3")

// texte tape, pret a etre insere : majuscules + exposants clavier reecrits
func typedText(t string) string {
	return exposantClavier.Replace(strings.ToUpper(t))
}

// insere du texte a la position du curseur de l'invite
func (s *Screen) replInsert(t string) {
	r := []rune(s.input)
	s.input = string(r[:s.inputCol]) + t + string(r[s.inputCol:])
	s.inputCol += len([]rune(t))
}

// remplace la ligne de saisie, curseur en fin
func (s *Screen) setInput(line string) {
	s.input = line
	s.inputCol = len([]rune(line))
}

// complete le mot en cours sur TAB. un seul candidat : on insere le mot entier +
// une espace ; plusieurs : on complete le prefixe commun, ou on liste si rien a etendre
func (s *Screen) autocomplete() {
	if s.complete == nil {
		return
	}
	r := []rune(s.input)
	start := s.inputCol // debut du mot : recule jusqu'a une espace
	for start > 0 && r[start-1] != ' ' {
		start--
	}
	word := string(r[start:s.inputCol])
	if word == "" {
		return
	}
	cands := s.complete(word)
	switch {
	case len(cands) == 0:
		return
	case len(cands) == 1:
		s.replReplaceWord(start, cands[0]+" ")
	default:
		if lcp := longestCommonPrefix(cands); len([]rune(lcp)) > len([]rune(word)) {
			s.replReplaceWord(start, lcp)
		} else {
			s.Print(strings.Join(cands, "  ")) // ambigu : liste les choix
		}
	}
	s.invalidate()
}

// remplace le mot [start:inputCol] de la saisie par word
func (s *Screen) replReplaceWord(start int, word string) {
	r := []rune(s.input)
	s.input = string(r[:start]) + word + string(r[s.inputCol:])
	s.inputCol = start + len([]rune(word))
}

// plus long prefixe commun a une liste de mots. compare par runes pour qu'un nom
// accentue reste un UTF-8 valide une fois coupe
func longestCommonPrefix(words []string) string {
	if len(words) == 0 {
		return ""
	}
	pre := []rune(words[0])
	for _, w := range words[1:] {
		r := []rune(w)
		n := 0
		for n < len(pre) && n < len(r) && pre[n] == r[n] {
			n++
		}
		pre = pre[:n]
		if len(pre) == 0 {
			return ""
		}
	}
	return string(pre)
}

// ajoute une commande a l'historique (max 100, pas de doublon consecutif) et remet
// le curseur d'historique en fin. passe 100, les plus vieilles tombent dans l'oubli
func (s *Screen) histAppend(line string) {
	if n := len(s.history); n == 0 || s.history[n-1] != line {
		s.history = append(s.history, line)
		if len(s.history) > 100 {
			s.history = s.history[len(s.history)-100:]
		}
	}
	s.histIdx = len(s.history)
}
