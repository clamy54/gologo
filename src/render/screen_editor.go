package render

import (
	"beroot.com/logo/turtle"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"image"
	"image/color"
	"image/draw"
	"strings"
)

// mot sous le curseur de l'editeur : la suite de caracteres autour du curseur,
// coupee par un separateur ou la fin de ligne. "" si on n'est sur aucun mot. sert au F1 contextuel
func (s *Screen) edWordAtCursor() string {
	if s.edRow < 0 || s.edRow >= len(s.edLines) {
		return ""
	}
	line := []rune(s.edLines[s.edRow])
	col := s.edCol
	if col > len(line) {
		col = len(line)
	}
	// bornes du mot : espaces, delimiteurs Logo ([](){}:";) et operateurs (+-*/=<>),
	// pour isoler une commande collee a un crochet ou une parenthese (STOP dans
	// "[STOP]", AV dans "(AV 50)"). les caracteres d'un nom de primitive (lettres,
	// chiffres, ? . #) n'en sont pas
	isSep := func(r rune) bool {
		switch r {
		case ' ', '\t', '[', ']', '(', ')', '{', '}', ':', '"', ';', '+', '-', '*', '/', '=', '<', '>':
			return true
		}
		return false
	}
	start, end := col, col
	for start > 0 && !isSep(line[start-1]) {
		start--
	}
	for end < len(line) && !isSep(line[end]) {
		end++
	}
	return string(line[start:end])
}

// resultat de l'editeur ED : texte final + s'il a ete valide
type editResult struct {
	text string
	ok   bool
}

// ouvre l'editeur ED (Ctrl+S sauve et quitte, Ctrl+X quitte sans sauver) et met la
// tache de fond en attente. logo.Editor. edLines est confie a l'UI ; le resultat
// repart par un canal, pas par un champ partage
func (s *Screen) Edit(initial string) (string, bool) {
	s.edLines = splitLines(initial)
	s.edRow, s.edCol, s.edScroll = 0, 0, 0
	s.edYank = ""
	s.edSel, s.edTyping, s.edUndo = false, false, nil
	s.edSearchTyping, s.edSearchInput = false, ""
	s.edEN = false         // a l'ouverture, le 1er Ctrl+T traduit vers l'anglais
	s.edActive.Store(true) // rend edLines visible a l'interface avant qu'elle ne lise
	s.invalidate()
	res := <-s.edDone
	s.invalidate()
	return res.text, res.ok
}

// ferme l'editeur une seule fois, avec le resultat res (ne fait rien si deja fait ;
// non bloquant). meme garde que closeHelp contre le double-envoi
func (s *Screen) closeEdit(res editResult) {
	if s.edActive.CompareAndSwap(true, false) {
		select {
		case s.edDone <- res:
		default:
		}
	}
}

