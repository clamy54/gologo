package logo

import "strings"

// une propriete d'objet (nom tel que donne, valeur). gardees dans l'ordre
// d'insertion pour que PLIST soit deterministe
type propEntry struct {
	name string // nom de propriete, casse preservee (affichee par LISTEPROP)
	val  Value
}

// index de la propriete prop dans la liste de obj, ou -1. objKey est deja en MAJ
func (i *Interp) findProp(objKey, prop string) int {
	for k, e := range i.plists[objKey] {
		if strings.EqualFold(e.name, prop) {
			return k
		}
	}
	return -1
}

func (i *Interp) registerPlist() {
	// DONNEPROP obj prop valeur (PPROP) : pose prop=valeur sur obj
	// remplace si elle existe, sinon ajout en fin de liste
	i.register(cmd(3, func(in *Interp, a []Value) error {
		obj, err := toWord(a[0])
		if err != nil {
			return err
		}
		prop, err := toWord(a[1])
		if err != nil {
			return err
		}
		key := strings.ToUpper(obj)
		if idx := in.findProp(key, prop); idx >= 0 {
			in.plists[key][idx].val = a[2]
		} else {
			in.plists[key] = append(in.plists[key], propEntry{name: prop, val: a[2]})
		}
		return nil
	}), "DONNEPROP")

	// PROP obj prop (GPROP) : valeur de prop sur obj, ou [ ] si absente
	i.register(&primitive{name: "PROP", arity: 2, reporter: true, fn: func(in *Interp, a []Value) (Value, error) {
		obj, err := toWord(a[0])
		if err != nil {
			return Value{}, err
		}
		prop, err := toWord(a[1])
		if err != nil {
			return Value{}, err
		}
		key := strings.ToUpper(obj)
		if idx := in.findProp(key, prop); idx >= 0 {
			return in.plists[key][idx].val, nil
		}
		return ListValue(nil), nil
	}}, "PROP")

	// EFPROP obj prop (REMPROP) : efface prop de obj. plus aucune prop -> l'objet disparait
	i.register(cmd(2, func(in *Interp, a []Value) error {
		obj, err := toWord(a[0])
		if err != nil {
			return err
		}
		prop, err := toWord(a[1])
		if err != nil {
			return err
		}
		key := strings.ToUpper(obj)
		idx := in.findProp(key, prop)
		if idx < 0 {
			return nil
		}
		list := in.plists[key]
		in.plists[key] = append(list[:idx], list[idx+1:]...)
		if len(in.plists[key]) == 0 {
			delete(in.plists, key)
		}
		return nil
	}), "EFPROP")

	// LISTEPROP obj (PLIST) : rend [ prop1 val1 prop2 val2 ... ], ou [ ] si rien
	i.register(&primitive{name: "LISTEPROP", arity: 1, reporter: true, fn: func(in *Interp, a []Value) (Value, error) {
		obj, err := toWord(a[0])
		if err != nil {
			return Value{}, err
		}
		entries := in.plists[strings.ToUpper(obj)]
		items := make([]Datum, 0, len(entries)*2)
		for _, e := range entries {
			items = append(items, valueToDatum(WordValue(e.name)), valueToDatum(e.val))
		}
		return ListValue(items), nil
	}}, "LISTEPROP")
}
