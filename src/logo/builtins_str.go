package logo

import (
	"fmt"
	"strings"
)

// primitives de manipulation de chaines (et de listes) ajoutees en v2.0.
// recherche facon FMSLogo (MEMBRE/SOUSCHAINE?/AVANT?), plus quelques extensions
// gologo (DECOUPE, ROGNE, SUBSTITUE, TRANCHE). tout compte en runes, pas en octets.
func (i *Interp) registerStrings() {
	op := func(arity int, fn func(*Interp, []Value) (Value, error), names ...string) {
		i.register(&primitive{arity: arity, reporter: true, fn: fn}, names...)
	}
	// variadique : arite fixe hors parentheses, le reste passe par (OP ...)
	vop := func(arity int, fn func(*Interp, []Value) (Value, error), names ...string) {
		i.register(&primitive{arity: arity, reporter: true, variadic: true, fn: fn}, names...)
	}

	// MEMBRE chose1 chose2 : le reste de chose2 a partir de la 1re occurrence de
	// chose1 (mot/liste vide si absent). cle pour retrouver une position.
	op(2, func(in *Interp, a []Value) (Value, error) { return member(a[0], a[1]) }, "MEMBRE")

	// SOUSCHAINE? chose1 chose2 : VRAI si chose1 apparait dans le mot chose2
	op(2, func(in *Interp, a []Value) (Value, error) {
		if a[0].Kind == KList || a[1].Kind == KList {
			return BoolValue(false), nil
		}
		// comme EGAL?, on ignore la casse
		return BoolValue(strings.Contains(strings.ToLower(a[1].String()), strings.ToLower(a[0].String()))), nil
	}, "SOUSCHAINE?")

	// AVANT? mot1 mot2 : VRAI si mot1 vient avant mot2 (ordre alphabetique, casse ignoree)
	op(2, func(in *Interp, a []Value) (Value, error) {
		return BoolValue(strings.ToLower(a[0].String()) < strings.ToLower(a[1].String())), nil
	}, "AVANT?")

	// DECOUPE mot separateur : liste des morceaux. le separateur peut etre un
	// caractere ou une chaine entiere.
	// les morceaux vides sont gardes (deux separateurs de suite -> un trou)
	op(2, func(in *Interp, a []Value) (Value, error) {
		sep := a[1].String()
		if sep == "" {
			return Value{}, fmt.Errorf("DECOUPE VEUT UN SEPARATEUR NON VIDE")
		}
		w, err := toWord(a[0])
		if err != nil {
			return Value{}, err
		}
		parts := strings.Split(w, sep)
		out := make([]Datum, len(parts))
		for k, p := range parts {
			out[k] = valueToDatum(WordValue(p))
		}
		return ListValue(out), nil
	}, "DECOUPE")

	// ROGNE / ROGNEDEBUT / ROGNEFIN : enleve les blancs (espaces, tabulations)
	// en debut et fin, en debut seul, en fin seul. les mots a espaces viennent de
	// l'entree (LISLIGNE, LISMOT), un mot litteral n'en porte pas
	op(1, func(in *Interp, a []Value) (Value, error) { return trimWord(a[0], strings.Trim) }, "ROGNE")
	op(1, func(in *Interp, a []Value) (Value, error) { return trimWord(a[0], strings.TrimLeft) }, "ROGNEDEBUT")
	op(1, func(in *Interp, a []Value) (Value, error) { return trimWord(a[0], strings.TrimRight) }, "ROGNEFIN")

	// SUBSTITUE ancien nouveau chaine : remplace toutes les occurrences de ancien.
	// (SUBSTITUE ancien nouveau chaine n) en remplace au plus n. ancien vide ->
	// erreur, sinon on tournerait en rond a ne jamais avancer
	vop(3, func(in *Interp, a []Value) (Value, error) {
		old, err := toWord(a[0])
		if err != nil {
			return Value{}, err
		}
		if old == "" {
			return Value{}, fmt.Errorf("SUBSTITUE N'AIME PAS LE MOT VIDE")
		}
		nouveau, err := toWord(a[1])
		if err != nil {
			return Value{}, err
		}
		s, err := toWord(a[2])
		if err != nil {
			return Value{}, err
		}
		n := -1 // -1 = toutes
		if len(a) >= 4 {
			c, err := intArg(a[3])
			if err != nil {
				return Value{}, err
			}
			n = c
		}
		return WordValue(strings.Replace(s, old, nouveau, n)), nil
	}, "SUBSTITUE")

	// OTETOUT chose elements : elements (liste ou mot) prive de tous les membres
	// egaux a chose. sur un mot, chose doit etre un seul caractere
	op(2, func(in *Interp, a []Value) (Value, error) { return removeAll(a[0], a[1]) }, "OTETOUT")

	// SANSDOUBLONS elements : copie sans doublons, on garde l'occurrence la plus a droite
	op(1, func(in *Interp, a []Value) (Value, error) { return remDup(a[0]) }, "SANSDOUBLONS")

	// TRANCHE sequence debut fin : la sous-sequence de debut a fin inclus (indices
	// a partir de 1, comme ITEM). (TRANCHE sequence debut) va jusqu'au bout.
	// le type suit l'entree, les bornes hors limites sont ramenees proprement
	vop(3, func(in *Interp, a []Value) (Value, error) { return slice(a) }, "TRANCHE")

	// REMPLACE seq n arg : seq avec l'element n remplace par arg (repris de XLogo).
	// version non destructive, rend une nouvelle valeur ; le type suit l'entree
	op(3, func(in *Interp, a []Value) (Value, error) { return replaceItem(a[0], a[1], a[2]) }, "REMPLACE")

	// AJOUTE seq n arg : insere arg en position n dans seq (repris de XLogo).
	// AJOUTE [ A B C ] 2 8 -> [ A 8 B C ]
	op(3, func(in *Interp, a []Value) (Value, error) { return insertItem(a[0], a[1], a[2]) }, "AJOUTE")

	// OTE seq debut longueur : seq prive de longueur elements a partir de debut
	// (indices a partir de 1). retire par POSITION, pas par valeur (cf OTETOUT)
	op(3, func(in *Interp, a []Value) (Value, error) { return deleteRange(a[0], a[1], a[2]) }, "OTE")

	// POSITION chose chaine : rang (a partir de 1) de la 1re occurrence, 0 si absent.
	// dans un mot, chose peut etre une sous-chaine ; dans une liste, un element
	op(2, func(in *Interp, a []Value) (Value, error) { return position(a[0], a[1]) }, "POSITION")
}