// touche dans l'editeur ED. Ctrl+S sauve et quitte, Ctrl+X quitte sans sauver.
// Shift+deplacement = selection ; frappe ou Suppr remplacent la selection.
// Ctrl+R efface jusqu'au bout de la ligne, Ctrl+B recolle
func (s *Screen) editorKey(ev event.Event) {
	if s.edSearchTyping { // saisie d'une recherche : capte tout (comme la page d'aide)
		s.edSearchKey(ev)
		return
	}
	switch e := ev.(type) {
	case key.EditEvent:
		if s.edSel { // taper remplace la selection
			s.edBeginEdit(false)
			s.edDeleteSelection()
		} else {
			s.edBeginEdit(true) // serie de frappes = un seul undo regroupe
		}
		s.edInsert(typedText(e.Text)) // majuscules + exposants reecrits
	case key.Event:
		if e.State != key.Press {
			return
		}
		s.edTyping = false                 // toute touche non-frappe cloture le groupe de frappes
		cmd := e.Modifiers.Contain(cmdMod) // Command sur Mac, Ctrl ailleurs
		shift := e.Modifiers.Contain(key.ModShift)
		switch {
		case e.Name == "S" && cmd: // sauver + quitter (CTRL+S)
			s.closeEdit(editResult{text: strings.Join(s.edLines, "\n"), ok: true})
		case e.Name == "X" && cmd: // quitter sans sauver (CTRL+X)
			s.closeEdit(editResult{ok: false})
		// Ctrl+Q ne ferme PLUS l'editeur (evitait un double Ctrl+Q qui quittait GoLogo
		// sans sauver) : touche ignoree volontairement
		case e.Name == "Z" && cmd: // annuler (CTRL+Z)
			s.edUndoPop()
		case e.Name == "A" && cmd: // selectionner tout (CTRL+A)
			s.edSelectAll()
		case e.Name == "T" && cmd: // traduit les instructions FR<->EN (CTRL+T)
			s.edTranslate()
		case e.Name == "F" && cmd: // ouvrir la recherche (CTRL+F)
			s.edStartSearch()
		case e.Name == "N" && cmd: // occurrence suivante (CTRL+N)
			s.edSearchNav(1)
		case e.Name == "P" && cmd: // occurrence precedente (CTRL+P)
			s.edSearchNav(-1)
		case e.Name == key.NameReturn || e.Name == key.NameEnter:
			s.edBeginEdit(false)
			if s.edSel {
				s.edDeleteSelection()
			}
			s.edNewline()
		case e.Name == key.NameDeleteBackward, e.Name == "D" && cmd:
			s.edBeginEdit(false)
			if s.edSel {
				s.edDeleteSelection()
			} else {
				s.edDeleteLeft()
			}
		case e.Name == key.NameDeleteForward:
			s.edBeginEdit(false)
			if s.edSel {
				s.edDeleteSelection()
			} else {
				s.edDeleteRight()
			}
		case (e.Name == "R" || e.Name == "K") && cmd: // efface jusqu'a la fin de ligne
			s.edBeginEdit(false)
			s.edSel = false
			s.edKillToEOL()
		case e.Name == "B" && cmd:
			s.edBeginEdit(false)
			s.edSel = false
			s.edInsert(s.edYank)
		case e.Name == key.NameLeftArrow:
			s.edStartSel(shift)
			s.edMove(-1, 0)
		case e.Name == key.NameRightArrow:
			s.edStartSel(shift)
			s.edMove(1, 0)
		case e.Name == key.NameUpArrow:
			s.edStartSel(shift)
			s.edMove(0, -1)
		case e.Name == key.NameDownArrow:
			s.edStartSel(shift)
			s.edMove(0, 1)
		case e.Name == key.NameHome:
			s.edStartSel(shift)
			s.edCol = 0
		case e.Name == key.NameEnd:
			s.edStartSel(shift)
			s.edCol = s.edLineLen(s.edRow)
		case e.Name == key.NamePageUp: // defile d'une page vers le haut
			s.edStartSel(shift)
			s.edMove(0, -edRows)
		case e.Name == key.NamePageDown: // defile d'une page vers le bas
			s.edStartSel(shift)
			s.edMove(0, edRows)
		}
	}
}

// Ctrl+T : traduit toutes les instructions du tampon FR<->EN. chaque appui inverse
// le sens (anglais, puis francais...). annulable par Ctrl+Z
func (s *Screen) edTranslate() {
	if s.translate == nil {
		return
	}
	s.edBeginEdit(false) // snapshot avant modification
	s.edSel = false
	out := s.translate(strings.Join(s.edLines, "\n"), !s.edEN)
	s.edLines = splitLines(out)
	s.edEN = !s.edEN
	if s.edRow > len(s.edLines)-1 {
		s.edRow = len(s.edLines) - 1
	}
	if s.edRow < 0 {
		s.edRow = 0
	}
	if l := s.edLineLen(s.edRow); s.edCol > l {
		s.edCol = l
	}
}

// un instantane de l'editeur, pour l'undo
type edSnap struct {
	lines    []string
	row, col int
}

// sauve l'etat courant (copie des lignes + curseur). on en garde 200 ; au-dela les
// plus vieux souvenirs partent a la poubelle
func (s *Screen) edSnapshot() {
	s.edUndo = append(s.edUndo, edSnap{append([]string(nil), s.edLines...), s.edRow, s.edCol})
	if len(s.edUndo) > 200 {
		s.edUndo = s.edUndo[len(s.edUndo)-200:]
	}
}

// sauve l'etat avant une modif. en frappe (typing), une seule sauvegarde pour toute
// la rafale de caracteres consecutifs
func (s *Screen) edBeginEdit(typing bool) {
	if typing {
		if !s.edTyping {
			s.edSnapshot()
			s.edTyping = true
		}
		return
	}
	s.edSnapshot()
}

