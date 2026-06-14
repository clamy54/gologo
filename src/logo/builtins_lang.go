package logo

import (
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"sort"
	"strings"
	"time"
)

// operations, reporters tortue et formes speciales (POUR, SI, STOP, RENDS, LOGO,
// EXEC, TANTQUE, REPETEPOUR...)
func (i *Interp) registerOperations() {
	op := func(arity int, fn func(*Interp, []Value) (Value, error), names ...string) {
		i.register(&primitive{arity: arity, reporter: true, fn: fn}, names...)
	}
	// operation variadique : arite fixe hors parentheses, mais (OP a b c ...) passe
	// tous les arguments a fn
	vop := func(arity int, fn func(*Interp, []Value) (Value, error), names ...string) {
		i.register(&primitive{arity: arity, reporter: true, variadic: true, fn: fn}, names...)
	}

	// arithmetique
	vop(2, func(in *Interp, a []Value) (Value, error) {
		return reduceNums(a, 0, addInt, func(x, y float64) float64 { return x + y })
	}, "SOMME")
	op(2, func(in *Interp, a []Value) (Value, error) {
		if v, ok := subInt(a[0], a[1]); ok {
			return v, nil
		}
		return numOp(a, func(x, y float64) float64 { return x - y })
	}, "DIFF", "DIF")
	vop(2, func(in *Interp, a []Value) (Value, error) {
		return reduceNums(a, 1, mulInt, func(x, y float64) float64 { return x * y })
	}, "PROD", "PRODUIT")
	op(2, func(in *Interp, a []Value) (Value, error) {
		x, y, err := twoNums(a)
		if err != nil {
			return Value{}, err
		}
		if y == 0 {
			return Value{}, fmt.Errorf("DIV N'AIME PAS 0")
		}
		return numResult(x / y), nil
	}, "DIV", "DIVISE")
	op(2, func(in *Interp, a []Value) (Value, error) {
		if bx, by, ok := twoInts(a); ok {
			if by.Sign() == 0 {
				return Value{}, fmt.Errorf("QUOT N'AIME PAS 0")
			}
			return intResult(new(big.Int).Quo(bx, by)), nil // tronque vers zero
		}
		x, y, err := twoNums(a)
		if err != nil {
			return Value{}, err
		}
		if y == 0 {
			return Value{}, fmt.Errorf("QUOT N'AIME PAS 0")
		}
		return numResult(math.Trunc(x / y)), nil
	}, "QUOT")
	op(2, func(in *Interp, a []Value) (Value, error) {
		if bx, by, ok := twoInts(a); ok {
			if by.Sign() == 0 {
				return Value{}, fmt.Errorf("RESTE N'AIME PAS 0")
			}
			return intResult(new(big.Int).Rem(bx, by)), nil // reste du signe de bx
		}
		x, y, err := twoNums(a)
		if err != nil {
			return Value{}, err
		}
		if y == 0 {
			return Value{}, fmt.Errorf("RESTE N'AIME PAS 0")
		}
		return numResult(math.Mod(x, y)), nil
	}, "RESTE")
	op(1, func(in *Interp, a []Value) (Value, error) { return intIdentityOr(a, math.Floor) }, "ENT")
	op(1, func(in *Interp, a []Value) (Value, error) {
		return num1(a, func(x float64) float64 { return math.Sin(x * math.Pi / 180) })
	}, "SIN", "SINUS")
	op(1, func(in *Interp, a []Value) (Value, error) {
		return num1(a, func(x float64) float64 { return math.Cos(x * math.Pi / 180) })
	}, "COS", "COSINUS")
	op(1, func(in *Interp, a []Value) (Value, error) {
		x, err := toNumber(a[0])
		if err != nil {
			return Value{}, err
		}
		if x < 0 {
			return Value{}, fmt.Errorf("RC N'AIME PAS %s", a[0].String())
		}
		return numResult(math.Sqrt(x)), nil
	}, "RC", "RACINE", "RAC")
	op(1, func(in *Interp, a []Value) (Value, error) {
		x, err := toNumber(a[0])
		if err != nil {
			return Value{}, err
		}
		return numResult(math.Exp(x)), nil
	}, "EXP")
	op(1, func(in *Interp, a []Value) (Value, error) {
		x, err := toNumber(a[0])
		if err != nil {
			return Value{}, err
		}
		if x <= 0 {
			return Value{}, fmt.Errorf("LN N'AIME PAS %s", a[0].String())
		}
		return numResult(math.Log(x)), nil
	}, "LN")
	op(1, func(in *Interp, a []Value) (Value, error) {
		x, err := toNumber(a[0])
		if err != nil {
			return Value{}, err
		}
		if x <= 0 {
			return Value{}, fmt.Errorf("LOG10 N'AIME PAS %s", a[0].String())
		}
		return numResult(math.Log10(x)), nil
	}, "LOG10")
	// HASARD n -> 0..n-1 ; (HASARD debut fin) -> debut..fin inclus (comme RANDOM)
	vop(1, func(in *Interp, a []Value) (Value, error) {
		if len(a) >= 2 {
			lo, err := intArg(a[0])
			if err != nil {
				return Value{}, err
			}
			hi, err := intArg(a[1])
			if err != nil {
				return Value{}, err
			}
			// largeur en int64 pour reperer un intervalle vide ou un debordement
			// (bornes tres ecartees), sinon randN recevrait un n negatif et planterait
			width := int64(hi) - int64(lo) + 1
			if width < 1 || width != int64(int(width)) {
				return Value{}, fmt.Errorf("HASARD N'AIME PAS %s", a[1].String())
			}
			return NumberValue(float64(lo + in.randN(int(width)))), nil
		}
		n, err := intArg(a[0])
		if err != nil {
			return Value{}, err
		}
		if n < 1 {
			return Value{}, fmt.Errorf("HASARD N'AIME PAS %s", a[0].String())
		}
		return NumberValue(float64(in.randN(n))), nil
	}, "HASARD")

	// arithmetique : extensions
	op(2, func(in *Interp, a []Value) (Value, error) {
		// exposant entier >= 0 sur une base entiere : puissance exacte en big,
		// si le resultat reste sous maxPowBits (sinon calcul flottant, donc Inf
		// pour les cas demesures - comme avant la v2)
		if bx, by, ok := twoInts(a); ok && by.Sign() >= 0 {
			if bits := bx.BitLen(); bits <= 1 || (by.IsInt64() && by.Int64() <= int64(maxPowBits/bits)) {
				return intResult(new(big.Int).Exp(bx, by, nil)), nil
			}
		}
		return numOp(a, math.Pow) // PUISSANCE a b = a^b
	}, "PUISSANCE")
	op(1, func(in *Interp, a []Value) (Value, error) {
		return intIdentityOr(a, math.Round) // ARRONDI : entier le plus proche
	}, "ARRONDI")
	op(1, func(in *Interp, a []Value) (Value, error) {
		return intIdentityOr(a, math.Trunc) // TRONQUE : partie entiere (vers zero)
	}, "TRONQUE")
	op(1, func(in *Interp, a []Value) (Value, error) {
		if b, ok := asIntOperand(a[0]); ok { // oppose exact d'un entier
			return intResult(new(big.Int).Neg(b)), nil
		}
		return num1(a, func(x float64) float64 { return -x }) // MOINS : oppose
	}, "MOINS")
	op(1, func(in *Interp, a []Value) (Value, error) {
		return num1(a, func(x float64) float64 { return math.Tan(x * math.Pi / 180) })
	}, "TANGENTE", "TAN")
	op(1, func(in *Interp, a []Value) (Value, error) { return reverse(a[0]) }, "INVERSE")
	op(0, func(in *Interp, a []Value) (Value, error) { return NumberValue(math.Pi), nil }, "PI")
	// OUEX a b : ou exclusif bit a bit de deux entiers >= 0. petit chemin int64, sinon big
	op(2, func(in *Interp, a []Value) (Value, error) {
		bx, ok := asIntOperand(a[0])
		if !ok {
			return Value{}, &badData{a[0].String()}
		}
		by, ok := asIntOperand(a[1])
		if !ok {
			return Value{}, &badData{a[1].String()}
		}
		// contrat : entiers non negatifs (sinon le complement infini de big.Int
		// donnerait un resultat negatif, contraire a la doc)
		if bx.Sign() < 0 {
			return Value{}, fmt.Errorf("OUEX N'AIME PAS %s", a[0].String())
		}
		if by.Sign() < 0 {
			return Value{}, fmt.Errorf("OUEX N'AIME PAS %s", a[1].String())
		}
		if bx.IsInt64() && by.IsInt64() {
			return intResultFromInt64(bx.Int64() ^ by.Int64()), nil
		}
		return intResult(new(big.Int).Xor(bx, by)), nil
	}, "OUEX")
	// VERSBASE n base : ecrit l'entier n dans la base donnee (2 a 36)
	op(2, func(in *Interp, a []Value) (Value, error) {
		base, err := baseArg(a[1])
		if err != nil {
			return Value{}, err
		}
		return toBaseWord(a[0], base)
	}, "VERSBASE")
	// DEPUISBASE mot base : lit le mot comme un nombre ecrit dans cette base
	op(2, func(in *Interp, a []Value) (Value, error) {
		base, err := baseArg(a[1])
		if err != nil {
			return Value{}, err
		}
		w, err := toWord(a[0])
		if err != nil {
			return Value{}, &badData{a[0].String()}
		}
		z, ok := new(big.Int).SetString(strings.TrimSpace(w), base)
		if !ok {
			return Value{}, &badData{a[0].String()} // pas un nombre valide dans cette base
		}
		return intResult(z), nil
	}, "DEPUISBASE")
	// raccourcis "kid-friendly" : montre un nombre en hexa ou en binaire
	op(1, func(in *Interp, a []Value) (Value, error) { return toBaseWord(a[0], 16) }, "HEXA")
	op(1, func(in *Interp, a []Value) (Value, error) { return toBaseWord(a[0], 2) }, "BINAIRE")

	// comparaisons
	op(2, func(in *Interp, a []Value) (Value, error) {
		return cmpOp(a, "<", func(x, y float64) bool { return x < y })
	}, "PLP?")
	op(2, func(in *Interp, a []Value) (Value, error) {
		return cmpOp(a, ">", func(x, y float64) bool { return x > y })
	}, "PLG?")
	op(2, func(in *Interp, a []Value) (Value, error) { return BoolValue(valuesEqual(a[0], a[1])), nil }, "EGAL?")

	// logique
	vop(2, func(in *Interp, a []Value) (Value, error) {
		for _, v := range a { // (ET a b c ...) : vrai si tous sont vrais
			if !v.IsTrue() {
				return BoolValue(false), nil
			}
		}
		return BoolValue(true), nil
	}, "ET")
	vop(2, func(in *Interp, a []Value) (Value, error) {
		for _, v := range a { // (OU a b c ...) : vrai si au moins un est vrai
			if v.IsTrue() {
				return BoolValue(true), nil
			}
		}
		return BoolValue(false), nil
	}, "OU")
	op(1, func(in *Interp, a []Value) (Value, error) { return BoolValue(!a[0].IsTrue()), nil }, "NON")
	op(0, func(in *Interp, a []Value) (Value, error) { return BoolValue(true), nil }, "VRAI")
	op(0, func(in *Interp, a []Value) (Value, error) { return BoolValue(false), nil }, "FAUX")

	// mots et listes : examiner
	op(1, func(in *Interp, a []Value) (Value, error) { return BoolValue(isEmpty(a[0])), nil }, "VIDE?")
	op(1, func(in *Interp, a []Value) (Value, error) { return BoolValue(a[0].Kind == KList), nil }, "LISTE?")
	// un tableau n'est ni une liste ni un mot (il a son propre predicat TABLEAU?)
	op(1, func(in *Interp, a []Value) (Value, error) {
		return BoolValue(a[0].Kind != KList && a[0].Kind != KArray), nil
	}, "MOT?")
	op(1, func(in *Interp, a []Value) (Value, error) {
		_, err := toNumber(a[0])
		return BoolValue(err == nil), nil
	}, "NOMBRE?")
	op(2, func(in *Interp, a []Value) (Value, error) {
		if a[1].Kind != KList {
			return Value{}, fmt.Errorf("MEMBRE? N'AIME PAS %s", a[1].String())
		}
		for _, d := range a[1].List {
			ev, _ := datumToValue(d)
			if valuesEqual(a[0], ev) {
				return BoolValue(true), nil
			}
		}
		return BoolValue(false), nil
	}, "MEMBRE?")
	op(1, func(in *Interp, a []Value) (Value, error) {
		w, err := toWord(a[0])
		if err != nil {
			return Value{}, err
		}
		if w == "" {
			return NumberValue(0), nil
		}
		return NumberValue(float64([]rune(w)[0])), nil
	}, "ASCII")
	op(1, func(in *Interp, a []Value) (Value, error) {
		n, err := intArg(a[0])
		if err != nil {
			return Value{}, err
		}
		// point de code Unicode valide : pas de negatif, pas au-dela du max, pas de surrogate
		if n < 0 || n > 0x10FFFF || (n >= 0xD800 && n <= 0xDFFF) {
			return Value{}, fmt.Errorf("CAR N'AIME PAS %s", a[0].String())
		}
		return WordValue(string(rune(n))), nil
	}, "CAR")
	op(1, func(in *Interp, a []Value) (Value, error) {
		switch a[0].Kind {
		case KList:
			return NumberValue(float64(len(a[0].List))), nil
		case KArray:
			return NumberValue(float64(len(a[0].Arr.Items))), nil
		default:
			return NumberValue(float64(len([]rune(a[0].String())))), nil
		}
	}, "COMPTE")

	// mots et listes : demonter
	op(1, func(in *Interp, a []Value) (Value, error) { return firstLast(a[0], true) }, "PREM", "PREMIER")
	op(1, func(in *Interp, a []Value) (Value, error) { return firstLast(a[0], false) }, "DER", "DERNIER")
	op(1, func(in *Interp, a []Value) (Value, error) { return butFirstLast(a[0], true) }, "SP", "SAUFPREMIER")
	op(1, func(in *Interp, a []Value) (Value, error) { return butFirstLast(a[0], false) }, "SD", "SAUFDERNIER")
	op(2, func(in *Interp, a []Value) (Value, error) {
		k, err := intArg(a[0]) // indice entier exact : ITEM 1.9 refuse, pas tronque
		if err != nil {
			return Value{}, err
		}
		if a[1].Kind == KList { // n-ieme membre de la liste
			if k < 1 || k > len(a[1].List) {
				return Value{}, fmt.Errorf("PAS ASSEZ D'ELEMENTS POUR ITEM")
			}
			return datumToValue(a[1].List[k-1])
		}
		if a[1].Kind == KArray { // n-ieme case du tableau (en tenant compte de l'origine)
			arr := a[1].Arr
			idx := k - arr.Origin
			if idx < 0 || idx >= len(arr.Items) {
				return Value{}, fmt.Errorf("PAS ASSEZ D'ELEMENTS POUR ITEM")
			}
			return arr.Items[idx], nil
		}
		r := []rune(a[1].String()) // sinon : n-ieme caractere du mot
		if k < 1 || k > len(r) {
			return Value{}, fmt.Errorf("PAS ASSEZ D'ELEMENTS POUR ITEM")
		}
		return WordValue(string(r[k-1])), nil
	}, "ITEM")

	// PIOCHE liste-ou-mot : un membre (ou caractere) tire au hasard (PICK FMSLogo)
	op(1, func(in *Interp, a []Value) (Value, error) {
		if a[0].Kind == KArray { // un tableau se pioche avec ITEM + HASARD
			return Value{}, &badData{a[0].String()}
		}
		if a[0].Kind == KList {
			if len(a[0].List) == 0 {
				return Value{}, &badData{a[0].String()}
			}
			return datumToValue(a[0].List[in.randN(len(a[0].List))])
		}
		r := []rune(a[0].String())
		if len(r) == 0 {
			return Value{}, &badData{a[0].String()}
		}
		return WordValue(string(r[in.randN(len(r))])), nil
	}, "PIOCHE")

	// mots et listes : construire (variadique entre parentheses)
	vop(2, func(in *Interp, a []Value) (Value, error) {
		var b strings.Builder
		for _, v := range a {
			w, err := toWord(v)
			if err != nil {
				return Value{}, err
			}
			b.WriteString(w)
		}
		return WordValue(b.String()), nil
	}, "MOT")
	vop(2, func(in *Interp, a []Value) (Value, error) {
		var out []Datum
		for _, v := range a {
			if v.Kind == KList {
				out = append(out, v.List...)
			} else {
				out = append(out, valueToDatum(v))
			}
		}
		return ListValue(out), nil
	}, "PH", "PHRASE")
	vop(2, func(in *Interp, a []Value) (Value, error) {
		out := make([]Datum, len(a))
		for k, v := range a {
			out[k] = valueToDatum(v)
		}
		return ListValue(out), nil
	}, "LISTE")
	op(2, func(in *Interp, a []Value) (Value, error) {
		if a[1].Kind != KList {
			return Value{}, fmt.Errorf("MP N'AIME PAS %s", a[1].String())
		}
		return ListValue(append([]Datum{valueToDatum(a[0])}, a[1].List...)), nil
	}, "MP", "METSPREMIER")
	op(2, func(in *Interp, a []Value) (Value, error) {
		if a[1].Kind != KList {
			return Value{}, fmt.Errorf("MD N'AIME PAS %s", a[1].String())
		}
		return ListValue(append(append([]Datum{}, a[1].List...), valueToDatum(a[0]))), nil
	}, "MD", "METSDERNIER")
	// TRIE liste : nombres par valeur, sinon ordre des mots. tri stable
	op(1, func(in *Interp, a []Value) (Value, error) {
		if a[0].Kind != KList {
			return Value{}, fmt.Errorf("TRIE N'AIME PAS %s", a[0].String())
		}
		items := append([]Datum(nil), a[0].List...)
		sort.SliceStable(items, func(i, j int) bool { return datumLess(items[i], items[j]) })
		return ListValue(items), nil
	}, "TRIE")

	// noms
	op(1, func(in *Interp, a []Value) (Value, error) {
		name, err := toWord(a[0])
		if err != nil {
			return Value{}, err
		}
		v, ok := in.lookupVar(name)
		if !ok {
			return Value{}, fmt.Errorf("PAS DE CHOSE DONNEE A %s", strings.ToUpper(name))
		}
		return v, nil
	}, "CHOSE")
	op(1, func(in *Interp, a []Value) (Value, error) {
		name, _ := toWord(a[0])
		_, ok := in.lookupVar(name)
		return BoolValue(ok), nil
	}, "NOM?")

	// examiner les procedures
	op(1, func(in *Interp, a []Value) (Value, error) {
		name, _ := toWord(a[0])
		return BoolValue(in.prims[strings.ToUpper(name)] != nil), nil
	}, "PRIM?")
	op(1, func(in *Interp, a []Value) (Value, error) {
		name, _ := toWord(a[0])
		return BoolValue(in.procs[strings.ToUpper(name)] != nil), nil
	}, "PROC?")

	// reporters tortue
	op(0, func(in *Interp, a []Value) (Value, error) {
		s := in.Turtle.State()
		return ListValue([]Datum{
			{Kind: DNumber, Num: round1(s.X)}, {Kind: DNumber, Num: round1(s.Y)},
		}), nil
	}, "POS")
	op(0, func(in *Interp, a []Value) (Value, error) { return NumberValue(round1(in.Turtle.State().Heading)), nil }, "CAP")
	op(0, func(in *Interp, a []Value) (Value, error) { return NumberValue(float64(in.Turtle.State().Pen)), nil }, "CC")
	op(0, func(in *Interp, a []Value) (Value, error) { return NumberValue(float64(in.Turtle.Background())), nil }, "CF")
	op(0, func(in *Interp, a []Value) (Value, error) {
		if c, set := in.Turtle.FloodColor(); set {
			return NumberValue(float64(c)), nil
		}
		return NumberValue(float64(in.Turtle.State().Pen)), nil // non fixee -> on prend le crayon
	}, "CR")
	op(0, func(in *Interp, a []Value) (Value, error) { return BoolValue(in.Turtle.State().PenDown), nil }, "BC?")
	op(0, func(in *Interp, a []Value) (Value, error) { return BoolValue(in.Turtle.State().Visible), nil }, "VISIBLE?")

	// controle
	i.register(&primitive{name: "POUR", special: formePour}, "POUR")
	i.register(&primitive{name: "ED", special: formeED}, "ED")
	i.register(&primitive{name: "AIDE", special: formeAide}, "AIDE")
	i.register(&primitive{name: "FIN", arity: 0, fn: func(in *Interp, a []Value) (Value, error) {
		return Value{}, fmt.Errorf("FIN N'A DE SENS QU'EN DEFINITION DE PROCEDURE")
	}}, "FIN")
	// SI (une branche) et SISINON (deux branches) partagent formeSi mais sont deux
	// primitives distinctes : chacune a sa propre fiche d'aide (cf. helpData), donc
	// les deux apparaissent dans l'aide debutant. formeSi accepte de toute facon une
	// 2e liste optionnelle, les deux noms se comportent donc pareil a l'execution.
	i.register(&primitive{name: "SI", special: formeSi}, "SI")
	i.register(&primitive{name: "SISINON", special: formeSi}, "SISINON")
	i.register(&primitive{name: "TANTQUE", special: formeTantque}, "TANTQUE")          // while
	i.register(&primitive{name: "REPETEPOUR", special: formeRepetepour}, "REPETEPOUR") // for
	i.register(&primitive{name: "SCENE", special: formeScene}, "SCENE")                // double tampon (anti-clignotement)
	// COMPTEUR : numero de l'iteration REPETE en cours (1..n), -1 hors boucle
	op(0, func(in *Interp, a []Value) (Value, error) {
		if len(in.repStack) == 0 {
			return NumberValue(-1), nil
		}
		return NumberValue(float64(in.repStack[len(in.repStack)-1])), nil
	}, "COMPTEUR")
	i.register(&primitive{name: "STOP", arity: 0, fn: func(in *Interp, a []Value) (Value, error) {
		return None, &ctrl{kind: ctlStop}
	}}, "STOP")
	i.register(&primitive{name: "RENDS", special: formeRends}, "RENDS", "RETOURNE", "RET")
	i.register(&primitive{name: "LOGO", arity: 0, fn: func(in *Interp, a []Value) (Value, error) {
		return None, &ctrl{kind: ctlLogo}
	}}, "LOGO", "STOPTOUT")
	// RAZ : apres confirmation, remet tout a zero (comme au demarrage)
	i.register(cmd(0, func(in *Interp, a []Value) error {
		q := "REINITIALISER ? (O/N) "
		if in.Lang() == "EN" {
			q = "RESET? (Y/N) "
		}
		ok, err := in.confirmYesNo(q)
		if err != nil {
			return err
		}
		if ok {
			in.resetAll()
		}
		return nil
	}), "RAZ")
	// QUITTE : on plie bagage, la tortue rentre se coucher
	i.register(&primitive{name: "QUITTE", arity: 0, fn: func(in *Interp, a []Value) (Value, error) {
		if err := in.closeAllFiles(); err != nil { // on referme proprement avant de plier bagage
			fmt.Fprintln(in.Out, "fermeture fichiers:", err)
		}
		return None, ErrQuitter
	}}, "QUITTE", "QUITTER")
	i.register(&primitive{name: "EXEC", arity: 1, reporter: true, fn: func(in *Interp, a []Value) (Value, error) {
		if a[0].Kind != KList {
			return Value{}, fmt.Errorf("EXEC N'AIME PAS %s", a[0].String())
		}
		return in.evalCode(a[0].List, false)
	}}, "EXEC")

	// EXECRESULTAT liste (RUNRESULT) : comme EXEC mais emballe le resultat dans une
	// liste : [valeur] si c'etait une operation, [] si c'etait une commande. ca permet
	// de distinguer "rien rendu" d'un resultat qui serait lui-meme une liste vide
	i.register(&primitive{name: "EXECRESULTAT", arity: 1, reporter: true, fn: func(in *Interp, a []Value) (Value, error) {
		if a[0].Kind != KList {
			return Value{}, fmt.Errorf("EXECRESULTAT N'AIME PAS %s", a[0].String())
		}
		v, err := in.evalCode(a[0].List, false)
		if err != nil {
			return Value{}, err
		}
		if v.HasValue() {
			return ListValue([]Datum{valueToDatum(v)}), nil
		}
		return ListValue(nil), nil
	}}, "EXECRESULTAT")

	// clavier / souris / manettes : voir registerIODevices (io_devices.go)

	// primitives courantes (Logo standard)
	op(1, func(in *Interp, a []Value) (Value, error) {
		w, err := toWord(a[0])
		if err != nil {
			return Value{}, err
		}
		return WordValue(strings.ToUpper(w)), nil
	}, "MAJUSCULE")
	op(1, func(in *Interp, a []Value) (Value, error) {
		w, err := toWord(a[0])
		if err != nil {
			return Value{}, err
		}
		return WordValue(strings.ToLower(w)), nil
	}, "MINUSCULE")
	op(1, func(in *Interp, a []Value) (Value, error) {
		if b, ok := asIntOperand(a[0]); ok { // valeur absolue exacte d'un entier
			return intResult(new(big.Int).Abs(b)), nil
		}
		x, err := toNumber(a[0])
		if err != nil {
			return Value{}, err
		}
		return numResult(math.Abs(x)), nil
	}, "VALABS")
	op(1, func(in *Interp, a []Value) (Value, error) {
		x, err := toNumber(a[0])
		if err != nil {
			return Value{}, err
		}
		return numResult(math.Atan(x) * 180 / math.Pi), nil
	}, "ARCTAN", "ATAN")
	// MODULO : reste de la division, du signe du diviseur (contrairement a RESTE)
	op(2, func(in *Interp, a []Value) (Value, error) {
		if bx, by, ok := twoInts(a); ok {
			if by.Sign() == 0 {
				return Value{}, fmt.Errorf("MODULO N'AIME PAS 0")
			}
			m := new(big.Int).Rem(bx, by)
			if m.Sign() != 0 && (m.Sign() < 0) != (by.Sign() < 0) {
				m.Add(m, by) // ramene au signe du diviseur
			}
			return intResult(m), nil
		}
		x, err := toNumber(a[0])
		if err != nil {
			return Value{}, err
		}
		y, err := toNumber(a[1])
		if err != nil {
			return Value{}, err
		}
		if y == 0 {
			return Value{}, fmt.Errorf("MODULO N'AIME PAS 0")
		}
		m := math.Mod(x, y)
		if m != 0 && (m < 0) != (y < 0) {
			m += y
		}
		return numResult(m), nil
	}, "MODULO")
	// VERSCAP [x y] : cap (0-360) a prendre pour viser le point depuis la tortue
	op(1, func(in *Interp, a []Value) (Value, error) {
		x, y, err := toCoords(a[0])
		if err != nil {
			return Value{}, err
		}
		st := in.Turtle.State()
		h := math.Atan2(x-st.X, y-st.Y) * 180 / math.Pi
		if h < 0 {
			h += 360
		}
		return NumberValue(round1(h)), nil
	}, "VERSCAP")
	op(0, func(in *Interp, a []Value) (Value, error) { return NumberValue(round1(in.Turtle.State().X)), nil }, "XCOR")
	op(0, func(in *Interp, a []Value) (Value, error) { return NumberValue(round1(in.Turtle.State().Y)), nil }, "YCOR")
	// DISTANCE [x y] : distance de la tortue au point
	op(1, func(in *Interp, a []Value) (Value, error) {
		x, y, err := toCoords(a[0])
		if err != nil {
			return Value{}, err
		}
		st := in.Turtle.State()
		return NumberValue(round1(math.Hypot(x-st.X, y-st.Y))), nil
	}, "DISTANCE")

	// pause
	i.register(cmd(1, func(in *Interp, a []Value) error {
		n, err := toNumber(a[0])
		if err != nil {
			return err
		}
		if n < 0 {
			return fmt.Errorf("ATTENDS N'AIME PAS %s", a[0].String())
		}
		// n soixantiemes de seconde (1 tick = 1/60 s, comme UCBLogo/FMSLogo :
		// ATTENDS 60 = 1 s). on dort par petits pas pour rester interruptible
		// (Ctrl+C met brk a true, on coupe alors avec INTERROMPU !)
		ms := int(math.Round(n * 1000.0 / 60.0))
		for slept := 0; slept < ms; slept += 20 {
			if in.brk.Load() {
				return ErrInterrompu
			}
			step := ms - slept
			if step > 20 {
				step = 20
			}
			time.Sleep(time.Duration(step) * time.Millisecond)
		}
		return nil
	}), "ATTENDS")
}

