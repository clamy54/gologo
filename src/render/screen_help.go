package render

import (
	"beroot.com/logo/logo"
	"fmt"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"image"
	"image/color"
	"image/draw"
	"strings"
)

// decoupe en lignes editables (toujours au moins une)
func splitLines(s string) []string {
	if s == "" {
		return []string{""}
	}
	return strings.Split(s, "\n")
}

// lignes de contenu visibles (une est reservee au pied de page)
const pgRows = (ScreenH-2*margin)/lineH - 1

// ouvre le navigateur d'aide (logo.Helper) et met la tache de fond en attente
// (Q/Echap ferme). start vide = mode liste ; sinon ouvre direct le detail de la commande
func (s *Screen) Help(names []string, details map[string][]string, start, lang string, switchLang logo.HelpSwitch) {
	s.pgNames = names
	s.pgDetails = details
	s.pgDetail = start
	s.pgLang = lang
	s.pgSwitch = switchLang
	s.pgOverlay = false
	s.pgExtended = true // primitive AIDE : aide complete
	s.pgScroll = 0
	s.pgSel = 0
	s.pgSearchTyping, s.pgSearchInput, s.pgSearchHits = false, "", nil
	for i, n := range names {
		if n == start {
			s.pgSel = i
			break
		}
	}
	s.pgActive.Store(true)
	s.invalidate()
	<-s.pgDone
	s.invalidate()
}

// ouvre l'aide generale (grille) par-dessus (F1), sans bloquer. extended=false :
// aide debutant (commandes d'origine) ; true (Shift+F1) : tout
func (s *Screen) openHelp(extended bool) { s.openHelpAt("", extended) }

// ouvre l'aide par-dessus. start="" => grille (liste) ; start=nom de fiche => direct
// la page de detail (et le curseur dessus dans la grille). extended : liste complete
// (true) ou commandes d'origine seulement (false). a la fermeture on revient a l'etat
// du dessous sans signaler pgDone
func (s *Screen) openHelpAt(start string, extended bool) {
	if s.pgOpen == nil || s.pgActive.Load() {
		return
	}
	names, details, lang, switchLang := s.pgOpen(extended)
	s.pgNames, s.pgDetails, s.pgDetail = names, details, start
	s.pgLang, s.pgSwitch = lang, switchLang
	s.pgExtended = extended
	s.pgSel, s.pgScroll = 0, 0
	s.pgSearchTyping, s.pgSearchInput, s.pgSearchHits = false, "", nil
	for i, n := range names { // place la selection sur la fiche ouverte
		if n == start {
			s.pgSel = i
			break
		}
	}
	s.pgOverlay = true
	s.pgActive.Store(true)
	s.invalidate()
}

// logo.Pager : imprime si le texte tient dans le REPL, sinon ouvre un afficheur
// plein ecran (facon "more") et met la tache de fond en attente jusqu'a Q
func (s *Screen) Page(title string, lines []string) {
	if len(lines) <= s.textCapacity() {
		for _, l := range lines {
			s.Print(l)
		}
		return
	}
	s.txtTitle = title
	s.txtLines = lines
	s.txtScroll = 0
	s.txtActive.Store(true)
	s.invalidate()
	<-s.txtDone
	s.invalidate()
}

// nombre de lignes de texte visibles dans le REPL (selon que le graphique est la ou
// non, et la limite ME), moins une pour l'invite
func (s *Screen) textCapacity() int {
	top := margin
	s.mu.Lock()
	g := s.graphics && !s.meTextOnly
	meLines := s.meLines
	s.mu.Unlock()
	if g {
		top = fieldY + fieldH + margin
	}
	n := (ScreenH - top - margin) / lineH
	if meLines < n {
		n = meLines
	}
	if n > textRows {
		n = textRows
	}
	n-- // une ligne reservee a l'invite
	if n < 1 {
		n = 1
	}
	return n
}

// ferme l'afficheur texte une seule fois (rien si deja fait ; non bloquant)
func (s *Screen) closePage() {
	if s.txtActive.CompareAndSwap(true, false) {
		select {
		case s.txtDone <- struct{}{}:
		default:
		}
	}
}

// borne le defilement du pager texte
func (s *Screen) clampPager() {
	if max := len(s.txtLines) - pgRows; s.txtScroll > max {
		s.txtScroll = max
	}
	if s.txtScroll < 0 {
		s.txtScroll = 0
	}
}

