package logo

// Calcul litteral : developper, factoriser, resoudre des expressions a
// inconnues. On vise l'ado post-Scratch pre-Python, pile a l'age ou on apprend
// a developper et factoriser en cours de maths.
//
// Le coeur est un petit moteur de polynomes a plusieurs variables, a
// coefficients rationnels exacts (math/big.Rat, jamais de flottant : sinon
// 9*x^2 finit en 8.9999999 et plus personne n'a confiance). Tout passe par la
// forme normale "somme de monomes", que DEVELOPPE imprime telle quelle et dont
// FACTORISE/RESOUDS relisent les coefficients.

import (
	"fmt"
	"math/big"
	"sort"
	"strings"
)

// au-dela, developper (x+1)^n produirait des milliers de termes pour rien :
// on prefere refuser poliment plutot que de faire ramer la machine.
const maxExposant = 1000

// plus grand diviseur teste dans les recherches par essais (racines rationnelles,
// extraction d'un carre sous un radical). garde-fou contre les grands nombres :
// au-dela, on renonce a chercher plus loin. le resultat reste correct, juste
// parfois moins simplifie (radical pas reduit, ou facteur non trouve).
const maxFacteur = 1000000

// un monome = un coefficient et les exposants de chaque variable.
// exemple : 9 x^2 -> coeff 9, exps {x:2}. la constante 5 a exps vide.
type term struct {
	coeff *big.Rat
	exps  map[string]int
}

// un polynome : ses monomes ranges par cle canonique (le monome sans coeff).
// la map regroupe d'office les termes semblables : ajouter 2x puis 3x donne 5x.
type poly map[string]*term

// monoKey : la signature texte d'un monome (variables triees, "x^2*y").
// sert de cle dans la map et a trier l'affichage. constante -> "".
func monoKey(exps map[string]int) string {
	if len(exps) == 0 {
		return ""
	}
	vars := make([]string, 0, len(exps))
	for v := range exps {
		vars = append(vars, v)
	}
	sort.Strings(vars)
	parts := make([]string, 0, len(vars))
	for _, v := range vars {
		if exps[v] == 1 {
			parts = append(parts, v)
		} else {
			parts = append(parts, fmt.Sprintf("%s^%d", v, exps[v]))
		}
	}
	return strings.Join(parts, "*")
}

// ajoute coeff*monome au polynome, en cumulant les termes semblables.
// un terme qui retombe a zero disparait : on ne traine pas de 0*x.
func (p poly) add(exps map[string]int, coeff *big.Rat) {
	if coeff.Sign() == 0 {
		return
	}
	k := monoKey(exps)
	if t, ok := p[k]; ok {
		t.coeff.Add(t.coeff, coeff)
		if t.coeff.Sign() == 0 {
			delete(p, k)
		}
		return
	}
	p[k] = &term{coeff: new(big.Rat).Set(coeff), exps: exps}
}

// le polynome constant (vide si la constante est nulle)
func constPoly(r *big.Rat) poly {
	p := poly{}
	p.add(map[string]int{}, r)
	return p
}

// le polynome reduit a une seule variable, exemple : x
func varPoly(name string) poly {
	p := poly{}
	p.add(map[string]int{name: 1}, big.NewRat(1, 1))
	return p
}

func addPoly(a, b poly) poly {
	r := poly{}
	for _, t := range a {
		r.add(cloneExps(t.exps), t.coeff)
	}
	for _, t := range b {
		r.add(cloneExps(t.exps), t.coeff)
	}
	return r
}

// multiplie tous les coefficients par s (0 -> polynome vide)
func scalePoly(a poly, s *big.Rat) poly {
	r := poly{}
	for _, t := range a {
		r.add(cloneExps(t.exps), new(big.Rat).Mul(t.coeff, s))
	}
	return r
}

func negPoly(a poly) poly { return scalePoly(a, big.NewRat(-1, 1)) }

func subPoly(a, b poly) poly { return addPoly(a, negPoly(b)) }

// produit de deux polynomes : chaque monome de a contre chaque monome de b.
func mulPoly(a, b poly) poly {
	r := poly{}
	for _, ta := range a {
		for _, tb := range b {
			exps := cloneExps(ta.exps)
			for v, e := range tb.exps {
				exps[v] += e
			}
			r.add(exps, new(big.Rat).Mul(ta.coeff, tb.coeff))
		}
	}
	return r
}

// une fraction rationnelle : numerateur / denominateur (deux polynomes). un
// simple polynome a 1 pour denominateur. les operations gardent la forme P/Q et
// la reduisent (a une variable) par le PGCD des polynomes.
type frac struct {
	num, den poly
}

func polyFrac(p poly) frac { return frac{p, constPoly(big.NewRat(1, 1))} }

func fracNeg(f frac) frac { return frac{negPoly(f.num), f.den} }

func fracAdd(a, b frac) frac {
	return frac{addPoly(mulPoly(a.num, b.den), mulPoly(b.num, a.den)), mulPoly(a.den, b.den)}.reduced()
}

func fracSub(a, b frac) frac {
	return frac{subPoly(mulPoly(a.num, b.den), mulPoly(b.num, a.den)), mulPoly(a.den, b.den)}.reduced()
}

func fracMul(a, b frac) frac {
	return frac{mulPoly(a.num, b.num), mulPoly(a.den, b.den)}.reduced()
}

func fracDiv(a, b frac) (frac, error) {
	if len(b.num) == 0 {
		return frac{}, fmt.Errorf("DIVISION PAR ZERO")
	}
	return frac{mulPoly(a.num, b.den), mulPoly(a.den, b.num)}.reduced(), nil
}

func fracPow(a frac, n int) frac {
	r := polyFrac(constPoly(big.NewRat(1, 1)))
	for i := 0; i < n; i++ {
		r = fracMul(r, a)
	}
	return r
}

