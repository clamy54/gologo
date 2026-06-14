package logo

// entiers exacts : float64 compte juste jusqu'a 2^53, au-dela il saute des
// valeurs. quand un calcul entier depasse cette borne, on passe sur math/big
// (KInt) ; en dessous on garde le float64, c'est plus rapide. les fractions et
// SIN/RC/EXP... restent flottantes - le big ne sert qu'aux entiers

import (
	"math"
	"math/big"
	"strconv"
	"strings"
)

// 2^53 vu comme un grand entier, pour comparer une valeur big a la borne d'exactitude
var bigMaxExact = new(big.Int).Lsh(big.NewInt(1), 53)

// taille maximale (en bits) d'une PUISSANCE calculee en exact : ~4M bits, soit
// un nombre d'environ 1,2 million de chiffres. au-dela on retombe en flottant,
// sinon 2 PUISSANCE 1e12 essaierait d'allouer 125 Go
const maxPowBits = 4 << 20

// resultat d'un calcul entier : on rebascule en float64 (KNumber) tant que la
// valeur tient exactement dans [-2^53, 2^53], sinon on garde l'entier exact
// (KInt). Ca evite de propager des KInt inutiles dans tout le reste du code.
func intResult(b *big.Int) Value {
	if new(big.Int).Abs(b).Cmp(bigMaxExact) <= 0 {
		return NumberValue(float64(b.Int64()))
	}
	return IntValue(b)
}

// vue "entier exact" d'une valeur, si c'en est un. un KInt en est toujours un,
// un KNumber seulement s'il est entier et dans [-2^53, 2^53]. au-dela un float64
// ment deja : pas question de le promouvoir en big, on figerait une fausse
// precision. un mot tout-chiffres compte aussi (typiquement lu dans un fichier)
func asIntOperand(v Value) (*big.Int, bool) {
	switch v.Kind {
	case KInt:
		return v.Int, true
	case KNumber:
		if !math.IsInf(v.Num, 0) && !math.IsNaN(v.Num) &&
			v.Num == math.Trunc(v.Num) && math.Abs(v.Num) <= maxExactIntFloat64 {
			return big.NewInt(int64(v.Num)), true
		}
	case KWord:
		return parseBigInt(v.Word)
	}
	return nil, false
}

// parse un mot comme entier en base 10 (signe optionnel, que des chiffres).
// rejette les flottants ("3.14", "1e9") et tout le reste.
func parseBigInt(s string) (*big.Int, bool) {
	if s == "" {
		return nil, false
	}
	b, ok := new(big.Int).SetString(s, 10)
	return b, ok
}

// litteral entier "trop grand pour float64" : appele a la lecture. on ne fabrique
// un entier exact que si le texte est un entier pur (signe + chiffres) ET deborde
// 2^53 - sinon on laisse le float64 habituel (plus simple, plus rapide).
func bigIntLiteral(text string) (*big.Int, bool) {
	if strings.ContainsAny(text, ".eE") {
		return nil, false
	}
	b, ok := parseBigInt(text)
	if !ok {
		return nil, false
	}
	if new(big.Int).Abs(b).Cmp(bigMaxExact) <= 0 {
		return nil, false // tient en float64, pas besoin de big
	}
	return b, true
}

// fabrique un Datum nombre a la lecture : entier exact (Big) si le litteral
// deborde 2^53, sinon le float64 habituel.
func numberDatum(num float64, text string) Datum {
	if b, ok := bigIntLiteral(text); ok {
		return Datum{Kind: DNumber, Big: b, Text: text}
	}
	return Datum{Kind: DNumber, Num: num, Text: text}
}

// vue int64 rapide d'une valeur, si c'est un entier exact qui tient en int64
// SANS depasser 2^53 (donc additionnable/soustrayable sans risque de debordement).
// pas d'allocation, contrairement a asIntOperand qui fabrique un big.Int. on
// exclut volontairement KInt (toujours > 2^53) : il passe par la voie big.
func asSmallInt(v Value) (int64, bool) {
	switch v.Kind {
	case KNumber:
		if !math.IsInf(v.Num, 0) && !math.IsNaN(v.Num) &&
			v.Num == math.Trunc(v.Num) && math.Abs(v.Num) <= maxExactIntFloat64 {
			return int64(v.Num), true
		}
	case KWord:
		if n, err := strconv.ParseInt(v.Word, 10, 64); err == nil && n >= -(1<<53) && n <= (1<<53) {
			return n, true
		}
	}
	return 0, false
}

// resultat d'un calcul entier deja en int64 : float64 tant que la valeur tient
// exactement (<= 2^53), sinon entier exact (KInt). evite le new(big.Int).Abs de
// intResult dans le cas courant.
func intResultFromInt64(n int64) Value {
	if n >= -(1<<53) && n <= (1<<53) {
		return NumberValue(float64(n))
	}
	return IntValue(big.NewInt(n))
}