// le reste de chose2 depuis la 1re occurrence de chose1 (cf MEMBRE)
func member(thing1, thing2 Value) (Value, error) {
	if thing2.Kind == KList {
		for idx, d := range thing2.List {
			ev, _ := datumToValue(d)
			if valuesEqual(thing1, ev) {
				return ListValue(append([]Datum{}, thing2.List[idx:]...)), nil
			}
		}
		return ListValue(nil), nil
	}
	// sur un mot : on cherche un caractere (comme MEMBRE? cote FMSLogo)
	r1 := []rune(thing1.String())
	if len(r1) != 1 {
		return WordValue(""), nil
	}
	r2 := []rune(thing2.String())
	for idx, c := range r2 {
		if strings.EqualFold(string(c), string(r1[0])) {
			return WordValue(string(r2[idx:])), nil
		}
	}
	return WordValue(""), nil
}

// applique une fonction de rognage (Trim/TrimLeft/TrimRight) sur les blancs
func trimWord(v Value, f func(string, string) string) (Value, error) {
	w, err := toWord(v)
	if err != nil {
		return Value{}, err
	}
	return WordValue(f(w, " \t")), nil
}

// elements prive de tous les membres egaux a chose (OTETOUT)
func removeAll(chose, elements Value) (Value, error) {
	if elements.Kind == KList {
		out := []Datum{}
		for _, d := range elements.List {
			ev, _ := datumToValue(d)
			if !valuesEqual(chose, ev) {
				out = append(out, d)
			}
		}
		return ListValue(out), nil
	}
	// sur un mot : on retire des caracteres, donc chose doit tenir en un seul
	r := []rune(elements.String())
	target := chose.String()
	var b strings.Builder
	for _, c := range r {
		if !strings.EqualFold(string(c), target) {
			b.WriteRune(c)
		}
	}
	return WordValue(b.String()), nil
}