// reduit la fraction : a une variable, on simplifie par le PGCD ; on met le
// denominateur de tete positif ; un denominateur constant se replie dans le
// numerateur (on retombe alors sur un simple polynome). a plusieurs variables,
// pas de PGCD (trop lourd) : la fraction reste telle quelle, juste normalisee.
func (f frac) reduced() frac {
	if len(f.num) == 0 {
		return polyFrac(poly{}) // zero
	}
	if len(fracVars(f)) <= 1 {
		g := polyGCD(f.num, f.den)
		f = frac{exactQuo(f.num, g), exactQuo(f.den, g)}
	}
	if f.den.sorted()[0].coeff.Sign() < 0 {
		f = frac{negPoly(f.num), negPoly(f.den)}
	}
	if c, ok := f.den.asConst(); ok {
		return frac{scalePoly(f.num, new(big.Rat).Inv(c)), constPoly(big.NewRat(1, 1))}
	}
	return f
}

// les variables presentes au numerateur ou au denominateur
func fracVars(f frac) []string {
	set := map[string]bool{}
	for _, v := range varsOf(f.num) {
		set[v] = true
	}
	for _, v := range varsOf(f.den) {
		set[v] = true
	}
	out := make([]string, 0, len(set))
	for v := range set {
		out = append(out, v)
	}
	return out
}

// PGCD de deux polynomes a une variable (Euclide), rendu unitaire
func polyGCD(a, b poly) poly {
	for len(b) != 0 {
		_, r := divPolyLong(a, b)
		a, b = b, r
	}
	if len(a) == 0 {
		return constPoly(big.NewRat(1, 1))
	}
	return scalePoly(a, new(big.Rat).Inv(a.sorted()[0].coeff)) // unitaire
}

// quotient d'une division exacte (le reste est nul par construction)
func exactQuo(p, g poly) poly {
	q, _ := divPolyLong(p, g)
	return q
}

// division euclidienne d'un polynome par un autre : rend le quotient et le reste
// (a = quotient*b + reste). on retranche a chaque tour le terme de tete tant
// qu'il se divise, sinon ce terme tombe au reste. l'ordre des monomes garantit
// que ca s'arrete. a une variable c'est la division euclidienne classique ; a
// plusieurs, le quotient depend de cet ordre (mais reste nul = vrai diviseur).
func divPolyLong(a, b poly) (poly, poly) {
	q := poly{}
	rem := poly{}
	cur := addPoly(poly{}, a) // copie de travail
	bl := b.sorted()[0]       // terme de tete du diviseur
	for len(cur) > 0 {
		cl := cur.sorted()[0]
		if t, ok := monoDivide(cl, bl); ok {
			q.add(cloneExps(t.exps), t.coeff)
			cur = subPoly(cur, mulPoly(termPoly(t), b))
		} else {
			rem.add(cloneExps(cl.exps), cl.coeff)
			cur = subPoly(cur, termPoly(cl))
		}
	}
	return q, rem
}

// le quotient de deux monomes (coeff divise, exposants soustraits), possible
// seulement si le second "rentre" dans le premier (tous ses exposants en deca).
func monoDivide(a, b *term) (*term, bool) {
	exps := cloneExps(a.exps)
	for v, e := range b.exps {
		if exps[v] < e {
			return nil, false
		}
		exps[v] -= e
		if exps[v] == 0 {
			delete(exps, v)
		}
	}
	return &term{coeff: new(big.Rat).Quo(a.coeff, b.coeff), exps: exps}, true
}

// un polynome reduit a un seul monome
func termPoly(t *term) poly {
	p := poly{}
	p.add(cloneExps(t.exps), t.coeff)
	return p
}

func cloneExps(e map[string]int) map[string]int {
	c := make(map[string]int, len(e))
	for k, v := range e {
		c[k] = v
	}
	return c
}

// la valeur constante du polynome, si c'en est une (sinon ok=false).
// le polynome vide vaut 0.
func (p poly) asConst() (*big.Rat, bool) {
	c := big.NewRat(0, 1)
	for k, t := range p {
		if k != "" {
			return nil, false
		}
		c = t.coeff
	}
	return c, true
}

// degre total du monome (somme des exposants)
func degOf(exps map[string]int) int {
	d := 0
	for _, e := range exps {
		d += e
	}
	return d
}

// compare deux monomes par exposant decroissant, variable par variable dans
// l'ordre alphabetique : a domine b, donc a^2b passe avant ab^2.
func lessMono(a, b map[string]int) bool {
	vars := map[string]bool{}
	for v := range a {
		vars[v] = true
	}
	for v := range b {
		vars[v] = true
	}
	names := make([]string, 0, len(vars))
	for v := range vars {
		names = append(names, v)
	}
	sort.Strings(names)
	for _, v := range names {
		if a[v] != b[v] {
			return a[v] > b[v]
		}
	}
	return false
}

// --- impression : forme normale -> chaine lisible ---

// les monomes ranges pour l'affichage : degre total decroissant, puis ordre
// lexicographique par exposant decroissant (a^3 avant a^2b avant ab^2 avant b^3)
func (p poly) sorted() []*term {
	ts := make([]*term, 0, len(p))
	for _, t := range p {
		ts = append(ts, t)
	}
	sort.Slice(ts, func(i, j int) bool {
		di, dj := degOf(ts[i].exps), degOf(ts[j].exps)
		if di != dj {
			return di > dj
		}
		return lessMono(ts[i].exps, ts[j].exps)
	})
	return ts
}

// rend le polynome facon "9x^2 - 25y^2" : le premier signe colle au terme, les
// suivants espaces avec + ou -.
func (p poly) String() string {
	if len(p) == 0 {
		return "0"
	}
	ts := p.sorted()
	var b strings.Builder
	for i, t := range ts {
		neg := t.coeff.Sign() < 0
		switch {
		case i == 0 && neg:
			b.WriteString("-")
		case i > 0 && neg:
			b.WriteString(" - ")
		case i > 0:
			b.WriteString(" + ")
		}
		abs := new(big.Rat).Abs(t.coeff)
		b.WriteString(printTerm(abs, t.exps))
	}
	return b.String()
}

// un monome positif : coefficient puis variables collees ("9x^2", "6xy").
// le coefficient 1 s'efface devant des variables (x, pas 1x), mais reste seul
// pour une constante.
func printTerm(coeff *big.Rat, exps map[string]int) string {
	mono := printMono(exps)
	if mono == "" {
		return printRat(coeff)
	}
	if coeff.Cmp(big.NewRat(1, 1)) == 0 {
		return mono
	}
	return printRat(coeff) + mono
}

