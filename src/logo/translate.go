package logo

import (
	"sort"
	"strings"
)

// decoupage d'un texte source en jetons, avec leur position : le meme decoupage que
// le lecteur (reader.go), mais sans rien jeter, pour pouvoir reecrire le texte en
// gardant sa mise en page. sert a la traduction (Ctrl+T) et au reperage des blocs
// POUR...FIN de l'editeur
type tokKind uint8

const (
	tokSpace   tokKind = iota // blancs, sauts de ligne compris
	tokComment                // ; ou # jusqu'a la fin de la ligne
	tokOpen                   // [ ( {
	tokClose                  // ] ) }
	tokWord                   // nom nu (ou nombre)
	tokQuoted                 // "mot
	tokVar                    // :nom
	tokOp                     // operateur infixe
)

// un jeton. start/end le situent dans le texte BRUT ; s est son texte tel que le
// lecteur le voit, c'est-a-dire sans les caracteres de continuation ('!' ou '~' en
// fin de ligne) : "POUR~" suivi d'un saut de ligne est bien le mot POUR
type srcTok struct {
	kind       tokKind
	start, end int // octets dans le texte brut
	depth      int // imbrication de crochets ( 0 = hors de toute liste )
	s          string
}

func isBlankByte(c byte) bool { return c == ' ' || c == '\t' || c == '\r' || c == '\n' }

// fin (exclue) du mot qui commence en pos, selon les regles de reader.readToken :
// '\' ou '$' protege le caractere suivant, un blanc protege termine le mot, et un
// nom nu s'arrete aussi sur un operateur infixe
func scanWord(src string, pos int, breakOnOp bool) int {
	start := pos
	for pos < len(src) {
		c := src[pos]
		if (c == '\\' || c == '$') && pos+1 < len(src) {
			nc := src[pos+1]
			pos += 2
			if isBlankByte(nc) {
				break
			}
			continue
		}
		if isDelim(rune(c)) {
			break
		}
		if breakOnOp && isInfixOp(rune(c)) {
			alt := c == '<' && pos+1 < len(src) && (src[pos+1] == '#' || src[pos+1] == 'b' || src[pos+1] == 'B')
			if (c == '+' || c == '-') && expSignPending(src[start:pos]) {
				pos++
				continue
			}
			if !alt {
				break
			}
		}
		pos++
	}
	return pos
}

func lexSource(raw string) []srcTok {
	// le lecteur retire les continuations avant de lire : on decoupe le meme texte
	// que lui, puis on reporte chaque jeton a sa place dans le texte brut
	src, cuts := stripContinuations(raw)
	rawPos := func(k int) int { // position brute de l'octet k du texte nettoye
		n := sort.Search(len(cuts), func(j int) bool { return cuts[j].at > k })
		if n == 0 {
			return k
		}
		return k + cuts[n-1].removed
	}
	var toks []srcTok
	depth := 0
	n := len(src)
	for i := 0; i < n; {
		c := src[i]
		j := i + 1
		kind := tokWord
		d := depth
		switch {
		case isBlankByte(c):
			kind = tokSpace
			for j < n && isBlankByte(src[j]) {
				j++
			}
		case c == ';' || c == '#':
			kind = tokComment
			for j < n && src[j] != '\n' {
				j++
			}
		case c == '[' || c == '(' || c == '{':
			kind = tokOpen
			depth++
		case c == ']' || c == ')' || c == '}':
			kind = tokClose
			if depth > 0 {
				depth--
			}
			d = depth
		case c == '"':
			kind = tokQuoted
			j = scanWord(src, i+1, false)
		case c == ':':
			kind = tokVar
			j = scanWord(src, i+1, true)
		case c == '-' && j < n && (isDigit(rune(src[j])) || src[j] == '.') &&
			(i == 0 || isBlankByte(src[i-1]) || strings.IndexByte("[({", src[i-1]) >= 0 || isInfixOp(rune(src[i-1]))):
			j = scanWord(src, j, true) // nombre negatif, comme le lit le lecteur
		case isInfixOp(rune(c)):
			kind = tokOp
			if j < n {
				if two := src[i : j+1]; two == "<=" || two == ">=" || two == "<>" {
					j++
				}
			}
		default:
			if k := scanWord(src, i, true); k > i {
				j = k
			}
		}
		toks = append(toks, srcTok{kind: kind, start: rawPos(i), end: rawPos(j-1) + 1, depth: d, s: src[i:j]})
		i = j
	}
	return toks
}

// retire les '\' d'echappement d'un nom
func unescape(w string) string {
	if !strings.ContainsAny(w, `\$`) {
		return w
	}
	var b strings.Builder
	for k := 0; k < len(w); k++ {
		if (w[k] == '\\' || w[k] == '$') && k+1 < len(w) {
			k++
		}
		b.WriteByte(w[k])
	}
	return b.String()
}