// defilement du pager texte ; Q/Echap ferme
func (s *Screen) pagerKey(ev event.Event) {
	if escOrQ(ev) || hardQuit(ev) {
		s.closePage()
		return
	}
	switch e := ev.(type) {
	case key.EditEvent:
		if e.Text == " " {
			s.txtScroll += pgRows
		}
	case key.Event:
		if e.State != key.Press {
			return
		}
		switch e.Name {
		case key.NameDownArrow:
			s.txtScroll++
		case key.NameUpArrow:
			s.txtScroll--
		case key.NamePageDown:
			s.txtScroll += pgRows
		case key.NamePageUp:
			s.txtScroll -= pgRows
		case key.NameHome:
			s.txtScroll = 0
		case key.NameEnd:
			s.txtScroll = len(s.txtLines)
		}
	}
	s.clampPager()
	s.invalidate()
}

// dessine le pager texte plein ecran : fond noir, lignes defilables, pied de page
// avec la position et les touches
func (s *Screen) composePager() {
	draw.Draw(s.frame, s.frame.Bounds(), uniBlack, image.Point{}, draw.Src)
	last := s.txtScroll
	for i := 0; i < pgRows; i++ {
		row := s.txtScroll + i
		if row >= len(s.txtLines) {
			break
		}
		drawString2x(s.frame, margin, margin+i*lineH, s.txtLines[row])
		last = row + 1
	}
	foot := fmt.Sprintf("%s  l.%d-%d/%d  [ESPACE/PgSuiv] defiler  [Q] quitter", s.txtTitle, s.txtScroll+1, last, len(s.txtLines))
	if s.getLang != nil && s.getLang() == "EN" {
		foot = fmt.Sprintf("%s  l.%d-%d/%d  [SPACE/PgDn] scroll  [Q] quit", s.txtTitle, s.txtScroll+1, last, len(s.txtLines))
	}
	drawText2x(s.frame, margin, ScreenH-margin-13*textScale, foot, helpFoot)
}

// ferme l'aide une seule fois (rien si deja fait ; non bloquant). en superposition,
// retour silencieux ; sinon on signale pgDone. le CAS unique evite le double-envoi
func (s *Screen) closeHelp() {
	if !s.pgActive.CompareAndSwap(true, false) {
		return
	}
	s.pgSearchTyping, s.pgSearchInput, s.pgSearchHits = false, "", nil // remet la recherche a zero
	if s.pgOverlay {
		s.pgOverlay = false
		s.invalidate()
		return
	}
	select {
	case s.pgDone <- struct{}{}:
	default:
	}
}

// dispo de la grille de commandes (mode liste) : nombre de colonnes et largeur de
// colonne, en cellules de caracteres
func (s *Screen) helpGrid() (cols, colW int) {
	maxLen := 1
	for _, n := range s.pgNames {
		if len(n) > maxLen {
			maxLen = len(n)
		}
	}
	colW = maxLen + 2
	cols = (ScreenW - 2*margin) / charW / colW
	if cols < 1 {
		cols = 1
	}
	return
}

// borne le defilement en mode detail
func (s *Screen) pgClampDetail() {
	if max := len(s.pgDetails[s.pgDetail]) - pgRows; s.pgScroll > max {
		s.pgScroll = max
	}
	if s.pgScroll < 0 {
		s.pgScroll = 0
	}
}

// route le clavier du navigateur d'aide selon le mode (liste/detail). Q vaut Echap.
// Ctrl+C / Ctrl+Q quittent de n'importe ou
func (s *Screen) helpKey(ev event.Event) {
	if ctrlL(ev) {
		s.helpSwitchLang()
		return
	}
	// F1 / Shift+F1 : bascule a chaud entre vue debutant (origine) et complete
	if e, ok := ev.(key.Event); ok && e.State == key.Press && e.Name == key.NameF1 {
		s.helpToggleMode(e.Modifiers.Contain(key.ModShift))
		return
	}
	if s.pgDetail == "" {
		s.helpListKey(ev)
	} else {
		s.helpDetailKey(ev)
	}
}

// vrai sur Ctrl+L (bascule de langue de l'aide)
func ctrlL(ev event.Event) bool {
	e, ok := ev.(key.Event)
	return ok && e.State == key.Press && e.Modifiers.Contain(cmdMod) && e.Name == "L"
}