// les variables d'un monome, triees et collees : x, xy, x^2y
func printMono(exps map[string]int) string {
	if len(exps) == 0 {
		return ""
	}
	vars := make([]string, 0, len(exps))
	for v := range exps {
		vars = append(vars, v)
	}
	sort.Strings(vars)
	var b strings.Builder
	for _, v := range vars {
		// en interne les variables sont en minuscule (insensibilite a la casse),
		// mais gologo affiche tout en majuscule : on rend X, pas x
		b.WriteString(strings.ToUpper(v))
		if exps[v] != 1 {
			fmt.Fprintf(&b, "^%d", exps[v])
		}
	}
	return b.String()
}

// un rationnel : entier si possible, sinon p/q
func printRat(r *big.Rat) string {
	if r.IsInt() {
		return r.Num().String()
	}
	return r.RatString()
}

// --- analyse : chaine -> polynome ---

// evalAlg lit une expression litterale et rend sa forme normale.
// accepte le * implicite (9x, 2(x+1), xy) et la casse libre (X = x).
func evalFrac(src string) (frac, error) {
	toks, err := lexAlg(src)
	if err != nil {
		return frac{}, err
	}
	ps := &algParser{toks: toks}
	f, err := ps.parseExpr()
	if err != nil {
		return frac{}, err
	}
	if ps.peek().kind != tkEnd {
		return frac{}, fmt.Errorf("JE NE COMPRENDS PAS LA SUITE DE L'EXPRESSION")
	}
	return f.reduced(), nil
}

// evalAlg : pour les usages qui exigent un vrai polynome (FACTORISE, RESOUS,
// EVALUE). une fraction a denominateur non constant est refusee proprement.
func evalAlg(src string) (poly, error) {
	f, err := evalFrac(src)
	if err != nil {
		return nil, err
	}
	c, ok := f.den.asConst()
	if !ok {
		return nil, fmt.Errorf("CETTE EXPRESSION N'EST PAS UN POLYNOME (DIVISION PAR UNE EXPRESSION)")
	}
	return scalePoly(f.num, new(big.Rat).Inv(c)), nil
}

type tkKind int

const (
	tkEnd tkKind = iota
	tkNum
	tkVar
	tkPlus
	tkMinus
	tkMul
	tkDiv
	tkPow
	tkLParen
	tkRParen
	tkEq
)

type algTok struct {
	kind tkKind
	num  *big.Rat
	name string
}

// lexAlg : decoupe la chaine en jetons. les variables sont des lettres seules
// (donc xy = x*y), les nombres des suites de chiffres avec point decimal
// optionnel. la puissance s'ecrit ^, ** ou les exposants ² et ³ : de quoi taper
// un exposant meme quand ^ est une touche morte (clavier AZERTY).
func lexAlg(src string) ([]algTok, error) {
	var toks []algTok
	rs := []rune(src)
	for i := 0; i < len(rs); {
		c := rs[i]
		switch {
		case c == ' ' || c == '\t':
			i++
		case c >= '0' && c <= '9' || c == '.':
			j := i
			for j < len(rs) && (rs[j] >= '0' && rs[j] <= '9' || rs[j] == '.') {
				j++
			}
			r := new(big.Rat)
			if _, ok := r.SetString(string(rs[i:j])); !ok {
				return nil, fmt.Errorf("NOMBRE INCORRECT : %s", string(rs[i:j]))
			}
			toks = append(toks, algTok{kind: tkNum, num: r})
			i = j
		case c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z':
			// une lettre = une variable, ramenee en minuscule (X et x confondus)
			toks = append(toks, algTok{kind: tkVar, name: strings.ToLower(string(c))})
			i++
		case c == '+':
			toks = append(toks, algTok{kind: tkPlus})
			i++
		case c == '-':
			toks = append(toks, algTok{kind: tkMinus})
			i++
		case c == '*':
			// ** = puissance (facile a taper, sans la touche morte ^ d'AZERTY)
			if i+1 < len(rs) && rs[i+1] == '*' {
				toks = append(toks, algTok{kind: tkPow})
				i += 2
			} else {
				toks = append(toks, algTok{kind: tkMul})
				i++
			}
		case c == '/':
			toks = append(toks, algTok{kind: tkDiv})
			i++
		case c == '^':
			toks = append(toks, algTok{kind: tkPow})
			i++
		case c == '²': // ²
			toks = append(toks, algTok{kind: tkPow}, algTok{kind: tkNum, num: big.NewRat(2, 1)})
			i++
		case c == '³': // ³
			toks = append(toks, algTok{kind: tkPow}, algTok{kind: tkNum, num: big.NewRat(3, 1)})
			i++
		case c == '(':
			toks = append(toks, algTok{kind: tkLParen})
			i++
		case c == ')':
			toks = append(toks, algTok{kind: tkRParen})
			i++
		case c == '=':
			toks = append(toks, algTok{kind: tkEq})
			i++
		default:
			return nil, fmt.Errorf("CARACTERE INATTENDU : %s", string(c))
		}
	}
	toks = append(toks, algTok{kind: tkEnd})
	return toks, nil
}

type algParser struct {
	toks []algTok
	pos  int
}

func (p *algParser) peek() algTok { return p.toks[p.pos] }
func (p *algParser) advance() algTok {
	t := p.toks[p.pos]
	p.pos++
	return t
}

// expr := terme ( (+|-) terme )*
func (p *algParser) parseExpr() (frac, error) {
	left, err := p.parseTerm()
	if err != nil {
		return frac{}, err
	}
	for {
		switch p.peek().kind {
		case tkPlus:
			p.advance()
			r, err := p.parseTerm()
			if err != nil {
				return frac{}, err
			}
			left = fracAdd(left, r)
		case tkMinus:
			p.advance()
			r, err := p.parseTerm()
			if err != nil {
				return frac{}, err
			}
			left = fracSub(left, r)
		default:
			return left, nil
		}
	}
}

