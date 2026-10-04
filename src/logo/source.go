package logo

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// forme SOURCE des valeurs et du code : ce que SAUVE, IM, IMNS et l'editeur ecrivent
// doit redonner les memes objets une fois relu. a ne pas confondre avec String(),
// qui sert a l'affichage (ECRIS) : la les mots perdent leur ", les groupes ( )
// deviennent des crochets et rien n'est protege.
//
// principe : tout Datum s'ecrit soit par un LITTERAL que le lecteur relit a
// l'identique, soit, quand il n'en a pas (mot a blanc interne, booleen, nombre non
// fini), par une EXPRESSION qui le reconstruit avec sa nature. une liste melange
// les deux : (PH [debut litteral] (LISTE expression) [suite litterale]). rien n'est
// jamais ecrit "au mieux" : ce qui ne peut pas s'ecrire fidelement est refuse.
//
// seule l'identite des tableaux n'est pas conservee : deux variables qui partagent
// un meme tableau en retrouvent chacune une copie independante

// ce qui n'a aucune ecriture source fidele
var errPasDeSource = errors.New("IMPOSSIBLE A ECRIRE EN SOURCE LOGO")

// caracteres que le lecteur prend pour des delimiteurs ou des echappements : il
// faut un '\' devant pour qu'ils restent dans le mot
func needsEscape(r rune) bool {
	switch r {
	case 0, ' ', '\t', '\n', '\r', '[', ']', '(', ')', '{', '}', ';', '\\', '$':
		return true
	}
	return false
}

func isBlank(r rune) bool { return r == ' ' || r == '\t' || r == '\n' || r == '\r' }

