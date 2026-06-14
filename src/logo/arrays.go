package logo

import (
	"fmt"
	"sort"
	"strings"
)

// plafond du nombre de cases d'un tableau : tres au-dela de tout usage reel, mais
// borne l'allocation pour qu'un TABLEAU 1e18 rende une erreur au lieu de paniquer
// (makeslice) ou de manger toute la memoire
const maxArrayCells = 10_000_000

// le cout d'un tableau MD tient-il sous le plafond ? on compte les cases de TOUS
// les etages, pas seulement le produit final : TABLEAUMD [ 10000000 1 ] a un
// produit de 10M mais fabrique aussi 10M de sous-tableaux. une taille < 1 est
// laissee a buildMD (erreur dediee) ; multiplications protegees de l'overflow
func mdSizeOK(sizes []int) bool {
	prod := 1  // produit des tailles deja vues
	total := 0 // cases allouees en tout, sous-tableaux compris
	for _, s := range sizes {
		if s < 1 {
			return true
		}
		if s > maxArrayCells || prod > maxArrayCells/s {
			return false
		}
		prod *= s
		total += prod
		if total > maxArrayCells {
			return false
		}
	}
	return true
}

// vrais tableaux (acces direct), piles et files, calques sur FMSLogo. ajoutes en
// v2.0. un tableau est mutable et passe par reference (cf type Array dans value.go) ;
// les piles/files modifient en place une variable contenant une liste.
func (i *Interp) registerArrays() {
	op := func(arity int, fn func(*Interp, []Value) (Value, error), names ...string) {
		i.register(&primitive{arity: arity, reporter: true, fn: fn}, names...)
	}
	vop := func(arity int, fn func(*Interp, []Value) (Value, error), names ...string) {
		i.register(&primitive{arity: arity, reporter: true, variadic: true, fn: fn}, names...)
	}

	// TABLEAU taille / (TABLEAU taille origine) : tableau de taille cases, chacune
	// initialisee a la liste vide. origine = indice de la 1re case (1 par defaut)
	vop(1, func(in *Interp, a []Value) (Value, error) {
		size, err := intArg(a[0])
		if err != nil {
			return Value{}, err
		}
		if size < 0 {
			return Value{}, fmt.Errorf("TABLEAU N'AIME PAS %s", a[0].String())
		}
		if size > maxArrayCells {
			return Value{}, fmt.Errorf("TABLEAU TROP GRAND")
		}
		origin := 1
		if len(a) >= 2 {
			o, err := intArg(a[1])
			if err != nil {
				return Value{}, err
			}
			origin = o
		}
		items := make([]Value, size)
		for k := range items {
			items[k] = ListValue(nil) // case vide = liste vide, facon FMSLogo
		}
		return ArrayValue(&Array{Items: items, Origin: origin}), nil
	}, "TABLEAU")

	// TABLEAUMD listetailles / (... origine) : tableau multi-dimensionnel
	vop(1, func(in *Interp, a []Value) (Value, error) {
		sizes, err := intList(a[0])
		if err != nil {
			return Value{}, err
		}
		if len(sizes) == 0 {
			return Value{}, fmt.Errorf("TABLEAUMD VEUT AU MOINS UNE DIMENSION")
		}
		if !mdSizeOK(sizes) {
			return Value{}, fmt.Errorf("TABLEAU TROP GRAND")
		}
		origin := 1
		if len(a) >= 2 {
			o, err := intArg(a[1])
			if err != nil {
				return Value{}, err
			}
			origin = o
		}
		arr, err := buildMD(sizes, origin)
		if err != nil {
			return Value{}, err
		}
		return ArrayValue(arr), nil
	}, "TABLEAUMD")

	// LISTEVERSTABLEAU liste / (... origine) : tableau ayant les memes elements
	vop(1, func(in *Interp, a []Value) (Value, error) {
		if a[0].Kind != KList {
			return Value{}, fmt.Errorf("LISTEVERSTABLEAU N'AIME PAS %s", a[0].String())
		}
		origin := 1
		if len(a) >= 2 {
			o, err := intArg(a[1])
			if err != nil {
				return Value{}, err
			}
			origin = o
		}
		items := make([]Value, len(a[0].List))
		for k, d := range a[0].List {
			v, err := datumToValue(d)
			if err != nil {
				return Value{}, err
			}
			items[k] = v
		}
		return ArrayValue(&Array{Items: items, Origin: origin}), nil
	}, "LISTEVERSTABLEAU")

	// TABLEAUVERSLISTE tableau : liste des cases (1re case en tete, quelle que soit
	// l'origine). copie de surface, pour traiter le tableau avec APPLIQUE etc.
	op(1, func(in *Interp, a []Value) (Value, error) {
		if a[0].Kind != KArray {
			return Value{}, fmt.Errorf("TABLEAUVERSLISTE N'AIME PAS %s", a[0].String())
		}
		out := make([]Datum, len(a[0].Arr.Items))
		for k, v := range a[0].Arr.Items {
			out[k] = valueToDatum(v)
		}
		return ListValue(out), nil
	}, "TABLEAUVERSLISTE")

	// ORDONNE tableau : trie les cases sur place (nombres par valeur, sinon mots).
	// commande qui modifie le tableau, comme FIXEITEM - pour garder l'original,
	// trier une COPIETABLEAU. la ou TRIE doit batir une liste, un tableau se trie
	// en place et tient la charge sur de gros volumes
	i.register(cmd(1, func(in *Interp, a []Value) error {
		if a[0].Kind != KArray {
			return fmt.Errorf("ORDONNE N'AIME PAS %s", a[0].String())
		}
		items := a[0].Arr.Items
		sort.SliceStable(items, func(i, j int) bool { return valueLess(items[i], items[j]) })
		return nil
	}), "ORDONNE")

	// ITEMMD listeindices tableaumd : case d'un tableau multi-dimensionnel
	op(2, func(in *Interp, a []Value) (Value, error) {
		if a[1].Kind != KArray {
			return Value{}, fmt.Errorf("ITEMMD N'AIME PAS %s", a[1].String())
		}
		idxs, err := intList(a[0])
		if err != nil {
			return Value{}, err
		}
		cur := a[1]
		for _, idx := range idxs {
			if cur.Kind != KArray {
				return Value{}, fmt.Errorf("ITEMMD : PAS ASSEZ DE DIMENSIONS")
			}
			pos := idx - cur.Arr.Origin
			if pos < 0 || pos >= len(cur.Arr.Items) {
				return Value{}, fmt.Errorf("PAS ASSEZ D'ELEMENTS POUR ITEMMD")
			}
			cur = cur.Arr.Items[pos]
		}
		return cur, nil
	}, "ITEMMD")

	// FIXEITEM indice tableau valeur : remplace la case indice. refuse de creer un
	// cycle (valeur ne doit pas contenir, meme en profondeur, le tableau modifie)
	i.register(cmd(3, func(in *Interp, a []Value) error {
		if a[1].Kind != KArray {
			return fmt.Errorf("FIXEITEM N'AIME PAS %s", a[1].String())
		}
		arr := a[1].Arr
		if reachesAnyArray(a[2], map[*Array]bool{arr: true}, map[*Array]bool{}) {
			return fmt.Errorf("FIXEITEM REFUSE UN TABLEAU CIRCULAIRE")
		}
		n, err := intArg(a[0])
		if err != nil {
			return err
		}
		pos := n - arr.Origin
		if pos < 0 || pos >= len(arr.Items) {
			return fmt.Errorf("PAS ASSEZ D'ELEMENTS POUR FIXEITEM")
		}
		arr.Items[pos] = a[2]
		return nil
	}), "FIXEITEM")

	// FIXEITEMMD listeindices tableaumd valeur : idem en multi-dimensionnel
	i.register(cmd(3, func(in *Interp, a []Value) error {
		if a[1].Kind != KArray {
			return fmt.Errorf("FIXEITEMMD N'AIME PAS %s", a[1].String())
		}
		idxs, err := intList(a[0])
		if err != nil {
			return err
		}
		if len(idxs) == 0 {
			return fmt.Errorf("FIXEITEMMD VEUT AU MOINS UN INDICE")
		}
		cur := a[1].Arr
		path := map[*Array]bool{cur: true}
		for _, idx := range idxs[:len(idxs)-1] {
			pos := idx - cur.Origin
			if pos < 0 || pos >= len(cur.Items) {
				return fmt.Errorf("PAS ASSEZ D'ELEMENTS POUR FIXEITEMMD")
			}
			nv := cur.Items[pos]
			if nv.Kind != KArray {
				return fmt.Errorf("FIXEITEMMD : PAS ASSEZ DE DIMENSIONS")
			}
			cur = nv.Arr
			path[cur] = true
		}
		if reachesAnyArray(a[2], path, map[*Array]bool{}) {
			return fmt.Errorf("FIXEITEMMD REFUSE UN TABLEAU CIRCULAIRE")
		}
		pos := idxs[len(idxs)-1] - cur.Origin
		if pos < 0 || pos >= len(cur.Items) {
			return fmt.Errorf("PAS ASSEZ D'ELEMENTS POUR FIXEITEMMD")
		}
		cur.Items[pos] = a[2]
		return nil
	}), "FIXEITEMMD")

	// TABLEAU? chose : VRAI si chose est un tableau
	op(1, func(in *Interp, a []Value) (Value, error) {
		return BoolValue(a[0].Kind == KArray), nil
	}, "TABLEAU?")

	// COPIETABLEAU tableau : copie PROFONDE (recursive sur les sous-tableaux).
	// le partage est preserve : un sous-tableau pointe deux fois n'est copie qu'une
	// fois. extension gologo (aucun Logo n'a de copie de tableau native)
	op(1, func(in *Interp, a []Value) (Value, error) {
		if a[0].Kind != KArray {
			return Value{}, fmt.Errorf("COPIETABLEAU N'AIME PAS %s", a[0].String())
		}
		return ArrayValue(deepCopyArray(a[0].Arr, map[*Array]*Array{})), nil
	}, "COPIETABLEAU")

	// piles et files : modifient en place une variable contenant une liste
	// (valeur initiale attendue : la liste vide [ ]).

	// EMPILE nomvar valeur : ajoute valeur en tete (pile)
	i.register(cmd(2, func(in *Interp, a []Value) error {
		name, list, err := in.listVar("EMPILE", a[0])
		if err != nil {
			return err
		}
		in.setVar(name, ListValue(append([]Datum{valueToDatum(a[1])}, list...)))
		return nil
	}), "EMPILE")

	// ENFILE nomvar valeur : ajoute valeur en fin (file)
	i.register(cmd(2, func(in *Interp, a []Value) error {
		name, list, err := in.listVar("ENFILE", a[0])
		if err != nil {
			return err
		}
		in.setVar(name, ListValue(append(append([]Datum{}, list...), valueToDatum(a[1]))))
		return nil
	}), "ENFILE")

	// DEPILE nomvar : rend et retire l'element de tete (pile, LIFO)
	op(1, func(in *Interp, a []Value) (Value, error) { return in.popFront("DEPILE", a[0]) }, "DEPILE")

	// DEFILE nomvar : rend et retire le plus ancien element, en tete (file, FIFO)
	op(1, func(in *Interp, a []Value) (Value, error) { return in.popFront("DEFILE", a[0]) }, "DEFILE")
}

