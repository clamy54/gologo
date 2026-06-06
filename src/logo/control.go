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
	var params []string
	for !e.atEnd() && e.data[e.pos].Kind == DVarRef {
		params = append(params, strings.ToUpper(e.next().Text))
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
	if e.i.Lang() == "EN" {
		fmt.Fprintf(e.i.Out, "%s DEFINED\n", name)
	} else {
		fmt.Fprintf(e.i.Out, "VOUS VENEZ DE DEFINIR %s\n", name)
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
			initial = in.procsText(names) // [ ] -> "" (editeur vierge)
		case DWord, DSymbol:
			e.pos++
			initial = in.procsText([]string{strings.ToUpper(d.Text)})
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
func (i *Interp) procsText(names []string) string {
	var b strings.Builder
	for _, n := range names {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		if p := i.procs[n]; p != nil {
			b.WriteString(p.sourceText())
		} else {
			fmt.Fprintf(&b, "POUR %s\n\nFIN", n) // nouvelle procedure : squelette vide
		}
	}
	return b.String()
}

// redefinit les procedures POUR...FIN (source brute conservee), puis execute les
// eventuelles autres lignes du texte
func (i *Interp) defineFromEditor(text string) error {
	blocks := scanProcBlocks(text)
	for name := range blocks {
		delete(i.procs, name) // autorise la redefinition (sinon "... EXISTE DEJA")
	}
	if err := i.RunString(text); err != nil {
		return err
	}
	for name, raw := range blocks {
		if p := i.procs[name]; p != nil {
			p.text = raw // source fidele pour la prochaine edition
		}
	}
	return nil
}

// debut (POUR/TO) et fin (FIN/END) d'une procedure, toute casse, FR et EN
func isProcStart(w string) bool { return strings.EqualFold(w, "POUR") || strings.EqualFold(w, "TO") }

func isProcEnd(w string) bool { return strings.EqualFold(w, "FIN") || strings.EqualFold(w, "END") }

// repere les blocs POUR <nom>...FIN et rend nom -> texte brut. fait a la main,
// sans le lecteur de code, pour preserver la mise en forme de l'editeur
func scanProcBlocks(text string) map[string]string {
	out := map[string]string{}
	lines := strings.Split(text, "\n")
	for i := 0; i < len(lines); i++ {
		f := strings.Fields(lines[i])
		if len(f) < 2 || !isProcStart(f[0]) {
			continue
		}
		name := strings.ToUpper(f[1])
		start := i
		for i < len(lines) && !isProcEnd(strings.TrimSpace(lines[i])) {
			i++
		}
		if i < len(lines) { // FIN trouvee
			out[name] = strings.Join(lines[start:i+1], "\n")
		}
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
	name := strings.ToUpper(se.next().Text) // la variable de boucle
	start, err := se.expr(0)
	if err != nil {
		return Value{}, err
	}
	end, err := se.expr(0)
	if err != nil {
		return Value{}, err
	}
	from, err := toNumber(start)
	if err != nil {
		return Value{}, err
	}
	to, err := toNumber(end)
	if err != nil {
		return Value{}, err
	}
	step := 1.0
	if !se.atEnd() {
		s, err := se.expr(0)
		if err != nil {
			return Value{}, err
		}
		if step, err = toNumber(s); err != nil {
			return Value{}, err
		}
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
	}
	return None, nil
}