// bascule l'aide FR<->EN (Ctrl+L) via pgSwitch, en gardant la commande affichee
// (liste ou detail) grace a sa traduction
func (s *Screen) helpSwitchLang() {
	if s.pgSwitch == nil {
		return
	}
	newLang := "EN"
	if s.pgLang == "EN" {
		newLang = "FR"
	}
	current := s.pgDetail // mode detail : nom affiche
	if current == "" && len(s.pgNames) > 0 {
		current = s.pgNames[s.pgSel] // mode liste : selection
	}
	names, details, newCurrent := s.pgSwitch(newLang, current)
	s.pgNames, s.pgDetails, s.pgLang = names, details, newLang
	s.pgSearchHits = nil  // les noms et le texte changent de langue : surbrillance obsolete
	if s.pgDetail != "" { // on etait dans le detail : y rester si traduit
		s.pgDetail = newCurrent
		s.pgScroll = 0
	}
	s.pgSel = 0
	for i, n := range names {
		if n == newCurrent {
			s.pgSel = i
			break
		}
	}
	s.invalidate()
}

// bascule l'aide entre vue debutant (commandes d'origine, F1) et complete (toutes
// les primitives, Shift+F1). reconstruit la liste via pgOpen et garde la commande
// affichee si elle existe encore (sinon retour grille). rien si on y est deja
func (s *Screen) helpToggleMode(extended bool) {
	if s.pgOpen == nil || extended == s.pgExtended {
		return
	}
	current := s.pgDetail // mode detail : nom affiche
	if current == "" && len(s.pgNames) > 0 {
		current = s.pgNames[s.pgSel] // mode liste : selection
	}
	names, details, lang, switchLang := s.pgOpen(extended)
	s.pgNames, s.pgDetails, s.pgLang, s.pgSwitch = names, details, lang, switchLang
	s.pgExtended = extended
	s.pgSearchHits = nil
	s.pgSel, s.pgScroll = 0, 0
	found := false
	for i, n := range names {
		if n == current {
			s.pgSel = i
			found = true
			break
		}
	}
	if s.pgDetail != "" && !found {
		s.pgDetail = "" // la fiche n'existe pas dans cette vue : retour a la grille
	}
	s.invalidate()
}

// ferme l'aide et insere le mot au curseur (editeur ou REPL). word passe par
// valeur, copie avant closeHelp
func (s *Screen) helpInsertWord(word string) {
	if word == "" {
		return
	}
	s.closeHelp() // revient a l'editeur/REPL (edActive/running inchanges)
	if s.edActive.Load() {
		s.edBeginEdit(false)
		if s.edSel {
			s.edDeleteSelection()
		}
		s.edInsert(word)
	} else {
		s.replInsert(word) // REPL : insere au curseur
	}
	s.invalidate()
}

// insere le nom de commande selectionne dans la grille (touche 'i' sur la liste)
func (s *Screen) helpInsertSelected() {
	if len(s.pgNames) == 0 {
		return
	}
	s.helpInsertWord(s.pgNames[s.pgSel])
}

// vrai si l'evenement est Echap ou Q (touche ou frappe)
func escOrQ(ev event.Event) bool {
	switch e := ev.(type) {
	case key.EditEvent:
		return strings.EqualFold(e.Text, "Q")
	case key.Event:
		return e.State == key.Press && e.Name == key.NameEscape
	}
	return false
}

// vrai sur Ctrl+C ou Cmd/Ctrl+Q (sortie immediate de l'aide). Ctrl+C reste
// l'interruption partout ; Q suit la touche commande (Command sur Mac)
func hardQuit(ev event.Event) bool {
	e, ok := ev.(key.Event)
	if !ok || e.State != key.Press {
		return false
	}
	return (e.Name == "C" && e.Modifiers.Contain(key.ModCtrl)) ||
		(e.Name == "Q" && e.Modifiers.Contain(cmdMod))
}