// randN rend un entier de [0,n) via le RNG local (in.rng) s'il est defini, sinon
// le generateur global (auto-seede a chaque lancement). Le bac a sable de l'aide
// fixe in.rng pour des exemples reproductibles.
func (in *Interp) randN(n int) int {
	if in.rng != nil {
		return in.rng.Intn(n)
	}
	return rand.Intn(n)
}

// convertit une ligne tapee en liste Logo (LL/READLIST). si elle est mal formee
// (crochet pas ferme), chaque mot devient un mot brut, sans erreur
func lineToList(line string) Value {
	data, err := Read(line)
	if err != nil {
		var ds []Datum
		for _, w := range strings.Fields(line) {
			ds = append(ds, Datum{Kind: DWord, Text: w})
		}
		return ListValue(ds)
	}
	return ListValue(data)
}

// lit un argument qui doit etre une liste (crochets ou groupe)
func (e *eval) listArg(name string) ([]Datum, error) {
	if e.atEnd() {
		return nil, fmt.Errorf("PAS ASSEZ DE DONNEES POUR %s", name)
	}
	d := e.next()
	if d.Kind == DList || d.Kind == DGroup {
		return d.List, nil
	}
	return nil, &badData{d.String()} // -> "SI N'AIME PAS ..." (nom ajoute par invoke)
}