// Ctrl+Z : restaure le dernier instantane
func (s *Screen) edUndoPop() {
	n := len(s.edUndo)
	if n == 0 {
		return
	}
	snap := s.edUndo[n-1]
	s.edUndo = s.edUndo[:n-1]
	s.edLines = snap.lines
	s.edRow, s.edCol = snap.row, snap.col
	if s.edRow > len(s.edLines)-1 {
		s.edRow = len(s.edLines) - 1
	}
	if l := s.edLineLen(s.edRow); s.edCol > l {
		s.edCol = l
	}
	s.edSel = false
	s.edTyping = false
}

// selection pendant un deplacement : Shift demarre ou prolonge (ancre = position
// courante), sans Shift on annule la selection
func (s *Screen) edStartSel(shift bool) {
	if !shift {
		s.edSel = false
		return
	}
	if !s.edSel {
		s.edSel = true
		s.edAnchorRow, s.edAnchorCol = s.edRow, s.edCol
	}
}

// Ctrl+A : selectionne tout le tampon. ancre en (0,0), curseur a la fin du document.
// rien a faire si le tampon est vide
func (s *Screen) edSelectAll() {
	last := len(s.edLines) - 1
	if last <= 0 && s.edLineLen(0) == 0 {
		return // tampon vide : rien a selectionner
	}
	s.edAnchorRow, s.edAnchorCol = 0, 0
	s.edRow, s.edCol = last, s.edLineLen(last)
	s.edSel = true
}

// --- recherche dans l'editeur (Ctrl+F), dans l'esprit de la page d'aide ---

// ouvre le champ de recherche (la barre du bas se vide pour la saisie)
func (s *Screen) edStartSearch() {
	s.edSearchTyping = true
	s.edSearchInput = ""
	s.edSel = false
	s.invalidate()
}

// saisie de la recherche : Echap annule, Entree valide (va a la 1re occurrence),
// Retour-arriere efface. texte force en MAJUSCULES, comme tout le Logo
func (s *Screen) edSearchKey(ev event.Event) {
	switch e := ev.(type) {
	case key.EditEvent:
		s.edSearchInput += strings.ToUpper(e.Text)
		s.invalidate()
	case key.Event:
		if e.State != key.Press {
			return
		}
		switch e.Name {
		case key.NameEscape:
			s.edSearchTyping, s.edSearchInput = false, "" // annule : plus de surbrillance
		case key.NameReturn, key.NameEnter:
			s.edSearchTyping = false
			s.edRunSearch()
		case key.NameDeleteBackward:
			if r := []rune(s.edSearchInput); len(r) > 0 {
				s.edSearchInput = string(r[:len(r)-1])
			}
		}
		s.invalidate()
	}
}

// debuts (ligne, colonne) de toutes les occurrences de query dans le tampon, dans
// l'ordre du document, sans tenir compte de la casse
func (s *Screen) edSearchMatches(query string) [][2]int {
	var out [][2]int
	if query == "" {
		return out
	}
	q := []rune(query)
	for r, line := range s.edLines {
		rl := []rune(line)
		for c := 0; c+len(q) <= len(rl); c++ {
			if strings.EqualFold(string(rl[c:c+len(q)]), query) {
				out = append(out, [2]int{r, c})
			}
		}
	}
	return out
}

// Entree : va a la 1re occurrence. rien trouve, rien de surligne
func (s *Screen) edRunSearch() {
	m := s.edSearchMatches(s.edSearchInput)
	if len(m) == 0 {
		s.edSearchInput = "" // pas de correspondance : pas de surbrillance
		return
	}
	s.edGoToMatch(m[0])
}

// surligne l'occurrence m (debut [ligne,colonne]) et pose le curseur juste apres.
// retient le debut pour Ctrl+N / Ctrl+P
func (s *Screen) edGoToMatch(m [2]int) {
	qlen := len([]rune(s.edSearchInput))
	s.edSearchRow, s.edSearchCol = m[0], m[1]
	s.edSel = false
	s.edRow = m[0]
	s.edCol = m[1] + qlen
	if l := s.edLineLen(s.edRow); s.edCol > l {
		s.edCol = l
	}
}