// deplace la selection vers une commande qui commence par la lettre tapee (sans
// tenir compte de la casse). si la selection courante commence DEJA par cette lettre,
// on passe a la suivante en bouclant (taper la meme lettre = tourner en rond dans ce
// groupe) ; sinon on va a la 1re. la page suit pgSel (cf helpPaging). rien si aucune ne colle
func (s *Screen) helpJumpToLetter(text string) {
	r := []rune(strings.TrimSpace(text))
	if len(r) == 0 {
		return
	}
	prefix := strings.ToUpper(string(r[0]))
	n := len(s.pgNames)
	if n == 0 {
		return
	}
	// deja sur cette lettre : on cherche la suivante en bouclant (k de 1 a n ; k==n
	// revient sur pgSel, donc une lettre unique reste en place)
	if s.pgSel < n && strings.HasPrefix(s.pgNames[s.pgSel], prefix) {
		for k := 1; k <= n; k++ {
			if i := (s.pgSel + k) % n; strings.HasPrefix(s.pgNames[i], prefix) {
				s.pgSel = i
				s.invalidate()
				return
			}
		}
		return
	}
	// sinon : la 1re commande qui commence par la lettre
	for i, nm := range s.pgNames {
		if strings.HasPrefix(nm, prefix) {
			s.pgSel = i
			s.invalidate()
			return
		}
	}
}

// nb max de caracteres dans la barre de recherche (le texte ne doit pas depasser la
// largeur de la ligne)
func (s *Screen) searchMaxLen() int {
	m := (ScreenW-2*margin)/charW - len(searchPrompt(s.pgLang)) - 1
	if m < 1 {
		m = 1
	}
	return m
}

func searchPrompt(lang string) string {
	if lang == "EN" {
		return "Search: "
	}
	return "Recherche : "
}

// ouvre le champ de recherche (la barre du bas se vide pour la saisie)
func (s *Screen) startSearch() {
	s.pgSearchTyping = true
	s.pgSearchInput = ""
	s.invalidate()
}

// saisie de la recherche dans la barre du bas : Echap annule, Entree valide (calcule
// les commandes a surligner), Retour-arriere efface
func (s *Screen) helpSearchKey(ev event.Event) {
	switch e := ev.(type) {
	case key.EditEvent:
		s.pgSearchInput += strings.ToUpper(e.Text) // pas de minuscules sur notre Logo
		if r := []rune(s.pgSearchInput); len(r) > s.searchMaxLen() {
			s.pgSearchInput = string(r[:s.searchMaxLen()])
		}
		s.invalidate()
	case key.Event:
		if e.State != key.Press {
			return
		}
		switch e.Name {
		case key.NameEscape:
			s.pgSearchTyping = false // abandonne la saisie ; la barre de raccourcis revient
		case key.NameReturn, key.NameEnter:
			s.helpRunSearch() // calcule les surbrillances
			s.pgSearchTyping = false
		case key.NameDeleteBackward:
			if r := []rune(s.pgSearchInput); len(r) > 0 {
				s.pgSearchInput = string(r[:len(r)-1])
			}
		}
		s.invalidate()
	}
}

// trouve les commandes dont la fiche contient la phrase saisie (hors section "Voir
// aussi"). phrase vide ou rien trouve : pas de surbrillance (tout blanc, comme a l'ouverture)
func (s *Screen) helpRunSearch() {
	s.pgSearchHits = nil
	q := strings.ToUpper(strings.TrimSpace(s.pgSearchInput))
	if q == "" {
		return
	}
	hits := map[string]bool{}
	for _, name := range s.pgNames {
		if helpPageContains(s.pgDetails[name], q, s.pgLang) {
			hits[name] = true
		}
	}
	if len(hits) > 0 {
		s.pgSearchHits = hits
		// Selectionne la 1re commande en jaune (la page affichee suit pgSel).
		for i, name := range s.pgNames {
			if hits[name] {
				s.pgSel = i
				break
			}
		}
	}
}

// deplace la selection sur le resultat de recherche (commande en jaune) suivant
// (dir=+1, Ctrl+N) ou precedent (dir=-1, Ctrl+P), sans boucler. rien s'il y a moins
// de deux resultats, ou plus rien dans cette direction. la page suit pgSel
func (s *Screen) helpSearchNav(dir int) {
	if len(s.pgSearchHits) < 2 { // 0 ou 1 resultat : aucune navigation
		return
	}
	n := len(s.pgNames)
	for k := 1; k <= n; k++ {
		i := s.pgSel + dir*k
		if i < 0 || i >= n {
			return // pas de bouclage : rien au-dela du bord
		}
		if s.pgSearchHits[s.pgNames[i]] {
			s.pgSel = i
			s.invalidate()
			return
		}
	}
}

