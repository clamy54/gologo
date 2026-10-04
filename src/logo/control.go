package logo

import (
	"fmt"
	"strings"
)

// lit "POUR nom :p1 :p2 ... <corps> FIN" et definit la procedure
func formePour(e *eval) (Value, error) {
	if e.atEnd() || e.data[e.pos].Kind != DSymbol {
		return Value{}, fmt.Errorf("POUR ATTEND UN NOM DE PROCEDURE")
	}
	name := strings.ToUpper(e.next().Text)
	if !validName(name) {
		return Value{}, fmt.Errorf("POUR ATTEND UN NOM DE PROCEDURE")
	}
	var params []string
	for !e.atEnd() && e.data[e.pos].Kind == DVarRef {
		params = append(params, strings.ToUpper(e.next().Text))
	}
	if err := checkParams(params); err != nil {
		return Value{}, err
	}
	var body []Datum
	found := false
	for !e.atEnd() {
		d := e.next()
		if d.Kind == DSymbol && isProcEnd(d.Text) {
			found = true
			break
		}
		body = append(body, d)
	}
	if !found {
		return Value{}, fmt.Errorf("FIN MANQUANT POUR %s", name)
	}
	if e.i.prims[name] != nil {
		return Value{}, fmt.Errorf("%s EXISTE DEJA", name)
	}
	e.i.procs[name] = &userProc{name: name, params: params, body: body}
	if !e.i.Quiet {
		msg := "VOUS VENEZ DE DEFINIR " + name
		if e.i.Lang() == "EN" {
			msg = name + " DEFINED"
		}
		// la procedure est definie quoi qu'il arrive ; si le message ne peut pas
		// s'ecrire (sortie cassee), on le signale au lieu de l'ignorer
		if err := e.i.printLine(msg); err != nil {
			return Value{}, err
		}
	}
	return None, nil
}

// ED : ED (dernier contenu), ED NOM, ED [A B...], ED [] (vide).
// a la validation le texte est interprete ; Ctrl+C abandonne sans rien redefinir
func formeED(e *eval) (Value, error) {
	in := e.i
	if in.editor == nil {
		return Value{}, fmt.Errorf("EDITEUR INDISPONIBLE")
	}
	if in.Turtle != nil {
		in.Turtle.StopAllMotion() // ouvrir l'editeur coupe les animations en cours
	}
	initial := in.edBuf // ED nu : on rouvre le dernier contenu
	if !e.atEnd() {
		switch d := e.data[e.pos]; d.Kind {
		case DList:
			e.pos++
			var names []string
			for _, n := range d.List {
				names = append(names, strings.ToUpper(n.Text))
			}
			var err error
			if initial, err = in.procsText(names); err != nil { // [ ] -> "" (editeur vierge)
				return Value{}, err
			}
		case DWord, DSymbol:
			e.pos++
			var err error
			if initial, err = in.procsText([]string{strings.ToUpper(d.Text)}); err != nil {
				return Value{}, err
			}
		}
	}
	text, ok := in.editor(initial)
	if !ok {
		return None, nil // Ctrl+C : abandon, rien n'est redefini
	}
	in.edBuf = text
	return None, in.defineFromEditor(text)
}

// concatene la source des procedures nommees (squelette vide si inconnue)
func (i *Interp) procsText(names []string) (string, error) {
	var b strings.Builder
	for _, n := range names {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		if p := i.procs[n]; p != nil {
			src, err := p.sourceText()
			if err != nil {
				return "", err
			}
			b.WriteString(src)
		} else {
			lit, ok := nameLiteral(n)
			if !ok {
				return "", &badData{n}
			}
			fmt.Fprintf(&b, "POUR %s\n\nFIN", lit) // nouvelle procedure : squelette vide
		}
	}
	return b.String(), nil
}

// les parametres d'une procedure : des noms non vides, tous differents. deux
// parametres de meme nom se partageraient une seule variable (le second argument
// ecraserait le premier sans rien dire)
func checkParams(params []string) error {
	seen := make(map[string]bool, len(params))
	for _, p := range params {
		if !validName(p) {
			return fmt.Errorf("NOM DE PARAMETRE INVALIDE : %s", p)
		}
		if seen[p] {
			return fmt.Errorf("PARAMETRE EN DOUBLE : %s", p)
		}
		seen[p] = true
	}
	return nil
}