// le nom (en MAJ, sans echappement) que porte un jeton mot
func (t srcTok) name() string { return strings.ToUpper(unescape(t.s)) }

type translator struct {
	i     *Interp
	src   string
	lang  string
	toks  []srcTok
	match []int          // crochet ouvrant -> indice de son fermant (len(toks) s'il manque)
	out   []string       // texte de remplacement d'un jeton ("" = inchange)
	code  map[int]bool   // crochets ouvrants dont le contenu est du code
	arity map[string]int // nombre de parametres des procedures definies dans le texte
}

// remplace dans src chaque nom de primitive par son equivalent dans la langue cible
// (toEN = anglais, sinon FR), en gardant la mise en page exacte. c'est le Ctrl+T de
// l'editeur.
//
// seul le CODE est traduit : le programme lui-meme et les listes d'instructions des
// primitives qui en executent (REPETE, SI, TANTQUE...). une liste de donnees reste
// telle quelle, sinon ECRIS [AVANCE] afficherait FORWARD. pour savoir quelle liste
// est laquelle on suit les arguments un par un, d'apres le nombre d'arguments de
// chaque primitive et procedure. des qu'on ne sait pas (procedure inconnue...), on
// ne traduit pas la liste : c'est sans danger, les noms des deux langues marchent
// toujours
func (i *Interp) TranslateProgram(src string, toEN bool) string {
	tr := &translator{i: i, src: src, lang: "FR", toks: lexSource(src), code: map[int]bool{}, arity: map[string]int{}}
	if toEN {
		tr.lang = "EN"
	}
	tr.out = make([]string, len(tr.toks))
	tr.match = make([]int, len(tr.toks))
	var open []int
	for k, t := range tr.toks {
		switch t.kind {
		case tokOpen:
			tr.match[k] = len(tr.toks)
			open = append(open, k)
		case tokClose:
			if n := len(open); n > 0 {
				tr.match[open[n-1]] = k
				open = open[:n-1]
			}
		}
	}
	// procedures definies dans le texte lui-meme : POUR nom :a :b
	for k, t := range tr.toks {
		if t.kind != tokWord || t.depth != 0 || !isProcStart(t.name()) {
			continue
		}
		m := tr.skipBlank(k+1, len(tr.toks))
		if m >= len(tr.toks) || tr.toks[m].kind != tokWord {
			continue
		}
		name, n := tr.toks[m].name(), 0
		for m = tr.skipBlank(m+1, len(tr.toks)); m < len(tr.toks) && tr.toks[m].kind == tokVar; m = tr.skipBlank(m+1, len(tr.toks)) {
			n++
		}
		tr.arity[name] = n
	}
	tr.level(0, len(tr.toks), true, 0)
	var b strings.Builder
	prev := 0
	for k, t := range tr.toks {
		b.WriteString(src[prev:t.start]) // caracteres de continuation entre deux jetons
		if tr.out[k] != "" {
			b.WriteString(tr.out[k])
		} else {
			b.WriteString(src[t.start:t.end])
		}
		prev = t.end
	}
	b.WriteString(src[prev:])
	return b.String()
}

// parcourt les jetons [from,to) d'un meme niveau de crochets ; code = ce niveau est
// du code a traduire
func (tr *translator) level(from, to int, code bool, depth int) {
	if depth > maxReadDepth {
		return // imbrication demesuree : on laisse le texte tel quel
	}
	for k := from; k < to; k++ {
		t := tr.toks[k]
		switch t.kind {
		case tokOpen:
			inner := false
			switch t.s {
			case "(":
				inner = code // un groupe est du code si ce qui l'entoure en est
			case "[":
				inner = tr.code[k]
			}
			end := tr.match[k]
			if end > to {
				end = to
			}
			tr.level(k+1, end, inner, depth+1)
			k = end
		case tokWord:
			if !code {
				continue
			}
			fr, ok := tr.primitive(k)
			if !ok {
				continue
			}
			if out := canonical(tr.lang, fr, helpData[fr]); out != "" {
				tr.out[k] = out
			}
			tr.markCodeLists(k, to, fr)
		}
	}
}

// le jeton k nomme-t-il une primitive ? rend sa cle d'aide (nom FR canonique)
func (tr *translator) primitive(k int) (string, bool) {
	up := tr.toks[k].name()
	if tr.i.prims[up] == nil {
		return "", false // pas une primitive disponible : nom de procedure, mot...
	}
	fr, _, ok := lookupHelp(up)
	return fr, ok
}

// premier jeton utile a partir de m (saute blancs et commentaires)
func (tr *translator) skipBlank(m, to int) int {
	for m < to && (tr.toks[m].kind == tokSpace || tr.toks[m].kind == tokComment) {
		m++
	}
	return m
}

// la liste [ ... ] qui commence en m (blancs sautes), ou ok=false
func (tr *translator) listAt(m, to int) (int, bool) {
	m = tr.skipBlank(m, to)
	if m < to && tr.toks[m].kind == tokOpen && tr.toks[m].s == "[" && tr.match[m] < to {
		return m, true
	}
	return m, false
}