// helpers numeriques / listes

// valeur numerique d'un element s'il en represente une
func datumNum(d Datum) (float64, bool) {
	v, err := datumToValue(d)
	if err != nil {
		return 0, false
	}
	n, err := toNumber(v)
	return n, err == nil
}

// ordonne deux elements pour TRIE : deux nombres par valeur, sinon par leur ecriture
func datumLess(a, b Datum) bool {
	av, ae := datumToValue(a)
	bv, be := datumToValue(b)
	if ae == nil && be == nil {
		if c, ok := intCmp(av, bv); ok { // grands entiers : compare exact
			return c < 0
		}
	}
	if an, aok := datumNum(a); aok {
		if bn, bok := datumNum(b); bok {
			return an < bn
		}
	}
	return a.String() < b.String()
}

// valide une base de numeration : un entier de 2 a 36 (sinon erreur)
func baseArg(v Value) (int, error) {
	n, err := toNumber(v)
	if err != nil {
		return 0, err
	}
	b := int(n)
	if float64(b) != n || b < 2 || b > 36 {
		return 0, &badData{v.String()}
	}
	return b, nil
}

// ecrit l'entier v dans la base donnee, en majuscules (FF plutot que ff)
func toBaseWord(v Value, base int) (Value, error) {
	b, ok := asIntOperand(v)
	if !ok {
		return Value{}, &badData{v.String()}
	}
	return WordValue(strings.ToUpper(b.Text(base))), nil
}

