package logo

import (
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

// type d'un element du flux source ou d'une liste
type DatumKind int

const (
	DNumber DatumKind = iota // 42, 3.14
	DWord                    // "abc  -> mot (sans le " initial)
	DVarRef                  // :x    -> reference de variable
	DSymbol                  // abc   -> nom a invoquer, ou mot en contexte donnee
	DList                    // [ ... ]
	DGroup                   // ( ... ) : liste d'instructions ou groupement, selon le contenu
	DOp                      // operateur infixe : + - * / = < > <= >= <>
	DArray                   // { ... } : tableau litteral (eventuellement { ... }@origine)
	DBool                    // booleen range dans une liste a l'execution (Text = VRAI ou FAUX)
)

// un element de code source, parfois imbrique (DList/DGroup/DArray)
type Datum struct {
	Kind   DatumKind
	Num    float64
	Text   string
	List   []Datum
	Unary  bool     // DOp '-' : moins unaire (negation) plutot que soustraction
	Origin int      // DArray : indice de la 1re case (origine), 1 par defaut
	Arr    *Array   // DArray issu d'un tableau a l'execution (garde l'identite)
	Big    *big.Int // DNumber : entier exact quand le litteral deborde 2^53
}

// plafond d'imbrication des donnees parcourues recursivement (affichage, copie,
// sauvegarde). une liste ou un tableau peut s'emboiter sans limite a l'execution
// (DONNE "L LISTE :L en boucle) : au-dela, la recursion Go ferait deborder la pile
const maxDataDepth = 10000

// un Datum tel qu'on l'affiche dans une liste (ECRIS/MONTRE). ce n'est PAS une forme
// relisible : les mots perdent leur " et les groupes ( ) deviennent des crochets.
// pour du source re-executable, voir source.go
func (d Datum) String() string { return d.str(0) }

func (d Datum) str(depth int) string {
	switch d.Kind {
	case DNumber:
		if d.Big != nil {
			return d.Big.String()
		}
		return formatNumber(d.Num)
	case DWord, DSymbol, DOp, DBool:
		return d.Text
	case DVarRef:
		return ":" + d.Text
	case DList, DGroup:
		if depth >= maxDataDepth {
			return "[...]" // trop profond : on coupe l'affichage plutot que la pile
		}
		parts := make([]string, len(d.List))
		for i, e := range d.List {
			parts[i] = e.str(depth + 1)
		}
		// entre crochets, la forme standard d'une liste
		return "[" + strings.Join(parts, " ") + "]"
	case DArray:
		if d.Arr != nil { // tableau deja construit (mis dans une liste a l'execution)
			return d.Arr.str(depth)
		}
		if depth >= maxDataDepth {
			return "{...}"
		}
		parts := make([]string, len(d.List))
		for i, e := range d.List {
			parts[i] = e.str(depth + 1)
		}
		s := "{" + strings.Join(parts, " ") + "}"
		if d.Origin != 1 {
			s += "@" + strconv.Itoa(d.Origin)
		}
		return s
	}
	return ""
}

// transforme une source Logo en suite de Datum (avec imbrications)
func Read(src string) ([]Datum, error) {
	r := &reader{src: []rune(joinContinuations(src))}
	return r.readSeq(0)
}

// retire les caracteres de continuation en fin de ligne : '!' (manuel MO5) et '~'
// (UCBLogo/MSWLogo, ex. "REPETE 6 ~" suivi de la liste sur les lignes suivantes). le
// caractere disparait, la ligne se prolonge, le saut de ligne reste (separateur). le
// '!' doit etre colle au saut, le '~' tolere des espaces avant. ailleurs, conserve
func joinContinuations(src string) string {
	s, _ := stripContinuations(src)
	return s
}

// une coupe faite par stripContinuations : a partir de l'octet `at` du texte
// nettoye, il manque `removed` octets par rapport au texte d'origine
type contCut struct{ at, removed int }

// le texte sans ses continuations, et la liste des coupes, qui permet de retrouver
// la position d'origine d'un octet du texte nettoye (cf lexSource). tout ce qui
// decoupe du source doit passer par ici pour voir le meme texte que le lecteur
func stripContinuations(src string) (string, []contCut) {
	if !strings.ContainsAny(src, "!~") {
		return src, nil
	}
	var b strings.Builder
	var cuts []contCut
	removed := 0
	n := len(src)
	for i := 0; i < n; i++ {
		c := src[i]
		if c == '!' { // MO5 : continuation seulement juste devant un saut de ligne
			j := i + 1
			if j < n && src[j] == '\r' {
				j++
			}
			if j < n && src[j] == '\n' {
				removed++ // on saute le '!', le saut de ligne reste (fait office d'espace)
				cuts = append(cuts, contCut{b.Len(), removed})
				continue
			}
		}
		if c == '~' { // UCBLogo/MSWLogo : tilde de continuation (espaces toleres avant le saut)
			j := i + 1
			for j < n && (src[j] == ' ' || src[j] == '\t') {
				j++
			}
			if j < n && src[j] == '\r' {
				j++
			}
			if j < n && src[j] == '\n' {
				removed += j - i // saute le '~' et les espaces, le saut de ligne reste
				cuts = append(cuts, contCut{b.Len(), removed})
				i = j - 1
				continue
			}
		}
		b.WriteByte(c)
	}
	return b.String(), cuts
}

type reader struct {
	src   []rune
	pos   int
	depth int // profondeur d'imbrication [ ] { } ( ), garde anti-debordement de pile
}

// limite d'imbrication a la lecture : tres au-dela de tout code reel, mais sous le
// seuil ou la recursion Go ferait deborder la pile sur une entree pathologique
const maxReadDepth = 4000

func isSpace(c rune) bool { return c == ' ' || c == '\t' || c == '\r' || c == '\n' }

// caracteres qui terminent un mot non quote (';' demarre un commentaire)
func isDelim(c rune) bool {
	return c == 0 || isSpace(c) || c == '[' || c == ']' || c == '(' || c == ')' ||
		c == '{' || c == '}' || c == ';'
}

// lit une suite de Datum jusqu'au terminateur term (0 = fin de source, ']' pour une
// liste, ')' pour un groupe)
func (r *reader) readSeq(term rune) ([]Datum, error) {
	if r.depth++; r.depth > maxReadDepth {
		r.depth--
		return nil, fmt.Errorf("IMBRICATION TROP PROFONDE")
	}
	defer func() { r.depth-- }()
	var out []Datum
	for {
		for r.pos < len(r.src) && isSpace(r.src[r.pos]) {
			r.pos++
		}
		if r.pos >= len(r.src) {
			if term != 0 {
				return nil, fmt.Errorf("%c manquant", term)
			}
			return out, nil
		}
		c := r.src[r.pos]
		switch {
		case c == ';' || c == '#': // commentaire jusqu'a la fin de ligne
			// on est en tete de token (la boucle a saute espaces et sauts de ligne) :
			// un '#' en debut de ligne/source ou apres une espace demarre un commentaire.
			// par contre '#' n'est pas un delimiteur de mot, donc colle a une note
			// (DO#, RE<#) il reste dans le mot et ne demarre rien
			for r.pos < len(r.src) && r.src[r.pos] != '\n' {
				r.pos++
			}
		case c == ']' || c == ')' || c == '}':
			if c != term {
				return nil, fmt.Errorf("%c inattendu", c)
			}
			r.pos++ // consomme le terminateur
			return out, nil
		case c == '[':
			r.pos++
			inner, err := r.readSeq(']')
			if err != nil {
				return nil, err
			}
			out = append(out, Datum{Kind: DList, List: inner})
		case c == '{':
			r.pos++
			inner, err := r.readSeq('}')
			if err != nil {
				return nil, err
			}
			// origine optionnelle juste apres l'accolade : { a b c }@0
			origin := 1
			if r.pos < len(r.src) && r.src[r.pos] == '@' {
				r.pos++
				w := ""
				if r.pos < len(r.src) && r.src[r.pos] == '-' { // origine negative : {a b}@-2
					w = "-"
					r.pos++
				}
				w += r.readWord()
				n, err := strconv.Atoi(w)
				if err != nil || n < -maxArrayOrigin || n > maxArrayOrigin {
					return nil, fmt.Errorf("ORIGINE DE TABLEAU INVALIDE : %s", w)
				}
				origin = n
			}
			out = append(out, Datum{Kind: DArray, List: inner, Origin: origin})
		case c == '(':
			r.pos++
			inner, err := r.readSeq(')')
			if err != nil {
				return nil, err
			}
			out = append(out, Datum{Kind: DGroup, List: inner})
		case c == '-' && r.pos+1 < len(r.src) && (isDigit(r.src[r.pos+1]) || r.src[r.pos+1] == '.') && minusIsLiteral(r.src, r.pos):
			// moins colle a un chiffre : nombre negatif. colle a une valeur precedente
			// (":p-1") ce serait une soustraction
			r.pos++
			w := r.readWord()
			if d, ok := numberLiteral("-" + w); ok {
				out = append(out, d)
			} else {
				out = append(out, Datum{Kind: DSymbol, Text: "-" + w})
			}
		case c == '-' && minusIsUnary(r.src, r.pos):
			// moins unaire colle a un operande non numerique ("-:x", "-(...)", "-FOO")
			// en contexte unaire (espace/debut/ouvrante/operateur a gauche). on marque
			// Unary pour que l'expression ne l'avale pas comme soustraction : "SETXY
			// :H -:H" = deux operandes :H et -(:H), pas ":H - :H"
			r.pos++
			out = append(out, Datum{Kind: DOp, Text: "-", Unary: true})
		case isInfixOp(c):
			r.pos++
			if r.pos < len(r.src) {
				two := string(c) + string(r.src[r.pos])
				if two == "<=" || two == ">=" || two == "<>" {
					r.pos++
					out = append(out, Datum{Kind: DOp, Text: two})
					continue
				}
			}
			out = append(out, Datum{Kind: DOp, Text: string(c)})
		case c == '"':
			r.pos++
			out = append(out, Datum{Kind: DWord, Text: r.readQuoted()})
		case c == ':':
			r.pos++
			out = append(out, Datum{Kind: DVarRef, Text: r.readWord()})
		default:
			start := r.pos
			w := r.readWord()
			if r.pos == start {
				// rien n'a ete consomme (octet nul : un delimiteur que personne ne
				// mange). sans cette erreur on relirait le meme caractere sans fin
				return nil, errCaractereNul
			}
			if d, ok := numberLiteral(w); ok {
				out = append(out, d)
			} else {
				out = append(out, Datum{Kind: DSymbol, Text: w})
			}
		}
	}
}

// un octet nul dans le source : fichier binaire ou abime, pas du Logo
var errCaractereNul = fmt.Errorf("CARACTERE NUL DANS LE PROGRAMME")

// w a-t-il la tete d'un nombre decimal ? seulement des chiffres, un point, un
// exposant et des signes, avec au moins un chiffre. ParseFloat tout seul accepte
// aussi "Inf", "NaN" ou l'hexa flottant, qui deviendraient des nombres par surprise
// (un mot ou une procedure nommes INF ou NAN)
func looksNumeric(w string) bool {
	digit := false
	for k := 0; k < len(w); k++ {
		switch c := w[k]; {
		case c >= '0' && c <= '9':
			digit = true
		case c == '.' || c == 'e' || c == 'E' || c == '+' || c == '-':
		default:
			return false
		}
	}
	return digit
}

// valeur d'un mot qui s'ecrit comme un nombre fini (ok=false sinon)
func parseNumber(w string) (float64, bool) {
	if !looksNumeric(w) {
		return 0, false
	}
	n, err := strconv.ParseFloat(w, 64)
	return n, err == nil
}

// lit w comme litteral numerique. un entier trop grand pour float64 (plus de 308
// chiffres) reste un entier exact au lieu de retomber en simple symbole
func numberLiteral(w string) (Datum, bool) {
	if !looksNumeric(w) {
		return Datum{}, false
	}
	n, err := strconv.ParseFloat(w, 64)
	if err == nil {
		return numberDatum(n, w), true
	}
	if errors.Is(err, strconv.ErrRange) {
		if b, ok := bigIntLiteral(w); ok {
			return Datum{Kind: DNumber, Big: b, Text: w}, true
		}
	}
	return Datum{}, false
}

// lit un nom (symbole ou variable apres ":") : s'arrete aux delimiteurs et aux
// operateurs infixes, pour que ":x+1" donne :x + 1
func (r *reader) readWord() string { return r.readToken(true) }

// lit un mot apres '"' : s'arrete seulement aux vrais delimiteurs, pas aux operateurs
// ('"-----' ou '"a-b' sont des mots entiers)
func (r *reader) readQuoted() string { return r.readToken(false) }

// lit un mot ; breakOnOp coupe aussi sur les operateurs infixes. '\' protege le
// caractere suivant, un espace protege termine le mot ('"label:\ cmd')
func (r *reader) readToken(breakOnOp bool) string {
	var b strings.Builder
	for r.pos < len(r.src) {
		c := r.src[r.pos]
		if (c == '\\' || c == '$') && r.pos+1 < len(r.src) { // '$' = echappement MO5, '\' le moderne
			nc := r.src[r.pos+1]
			b.WriteRune(nc)
			r.pos += 2
			if nc == ' ' || nc == '\t' || nc == '\n' || nc == '\r' {
				break
			}
			continue
		}
		if isDelim(c) {
			break
		}
		if breakOnOp && isInfixOp(c) && !altSuffix(r.src, r.pos) {
			// exposant signe d'un nombre : "1e-3", "2E+5" -> le + ou - fait partie du nombre
			if (c == '+' || c == '-') && expSignPending(b.String()) {
				b.WriteRune(c)
				r.pos++
				continue
			}
			break // '<' colle a #/b (alteration musicale) ne coupe pas : l'operateur < veut des espaces
		}
		b.WriteRune(c)
		r.pos++
	}
	return b.String()
}

// vrai si le mot accumule est un debut de nombre en notation scientifique qui
// attend le signe de son exposant : chiffres/point puis un e/E final ("1e", "2.5E")
func expSignPending(s string) bool {
	if len(s) < 2 || (s[len(s)-1] != 'e' && s[len(s)-1] != 'E') {
		return false
	}
	if prev := s[len(s)-2]; prev < '0' || prev > '9' { // un chiffre juste avant le e
		return false
	}
	c0 := s[0]
	return (c0 >= '0' && c0 <= '9') || c0 == '.'
}

// vrai si src[pos] est un '<' suivi d'une alteration musicale (#, b, B), ex. "RE<#".
// l'operateur '<' veut des espaces, donc un '<' colle est forcement une alteration
func altSuffix(src []rune, pos int) bool {
	if src[pos] != '<' || pos+1 >= len(src) {
		return false
	}
	switch src[pos+1] {
	case '#', 'b', 'B':
		return true
	}
	return false
}

func isInfixOp(c rune) bool {
	switch c {
	case '+', '-', '*', '/', '=', '<', '>':
		return true
	}
	return false
}

// vrai si le '-' en pos introduit un nombre negatif, faux si c'est une soustraction
// (colle a une valeur precedente, ex. ":p-1")
func minusIsLiteral(src []rune, pos int) bool {
	if pos == 0 {
		return true
	}
	switch p := src[pos-1]; p {
	case ' ', '\t', '\n', '\r', '[', '(', '{':
		return true
	default:
		return isInfixOp(p)
	}
}

// vrai si le '-' en pos est un moins unaire (negation) et pas une soustraction,
// d'apres l'espacement facon UCBLogo : contexte gauche unaire (debut, espace,
// ouvrante, autre operateur) ET colle a un operande a droite. ainsi "SETXY :H -:H"
// donne deux operandes :H et -(:H), alors que ":H - :H" ou ":H-:H" sont des
// soustractions. le nombre negatif litteral ("-5") a deja ete regle avant.
// ces quelques regles d'espacement coutent plus de cheveux blancs qu'elles n'en ont l'air
func minusIsUnary(src []rune, pos int) bool {
	if !minusIsLiteral(src, pos) { // contexte gauche non unaire -> soustraction
		return false
	}
	n := pos + 1
	if n >= len(src) {
		return false
	}
	c := src[n] // un operande doit etre colle a droite (ni espace, ni fermante)
	return !isSpace(c) && c != ')' && c != ']' && c != ';' && !isInfixOp(c)
}

func isDigit(c rune) bool { return c >= '0' && c <= '9' }

// convertit un Datum en Value (en contexte donnee) : symbole nu -> mot,
// sous-liste/groupe -> liste
func datumToValue(d Datum) (Value, error) {
	switch d.Kind {
	case DNumber:
		if d.Big != nil {
			return IntValue(d.Big), nil
		}
		return NumberValue(d.Num), nil
	case DWord, DSymbol, DOp:
		return WordValue(d.Text), nil
	case DBool:
		return BoolValue(d.Text == "VRAI"), nil
	case DVarRef:
		return WordValue(":" + d.Text), nil
	case DList, DGroup:
		return ListValue(d.List), nil
	case DArray:
		if d.Arr != nil { // deja construit : on garde le meme tableau (identite)
			return ArrayValue(d.Arr), nil
		}
		// litteral { ... } : on fabrique un tableau frais a chaque evaluation
		items := make([]Value, len(d.List))
		for i, e := range d.List {
			v, err := datumToValue(e)
			if err != nil {
				return Value{}, err
			}
			items[i] = v
		}
		return ArrayValue(&Array{Items: items, Origin: d.Origin}), nil
	}
	return Value{}, fmt.Errorf("OBJET INATTENDU DANS LA LISTE")
}

// convertit une Value en Datum pour (re)construire des listes
func valueToDatum(v Value) Datum {
	switch v.Kind {
	case KNumber:
		return Datum{Kind: DNumber, Num: v.Num, Text: formatNumber(v.Num)}
	case KInt:
		return Datum{Kind: DNumber, Big: v.Int, Text: v.Int.String()}
	case KWord:
		return Datum{Kind: DWord, Text: v.Word}
	case KBool:
		// un vrai booleen, pas le mot VRAI : sorti de la liste il redevient booleen
		return Datum{Kind: DBool, Text: v.String()}
	case KList:
		return Datum{Kind: DList, List: v.List}
	case KArray:
		return Datum{Kind: DArray, Arr: v.Arr, Origin: v.Arr.Origin}
	}
	return Datum{Kind: DWord, Text: ""}
}