// recupere une variable-liste par son nom (pour EMPILE/DEPILE/ENFILE/DEFILE)
func (in *Interp) listVar(op string, nameVal Value) (string, []Datum, error) {
	name, err := toWord(nameVal)
	if err != nil {
		return "", nil, err
	}
	v, ok := in.lookupVar(name)
	if !ok {
		return "", nil, fmt.Errorf("PAS DE CHOSE DONNEE A %s", strings.ToUpper(name))
	}
	if v.Kind != KList {
		return "", nil, fmt.Errorf("%s N'AIME PAS %s", op, v.String())
	}
	return name, v.List, nil
}

// retire et rend l'element de tete d'une variable-liste (DEPILE/DEFILE)
func (in *Interp) popFront(op string, nameVal Value) (Value, error) {
	name, list, err := in.listVar(op, nameVal)
	if err != nil {
		return Value{}, err
	}
	if len(list) == 0 {
		return Value{}, fmt.Errorf("%s N'AIME PAS LA LISTE VIDE", op)
	}
	in.setVar(name, ListValue(append([]Datum{}, list[1:]...)))
	return datumToValue(list[0])
}

// liste d'entiers depuis une valeur liste (indices de TABLEAUMD/ITEMMD/FIXEITEMMD)
func intList(v Value) ([]int, error) {
	if v.Kind != KList {
		return nil, fmt.Errorf("LISTE D'INDICES ATTENDUE")
	}
	out := make([]int, len(v.List))
	for i, d := range v.List {
		dv, err := datumToValue(d)
		if err != nil {
			return nil, fmt.Errorf("INDICE INVALIDE : %s", d.String())
		}
		n, err := intArg(dv) // entier exact : pas d'arrondi d'un grand indice
		if err != nil {
			return nil, fmt.Errorf("INDICE INVALIDE : %s", d.String())
		}
		out[i] = n
	}
	return out, nil
}