// la phrase qUpper (deja en MAJUSCULES) apparait-elle dans la fiche, en laissant de
// cote la section "Voir aussi" / "See also" et tout ce qui suit ?
func helpPageContains(lines []string, qUpper, lang string) bool {
	seeAlso := "VOIR AUSSI"
	if lang == "EN" {
		seeAlso = "SEE ALSO"
	}
	var b strings.Builder
	for _, l := range lines {
		if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(l)), seeAlso) {
			break // exclut "Voir aussi" et ce qui suit
		}
		b.WriteString(strings.ToUpper(l))
		b.WriteByte('\n')
	}
	return strings.Contains(b.String(), qUpper)
}

// navigation dans la grille de commandes
func (s *Screen) helpListKey(ev event.Event) {
	if s.pgSearchTyping { // saisie d'une recherche : capte tout
		s.helpSearchKey(ev)
		return
	}
	if escOrQ(ev) || hardQuit(ev) {
		s.closeHelp() // Q/Echap depuis la liste : on quitte
		return
	}
	if ed, ok := ev.(key.EditEvent); ok {
		if ed.Text == "/" { // '/' ouvre la recherche
			s.startSearch()
			return
		}
		// une lettre tapee : saute a la 1re commande de cette lettre (la page qui
		// contient la selection est recalculee toute seule)
		s.helpJumpToLetter(ed.Text)
		return
	}
	e, ok := ev.(key.Event)
	if !ok || e.State != key.Press {
		return
	}
	// Ctrl+I : insere le mot selectionne dans l'editeur/REPL puis ferme
	if e.Name == "I" && e.Modifiers.Contain(cmdMod) {
		s.helpInsertSelected()
		return
	}
	// Ctrl+F ouvre la recherche
	if e.Name == "F" && e.Modifiers.Contain(cmdMod) {
		s.startSearch()
		return
	}
	// Ctrl+N / Ctrl+P : resultat de recherche suivant ou precedent (en jaune)
	if e.Modifiers.Contain(cmdMod) && e.Name == "N" {
		s.helpSearchNav(1)
		return
	}
	if e.Modifiers.Contain(cmdMod) && e.Name == "P" {
		s.helpSearchNav(-1)
		return
	}
	cols, _ := s.helpGrid()
	n := len(s.pgNames)
	switch e.Name {
	case key.NameReturn, key.NameEnter:
		if n > 0 {
			s.pgDetail = s.pgNames[s.pgSel] // ouvrir le detail
			s.pgScroll = 0
		}
	case key.NameRightArrow:
		if s.pgSel < n-1 {
			s.pgSel++
		}
	case key.NameLeftArrow:
		if s.pgSel > 0 {
			s.pgSel--
		}
	case key.NameDownArrow:
		if s.pgSel+cols < n {
			s.pgSel += cols
		}
	case key.NameUpArrow:
		if s.pgSel-cols >= 0 {
			s.pgSel -= cols
		}
	case key.NamePageDown:
		perPage, _, _ := s.helpPaging()
		if s.pgSel+perPage < n {
			s.pgSel += perPage
		} else {
			s.pgSel = n - 1
		}
	case key.NamePageUp:
		perPage, _, _ := s.helpPaging()
		if s.pgSel-perPage >= 0 {
			s.pgSel -= perPage
		} else {
			s.pgSel = 0
		}
	case key.NameHome:
		s.pgSel = 0
	case key.NameEnd:
		s.pgSel = n - 1
	}
	s.invalidate()
}

// defilement du detail ; Q/Echap revient a la liste
func (s *Screen) helpDetailKey(ev event.Event) {
	if hardQuit(ev) {
		s.closeHelp()
		return
	}
	if escOrQ(ev) {
		s.pgDetail = "" // retour a la liste
		s.invalidate()
		return
	}
	switch e := ev.(type) {
	case key.EditEvent:
		if e.Text == " " {
			s.pgScroll += pgRows
		}
	case key.Event:
		if e.State != key.Press {
			return
		}
		// Ctrl+I : insere la commande affichee dans l'editeur/REPL
		if e.Name == "I" && e.Modifiers.Contain(cmdMod) {
			s.helpInsertWord(s.pgDetail)
			return
		}
		switch e.Name {
		case key.NameDownArrow:
			s.pgScroll++
		case key.NameUpArrow:
			s.pgScroll--
		case key.NamePageDown:
			s.pgScroll += pgRows
		case key.NamePageUp:
			s.pgScroll -= pgRows
		case key.NameHome:
			s.pgScroll = 0
		case key.NameEnd:
			s.pgScroll = len(s.pgDetails[s.pgDetail])
		}
	}
	s.pgClampDetail()
	s.invalidate()
}