// terme := facteur ( (*|/) facteur | facteur )*   (le dernier cas = * implicite)
func (p *algParser) parseTerm() (frac, error) {
	left, err := p.parseFactor()
	if err != nil {
		return frac{}, err
	}
	for {
		switch p.peek().kind {
		case tkMul:
			p.advance()
			r, err := p.parseFactor()
			if err != nil {
				return frac{}, err
			}
			left = fracMul(left, r)
		case tkDiv:
			p.advance()
			r, err := p.parseFactor()
			if err != nil {
				return frac{}, err
			}
			left, err = fracDiv(left, r)
			if err != nil {
				return frac{}, err
			}
		case tkNum, tkVar, tkLParen:
			// rien entre les deux : multiplication implicite (9x, 2(x+1), xy)
			r, err := p.parseFactor()
			if err != nil {
				return frac{}, err
			}
			left = fracMul(left, r)
		default:
			return left, nil
		}
	}
}

// facteur := (-|+) facteur | base (^ entier)?
// le moins unaire est moins prioritaire que la puissance : -x^2 = -(x^2)
func (p *algParser) parseFactor() (frac, error) {
	switch p.peek().kind {
	case tkMinus:
		p.advance()
		f, err := p.parseFactor()
		if err != nil {
			return frac{}, err
		}
		return fracNeg(f), nil
	case tkPlus:
		p.advance()
		return p.parseFactor()
	}
	base, err := p.parseBase()
	if err != nil {
		return frac{}, err
	}
	if p.peek().kind == tkPow {
		p.advance()
		t := p.peek()
		if t.kind != tkNum {
			return frac{}, fmt.Errorf("APRES ^ IL FAUT UN EXPOSANT ENTIER")
		}
		if !t.num.IsInt() || t.num.Sign() < 0 {
			return frac{}, fmt.Errorf("L'EXPOSANT DOIT ETRE UN ENTIER POSITIF")
		}
		// garde-fou : un exposant enorme ferait exploser le developpement
		n := t.num.Num()
		if !n.IsInt64() || n.Int64() > maxExposant {
			return frac{}, fmt.Errorf("EXPOSANT TROP GRAND (MAX %d)", maxExposant)
		}
		p.advance()
		base = fracPow(base, int(n.Int64()))
	}
	return base, nil
}

// base := nombre | variable | ( expr )
func (p *algParser) parseBase() (frac, error) {
	switch t := p.peek(); t.kind {
	case tkNum:
		p.advance()
		return polyFrac(constPoly(t.num)), nil
	case tkVar:
		p.advance()
		return polyFrac(varPoly(t.name)), nil
	case tkLParen:
		p.advance()
		e, err := p.parseExpr()
		if err != nil {
			return frac{}, err
		}
		if p.peek().kind != tkRParen {
			return frac{}, fmt.Errorf("IL MANQUE UNE PARENTHESE FERMANTE")
		}
		p.advance()
		return e, nil
	default:
		return frac{}, fmt.Errorf("IL MANQUE UN NOMBRE OU UNE VARIABLE")
	}
}

// developpeStr : le resultat affiche par DEVELOPPE. L'expression est evaluee
// comme fraction rationnelle reduite. Si le denominateur est constant, c'est un
// simple polynome. Sinon on montre la division : partie entiere + reste/diviseur.
func developpeStr(src string) (string, error) {
	f, err := evalFrac(src)
	if err != nil {
		return "", err
	}
	return fracRender(f)
}

// met en mots une fraction : un simple polynome si le denominateur est constant,
// sinon partie entiere + reste/diviseur.
func fracRender(f frac) (string, error) {
	if len(f.den) == 0 {
		return "", fmt.Errorf("DIVISION PAR ZERO")
	}
	if c, ok := f.den.asConst(); ok {
		return scalePoly(f.num, new(big.Rat).Inv(c)).String(), nil
	}
	q, r := divPolyLong(f.num, f.den)
	return fracString(q, r, f.den), nil
}

// met en forme "quotient + reste/diviseur" (la valeur exacte de A/B). Le reste
// est rendu en valeur absolue, son signe enchaine le quotient (+ ou -). Reste
// nul : juste le quotient. Quotient nul : juste la fraction.
func fracString(q, r, den poly) string {
	if len(r) == 0 {
		return q.String()
	}
	denStr := den.String()
	if len(den) > 1 {
		denStr = "(" + denStr + ")"
	}
	neg, mag := splitPolySign(r)
	num := mag.String()
	if len(mag) > 1 {
		num = "(" + num + ")"
	}
	frac := num + "/" + denStr
	if len(q) == 0 {
		if neg {
			return "-" + frac
		}
		return frac
	}
	if neg {
		return q.String() + " - " + frac
	}
	return q.String() + " + " + frac
}

// rend le signe du terme de tete a part, pour que le reste s'affiche positif
func splitPolySign(r poly) (bool, poly) {
	if len(r) == 0 {
		return false, r
	}
	if r.sorted()[0].coeff.Sign() < 0 {
		return true, negPoly(r)
	}
	return false, r
}

// --- outils sur les polynomes (resolution, factorisation) ---

