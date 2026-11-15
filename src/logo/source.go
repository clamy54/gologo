package logo

import (
	"errors"
	"fmt"
	"math"
	"math/big"
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
		case ops && isInfixOp(r) && (k == 0 || !altSuffix(rs, k)): // "RE<#" mais pas "<B" en tete
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
	if err := checkSource(v); err != nil {
		return "", err
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
	if err := checkSource(ListValue(p.body)); err != nil {
		return "", err
	}
	// un corps qui commence par :x serait relu comme un parametre de plus
	lead := len(p.body) > 0 && p.body[0].Kind == DVarRef
	if body, ok := seqLiteral(p.body); ok && !hasProcEnd(p.body) && !lead {
		return guardLineEnd(procTitle(p)) + "\n" + guardLineEnd(body) + "\nFIN", nil
	}
	params := make([]string, len(p.params))
	for k, par := range p.params {
		params[k], _ = nameLiteral(par)
		if strings.HasPrefix(par, ":") {
			params[k] = `\:` + params[k] // DEFINIS retire un ':' de tete
		}
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

// une liste ou un tableau est-il emboite au-dela de maxDataDepth ?
func tooDeep(v Value) bool {
	_, deep := measure(v)
	return deep
}

// avant d'ecrire v en source : ni trop emboite, ni plus gros que ce que RAMENE
// relit. refuse avant de batir le texte
func checkSource(v Value) error {
	size, deep := measure(v)
	if deep {
		return errTropProfond
	}
	if size > maxSourceBytes {
		return errTropGros
	}
	return nil
}

// un conteneur vu par measure : tranche de liste (cle = debut + longueur) ou tableau
type sliceKey struct {
	p *Datum
	n int
}

type node struct {
	key  any
	list []Datum
	arr  []Value
}

func listNode(l []Datum) node {
	if len(l) == 0 {
		return node{key: sliceKey{}}
	}
	return node{key: sliceKey{&l[0], len(l)}, list: l}
}

func valueNode(v Value) (node, bool) {
	switch {
	case v.Kind == KArray && v.Arr != nil:
		return node{key: v.Arr, arr: v.Arr.Items}, true
	case v.Kind == KList:
		return listNode(v.List), true
	}
	return node{}, false
}

func datumNode(d Datum) (node, bool) {
	switch {
	case d.Kind == DArray && d.Arr != nil:
		return node{key: d.Arr, arr: d.Arr.Items}, true
	case d.Kind == DList || d.Kind == DGroup || d.Kind == DArray:
		return listNode(d.List), true
	}
	return node{}, false
}

// minorant du nombre de chiffres decimaux de z (log10(2) arrondi par defaut)
func bigDigits(z *big.Int) int64 {
	if b := z.BitLen(); b > 1 {
		return int64(b-1)*30102/100000 + 1
	}
	return 1
}

// octets minimum du source d'un element non conteneur : un mot au moins ses
// caracteres, un nombre son ecriture normalisee (pas celle du texte lu)
func valueAtomSize(v Value) int64 {
	switch v.Kind {
	case KWord:
		return max(1, int64(len(v.Word)))
	case KNumber:
		return int64(len(numberSource(v.Num)))
	case KInt:
		return bigDigits(v.Int)
	case KBool:
		return 4
	}
	return 1
}

func datumAtomSize(d Datum) int64 {
	if d.Kind == DNumber {
		if d.Big != nil {
			return bigDigits(d.Big)
		}
		return int64(len(numberSource(d.Num)))
	}
	return max(1, int64(len(d.Text)))
}

func satAdd(a, b int64) int64 {
	if s := a + b; s < 1<<60 {
		return s
	}
	return 1 << 60
}

// emboitement et taille depliee de v (minorant des octets de son source : une
// sous-liste partagee s'ecrit autant de fois qu'elle apparait). memoise par objet :
// lineaire en nombre d'objets distincts, meme si chaque etage reprend deux fois le
// meme sous-objet. iteratif ; deep des que l'emboitement depasse maxDataDepth
func measure(v Value) (size int64, deep bool) {
	root, ok := valueNode(v)
	if !ok {
		return valueAtomSize(v), false
	}
	type frame struct {
		n       node
		k       int
		h, size int64
	}
	type shape struct{ h, size int64 }
	memo := map[any]shape{}
	busy := map[any]bool{root.key: true}
	stack := []*frame{{n: root, size: 2}}
	var res shape
	for len(stack) > 0 {
		f := stack[len(stack)-1]
		var c node
		var isNode bool
		var atom int64
		switch {
		case f.k < len(f.n.list):
			d := f.n.list[f.k]
			if c, isNode = datumNode(d); !isNode {
				atom = datumAtomSize(d)
			}
		case f.k < len(f.n.arr):
			if c, isNode = valueNode(f.n.arr[f.k]); !isNode {
				atom = valueAtomSize(f.n.arr[f.k])
			}
		default:
			stack = stack[:len(stack)-1]
			delete(busy, f.n.key)
			sh := shape{f.h + 1, f.size}
			memo[f.n.key] = sh
			if len(stack) == 0 {
				res = sh
				continue
			}
			p := stack[len(stack)-1]
			p.h, p.size = max(p.h, sh.h), satAdd(p.size, sh.size)
			continue
		}
		f.k++
		if !isNode {
			f.size = satAdd(f.size, atom)
			continue
		}
		if sh, ok := memo[c.key]; ok {
			f.h, f.size = max(f.h, sh.h), satAdd(f.size, sh.size)
			continue
		}
		if busy[c.key] || len(stack) >= maxDataDepth {
			return 0, true
		}
		busy[c.key] = true
		stack = append(stack, &frame{n: c, size: 2})
	}
	return res.size, res.h > maxDataDepth
}