// va a l'occurrence suivante (dir=+1, Ctrl+N) ou precedente (dir=-1, Ctrl+P) depuis
// la courante, sans boucler. ne fait rien s'il y a moins de deux occurrences (comme
// la navigation des resultats dans la page d'aide)
func (s *Screen) edSearchNav(dir int) {
	if s.edSearchInput == "" {
		return
	}
	m := s.edSearchMatches(s.edSearchInput)
	if len(m) < 2 {
		return // 0 ou 1 occurrence : aucune navigation
	}
	cr, cc := s.edSearchRow, s.edSearchCol
	if dir > 0 {
		for _, p := range m {
			if p[0] > cr || (p[0] == cr && p[1] > cc) {
				s.edGoToMatch(p)
				s.invalidate()
				return
			}
		}
	} else {
		for i := len(m) - 1; i >= 0; i-- {
			if p := m[i]; p[0] < cr || (p[0] == cr && p[1] < cc) {
				s.edGoToMatch(p)
				s.invalidate()
				return
			}
		}
	}
}

// zone selectionnee remise dans l'ordre (debut <= fin), borne haute incluse (+1).
// ce qui est surligne est ce qui sera copie. ancre == curseur => plage vide
func (s *Screen) selRange() (r0, c0, r1, c1 int) {
	r0, c0, r1, c1 = s.edAnchorRow, s.edAnchorCol, s.edRow, s.edCol
	if r1 < r0 || (r1 == r0 && c1 < c0) {
		r0, c0, r1, c1 = r1, c1, r0, c0
	}
	empty := r0 == r1 && c0 == c1
	if !empty && c1 < s.edLineLen(r1) { // inclut le caractere sous la borne haute
		c1++
	}
	return
}

// texte a copier sur Ctrl+C : la selection si elle existe, sinon le seul caractere
// sous le curseur ("" en fin de ligne)
func (s *Screen) copyText() string {
	if s.edSel {
		return s.selectionText()
	}
	if r := []rune(s.edLines[s.edRow]); s.edCol < len(r) {
		return string(r[s.edCol])
	}
	return ""
}

// le texte selectionne, lignes jointes par \n
func (s *Screen) selectionText() string {
	r0, c0, r1, c1 := s.selRange()
	if r0 == r1 {
		line := []rune(s.edLines[r0])
		return string(line[c0:c1])
	}
	var b strings.Builder
	b.WriteString(string([]rune(s.edLines[r0])[c0:]))
	for r := r0 + 1; r < r1; r++ {
		b.WriteByte('\n')
		b.WriteString(s.edLines[r])
	}
	b.WriteByte('\n')
	b.WriteString(string([]rune(s.edLines[r1])[:c1]))
	return b.String()
}

// supprime le texte selectionne, curseur au debut, selection videe
func (s *Screen) edDeleteSelection() {
	r0, c0, r1, c1 := s.selRange()
	first := []rune(s.edLines[r0])
	last := []rune(s.edLines[r1])
	merged := string(first[:c0]) + string(last[c1:])
	s.edLines[r0] = merged
	if r1 > r0 {
		s.edLines = append(s.edLines[:r0+1], s.edLines[r1+1:]...)
	}
	s.edRow, s.edCol = r0, c0
	s.edSel = false
}

func (s *Screen) edLineLen(row int) int { return len([]rune(s.edLines[row])) }

// insere du texte (sans saut de ligne) au curseur
func (s *Screen) edInsert(t string) {
	if t == "" {
		return
	}
	r := []rune(s.edLines[s.edRow])
	ins := []rune(t)
	out := append(append(append([]rune{}, r[:s.edCol]...), ins...), r[s.edCol:]...)
	s.edLines[s.edRow] = string(out)
	s.edCol += len(ins)
}

// coupe la ligne courante au curseur
func (s *Screen) edNewline() {
	r := []rune(s.edLines[s.edRow])
	left, right := string(r[:s.edCol]), string(r[s.edCol:])
	s.edLines[s.edRow] = left
	rest := append([]string{right}, s.edLines[s.edRow+1:]...)
	s.edLines = append(s.edLines[:s.edRow+1], rest...)
	s.edRow++
	s.edCol = 0
}