// construit un tableau multi-dimensionnel : une dimension par taille, les cases
// les plus profondes valent la liste vide. les tailles doivent etre positives
func buildMD(sizes []int, origin int) (*Array, error) {
	size := sizes[0]
	if size < 1 {
		return nil, fmt.Errorf("TABLEAUMD VEUT DES TAILLES POSITIVES")
	}
	items := make([]Value, size)
	if len(sizes) == 1 {
		for k := range items {
			items[k] = ListValue(nil)
		}
		return &Array{Items: items, Origin: origin}, nil
	}
	for k := range items {
		sub, err := buildMD(sizes[1:], origin)
		if err != nil {
			return nil, err
		}
		items[k] = ArrayValue(sub)
	}
	return &Array{Items: items, Origin: origin}, nil
}

// v atteint-il (en profondeur) l'un des tableaux cibles ? sert a refuser les cycles
// dans FIXEITEM/FIXEITEMMD. descend dans les tableaux ET dans les listes (une liste
// peut contenir un tableau). seen evite de retraiter un meme tableau (DAG)
func reachesAnyArray(v Value, targets, seen map[*Array]bool) bool {
	switch v.Kind {
	case KArray:
		a := v.Arr
		if targets[a] {
			return true
		}
		if seen[a] {
			return false
		}
		seen[a] = true
		for _, it := range a.Items {
			if reachesAnyArray(it, targets, seen) {
				return true
			}
		}
	case KList:
		for _, d := range v.List {
			if reachesDatumArray(d, targets, seen) {
				return true
			}
		}
	}
	return false
}