// copie sans doublons, l'occurrence la plus a droite l'emporte (SANSDOUBLONS)
func remDup(v Value) (Value, error) {
	if v.Kind == KList {
		out := []Datum{}
		for i, d := range v.List {
			ev, _ := datumToValue(d)
			dernier := true // ce membre est-il sa derniere occurrence ?
			for _, e2 := range v.List[i+1:] {
				ev2, _ := datumToValue(e2)
				if valuesEqual(ev, ev2) {
					dernier = false
					break
				}
			}
			if dernier {
				out = append(out, d)
			}
		}
		return ListValue(out), nil
	}
	r := []rune(v.String())
	var b strings.Builder
	for i, c := range r {
		dernier := true
		for _, c2 := range r[i+1:] {
			if strings.EqualFold(string(c), string(c2)) {
				dernier = false
				break
			}
		}
		if dernier {
			b.WriteRune(c)
		}
	}
	return WordValue(b.String()), nil
}

// sous-sequence [debut..fin] d'un mot, d'une liste ou d'un tableau (TRANCHE)
func slice(a []Value) (Value, error) {
	debut, err := intArg(a[1])
	if err != nil {
		return Value{}, err
	}
	if a[0].Kind == KArray { // tableau -> nouveau tableau d'origine 1 (copie fraiche)
		arr := a[0].Arr
		src := arr.Items
		n := len(src)
		fin := arr.Origin + n - 1 // borne haute par defaut : la derniere case
		if len(a) >= 3 {
			f, err := intArg(a[2])
			if err != nil {
				return Value{}, err
			}
			fin = f
		}
		// indices exprimes dans l'origine du tableau (comme ITEM), ramenes en positionnel
		lo, hi := clampRange(debut-arr.Origin+1, fin-arr.Origin+1, n)
		if lo > hi {
			return ArrayValue(&Array{Items: []Value{}, Origin: 1}), nil
		}
		return ArrayValue(&Array{Items: append([]Value{}, src[lo-1:hi]...), Origin: 1}), nil
	}
	if a[0].Kind == KList {
		n := len(a[0].List)
		fin := n
		if len(a) >= 3 {
			f, err := intArg(a[2])
			if err != nil {
				return Value{}, err
			}
			fin = f
		}
		lo, hi := clampRange(debut, fin, n)
		if lo > hi {
			return ListValue(nil), nil
		}
		return ListValue(append([]Datum{}, a[0].List[lo-1:hi]...)), nil
	}
	r := []rune(a[0].String())
	n := len(r)
	fin := n
	if len(a) >= 3 {
		f, err := intArg(a[2])
		if err != nil {
			return Value{}, err
		}
		fin = f
	}
	lo, hi := clampRange(debut, fin, n)
	if lo > hi {
		return WordValue(""), nil
	}
	return WordValue(string(r[lo-1 : hi])), nil
}

// ramene [debut..fin] dans [1..n] sans rouspeter (bornes hors limites tolerees)
func clampRange(debut, fin, n int) (int, int) {
	if debut < 1 {
		debut = 1
	}
	if fin > n {
		fin = n
	}
	return debut, fin
}

// seq avec l'element n remplace par arg (REMPLACE)
func replaceItem(seq, nv, arg Value) (Value, error) {
	if seq.Kind == KArray { // sur un tableau, c'est FIXEITEM (mutation en place)
		return Value{}, &badData{seq.String()}
	}
	n, err := intArg(nv)
	if err != nil {
		return Value{}, err
	}
	k := n
	if seq.Kind == KList {
		if k < 1 || k > len(seq.List) {
			return Value{}, fmt.Errorf("PAS ASSEZ D'ELEMENTS POUR REMPLACE")
		}
		out := append([]Datum{}, seq.List...)
		out[k-1] = valueToDatum(arg)
		return ListValue(out), nil
	}
	r := []rune(seq.String())
	if k < 1 || k > len(r) {
		return Value{}, fmt.Errorf("PAS ASSEZ D'ELEMENTS POUR REMPLACE")
	}
	w, err := toWord(arg)
	if err != nil {
		return Value{}, err
	}
	return WordValue(string(r[:k-1]) + w + string(r[k:])), nil
}