// efface le caractere a gauche, ou recolle a la ligne du dessus
func (s *Screen) edDeleteLeft() {
	if s.edCol > 0 {
		r := []rune(s.edLines[s.edRow])
		s.edLines[s.edRow] = string(append(r[:s.edCol-1], r[s.edCol:]...))
		s.edCol--
		return
	}
	if s.edRow > 0 {
		prev := s.edLines[s.edRow-1]
		s.edCol = len([]rune(prev))
		s.edLines[s.edRow-1] = prev + s.edLines[s.edRow]
		s.edLines = append(s.edLines[:s.edRow], s.edLines[s.edRow+1:]...)
		s.edRow--
	}
}

// EFF : efface sous le curseur, ou recolle la ligne suivante
func (s *Screen) edDeleteRight() {
	r := []rune(s.edLines[s.edRow])
	if s.edCol < len(r) {
		s.edLines[s.edRow] = string(append(r[:s.edCol], r[s.edCol+1:]...))
		return
	}
	if s.edRow < len(s.edLines)-1 {
		s.edLines[s.edRow] += s.edLines[s.edRow+1]
		s.edLines = append(s.edLines[:s.edRow+1], s.edLines[s.edRow+2:]...)
	}
}

// efface du curseur a la fin de ligne, garde le bout coupe pour Ctrl+B (Ctrl+R)
func (s *Screen) edKillToEOL() {
	r := []rune(s.edLines[s.edRow])
	s.edYank = string(r[s.edCol:])
	s.edLines[s.edRow] = string(r[:s.edCol])
}

// bouge le curseur de (dx,dy) cellules, avec passage de ligne et bornage
func (s *Screen) edMove(dx, dy int) {
	if dy != 0 {
		s.edRow += dy
		if s.edRow < 0 {
			s.edRow = 0
		}
		if s.edRow > len(s.edLines)-1 {
			s.edRow = len(s.edLines) - 1
		}
		if n := s.edLineLen(s.edRow); s.edCol > n {
			s.edCol = n
		}
		return
	}
	s.edCol += dx
	if s.edCol < 0 { // recule vers la fin de la ligne precedente
		if s.edRow > 0 {
			s.edRow--
			s.edCol = s.edLineLen(s.edRow)
		} else {
			s.edCol = 0
		}
	} else if s.edCol > s.edLineLen(s.edRow) { // avance vers le debut de la suivante
		if s.edRow < len(s.edLines)-1 {
			s.edRow++
			s.edCol = 0
		} else {
			s.edCol = s.edLineLen(s.edRow)
		}
	}
}

// une ligne visuelle de l'editeur (apres repli). startCol = debut dans la ligne
// logique ; contLen = nb de caracteres de contenu ; text = le contenu (+ "!" si ca continue)
type edVisRow struct {
	logRow   int
	startCol int
	contLen  int
	text     string
}

// coupe les lignes longues en segments affichables ; le debordement est signale par
// un "!" en derniere colonne (juste visuel, le "!" n'est pas dans le texte)
func edVisRows(lines []string) []edVisRow {
	content := edCols - 1
	var out []edVisRow
	for li, line := range lines {
		r := []rune(line)
		start := 0
		for len(r)-start > content {
			out = append(out, edVisRow{li, start, content, string(r[start:start+content]) + "!"})
			start += content
		}
		out = append(out, edVisRow{li, start, len(r) - start, string(r[start:])})
	}
	return out
}

// combien de lignes visuelles pour une ligne logique de n runes
func edVisCount(n, content int) int {
	if n <= content {
		return 1
	}
	return (n + content - 1) / content
}

