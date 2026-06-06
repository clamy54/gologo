package logo

import (
	"fmt"
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
)

// un element de code source, parfois imbrique (DList/DGroup)
type Datum struct {
	Kind  DatumKind
	Num   float64
	Text  string
	List  []Datum
	Unary bool // DOp '-' : moins unaire (negation) plutot que soustraction
}

// un Datum sous forme source (pour afficher des listes)
func (d Datum) String() string {
	switch d.Kind {
	case DNumber:
		return formatNumber(d.Num)
	case DWord, DSymbol, DOp:
		return d.Text
	case DVarRef:
		return ":" + d.Text
	case DList, DGroup:
		parts := make([]string, len(d.List))
		for i, e := range d.List {
			parts[i] = e.String()
		}
		// entre crochets, la forme standard d'une liste
		return "[" + strings.Join(parts, " ") + "]"
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
	var b strings.Builder
	rs := []rune(src)
	for i := 0; i < len(rs); i++ {
		if rs[i] == '!' { // MO5 : continuation seulement juste devant un saut de ligne
			j := i + 1
			if j < len(rs) && rs[j] == '\r' {
				j++
			}
			if j < len(rs) && rs[j] == '\n' {
				continue // on saute le '!', le saut de ligne reste (fait office d'espace)
			}
		}
		if rs[i] == '~' { // UCBLogo/MSWLogo : tilde de continuation (espaces toleres avant le saut)
			j := i + 1
			for j < len(rs) && (rs[j] == ' ' || rs[j] == '\t') {
				j++
			}
			if j < len(rs) && rs[j] == '\r' {
				j++
			}
			if j < len(rs) && rs[j] == '\n' {
				i = j - 1 // saute le '~' et les espaces, le saut de ligne reste
				continue
			}
		}
		b.WriteRune(rs[i])
	}
	return b.String()
}

type reader struct {
	src []rune
	pos int
}

func isSpace(c rune) bool { return c == ' ' || c == '\t' || c == '\r' || c == '\n' }

// caracteres qui terminent un mot non quote (';' demarre un commentaire)
func isDelim(c rune) bool {
	return c == 0 || isSpace(c) || c == '[' || c == ']' || c == '(' || c == ')' || c == ';'
}

// lit une suite de Datum jusqu'au terminateur term (0 = fin de source, ']' pour une
// liste, ')' pour un groupe)
func (r *reader) readSeq(term rune) ([]Datum, error) {
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
		case c == ']' || c == ')':
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
			if n, err := strconv.ParseFloat(w, 64); err == nil {
				out = append(out, Datum{Kind: DNumber, Num: -n, Text: "-" + w})
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
			w := r.readWord()
			if n, err := strconv.ParseFloat(w, 64); err == nil {
				out = append(out, Datum{Kind: DNumber, Num: n, Text: w})
			} else {
				out = append(out, Datum{Kind: DSymbol, Text: w})
			}
		}
	}
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
			break // '<' colle a #/b (alteration musicale) ne coupe pas : l'operateur < veut des espaces
		}
		b.WriteRune(c)
		r.pos++
	}
	return b.String()
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
	case ' ', '\t', '\n', '\r', '[', '(':
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
		return NumberValue(d.Num), nil
	case DWord, DSymbol, DOp:
		return WordValue(d.Text), nil
	case DVarRef:
		return WordValue(":" + d.Text), nil
	case DList, DGroup:
		return ListValue(d.List), nil
	}
	return Value{}, fmt.Errorf("OBJET INATTENDU DANS LA LISTE")
}

// convertit une Value en Datum pour (re)construire des listes
func valueToDatum(v Value) Datum {
	switch v.Kind {
	case KNumber:
		return Datum{Kind: DNumber, Num: v.Num, Text: formatNumber(v.Num)}
	case KWord:
		return Datum{Kind: DWord, Text: v.Word}
	case KBool:
		return Datum{Kind: DSymbol, Text: v.String()}
	case KList:
		return Datum{Kind: DList, List: v.List}
	}
	return Datum{Kind: DWord, Text: ""}
}