func twoNums(a []Value) (float64, float64, error) {
	x, err := toNumber(a[0])
	if err != nil {
		return 0, 0, err
	}
	y, err := toNumber(a[1])
	if err != nil {
		return 0, 0, err
	}
	return x, y, nil
}

func numOp(a []Value, f func(x, y float64) float64) (Value, error) {
	x, y, err := twoNums(a)
	if err != nil {
		return Value{}, err
	}
	return numResult(f(x, y)), nil
}

// inverse l'ordre des elements d'une liste, ou des caracteres d'un mot (INVERSE)
func reverse(v Value) (Value, error) {
	if v.Kind == KList {
		out := make([]Datum, len(v.List))
		for i, d := range v.List {
			out[len(v.List)-1-i] = d
		}
		return ListValue(out), nil
	}
	w, err := toWord(v)
	if err != nil {
		return Value{}, err
	}
	r := []rune(w)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return WordValue(string(r)), nil
}

// cumule les nombres de a en partant de init (SOMME/PROD variadiques). tant que
// les operandes sont entiers, on replie avec comb (chemin int64/big exact) ; des
// qu'une operande est fractionnaire, on bascule en flottant avec f pour le reste.
func reduceNums(a []Value, init float64, comb func(l, r Value) (Value, bool), f func(x, y float64) float64) (Value, error) {
	acc := NumberValue(init)
	facc := init
	floatMode := false
	for _, v := range a {
		if !floatMode {
			if r, ok := comb(acc, v); ok {
				acc = r
				continue
			}
			// v n'est pas un entier : on passe en flottant pour la suite
			facc, _ = toNumber(acc) // acc est entier, conversion exacte
			floatMode = true
		}
		n, err := toNumber(v)
		if err != nil {
			return Value{}, err
		}
		facc = f(facc, n)
	}
	if floatMode {
		return numResult(facc), nil
	}
	return acc, nil
}

