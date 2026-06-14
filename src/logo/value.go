// Package logo : l'interpreteur Logo (lecture, valeurs, evaluateur, primitives).
// aucune dependance graphique, le rendu passe par turtle.Canvas
package logo

import (
	"math"
	"math/big"
	"strconv"
	"strings"
)

// type d'une valeur Logo (nombre, mot, liste...)
type Kind int

const (
	KNone   Kind = iota // "rien", ce que rend une commande (valeur zero)
	KNumber             // nombre (float64)
	KWord               // mot (chaine)
	KList               // liste (suite de Datum non evalues)
	KBool               // booleen (VRAI/FAUX)
	KArray              // tableau (cases mutables), passe par reference
	KInt                // entier exact (au-dela de 2^53, la ou float64 decroche)
)

// ce que rend une commande qui ne produit aucun objet
var None = Value{Kind: KNone}

// vrai objet Logo, par opposition a "rien"
func (v Value) HasValue() bool { return v.Kind != KNone }

// une valeur Logo, type connu seulement a l'execution
type Value struct {
	Kind Kind
	Num  float64
	Word string
	Bool bool
	List []Datum  // contenu d'une liste (elements non evalues, re-executables)
	Arr  *Array   // contenu d'un tableau (KArray)
	Int  *big.Int // valeur d'un entier exact (KInt)
}

// un tableau Logo : des cases evaluees + une origine (indice de la 1re case, 1 par
// defaut). pointeur, donc copier la Value partage le meme tableau (modifier l'un
// modifie l'autre) - exactement comme en FMSLogo. pour dupliquer : COPIETABLEAU
type Array struct {
	Items  []Value
	Origin int
}

func ArrayValue(a *Array) Value { return Value{Kind: KArray, Arr: a} }

func NumberValue(n float64) Value { return Value{Kind: KNumber, Num: n} }

// un entier exact (KInt). cf bignum.go pour la fabrication "intelligente" qui
// rebascule en float64 quand la valeur tient sans perte (intResult)
func IntValue(b *big.Int) Value { return Value{Kind: KInt, Int: b} }

// chiffres significatifs pour arrondir les resultats et gommer le bruit flottant
// 11 = compromis lisibilite / precision
const sigDigits = 11

// 2^53 : au-dela float64 ne compte plus les entiers un par un. en deca, tout
// entier est exact, donc inutile (et nuisible) de le raboter a 11 chiffres.
const maxExactIntFloat64 = float64(1 << 53)

// arrondit x a sigDigits chiffres significatifs (gomme le bruit flottant)
// laisse tranquille 0, l'infini et NaN
func roundSig(x float64) float64 {
	if x == 0 || math.IsNaN(x) || math.IsInf(x, 0) {
		return x
	}
	// un entier qui tient exactement dans float64 : on n'y touche pas, l'arrondi
	// a 11 chiffres ne servait qu'a gommer le bruit des fractions (0.1*10=1)
	if x == math.Trunc(x) && math.Abs(x) <= maxExactIntFloat64 {
		return x
	}
	mag := math.Floor(math.Log10(math.Abs(x)))
	factor := math.Pow(10, float64(sigDigits-1)-mag)
	if factor == 0 || math.IsInf(factor, 0) { // garde-fou pour les valeurs extremes
		return x
	}
	return math.Round(x*factor) / factor
}

// valeur nombre issue d'un calcul, arrondie (cf sigDigits/roundSig)
func numResult(n float64) Value { return NumberValue(roundSig(n)) }

func WordValue(s string) Value { return Value{Kind: KWord, Word: s} }

func BoolValue(b bool) Value { return Value{Kind: KBool, Bool: b} }

func ListValue(items []Datum) Value { return Value{Kind: KList, List: items} }

// veracite Logo d'une valeur (pour SI/TANTQUE)
func (v Value) IsTrue() bool {
	switch v.Kind {
	case KBool:
		return v.Bool
	case KWord:
		return strings.EqualFold(v.Word, "VRAI")
	default:
		return false
	}
}

// la valeur telle que Logo l'afficherait (ECRIS)
func (v Value) String() string {
	switch v.Kind {
	case KNumber:
		return formatNumber(v.Num)
	case KInt:
		return v.Int.String()
	case KWord:
		return v.Word
	case KBool:
		if v.Bool {
			return "VRAI"
		}
		return "FAUX"
	case KList:
		parts := make([]string, len(v.List))
		for i, d := range v.List {
			parts[i] = d.String()
		}
		return strings.Join(parts, " ")
	case KArray:
		return v.Arr.String()
	}
	return ""
}

// un tableau facon FMSLogo : { a b c }, avec @origine si elle n'est pas 1
func (a *Array) String() string {
	parts := make([]string, len(a.Items))
	for i, e := range a.Items {
		parts[i] = e.String()
	}
	s := "{" + strings.Join(parts, " ") + "}"
	if a.Origin != 1 {
		s += "@" + strconv.Itoa(a.Origin)
	}
	return s
}

// petits entiers deja formates : une conversion de petit entier en mot (indices
// de boucle, coordonnees de grille, chiffres d'une cle...) revient tres souvent
// dans les boucles serrees. les pre-formater evite une allocation a chaque fois.
var smallInts [1024]string

func init() {
	for i := range smallInts {
		smallInts[i] = strconv.Itoa(i)
	}
}

// nombre facon Logo : entier sans decimale, sinon decimale minimale
func formatNumber(n float64) string {
	if n == float64(int64(n)) {
		i := int64(n)
		if i >= 0 && i < int64(len(smallInts)) {
			return smallInts[i]
		}
		return strconv.FormatInt(i, 10)
	}
	return strconv.FormatFloat(n, 'g', -1, 64)
}