// ecrit les caracteres d'un mot en protegeant ce qui doit l'etre. ops = proteger
// aussi les operateurs infixes (nom nu ou variable : le lecteur y coupe le mot).
// rend false si le mot ne tient pas dans un seul litteral : le lecteur termine le
// mot juste apres un blanc protege, donc un blanc ailleurs qu'en fin ne se relit pas
func writeEscaped(b *strings.Builder, w string, ops bool) bool {
	rs := []rune(w)
	for k, r := range rs {
		if isBlank(r) && k < len(rs)-1 {
			return false
		}
		switch {
		case needsEscape(r):
			b.WriteByte('\\')
		case ops && isInfixOp(r) && !altSuffix(rs, k):
			b.WriteByte('\\')
		case ops && k == 0 && (r == '"' || r == ':' || r == '#'):
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return true
}

// litteral "mot (ok=false s'il contient un blanc ailleurs qu'en derniere position)
func wordLiteral(w string) (string, bool) {
	var b strings.Builder
	b.WriteByte('"')
	ok := writeEscaped(&b, w, false)
	return b.String(), ok
}

// nom nu (procedure, parametre, mot d'une liste) avec ses echappements. ok=false
// s'il ne peut pas se relire comme un nom : vide, a blanc interne, ou qui serait
// pris pour un nombre
func nameLiteral(w string) (string, bool) {
	if w == "" {
		return "", false
	}
	if _, num := numberLiteral(w); num {
		return "", false
	}
	var b strings.Builder
	ok := writeEscaped(&b, w, true)
	return b.String(), ok
}

// name peut-il nommer une procedure ou un parametre ? il doit pouvoir s'ecrire et
// se relire comme un nom nu (au besoin avec des echappements)
func validName(name string) bool {
	_, ok := nameLiteral(name)
	return ok
}

// un mot sous forme d'expression relisible. presque toujours un simple litteral ;
// un mot a blancs internes est recolle par MOT, morceau par morceau (chaque morceau
// finit sur son blanc protege)
func wordSource(w string) string {
	if lit, ok := wordLiteral(w); ok {
		return lit
	}
	var parts []string
	rs := []rune(w)
	start := 0
	for k, r := range rs {
		if isBlank(r) {
			lit, _ := wordLiteral(string(rs[start : k+1]))
			parts = append(parts, lit)
			start = k + 1
		}
	}
	if start < len(rs) {
		lit, _ := wordLiteral(string(rs[start:]))
		parts = append(parts, lit)
	}
	return "(MOT " + strings.Join(parts, " ") + ")"
}

// un nombre sous forme d'expression. l'infini et NaN n'ont pas de litteral : on
// ecrit le calcul qui les redonne
func numberSource(n float64) string {
	switch {
	case math.IsNaN(n):
		return "((1e308 * 10) - (1e308 * 10))"
	case math.IsInf(n, 1):
		return "(1e308 * 10)"
	case math.IsInf(n, -1):
		return "(-1e308 * 10)"
	}
	return formatNumber(n)
}

func boolSource(b bool) string {
	if b {
		return "VRAI" // la primitive : on retrouve un vrai booleen
	}
	return "FAUX"
}

// une suite de Datum en litteraux separes par une espace, ou ok=false des qu'un
// element n'en a pas. un moins unaire reste colle a son operande ("-:X"), sinon il
// serait relu comme une soustraction
func seqLiteral(ds []Datum) (string, bool) {
	var b strings.Builder
	glue := true // pas d'espace avant le 1er element, ni apres un moins unaire
	for _, d := range ds {
		s, ok := datumLiteral(d)
		if !ok {
			return "", false
		}
		if !glue {
			b.WriteByte(' ')
		}
		b.WriteString(s)
		glue = d.Kind == DOp && d.Unary
	}
	return b.String(), true
}

// le litteral d'un Datum : le texte que le lecteur relit en ce meme Datum.
// ok=false s'il n'en a pas (mot a blanc interne, booleen, nombre non fini, ou un
// conteneur qui en renferme un)
func datumLiteral(d Datum) (string, bool) {
	switch d.Kind {
	case DNumber:
		if d.Big != nil {
			return d.Big.String(), true
		}
		if math.IsNaN(d.Num) || math.IsInf(d.Num, 0) {
			return "", false
		}
		return formatNumber(d.Num), true
	case DWord:
		return wordLiteral(d.Text)
	case DSymbol:
		return nameLiteral(d.Text)
	case DVarRef:
		if d.Text == "" {
			return ":", true
		}
		s, ok := nameLiteral(d.Text)
		return ":" + s, ok
	case DOp:
		return d.Text, true
	case DList:
		s, ok := seqLiteral(d.List)
		return "[" + s + "]", ok
	case DGroup:
		s, ok := seqLiteral(d.List)
		return "(" + s + ")", ok
	case DArray:
		items, origin := arrayDatums(d)
		s, ok := seqLiteral(items)
		s = "{" + s + "}"
		if origin != 1 {
			s += "@" + strconv.Itoa(origin)
		}
		return s, ok
	}
	return "", false // DBool : pas de litteral, VRAI nu serait relu comme un nom
}

// les cases d'un tableau vues en Datum, et son origine (tableau litteral du source
// ou tableau construit a l'execution)
func arrayDatums(d Datum) ([]Datum, int) {
	if d.Arr == nil {
		return d.List, d.Origin
	}
	items := make([]Datum, len(d.Arr.Items))
	for k, v := range d.Arr.Items {
		items[k] = valueToDatum(v)
	}
	return items, d.Arr.Origin
}

// une liste sous forme d'expression : le litteral [ ... ] quand tous ses elements
// en ont un (le cas de loin le plus courant). sinon (PH ...) recolle des tranches :
// les elements a litteral restent tels quels entre crochets, noms nus, variables et
// groupes compris, et chaque element sans litteral est reconstruit par (LISTE expr)
func listSource(items []Datum) (string, error) {
	if s, ok := seqLiteral(items); ok {
		return "[" + s + "]", nil
	}
	var parts []string
	run := 0 // debut de la tranche litterale en cours
	flush := func(end int) {
		if end > run {
			s, _ := seqLiteral(items[run:end])
			parts = append(parts, "["+s+"]")
		}
	}
	for k, d := range items {
		if _, lit := datumLiteral(d); lit {
			continue
		}
		flush(k)
		e, err := datumExpr(d)
		if err != nil {
			return "", err
		}
		parts = append(parts, "(LISTE "+e+")")
		run = k + 1
	}
	flush(len(items))
	return "(PH " + strings.Join(parts, " ") + ")", nil
}

// expression dont la valeur, rangee dans une liste, redonne exactement le Datum d
func datumExpr(d Datum) (string, error) {
	switch d.Kind {
	case DWord:
		return wordSource(d.Text), nil
	case DNumber:
		if d.Big != nil {
			return d.Big.String(), nil
		}
		return numberSource(d.Num), nil
	case DBool:
		return boolSource(d.Text == "VRAI"), nil
	case DList:
		return listSource(d.List)
	case DArray:
		items, origin := arrayDatums(d)
		return arrayExpr(items, origin)
	}
	// un nom, une variable ou un groupe sans litteral : rien ne les reconstruit avec
	// leur nature (le lecteur lui-meme ne peut pas en produire)
	return "", errPasDeSource
}

// expression qui reconstruit un tableau case par case (LISTEVERSTABLEAU relit
// chaque element comme le ferait un tableau litteral, booleens compris)
func arrayExpr(items []Datum, origin int) (string, error) {
	l, err := listSource(items)
	if err != nil {
		return "", err
	}
	return "(LISTEVERSTABLEAU " + l + " " + strconv.Itoa(origin) + ")", nil
}

// une valeur sous forme d'expression relisible : ce qu'on ecrit derriere DONNE "nom.
// l'appelant a deja ecarte les valeurs emboitees trop profond (tooDeep)
func valueSource(v Value) (string, error) {
	switch v.Kind {
	case KNumber:
		return numberSource(v.Num), nil
	case KInt:
		return v.Int.String(), nil
	case KWord:
		return wordSource(v.Word), nil
	case KBool:
		return boolSource(v.Bool), nil
	case KList:
		return listSource(v.List)
	case KArray:
		d := valueToDatum(v)
		if s, ok := datumLiteral(d); ok {
			return s, nil
		}
		items, origin := arrayDatums(d)
		return arrayExpr(items, origin)
	}
	return "", errPasDeSource
}

// une ligne de source finie par '!' ou '~' serait prise pour une continuation a la
// relecture (cf stripContinuations) et perdrait ce caractere : un commentaire vide
// derriere le desamorce
func guardLineEnd(line string) string {
	if strings.HasSuffix(line, "!") || strings.HasSuffix(line, "~") {
		return line + " ;"
	}
	return line
}

// l'instruction DONNE qui recree la variable name avec la valeur v
func donneSource(name string, v Value) (string, error) {
	if tooDeep(v) {
		return "", errTropProfond
	}
	s, err := valueSource(v)
	if err != nil {
		return "", err
	}
	return guardLineEnd("DONNE " + wordSource(name) + " " + s), nil
}

// la premiere ligne "POUR nom :p1 ..." d'une procedure (noms echappes au besoin)
func procTitle(p *userProc) string {
	name, _ := nameLiteral(p.name)
	t := "POUR " + name
	for _, par := range p.params {
		s, _ := nameLiteral(par)
		t += " :" + s
	}
	return t
}

// texte source de la proc : la saisie brute de l'editeur si on l'a, sinon on la
// rebatit. la forme POUR ... FIN quand tout le corps a un litteral ; sinon (corps
// bati a l'execution par DEFINIS, avec un mot a blanc interne, un booleen...) la
// forme DEFINIS "nom [params] expression, qui reconstruit le corps element par
// element. dans les deux cas la procedure relue a le meme corps
func (p *userProc) sourceText() (string, error) {
	if p.text != "" {
		return p.text, nil
	}
	if tooDeep(ListValue(p.body)) {
		return "", errTropProfond
	}
	if body, ok := seqLiteral(p.body); ok && !hasProcEnd(p.body) {
		return procTitle(p) + "\n" + guardLineEnd(body) + "\nFIN", nil
	}
	params := make([]string, len(p.params))
	for k, par := range p.params {
		params[k], _ = nameLiteral(par)
	}
	body, err := listSource(p.body)
	if err != nil {
		return "", fmt.Errorf("%s : %w", p.name, err)
	}
	return guardLineEnd("DEFINIS " + wordSource(p.name) + " [" + strings.Join(params, " ") + "] " + body), nil
}

// le corps contient-il, a son niveau, un FIN/END nu ? ecrit entre POUR et FIN il
// terminerait la definition avant l'heure
func hasProcEnd(body []Datum) bool {
	for _, d := range body {
		if d.Kind == DSymbol && isProcEnd(d.Text) {
			return true
		}
	}
	return false
}

// une liste ou un tableau est-il emboite au-dela de maxDataDepth ? parcours sans
// recursion, justement parce qu'une telle valeur ferait deborder la pile
func tooDeep(v Value) bool {
	type item struct {
		v     Value
		depth int
	}
	seen := map[*Array]bool{}
	stack := []item{{v, 0}}
	for len(stack) > 0 {
		it := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if it.depth >= maxDataDepth {
			return true
		}
		switch it.v.Kind {
		case KArray:
			if seen[it.v.Arr] {
				continue
			}
			seen[it.v.Arr] = true
			for _, c := range it.v.Arr.Items {
				if c.Kind == KArray || c.Kind == KList {
					stack = append(stack, item{c, it.depth + 1})
				}
			}
		case KList:
			for _, d := range it.v.List {
				switch {
				case d.Kind == DArray && d.Arr != nil:
					stack = append(stack, item{ArrayValue(d.Arr), it.depth + 1})
				case d.Kind == DList || d.Kind == DGroup || d.Kind == DArray:
					stack = append(stack, item{ListValue(d.List), it.depth + 1})
				}
			}
		}
	}
	return false
}