func reachesDatumArray(d Datum, targets, seen map[*Array]bool) bool {
	switch d.Kind {
	case DArray:
		if d.Arr != nil {
			return reachesAnyArray(ArrayValue(d.Arr), targets, seen)
		}
	case DList:
		for _, e := range d.List {
			if reachesDatumArray(e, targets, seen) {
				return true
			}
		}
	}
	return false
}

// copie profonde d'un tableau ; memo (par identite) preserve le partage : deux
// cases pointant le meme sous-tableau ne le copient qu'une fois
func deepCopyArray(a *Array, memo map[*Array]*Array) *Array {
	if c, ok := memo[a]; ok {
		return c
	}
	c := &Array{Items: make([]Value, len(a.Items)), Origin: a.Origin}
	memo[a] = c
	for i, v := range a.Items {
		switch v.Kind {
		case KArray:
			c.Items[i] = ArrayValue(deepCopyArray(v.Arr, memo))
		case KList:
			// une liste peut cacher un tableau (LISTE :tab) : on copie aussi en profondeur
			c.Items[i] = ListValue(deepCopyDatums(v.List, memo))
		default:
			c.Items[i] = v // mots/nombres immuables : partage sans risque
		}
	}
	return c
}

// copie profonde d'une suite de datums (corps de liste) : recopie les tableaux
// rencontres (directement ou via des sous-listes), partage le reste. meme memo
// que deepCopyArray pour preserver le partage et casser les cycles
func deepCopyDatums(ds []Datum, memo map[*Array]*Array) []Datum {
	out := make([]Datum, len(ds))
	for i, d := range ds {
		switch d.Kind {
		case DArray:
			if d.Arr != nil {
				cp := deepCopyArray(d.Arr, memo)
				out[i] = Datum{Kind: DArray, Arr: cp, Origin: cp.Origin}
				continue
			}
			out[i] = d
		case DList:
			out[i] = Datum{Kind: DList, List: deepCopyDatums(d.List, memo)}
		default:
			out[i] = d
		}
	}
	return out
}