// dessine le navigateur d'aide (liste ou detail) dans s.frame
func (s *Screen) composeHelp() {
	draw.Draw(s.frame, s.frame.Bounds(), uniBlack, image.Point{}, draw.Src)
	if s.pgDetail == "" {
		s.composeHelpList()
	} else {
		s.composeHelpDetail()
	}
}

var helpFoot = color.RGBA{170, 255, 255, 255} // cyan pale pour le pied de page
var helpHit = color.RGBA{255, 255, 0, 255}    // jaune : commande trouvee par la recherche

// dessine la grille des commandes, paginee (Pg.Prec/Pg.Suiv). la pagination a ete
// ajoutee parce que les noms longs depassaient pgRows et coupaient la liste
func (s *Screen) composeHelpList() {
	cols, colW := s.helpGrid()
	perPage, pages, page := s.helpPaging()
	start := page * perPage
	end := start + perPage
	if end > len(s.pgNames) {
		end = len(s.pgNames)
	}
	for i := start; i < end; i++ {
		name := s.pgNames[i]
		local := i - start
		r, c := local/cols, local%cols
		x, y := margin+c*colW*charW, margin+r*lineH
		switch {
		case i == s.pgSel: // surbrillance de la selection : bloc gris clair, texte noir
			fillRect(s.frame, x-2, y-2, len(name)*charW+4, 13*textScale+2, textDefault)
			drawText2x(s.frame, x, y, name, color.RGBA{0, 0, 0, 255})
		case s.pgSearchHits[name]: // resultat de recherche : jaune
			drawText2x(s.frame, x, y, name, helpHit)
		default:
			drawString2x(s.frame, x, y, name)
		}
	}
	footY := ScreenH - margin - 13*textScale
	if s.pgSearchTyping { // la barre du bas devient le champ de recherche
		drawText2x(s.frame, margin, footY, searchPrompt(s.pgLang)+s.pgSearchInput+"_", helpFoot)
		return
	}
	c := modAbbr
	// bascule debutant <-> complet : montre la touche de l'autre vue
	toggle := "  [Maj+F1] tout"
	if s.pgExtended {
		toggle = "  [F1] origine"
	}
	foot := fmt.Sprintf("AIDE  p.%d/%d  [%s-i] inserer  [%s-F] chercher  [%s-L] langue%s  [Q]", page+1, pages, c, c, c, toggle)
	if s.pgLang == "EN" {
		toggle = "  [Shift+F1] all"
		if s.pgExtended {
			toggle = "  [F1] basics"
		}
		foot = fmt.Sprintf("HELP  p.%d/%d  [%s-i] insert  [%s-F] search  [%s-L] lang%s  [Q]", page+1, pages, c, c, c, toggle)
	}
	drawText2x(s.frame, margin, footY, foot, helpFoot)
}

// taille d'une page (perPage), nombre total de pages, et page qui contient la
// selection courante. perPage >= 1 ; pages >= 1
func (s *Screen) helpPaging() (perPage, pages, page int) {
	cols, _ := s.helpGrid()
	perPage = cols * pgRows
	if perPage < 1 {
		perPage = 1
	}
	pages = (len(s.pgNames) + perPage - 1) / perPage
	if pages < 1 {
		pages = 1
	}
	page = s.pgSel / perPage
	return
}

// dessine la fiche d'aide d'une commande
func (s *Screen) composeHelpDetail() {
	lines := s.pgDetails[s.pgDetail]
	for i := 0; i < pgRows; i++ {
		row := s.pgScroll + i
		if row >= len(lines) {
			break
		}
		drawString2x(s.frame, margin, margin+i*lineH, lines[row])
	}
	foot := "AIDE " + s.pgDetail + "   [" + modAbbr + "-i] inserer   [" + modAbbr + "-L] langue   [Q/Echap] retour"
	if s.pgLang == "EN" {
		foot = "HELP " + s.pgDetail + "   [" + modAbbr + "-i] insert   [" + modAbbr + "-L] lang   [Q/Esc] back"
	}
	drawText2x(s.frame, margin, ScreenH-margin-13*textScale, foot, helpFoot)
}