// nombre d'arguments du nom up, si on le connait : primitive ordinaire, procedure
// du texte ou procedure deja definie. inconnu pour une forme speciale (elle lit ses
// arguments a sa facon) ou un nom jamais vu
func (tr *translator) arityOf(up string) (int, bool) {
	if p := tr.i.prims[up]; p != nil {
		return p.arity, p.special == nil
	}
	if n, ok := tr.arity[up]; ok {
		return n, true
	}
	if p := tr.i.procs[up]; p != nil {
		return len(p.params), true
	}
	return 0, false
}

// saute une expression complete a partir de m : un terme, puis d'eventuels
// operateurs infixes et leurs termes. rend l'indice qui la suit ; ok=false si on ne
// sait pas la delimiter avec certitude
func (tr *translator) skipExpr(m, to, depth int) (int, bool) {
	m, ok := tr.skipTerm(m, to, depth)
	for ok {
		j := tr.skipBlank(m, to)
		if j >= to || tr.toks[j].kind != tokOp {
			break
		}
		m, ok = tr.skipTerm(j+1, to, depth)
	}
	return m, ok
}

// saute un terme : valeur directe, liste ou groupe, ou appel avec ses arguments
func (tr *translator) skipTerm(m, to, depth int) (int, bool) {
	m = tr.skipBlank(m, to)
	if m >= to || depth > 200 {
		return m, false
	}
	t := tr.toks[m]
	switch t.kind {
	case tokOpen:
		if tr.match[m] >= to {
			return m, false
		}
		return tr.match[m] + 1, true
	case tokQuoted, tokVar:
		return m + 1, true
	case tokOp:
		if t.s == "-" { // moins unaire
			return tr.skipTerm(m+1, to, depth+1)
		}
	case tokWord:
		up := t.name()
		if _, num := numberLiteral(up); num {
			return m + 1, true
		}
		n, ok := tr.arityOf(up)
		if !ok {
			return m, false
		}
		m++
		for ; n > 0 && ok; n-- {
			m, ok = tr.skipExpr(m, to, depth+1)
		}
		return m, ok
	}
	return m, false
}

// marque comme code les listes d'instructions de la primitive fr placee en k, en
// suivant ses arguments un par un. au moindre doute sur leur decoupage, rien n'est
// marque
func (tr *translator) markCodeLists(k, to int, fr string) {
	m := k + 1
	// la liste attendue en m est du code ; avance m derriere elle
	codeList := func() bool {
		l, ok := tr.listAt(m, to)
		if ok {
			tr.code[l] = true
			m = tr.match[l] + 1
		}
		return ok
	}
	expr := func() bool {
		var ok bool
		m, ok = tr.skipExpr(m, to, 0)
		return ok
	}
	switch fr {
	case "SCENE", "SIVRAI", "SIFAUX", "EXEC", "EXECRESULTAT":
		codeList()
	case "TANTQUE": // [condition] [instructions]
		if codeList() {
			codeList()
		}
	case "REPETE", "PIEGE", "DEMANDE", "APPLIQUE", "FILTRE", "REDUIS":
		// un argument (nombre, etiquette, tortues, liste a parcourir), puis le bloc
		if expr() {
			codeList()
		}
	case "REPETEPOUR": // [var debut fin pas] reste une donnee, puis le bloc
		if l, ok := tr.listAt(m, to); ok {
			m = tr.match[l] + 1
			codeList()
		}
	case "SI", "SISINON": // predicat [alors] puis [sinon] eventuel
		if expr() && codeList() {
			codeList()
		}
	case "POURCHAQUE":
		// liste [gabarit], ou forme XLogo : "var liste_ou_mot [commande]
		first := tr.skipBlank(m, to)
		if !expr() {
			return
		}
		if tr.toks[first].kind == tokQuoted && !expr() {
			return
		}
		codeList()
	case "DEFINIS":
		// "nom [ [params] [ligne1] [ligne2]... ] : les lignes sont du code ;
		// "nom [params] [corps] : le corps
		if !expr() {
			return
		}
		l, ok := tr.listAt(m, to)
		if !ok {
			return
		}
		if subs := tr.sublists(l); len(subs) > 0 && tr.skipBlank(l+1, to) == subs[0] {
			for _, s := range subs[1:] {
				tr.code[s] = true
			}
			return
		}
		m = tr.match[l] + 1
		codeList()
	}
}

// les sous-listes directes de la liste ouverte en k
func (tr *translator) sublists(k int) []int {
	var subs []int
	for m := k + 1; m < tr.match[k] && m < len(tr.toks); m++ {
		if tr.toks[m].kind == tokOpen {
			if tr.toks[m].s == "[" {
				subs = append(subs, m)
			}
			m = tr.match[m]
		}
	}
	return subs
}