// redefinit les procedures POUR...FIN (source brute conservee), puis execute les
// eventuelles autres lignes du texte
func (i *Interp) defineFromEditor(text string) error {
	blocks := scanProcBlocks(text)
	old := make(map[string]*userProc, len(blocks))
	for name := range blocks {
		if p := i.procs[name]; p != nil {
			old[name] = p // garde l'ancienne def pour la restaurer si l'edition est fautive
		}
		delete(i.procs, name) // le texte ne doit pas s'appuyer sur l'ancienne version
	}
	err := i.RunString(text)
	if _, stop := err.(*ctrl); err != nil && !stop {
		// edition invalide : on annule les definitions du texte (meme celles deja
		// refaites avant l'erreur) et on retrouve les procedures d'avant. le reste
		// de ce que le texte a execute avant l'erreur (variables, dessin, fichiers)
		// n'est pas defait
		for name := range blocks {
			if p := old[name]; p != nil {
				i.procs[name] = p
			} else {
				delete(i.procs, name)
			}
		}
		return err
	}
	for name, raw := range blocks {
		if p := i.procs[name]; p != nil {
			p.text = raw // source fidele pour la prochaine edition
		}
	}
	return err // nil, ou le signal LOGO/RAZ qui continue de remonter
}

// debut (POUR/TO) et fin (FIN/END) d'une procedure, toute casse, FR et EN
func isProcStart(w string) bool { return strings.EqualFold(w, "POUR") || strings.EqualFold(w, "TO") }

func isProcEnd(w string) bool { return strings.EqualFold(w, "FIN") || strings.EqualFold(w, "END") }

// repere les blocs POUR <nom>...FIN et rend nom -> texte brut, pour preserver la
// mise en forme de l'editeur. le decoupage suit celui du lecteur (cf lexSource) :
// un FIN suivi d'un commentaire, plusieurs definitions sur une ligne, un POUR dans
// un commentaire, une continuation ("POUR~" en fin de ligne) ou un mot-cle ecrit
// avec un echappement sont vus comme le lecteur les verra
func scanProcBlocks(text string) map[string]string {
	out := map[string]string{}
	toks := lexSource(text)
	for k := 0; k < len(toks); k++ {
		t := toks[k]
		if t.kind != tokWord || t.depth != 0 || !isProcStart(t.name()) {
			continue
		}
		// le nom : le prochain jeton utile, un nom nu
		n := k + 1
		for n < len(toks) && (toks[n].kind == tokSpace || toks[n].kind == tokComment) {
			n++
		}
		if n >= len(toks) || toks[n].kind != tokWord || toks[n].depth != 0 {
			continue
		}
		name := toks[n].name()
		// la fin : le prochain FIN/END au meme niveau (hors listes)
		end := -1
		for m := n + 1; m < len(toks); m++ {
			if toks[m].kind == tokWord && toks[m].depth == 0 && isProcEnd(toks[m].name()) {
				end = m
				break
			}
		}
		if end < 0 {
			break // FIN manquant : le lecteur le dira
		}
		stop := toks[end].end
		// un commentaire qui suit FIN sur la meme ligne fait partie du bloc
		for m := end + 1; m < len(toks); m++ {
			if toks[m].kind == tokComment {
				stop = toks[m].end
				break
			}
			if toks[m].kind != tokSpace || strings.Contains(toks[m].s, "\n") {
				break
			}
		}
		out[name] = text[t.start:stop]
		k = end
	}
	return out
}

// RENDS obj (alias RETOURNE/RET, anglais OUTPUT/OP) : termine la procedure en
// rendant obj. si obj est juste un appel d'operation a une procedure utilisateur
// et que RENDS est en position terminale, on traite l'appel comme terminal
// (recursion terminale deroulee par callProc). sinon, evaluation normale
func formeRends(e *eval) (Value, error) {
	tail := e.headTail // pose par invoke0 : RENDS est-il la derniere instruction ?
	if e.atEnd() {
		return Value{}, fmt.Errorf("PAS ASSEZ DE DONNEES POUR RENDS")
	}
	if tail {
		if d := e.data[e.pos]; d.Kind == DSymbol {
			up := strings.ToUpper(d.Text)
			if proc := e.i.procs[up]; proc != nil && e.i.prims[up] == nil {
				e.pos++
				args, err := e.readArgs(up, len(proc.params))
				if err != nil {
					return Value{}, err
				}
				if e.atEnd() { // RENDS <proc> <args> : appel terminal (valeur attendue)
					return None, &tailCall{proc: proc, args: args, output: true}
				}
				// un operateur suit (ex. RENDS P :x + 1) : plus terminal. on appelle la
				// procedure puis on poursuit l'expression avec sa valeur
				v, err := e.i.callProc(proc, args)
				if err != nil {
					return Value{}, err
				}
				v, err = e.exprFrom(v, 0)
				if err != nil {
					return Value{}, err
				}
				return rendsResult(v)
			}
		}
	}
	v, err := e.expr(0)
	if err != nil {
		return Value{}, err
	}
	return rendsResult(v)
}

// emballe la valeur d'un RENDS en signal de sortie, en exigeant qu'elle existe :
// RENDS d'une commande muette donne "PAS ASSEZ DE DONNEES POUR RENDS"
func rendsResult(v Value) (Value, error) {
	if !v.HasValue() {
		return Value{}, fmt.Errorf("PAS ASSEZ DE DONNEES POUR RENDS")
	}
	return None, &ctrl{kind: ctlOutput, val: v}
}