func num1(a []Value, f func(float64) float64) (Value, error) {
	x, err := toNumber(a[0])
	if err != nil {
		return Value{}, err
	}
	return numResult(f(x)), nil
}

// op vaut "<" ou ">" : on compare en entier exact si les deux operandes en sont
// (sinon en flottant). evite que PLP?/PLG? reperdent la precision au-dela de 2^53.
func cmpOp(a []Value, op string, f func(x, y float64) bool) (Value, error) {
	if c, ok := intCmp(a[0], a[1]); ok {
		return BoolValue(cmpHolds(op, c)), nil
	}
	x, y, err := twoNums(a)
	if err != nil {
		return Value{}, err
	}
	return BoolValue(f(x, y)), nil
}

func isEmpty(v Value) bool {
	switch v.Kind {
	case KList:
		return len(v.List) == 0
	case KWord:
		return v.Word == ""
	}
	return false
}

// premier (first=true) ou dernier element/caractere
func firstLast(v Value, first bool) (Value, error) {
	if v.Kind == KArray { // un tableau s'indexe avec ITEM, pas avec PREM/DER
		return Value{}, &badData{v.String()}
	}
	switch v.Kind {
	case KList:
		if len(v.List) == 0 {
			return Value{}, fmt.Errorf("PREM/DER N'AIME PAS LA LISTE VIDE")
		}
		if first {
			return datumToValue(v.List[0])
		}
		return datumToValue(v.List[len(v.List)-1])
	default:
		r := []rune(v.String())
		if len(r) == 0 {
			return Value{}, fmt.Errorf("PREM/DER N'AIME PAS LE MOT VIDE")
		}
		if first {
			return WordValue(string(r[0])), nil
		}
		return WordValue(string(r[len(r)-1])), nil
	}
}

// la liste/mot sans le premier (first=true) ou sans le dernier element
func butFirstLast(v Value, first bool) (Value, error) {
	if v.Kind == KArray { // pas de "sauf le premier" sur un tableau
		return Value{}, &badData{v.String()}
	}
	switch v.Kind {
	case KList:
		if len(v.List) == 0 {
			return Value{}, fmt.Errorf("SP/SD N'AIME PAS LA LISTE VIDE")
		}
		if first {
			return ListValue(append([]Datum{}, v.List[1:]...)), nil
		}
		return ListValue(append([]Datum{}, v.List[:len(v.List)-1]...)), nil
	default:
		r := []rune(v.String())
		if len(r) == 0 {
			return Value{}, fmt.Errorf("SP/SD N'AIME PAS LE MOT VIDE")
		}
		if first {
			return WordValue(string(r[1:])), nil
		}
		return WordValue(string(r[:len(r)-1])), nil
	}
}

// arrondi au dixieme le plus proche (pas une troncature)
// applique aux entrees AV/TD/FCAP/FPOS/POINT et au rendu de POS/CAP
func round1(x float64) float64 { return math.Round(x*10) / 10 }
