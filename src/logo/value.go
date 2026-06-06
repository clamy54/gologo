// Package logo : l'interpreteur Logo (lecture, valeurs, evaluateur, primitives).
// aucune dependance graphique, le rendu passe par turtle.Canvas
package logo

import (
	"math"
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
	List []Datum // contenu d'une liste (elements non evalues, re-executables)
}

func NumberValue(n float64) Value { return Value{Kind: KNumber, Num: n} }

// chiffres significatifs pour arrondir les resultats et gommer le bruit flottant
// 11 = compromis lisibilite / precision
const sigDigits = 11

// arrondit x a sigDigits chiffres significatifs (gomme le bruit flottant)
// laisse tranquille 0, l'infini et NaN
func roundSig(x float64) float64 {
	if x == 0 || math.IsNaN(x) || math.IsInf(x, 0) {
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
	}
	return ""
}

// nombre facon Logo : entier sans decimale, sinon decimale minimale
func formatNumber(n float64) string {
	if n == float64(int64(n)) {
		return strconv.FormatInt(int64(n), 10)
	}
	return strconv.FormatFloat(n, 'g', -1, 64)
}