// lit "SI pred liste" ou "SI pred liste1 liste2"
func formeSi(e *eval) (Value, error) {
	tail := e.headTail // SI candidat a la position terminale (pose par invoke0)
	pred, err := e.expr(0)
	if err != nil {
		return Value{}, err
	}
	then, err := e.listArg("SI")
	if err != nil {
		return Value{}, err
	}
	var els []Datum
	hasElse := false
	if !e.atEnd() && (e.data[e.pos].Kind == DList || e.data[e.pos].Kind == DGroup) {
		els, err = e.listArg("SI")
		if err != nil {
			return Value{}, err
		}
		hasElse = true
	}
	// la branche prise est terminale si le SI lui-meme l'etait (derniere instruction) :
	// la recursion terminale traverse donc le SI
	branchTail := tail && e.atEnd()
	if pred.IsTrue() {
		return None, e.i.runSeqTail(then, branchTail)
	}
	if hasElse {
		return None, e.i.runSeqTail(els, branchTail)
	}
	return None, nil
}

// TANTQUE [condition] [instructions] : repete les instructions tant que la
// condition rend VRAI
func formeTantque(e *eval) (Value, error) {
	cond, err := e.listArg("TANTQUE")
	if err != nil {
		return Value{}, err
	}
	body, err := e.listArg("TANTQUE")
	if err != nil {
		return Value{}, err
	}
	for {
		if e.i.brk.Load() { // Ctrl+C verifie meme quand le corps est vide
			return Value{}, ErrInterrompu
		}
		v, err := e.i.evalCode(cond, false)
		if err != nil {
			return Value{}, err
		}
		if !v.IsTrue() {
			return None, nil
		}
		if err := e.i.runSeq(body); err != nil {
			return Value{}, err
		}
	}
}

// SCENE [instructions] : dessine le bloc dans un tampon cache puis l'affiche d'un
// coup (double tampon). evite le clignotement des animations qui NETTOIE+redessinent
// a chaque image. EndFrame est garanti par defer meme sur erreur ou STOP, pour ne
// jamais laisser l'ecran fige. sans ecran (texte/headless), execute juste le bloc
func formeScene(e *eval) (Value, error) {
	body, err := e.listArg("SCENE")
	if err != nil {
		return Value{}, err
	}
	if fb, ok := e.i.frameBuffer(); ok {
		fb.BeginFrame()
		defer fb.EndFrame()
	}
	return None, e.i.runSeq(body)
}

// REPETEPOUR [var debut fin (pas)] [instructions] : var prend les valeurs de debut
// a fin (par pas, defaut 1). var est locale a la boucle
func formeRepetepour(e *eval) (Value, error) {
	spec, err := e.listArg("REPETEPOUR")
	if err != nil {
		return Value{}, err
	}
	body, err := e.listArg("REPETEPOUR")
	if err != nil {
		return Value{}, err
	}
	se := &eval{i: e.i, data: spec}
	if se.atEnd() {
		return Value{}, &badData{"[]"} // "REPETEPOUR N'AIME PAS ..."
	}
	specText := Datum{Kind: DList, List: spec}.String()
	nd := se.next() // la variable de boucle : un nom (nu ou "mot)
	if (nd.Kind != DSymbol && nd.Kind != DWord) || nd.Text == "" {
		return Value{}, &badData{specText}
	}
	name := strings.ToUpper(nd.Text)
	start, err := se.expr(0)
	if err != nil {
		return Value{}, err
	}
	end, err := se.expr(0)
	if err != nil {
		return Value{}, err
	}
	from, err := toFinite(start)
	if err != nil {
		return Value{}, err
	}
	to, err := toFinite(end)
	if err != nil {
		return Value{}, err
	}
	step := 1.0
	if !se.atEnd() {
		s, err := se.expr(0)
		if err != nil {
			return Value{}, err
		}
		if step, err = toFinite(s); err != nil {
			return Value{}, err
		}
	}
	if !se.atEnd() { // [var debut fin pas] et rien d'autre
		return Value{}, &badData{specText}
	}
	if step == 0 {
		return Value{}, &badData{"0"}
	}
	// la variable de boucle est locale : on l'empile comme pour un appel
	frame := map[string]Value{name: NumberValue(from)}
	e.i.frames = append(e.i.frames, frame)
	defer func() { e.i.frames = e.i.frames[:len(e.i.frames)-1] }()
	for v := from; (step > 0 && v <= to) || (step < 0 && v >= to); v += step {
		if e.i.brk.Load() {
			return Value{}, ErrInterrompu
		}
		frame[name] = NumberValue(v)
		if err := e.i.runSeq(body); err != nil {
			return Value{}, err
		}
		if v+step == v {
			break // pas trop petit devant v (au-dela de 2^53) : v n'avancerait plus jamais
		}
	}
	return None, nil
}