// les variables presentes dans le polynome, triees
func varsOf(p poly) []string {
	set := map[string]bool{}
	for _, t := range p {
		for v := range t.exps {
			set[v] = true
		}
	}
	out := make([]string, 0, len(set))
	for v := range set {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

// plus haut exposant de la variable v dans le polynome
func maxDeg(p poly, v string) int {
	d := 0
	for _, t := range p {
		if t.exps[v] > d {
			d = t.exps[v]
		}
	}
	return d
}

// coefficient du terme en v^k (k=0 : la constante). 0 si absent.
// suppose un polynome a une seule variable.
func coeffDeg(p poly, v string, k int) *big.Rat {
	want := 0
	if k > 0 {
		want = 1
	}
	for _, t := range p {
		if t.exps[v] == k && len(t.exps) == want {
			return new(big.Rat).Set(t.coeff)
		}
	}
	return big.NewRat(0, 1)
}

func polyEqual(a, b poly) bool { return len(subPoly(a, b)) == 0 }

// multiplie un rationnel par un petit entier
func ratScale(r *big.Rat, k int64) *big.Rat {
	return new(big.Rat).Mul(r, big.NewRat(k, 1))
}

// racine carree exacte d'un entier >= 0 (ok=false si ce n'est pas un carre)
func intSqrt(n *big.Int) (*big.Int, bool) {
	if n.Sign() < 0 {
		return nil, false
	}
	s := new(big.Int).Sqrt(n)
	if new(big.Int).Mul(s, s).Cmp(n) == 0 {
		return s, true
	}
	return nil, false
}

// racine carree exacte d'un rationnel : exacte seulement si numerateur et
// denominateur sont tous deux des carres parfaits
func ratSqrt(r *big.Rat) (*big.Rat, bool) {
	if r.Sign() < 0 {
		return nil, false
	}
	sn, okn := intSqrt(r.Num())
	sd, okd := intSqrt(r.Denom())
	if !okn || !okd {
		return nil, false
	}
	return new(big.Rat).SetFrac(sn, sd), true
}

// ppcm de deux entiers positifs
func lcmInt(a, b *big.Int) *big.Int {
	g := gcdBig(a, b)
	if g.Sign() == 0 {
		return big.NewInt(1)
	}
	return new(big.Int).Div(new(big.Int).Mul(a, b), g)
}

// sort le plus grand carre d'un entier : m = g^2 * d, d sans facteur carre.
// rend g et d (par divisions d'essai, largement suffisant a taille scolaire).
func squareFreePart(m *big.Int) (g, d *big.Int) {
	g = big.NewInt(1)
	d = new(big.Int).Set(m)
	i := big.NewInt(2)
	limite := big.NewInt(maxFacteur)
	for {
		if i.Cmp(limite) > 0 {
			break // trop loin : on laisse le reste sous le radical (toujours juste)
		}
		i2 := new(big.Int).Mul(i, i)
		if i2.Cmp(d) > 0 {
			break
		}
		for {
			q, r := new(big.Int).DivMod(d, i2, new(big.Int))
			if r.Sign() != 0 {
				break
			}
			d = q
			g.Mul(g, i)
		}
		i.Add(i, big.NewInt(1))
	}
	return g, d
}

// les deux racines exactes d'un trinome a discriminant non carre, sous forme
// (p +/- q√d)/c. disc > 0 et n'est pas un carre parfait.
func quadRadicalRoots(a, b, disc *big.Rat) (string, string) {
	// √disc = g√d / Denom(disc), avec disc = Num/Denom
	g, d := squareFreePart(new(big.Int).Mul(disc.Num(), disc.Denom()))
	twoA := ratScale(a, 2)
	r1 := new(big.Rat).Quo(new(big.Rat).Neg(b), twoA)                   // partie rationnelle
	r2 := new(big.Rat).Quo(new(big.Rat).SetFrac(g, disc.Denom()), twoA) // coeff du radical
	r2.Abs(r2)                                                          // le +/- porte le signe
	c := lcmInt(r1.Denom(), r2.Denom())
	p := new(big.Int).Mul(r1.Num(), new(big.Int).Div(c, r1.Denom()))
	q := new(big.Int).Mul(r2.Num(), new(big.Int).Div(c, r2.Denom()))
	return radRoot(p, q, d, c, true), radRoot(p, q, d, c, false)
}

// met en forme une racine (p +/- q√d)/c, sans afficher les 1 ni les /1 inutiles
func radRoot(p, q, d, c *big.Int, minus bool) string {
	surd := "√" + d.String()
	if q.Cmp(big.NewInt(1)) != 0 {
		surd = q.String() + surd
	}
	var num string
	switch {
	case p.Sign() == 0 && minus:
		num = "-" + surd
	case p.Sign() == 0:
		num = surd
	case minus:
		num = p.String() + " - " + surd
	default:
		num = p.String() + " + " + surd
	}
	if c.Cmp(big.NewInt(1)) == 0 {
		return num
	}
	if p.Sign() == 0 {
		return num + "/" + c.String() // ex -√2/2
	}
	return "(" + num + ")/" + c.String()
}

// --- RESOUS : equations du 1er et 2nd degre a une inconnue ---

// les mots du resultat changent avec la langue (les nombres, eux, sont neutres)
func solveWord(en bool, fr, ang string) string {
	if en {
		return ang
	}
	return fr
}

func solveEquation(src, lang string) (string, error) {
	en := lang == "EN"
	sides := strings.Split(src, "=")
	if len(sides) != 2 {
		return "", fmt.Errorf("UNE EQUATION DOIT AVOIR UN SEUL SIGNE = (EXEMPLE [ 2x + 3 = 7 ])")
	}
	left, err := evalAlg(sides[0])
	if err != nil {
		return "", err
	}
	right, err := evalAlg(sides[1])
	if err != nil {
		return "", err
	}
	// tout du meme cote : p = 0
	p := subPoly(left, right)
	vars := varsOf(p)
	if len(vars) > 1 {
		return "", fmt.Errorf("JE NE RESOUS QU'UNE EQUATION A UNE SEULE INCONNUE")
	}
	if len(vars) == 0 {
		// plus d'inconnue : ou bien c'est toujours vrai, ou bien jamais
		c, _ := p.asConst()
		if c.Sign() == 0 {
			return solveWord(en, "toujours vrai : n'importe quel nombre convient",
				"always true: any number works"), nil
		}
		return solveWord(en, "aucune solution", "no solution"), nil
	}
	v := vars[0]
	vu := strings.ToUpper(v) // affichage en majuscule (v reste la cle interne en minuscule)
	deg := maxDeg(p, v)
	if deg > 2 {
		return "", fmt.Errorf("JE NE RESOUS QUE LE 1er ET LE 2nd DEGRE")
	}
	a := coeffDeg(p, v, 2)
	b := coeffDeg(p, v, 1)
	c := coeffDeg(p, v, 0)
	if deg == 1 {
		// b x + c = 0
		x := new(big.Rat).Neg(new(big.Rat).Quo(c, b))
		return fmt.Sprintf("%s = %s", vu, printRat(x)), nil
	}
	// a x^2 + b x + c = 0 : discriminant b^2 - 4ac
	disc := new(big.Rat).Sub(new(big.Rat).Mul(b, b), ratScale(new(big.Rat).Mul(a, c), 4))
	twoA := ratScale(a, 2)
	ou := solveWord(en, "ou", "or")
	switch disc.Sign() {
	case -1:
		return solveWord(en, "pas de solution reelle", "no real solution"), nil
	case 0:
		x := new(big.Rat).Quo(new(big.Rat).Neg(b), twoA)
		return fmt.Sprintf("%s = %s %s", vu, printRat(x),
			solveWord(en, "(solution double)", "(double solution)")), nil
	}
	// disc > 0 : deux solutions
	if rt, ok := ratSqrt(disc); ok {
		x1 := new(big.Rat).Quo(new(big.Rat).Add(new(big.Rat).Neg(b), rt), twoA)
		x2 := new(big.Rat).Quo(new(big.Rat).Sub(new(big.Rat).Neg(b), rt), twoA)
		lo, hi := printRat(x1), printRat(x2)
		if x1.Cmp(x2) > 0 {
			lo, hi = hi, lo
		}
		return fmt.Sprintf("%s = %s %s %s = %s", vu, lo, ou, vu, hi), nil
	}
	// racines irrationnelles : forme exacte avec radical, (p +/- q√d)/c
	lo, hi := quadRadicalRoots(a, b, disc)
	return fmt.Sprintf("%s = %s %s %s = %s", vu, lo, ou, vu, hi), nil
}

// --- EVALUE : remplace des variables par des valeurs ---

// r^n pour n entier >= 0
func ratPow(r *big.Rat, n int) *big.Rat {
	res := big.NewRat(1, 1)
	for i := 0; i < n; i++ {
		res.Mul(res, r)
	}
	return res
}

// remplace dans le polynome les variables donnees par leur valeur. les variables
// absentes de la table restent telles quelles (on peut donc evaluer partiellement).
func substitute(p poly, vals map[string]*big.Rat) poly {
	out := poly{}
	for _, t := range p {
		coeff := new(big.Rat).Set(t.coeff)
		exps := map[string]int{}
		for v, e := range t.exps {
			if val, ok := vals[v]; ok {
				coeff.Mul(coeff, ratPow(val, e))
			} else {
				exps[v] = e
			}
		}
		out.add(exps, coeff)
	}
	return out
}

// lit la liste de couples "variable valeur" ( [ x 3 y 5 ] ). la valeur est tout
// ce qui suit jusqu'a la prochaine lettre seule, et doit valoir un nombre.
func parsePairs(items []Datum) (map[string]*big.Rat, error) {
	vals := map[string]*big.Rat{}
	for i := 0; i < len(items); {
		name := strings.ToLower(items[i].String())
		if len(name) != 1 || name[0] < 'a' || name[0] > 'z' {
			return nil, fmt.Errorf("ATTENDU UNE VARIABLE (UNE LETTRE), VU %q", items[i].String())
		}
		i++
		start := i
		for i < len(items) && !isSingleLetter(items[i]) {
			i++
		}
		if i == start {
			return nil, fmt.Errorf("IL MANQUE UNE VALEUR POUR %s", strings.ToUpper(name))
		}
		p, err := evalAlg(datumsToAlg(items[start:i]))
		if err != nil {
			return nil, err
		}
		c, ok := p.asConst()
		if !ok {
			return nil, fmt.Errorf("LA VALEUR DE %s DOIT ETRE UN NOMBRE", strings.ToUpper(name))
		}
		vals[name] = c
	}
	return vals, nil
}

// vrai si le datum est une seule lettre (un nom de variable), pas un nombre
func isSingleLetter(d Datum) bool {
	s := d.String()
	if len(s) != 1 {
		return false
	}
	c := s[0]
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// --- FACTORISE : facteur commun, identites remarquables, trinome ---

func factorize(p poly) string {
	if len(p) == 0 {
		return "0"
	}
	content, mono, prim := pullCommon(p)
	factors := factorPrimitive(prim)
	return renderFactored(content, mono, factors)
}

// pgcd de deux entiers positifs (0 neutre)
func gcdBig(a, b *big.Int) *big.Int {
	return new(big.Int).GCD(nil, nil, new(big.Int).Abs(a), new(big.Int).Abs(b))
}

// sort le facteur commun : un coefficient rationnel et un monome present partout.
// rend (contenu, monome commun, partie primitive a coefficients entiers premiers
// entre eux et de tete positive).
func pullCommon(p poly) (*big.Rat, map[string]int, poly) {
	// monome commun : le plus petit exposant de chaque variable
	mono := map[string]int{}
	first := true
	for _, t := range p {
		if first {
			for v, e := range t.exps {
				mono[v] = e
			}
			first = false
			continue
		}
		for v := range mono {
			if e := t.exps[v]; e < mono[v] {
				mono[v] = e
			}
		}
	}
	for v, e := range mono {
		if e == 0 {
			delete(mono, v)
		}
	}
	// contenu : pgcd des coefficients (apres mise au meme denominateur)
	denom := big.NewInt(1)
	for _, t := range p {
		d := t.coeff.Denom()
		denom = new(big.Int).Div(new(big.Int).Mul(denom, d), gcdBig(denom, d))
	}
	g := big.NewInt(0)
	for _, t := range p {
		ni := new(big.Int).Mul(t.coeff.Num(), new(big.Int).Div(denom, t.coeff.Denom()))
		g = gcdBig(g, ni)
	}
	content := new(big.Rat).SetFrac(g, denom)
	inv := new(big.Rat).Inv(content)
	prim := poly{}
	for _, t := range p {
		exps := cloneExps(t.exps)
		for v, e := range mono {
			exps[v] -= e
			if exps[v] == 0 {
				delete(exps, v)
			}
		}
		prim.add(exps, new(big.Rat).Mul(t.coeff, inv))
	}
	// tete positive : on pousse le signe dans le contenu
	if lead := prim.sorted(); len(lead) > 0 && lead[0].coeff.Sign() < 0 {
		content.Neg(content)
		prim = negPoly(prim)
	}
	return content, mono, prim
}

// factorise la partie primitive en une liste de facteurs (produit = entree).
// a une variable : on extrait toutes les racines rationnelles (tout degre). a
// plusieurs : on s'en tient aux identites remarquables.
func factorPrimitive(q poly) []poly {
	if len(q) <= 1 {
		return []poly{q}
	}
	if len(varsOf(q)) == 1 {
		return factorUnivariate(q, varsOf(q)[0])
	}
	if fs := diffSquares(q); fs != nil {
		out := []poly{}
		for _, f := range fs {
			out = append(out, factorPrimitive(f)...)
		}
		return out
	}
	if base := perfectSquare(q); base != nil {
		sub := factorPrimitive(base)
		return append(sub, sub...) // q = base^2
	}
	return []poly{q} // irreductible avec nos moyens : on laisse tel quel
}

// factorise un polynome a une variable en facteurs lineaires (une par racine
// rationnelle) suivis de ce qui reste sans racine rationnelle. ce reste est
// rendu tel quel : x^2+1 est irreductible sur les rationnels, on ne le casse pas.
func factorUnivariate(q poly, v string) []poly {
	var factors []poly
	cur := q
	for maxDeg(cur, v) >= 1 {
		r, ok := rationalRoot(cur, v)
		if !ok {
			break
		}
		lin := linearFactor(v, r)
		factors = append(factors, lin)
		cur, _ = divPolyLong(cur, lin) // division exacte (r est racine)
	}
	return append(factors, cur) // le reste (constante 1 absorbee, ou facteur sans racine)
}

// cherche une racine rationnelle p/q d'un polynome a une variable, via le
// theoreme des racines rationnelles : p divise le terme constant, q le terme de
// tete. rend false si aucune (ou si les coefficients debordent).
func rationalRoot(p poly, v string) (*big.Rat, bool) {
	deg := maxDeg(p, v)
	a0 := coeffDeg(p, v, 0)
	if a0.Sign() == 0 {
		return big.NewRat(0, 1), true // 0 est racine (rare ici : monome commun deja sorti)
	}
	an := coeffDeg(p, v, deg)
	if !a0.Num().IsInt64() || !an.Num().IsInt64() {
		return nil, false
	}
	for _, pp := range divisors(a0.Num()) {
		for _, qq := range divisors(an.Num()) {
			for _, s := range []int64{1, -1} {
				r := new(big.Rat).SetFrac(big.NewInt(s*pp), big.NewInt(qq))
				if evalPolyAt(p, v, r).Sign() == 0 {
					return r, true
				}
			}
		}
	}
	return nil, false
}

// la valeur du polynome (a une variable) en r
func evalPolyAt(p poly, v string, r *big.Rat) *big.Rat {
	c, _ := substitute(p, map[string]*big.Rat{v: r}).asConst()
	return c
}

// les diviseurs positifs de |n| (n tient dans un int64). on ne teste pas au-dela
// de maxFacteur : sur un tres grand n on peut rater une racine (le polynome reste
// alors non factorise), mais on ne fait jamais ramer la machine.
func divisors(n *big.Int) []int64 {
	m := new(big.Int).Abs(n).Int64()
	var ds []int64
	for i := int64(1); i*i <= m && i <= maxFacteur; i++ {
		if m%i == 0 {
			ds = append(ds, i)
			if i != m/i {
				ds = append(ds, m/i)
			}
		}
	}
	return ds
}

// la racine carree d'un monome (coeff carre parfait, exposants pairs), ou nil
func sqrtMonomial(t *term) poly {
	if t.coeff.Sign() <= 0 {
		return nil
	}
	c, ok := ratSqrt(t.coeff)
	if !ok {
		return nil
	}
	exps := map[string]int{}
	for v, e := range t.exps {
		if e%2 != 0 {
			return nil
		}
		exps[v] = e / 2
	}
	r := poly{}
	r.add(exps, c)
	return r
}

// difference de deux carres A^2 - B^2 -> [A-B, A+B], sinon nil
func diffSquares(q poly) []poly {
	if len(q) != 2 {
		return nil
	}
	ts := q.sorted()
	var pos, neg *term
	for _, t := range ts {
		if t.coeff.Sign() > 0 {
			pos = t
		} else {
			neg = t
		}
	}
	if pos == nil || neg == nil {
		return nil
	}
	A := sqrtMonomial(pos)
	B := sqrtMonomial(&term{coeff: new(big.Rat).Abs(neg.coeff), exps: neg.exps})
	if A == nil || B == nil {
		return nil
	}
	return []poly{subPoly(A, B), addPoly(A, B)}
}

// carre parfait A^2 +/- 2AB + B^2 -> le binome A+B ou A-B, sinon nil
func perfectSquare(q poly) poly {
	if len(q) != 3 {
		return nil
	}
	ts := q.sorted()
	for i := 0; i < 3; i++ {
		for j := i + 1; j < 3; j++ {
			A := sqrtMonomial(ts[i])
			B := sqrtMonomial(ts[j])
			if A == nil || B == nil {
				continue
			}
			mid := poly{}
			k := 3 - i - j
			mid.add(cloneExps(ts[k].exps), ts[k].coeff)
			two := mulPoly(scalePoly(A, big.NewRat(2, 1)), B)
			if polyEqual(mid, two) {
				return addPoly(A, B)
			}
			if polyEqual(mid, negPoly(two)) {
				return subPoly(A, B)
			}
		}
	}
	return nil
}

// le facteur lineaire entier de racine r = n/d : d*v - n (d>0 toujours)
func linearFactor(v string, r *big.Rat) poly {
	p := poly{}
	p.add(map[string]int{v: 1}, new(big.Rat).SetInt(r.Denom()))
	p.add(map[string]int{}, new(big.Rat).Neg(new(big.Rat).SetInt(r.Num())))
	return p
}

// assemble le resultat : contenu, monome commun, puis les facteurs entre
// parentheses (facteurs identiques regroupes en puissances).
func renderFactored(content *big.Rat, mono map[string]int, factors []poly) string {
	// les facteurs constants rentrent dans le contenu
	var real []poly
	for _, f := range factors {
		if c, ok := f.asConst(); ok {
			content = new(big.Rat).Mul(content, c)
		} else {
			real = append(real, f)
		}
	}
	// regroupe les facteurs egaux, en gardant un ordre lisible
	type grp struct {
		p poly
		n int
	}
	var groups []grp
	for _, f := range real {
		found := false
		for i := range groups {
			if polyEqual(groups[i].p, f) {
				groups[i].n++
				found = true
				break
			}
		}
		if !found {
			groups = append(groups, grp{f, 1})
		}
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].p.String() < groups[j].p.String() })

	// un seul facteur, sans coefficient ni monome : on l'ecrit sans parentheses
	monoStr := printMono(mono)
	one := content.Cmp(big.NewRat(1, 1)) == 0
	if one && monoStr == "" && len(groups) == 1 && groups[0].n == 1 {
		return groups[0].p.String()
	}

	var b strings.Builder
	if content.Sign() < 0 {
		b.WriteString("-")
	}
	abs := new(big.Rat).Abs(content)
	// le coefficient 1 s'efface s'il y a des variables ou des facteurs
	if abs.Cmp(big.NewRat(1, 1)) != 0 || (monoStr == "" && len(groups) == 0) {
		b.WriteString(printRat(abs))
	}
	b.WriteString(monoStr)
	for _, g := range groups {
		fmt.Fprintf(&b, "(%s)", g.p.String())
		if g.n > 1 {
			fmt.Fprintf(&b, "^%d", g.n)
		}
	}
	return b.String()
}

// argToAlg : recupere l'expression a analyser depuis l'argument Logo.
// on l'exige entre crochets : un mot Logo ne peut porter ni espace ni
// parenthese, la liste est donc la seule forme qui passe tout. autant n'en
// garder qu'une.
func argToAlg(v Value) (string, error) {
	if v.Kind != KList {
		return "", fmt.Errorf("DONNE L'EXPRESSION ENTRE CROCHETS, PAR EXEMPLE [ (X+1)(X-1) ]")
	}
	return datumsToAlg(v.List), nil
}

// reconstruit le texte de l'expression a partir des Datum d'une liste. une
// sous-liste ou un groupe ( ... ) redevient une vraie parenthese, le reste
// s'imprime tel quel et les espaces separateurs ne genent pas l'analyse.
func datumsToAlg(ds []Datum) string {
	parts := make([]string, len(ds))
	for i, d := range ds {
		parts[i] = datumToAlg(d)
	}
	return strings.Join(parts, " ")
}

func datumToAlg(d Datum) string {
	if d.Kind == DList || d.Kind == DGroup {
		return "(" + datumsToAlg(d.List) + ")"
	}
	return d.String()
}

// --- primitives ---

func (i *Interp) registerCalcul() {
	op := func(arity int, fn func(*Interp, []Value) (Value, error), names ...string) {
		i.register(&primitive{arity: arity, reporter: true, fn: fn}, names...)
	}

	// DEVELOPPE expr : developpe et reduit l'expression litterale
	// DEVELOPPE [ (3x+5y)(3x-5y) ]  ->  9x^2 - 25y^2
	op(1, func(in *Interp, a []Value) (Value, error) {
		w, err := argToAlg(a[0])
		if err != nil {
			return Value{}, err
		}
		if strings.Contains(w, "=") {
			return Value{}, fmt.Errorf("DEVELOPPE VEUT UNE EXPRESSION, PAS UNE EGALITE")
		}
		s, err := developpeStr(w)
		if err != nil {
			return Value{}, err
		}
		return WordValue(s), nil
	}, "DEVELOPPE")

	// FACTORISE expr : factorise (facteur commun, identites remarquables, trinome)
	// FACTORISE [ 9x^2 - 25y^2 ]  ->  (3x + 5y)(3x - 5y)
	op(1, func(in *Interp, a []Value) (Value, error) {
		w, err := argToAlg(a[0])
		if err != nil {
			return Value{}, err
		}
		if strings.Contains(w, "=") {
			return Value{}, fmt.Errorf("FACTORISE VEUT UNE EXPRESSION, PAS UNE EGALITE")
		}
		p, err := evalAlg(w)
		if err != nil {
			return Value{}, err
		}
		return WordValue(factorize(p)), nil
	}, "FACTORISE")

	// RESOUS equation : resout une equation a une inconnue (1er ou 2nd degre)
	// RESOUS [ x^2 - 5x + 6 = 0 ]  ->  x = 2 ou x = 3
	op(1, func(in *Interp, a []Value) (Value, error) {
		w, err := argToAlg(a[0])
		if err != nil {
			return Value{}, err
		}
		if !strings.Contains(w, "=") {
			return Value{}, fmt.Errorf("UNE EQUATION A UN SIGNE = (EXEMPLE [ 2x + 3 = 7 ])")
		}
		s, err := solveEquation(w, in.Lang())
		if err != nil {
			return Value{}, err
		}
		return WordValue(s), nil
	}, "RESOUS", "RESOUDS")

	// EVALUE expr valeurs : remplace les variables par des nombres
	// EVALUE [ x^2 + 1 ] [ x 3 ]  ->  10
	op(2, func(in *Interp, a []Value) (Value, error) {
		w, err := argToAlg(a[0])
		if err != nil {
			return Value{}, err
		}
		if strings.Contains(w, "=") {
			return Value{}, fmt.Errorf("EVALUE VEUT UNE EXPRESSION, PAS UNE EGALITE")
		}
		// fraction-based : EVALUE marche aussi sur 1/(x+1) (rend une valeur ou une
		// fraction reduite), pas seulement sur les polynomes.
		f, err := evalFrac(w)
		if err != nil {
			return Value{}, err
		}
		if a[1].Kind != KList {
			return Value{}, fmt.Errorf("DONNE LES VALEURS ENTRE CROCHETS, PAR EXEMPLE [ X 3 ]")
		}
		vals, err := parsePairs(a[1].List)
		if err != nil {
			return Value{}, err
		}
		den := substitute(f.den, vals)
		if len(den) == 0 {
			return Value{}, fmt.Errorf("DIVISION PAR ZERO")
		}
		s, err := fracRender(frac{substitute(f.num, vals), den}.reduced())
		if err != nil {
			return Value{}, err
		}
		return WordValue(s), nil
	}, "EVALUE")
}