// dessine la vue de l'editeur ED dans s.frame
func (s *Screen) composeEditor() {
	bgc := rgba(turtle.Bleu)     // fond bleu (4)
	fgc := rgba(turtle.Color(6)) // cyan (texte + bord)
	// cadre cyan plein, puis interieur bleu
	draw.Draw(s.frame, s.frame.Bounds(), uniform(fgc), image.Point{}, draw.Src)
	inner := image.Rect(edBorder, edBorder, ScreenW-edBorder, ScreenH-edBorder)
	draw.Draw(s.frame, inner, uniform(bgc), image.Point{}, draw.Src)

	content := edCols - 1
	vis := edVisRows(s.edLines)

	// position visuelle du curseur (ligne visuelle globale + colonne)
	curBase := 0
	for li := 0; li < s.edRow; li++ {
		curBase += edVisCount(len([]rune(s.edLines[li])), content)
	}
	vrowInLine := s.edCol / content
	if vc := edVisCount(len([]rune(s.edLines[s.edRow])), content); vrowInLine >= vc {
		vrowInLine = vc - 1 // curseur en fin de ligne pleine : derniere colonne
	}
	curVis := curBase + vrowInLine
	curCol := s.edCol - vrowInLine*content

	// defilement (en lignes visuelles) : on garde le curseur visible
	if curVis < s.edScroll {
		s.edScroll = curVis
	}
	if curVis >= s.edScroll+edRows {
		s.edScroll = curVis - edRows + 1
	}

	var r0, c0, r1, c1 int
	if s.edSel {
		r0, c0, r1, c1 = s.selRange()
	}
	cellH := 13*textScale + 2
	for i := 0; i < edRows; i++ {
		vr := s.edScroll + i
		if vr >= len(vis) {
			break
		}
		v := vis[vr]
		y := edTextY + i*lineH
		// selection : on croise la plage [lo,hi) de la ligne logique avec le
		// contenu de cette ligne visuelle [startCol, startCol+contLen)
		if s.edSel && v.logRow >= r0 && v.logRow <= r1 {
			lo, hi := 0, len([]rune(s.edLines[v.logRow]))
			if v.logRow == r0 {
				lo = c0
			}
			if v.logRow == r1 {
				hi = c1
			}
			a, b := lo-v.startCol, hi-v.startCol
			if a < 0 {
				a = 0
			}
			if b > v.contLen {
				b = v.contLen
			}
			if b > a {
				fillRect(s.frame, edTextX+a*charW, y-2, (b-a)*charW, cellH, fgc) // bloc cyan
				drawText2x(s.frame, edTextX, y, v.text, fgc)                     // ligne en cyan
				seg := []rune(s.edLines[v.logRow])[v.startCol : v.startCol+v.contLen]
				drawText2x(s.frame, edTextX+a*charW, y, string(seg[a:b]), bgc) // selection en bleu
				continue
			}
		}
		drawText2x(s.frame, edTextX, y, v.text, fgc)
		// recherche : l'occurrence courante surlignee en jaune (texte bleu sur bloc jaune)
		if s.edSearchInput != "" && !s.edSearchTyping && v.logRow == s.edSearchRow {
			qlen := len([]rune(s.edSearchInput))
			a, b := s.edSearchCol-v.startCol, s.edSearchCol+qlen-v.startCol
			if a < 0 {
				a = 0
			}
			if b > v.contLen {
				b = v.contLen
			}
			if b > a {
				fillRect(s.frame, edTextX+a*charW, y-2, (b-a)*charW, cellH, helpHit)
				seg := []rune(s.edLines[v.logRow])[v.startCol : v.startCol+v.contLen]
				drawText2x(s.frame, edTextX+a*charW, y, string(seg[a:b]), bgc)
			}
		}
	}

	// curseur : bloc cyan inverse, le caractere redessine en bleu par-dessus
	if curVis >= s.edScroll && curVis < s.edScroll+edRows {
		cx := edTextX + curCol*charW
		cy := edTextY + (curVis-s.edScroll)*lineH
		fillRect(s.frame, cx, cy-2, charW, cellH, fgc)
		if r := []rune(s.edLines[s.edRow]); s.edCol < len(r) {
			drawText2x(s.frame, cx, cy, string(r[s.edCol]), bgc)
		}
	}

	s.composeEditStatus(fgc, bgc)
}

// dessine la barre de raccourcis en bas (video inverse, fond cyan / texte bleu)
func (s *Screen) composeEditStatus(fgc, bgc color.RGBA) {
	y := ScreenH - edBorder - lineH
	fillRect(s.frame, edBorder, y, ScreenW-2*edBorder, lineH, fgc)
	lang := "FR"
	if s.getLang != nil {
		lang = s.getLang()
	}
	txt := "F1 aide   " + modName + "+F chercher   " + modName + "+S sauver & quitter   " + modName + "+X quitter sans sauver"
	if lang == "EN" {
		txt = "F1 help   " + modName + "+F search   " + modName + "+S save & quit   " + modName + "+X quit without saving"
	}
	if s.edSearchTyping { // la barre du bas devient le champ de recherche
		txt = searchPrompt(lang) + s.edSearchInput + "_"
	}
	drawText2x(s.frame, edTextX, y+5, txt, bgc)
}