// arg insere en position n dans seq (AJOUTE) ; n va de 1 a longueur+1
func insertItem(seq, nv, arg Value) (Value, error) {
	if seq.Kind == KArray { // un tableau a une taille fixe : pas d'insertion
		return Value{}, &badData{seq.String()}
	}
	n, err := intArg(nv)
	if err != nil {
		return Value{}, err
	}
	k := n
	if seq.Kind == KList {
		if k < 1 || k > len(seq.List)+1 {
			return Value{}, fmt.Errorf("POSITION INVALIDE POUR AJOUTE")
		}
		out := append([]Datum{}, seq.List[:k-1]...)
		out = append(out, valueToDatum(arg))
		out = append(out, seq.List[k-1:]...)
		return ListValue(out), nil
	}
	r := []rune(seq.String())
	if k < 1 || k > len(r)+1 {
		return Value{}, fmt.Errorf("POSITION INVALIDE POUR AJOUTE")
	}
	w, err := toWord(arg)
	if err != nil {
		return Value{}, err
	}
	return WordValue(string(r[:k-1]) + w + string(r[k-1:])), nil
}

// seq prive de longueur elements a partir de debut (OTE). plage hors limites
// ramenee proprement ; une longueur <= 0 ne retire rien
func deleteRange(seq, debutv, longv Value) (Value, error) {
	if seq.Kind == KArray { // un tableau a une taille fixe : pas de suppression
		return Value{}, &badData{seq.String()}
	}
	debut, err := intArg(debutv)
	if err != nil {
		return Value{}, err
	}
	long, err := intArg(longv)
	if err != nil {
		return Value{}, err
	}
	if seq.Kind == KList {
		n := len(seq.List)
		lo, hi := clampDelete(debut, long, n)
		if long <= 0 || lo > hi {
			return ListValue(append([]Datum{}, seq.List...)), nil
		}
		out := append([]Datum{}, seq.List[:lo-1]...)
		return ListValue(append(out, seq.List[hi:]...)), nil
	}
	r := []rune(seq.String())
	n := len(r)
	lo, hi := clampDelete(debut, long, n)
	if long <= 0 || lo > hi {
		return WordValue(string(r)), nil
	}
	return WordValue(string(r[:lo-1]) + string(r[hi:])), nil
}

// bornes 1-based [lo,hi] des elements a retirer (OTE), sans jamais deborder l'int :
// une longueur plus grande que la sequence retire simplement jusqu'au bout
func clampDelete(debut, long, n int) (int, int) {
	lo := debut
	if lo < 1 {
		lo = 1
	}
	if long > n { // plus long que toute la sequence : on vise la fin, pas de lo+long
		return lo, n
	}
	hi := lo + long - 1
	if hi > n {
		hi = n
	}
	return lo, hi
}

// rang de la 1re occurrence de chose, 0 si absent (POSITION). sous-chaine dans
// un mot, element dans une liste ; casse ignoree comme partout ailleurs
func position(chose, chaine Value) (Value, error) {
	if chaine.Kind == KList {
		for idx, d := range chaine.List {
			ev, _ := datumToValue(d)
			if valuesEqual(chose, ev) {
				return NumberValue(float64(idx + 1)), nil
			}
		}
		return NumberValue(0), nil
	}
	r := []rune(chaine.String())
	sub := []rune(chose.String())
	if len(sub) == 0 { // une sous-chaine vide n'a pas de position : absente (0)
		return NumberValue(0), nil
	}
	for i := 0; i+len(sub) <= len(r); i++ {
		ok := true
		for j := range sub {
			if !strings.EqualFold(string(r[i+j]), string(sub[j])) {
				ok = false
				break
			}
		}
		if ok {
			return NumberValue(float64(i + 1)), nil
		}
	}
	return NumberValue(0), nil
}