// addition entiere : chemin int64 rapide (0 allocation) si les deux operandes
// sont de petits entiers, sinon voie big. somme de deux valeurs <= 2^53 tient
// toujours dans un int64 (<= 2^54), pas de debordement a craindre.
func addInt(l, r Value) (Value, bool) {
	if x, ok := asSmallInt(l); ok {
		if y, ok2 := asSmallInt(r); ok2 {
			return intResultFromInt64(x + y), true
		}
	}
	return intBin(l, r, (*big.Int).Add)
}

func subInt(l, r Value) (Value, bool) {
	if x, ok := asSmallInt(l); ok {
		if y, ok2 := asSmallInt(r); ok2 {
			return intResultFromInt64(x - y), true
		}
	}
	return intBin(l, r, (*big.Int).Sub)
}

// produit entier : le produit de deux valeurs <= 2^53 peut depasser l'int64, on
// detecte le debordement (p/x != y) et on ne fabrique un big.Int qu'a ce moment.
func mulInt(l, r Value) (Value, bool) {
	if x, ok := asSmallInt(l); ok {
		if y, ok2 := asSmallInt(r); ok2 {
			p := x * y
			if x == 0 || p/x == y {
				return intResultFromInt64(p), true
			}
			return IntValue(new(big.Int).Mul(big.NewInt(x), big.NewInt(y))), true
		}
	}
	return intBin(l, r, (*big.Int).Mul)
}

// l < r en traitant les deux comme des nombres si possible (sans allouer dans le
// cas courant). pour le tri de tableau (ORDONNE) : meme ordre que TRIE/datumLess,
// mais sans passer par valueToDatum ni big.Int sur des nombres ordinaires.
func valueLess(a, b Value) bool {
	if x, ok := asSmallInt(a); ok {
		if y, ok2 := asSmallInt(b); ok2 {
			return x < y
		}
	}
	if a.Kind == KNumber && b.Kind == KNumber {
		return a.Num < b.Num
	}
	if c, ok := intCmp(a, b); ok { // grands entiers exacts
		return c < 0
	}
	an, aerr := toNumber(a)
	bn, berr := toNumber(b)
	if aerr == nil && berr == nil {
		return an < bn
	}
	return a.String() < b.String()
}

// applique un operateur binaire entier (Add, Sub, Mul...) si les deux operandes
// sont des entiers exacts. rend (resultat, true) dans ce cas, (_, false) sinon
// (l'appelant retombe alors sur le calcul flottant).
func intBin(l, r Value, f func(z, x, y *big.Int) *big.Int) (Value, bool) {
	x, ok1 := asIntOperand(l)
	y, ok2 := asIntOperand(r)
	if !ok1 || !ok2 {
		return Value{}, false
	}
	return intResult(f(new(big.Int), x, y)), true
}

// les deux arguments vus comme entiers exacts, si les deux en sont (sinon false).
// pour QUOT/RESTE/MODULO qui ont leur propre garde-fou (division par zero).
func twoInts(a []Value) (*big.Int, *big.Int, bool) {
	x, ok1 := asIntOperand(a[0])
	y, ok2 := asIntOperand(a[1])
	if !ok1 || !ok2 {
		return nil, nil, false
	}
	return x, y, true
}

// compare deux valeurs comme entiers exacts si elles en sont. rend (-1/0/1, true)
// ou (_, false) si l'une n'est pas entiere.
func intCmp(l, r Value) (int, bool) {
	x, ok1 := asIntOperand(l)
	y, ok2 := asIntOperand(r)
	if !ok1 || !ok2 {
		return 0, false
	}
	return x.Cmp(y), true
}

// ENT/ARRONDI/TRONQUE sur un entier, c'est l'identite (deja entier). sinon on
// applique la fonction flottante habituelle (Floor/Round/Trunc).
func intIdentityOr(a []Value, f func(float64) float64) (Value, error) {
	if b, ok := asIntOperand(a[0]); ok {
		return intResult(b), nil
	}
	return num1(a, f)
}

// -1/0/1 selon que x est plus petit, egal ou plus grand que y (facon big.Cmp)
func cmpInt64(x, y int64) int {
	switch {
	case x < y:
		return -1
	case x > y:
		return 1
	}
	return 0
}

// un resultat de comparaison (c = -1/0/1, facon big.Cmp) satisfait-il l'operateur ?
func cmpHolds(op string, c int) bool {
	switch op {
	case "<":
		return c < 0
	case ">":
		return c > 0
	case "<=":
		return c <= 0
	case ">=":
		return c >= 0
	}
	return false
}
