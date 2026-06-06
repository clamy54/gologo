package logo

import (
	"fmt"
	"strings"
)

// iterateurs de liste a gabarit (Logo "adulte"). le gabarit est du code ou
// l'element courant se lit via :? (et :?1 / :?2 pour REDUIS)

// evalue une suite de Datum comme expression et rend sa valeur (ex. [ :? * 2 ])
func (i *Interp) evalExpr(data []Datum) (Value, error) {
	ev := &eval{i: i, data: data}
	return ev.expr(0)
}

// contexte local des variables de gabarit (:? etc.)
func (i *Interp) pushSlots(s map[string]Value) { i.frames = append(i.frames, s) }
func (i *Interp) popSlot()                     { i.frames = i.frames[:len(i.frames)-1] }

// execute body pour chaque element de items, l'element courant lie a varName
// (slot local lu par ":varName"). interruptible via runSeq
func (i *Interp) forEachSlot(varName string, items, body []Datum) error {
	for _, d := range items {
		v, err := datumToValue(d)
		if err != nil {
			return err
		}
		i.pushSlots(map[string]Value{varName: v})
		err = i.runSeq(body)
		i.popSlot()
		if err != nil {
			return err
		}
	}
	return nil
}

// elements a parcourir : membres d'une liste, ou caracteres d'un mot/nombre
// (la forme XLogo de POURCHAQUE accepte liste OU mot)
func iterDatums(v Value) []Datum {
	switch v.Kind {
	case KList:
		return v.List
	case KWord, KNumber:
		rs := []rune(v.String())
		out := make([]Datum, len(rs))
		for k, r := range rs {
			out[k] = Datum{Kind: DWord, Text: string(r)}
		}
		return out
	}
	return nil
}

// POURCHAQUE a deux formes, devinees au type du 1er argument :
//   liste en tete -> notre forme : POURCHAQUE liste [ gabarit avec :? ]
//   mot en tete   -> forme XLogo : POURCHAQUE "var liste_ou_mot [ commande avec :var ]
func formePourchaque(e *eval) (Value, error) {
	first, err := e.expr(0)
	if err != nil {
		return Value{}, err
	}
	if !first.HasValue() {
		return Value{}, fmt.Errorf("PAS ASSEZ DE DONNEES POUR POURCHAQUE")
	}
	if first.Kind == KList { // notre forme (:?)
		tmpl, err := e.expr(0)
		if err != nil {
			return Value{}, err
		}
		if tmpl.Kind != KList {
			return Value{}, &badData{tmpl.String()}
		}
		return None, e.i.forEachSlot("?", first.List, tmpl.List)
	}
	// forme XLogo : "var liste_ou_mot commande
	varName, err := toWord(first)
	if err != nil {
		return Value{}, err
	}
	data, err := e.expr(0)
	if err != nil {
		return Value{}, err
	}
	if !data.HasValue() {
		return Value{}, fmt.Errorf("PAS ASSEZ DE DONNEES POUR POURCHAQUE")
	}
	body, err := e.expr(0)
	if err != nil {
		return Value{}, err
	}
	if body.Kind != KList {
		return Value{}, &badData{body.String()}
	}
	return None, e.i.forEachSlot(strings.ToUpper(varName), iterDatums(data), body.List)
}

func (i *Interp) registerIterators() {
	// POURCHAQUE : deux formes (cf formePourchaque)
	i.register(&primitive{name: "POURCHAQUE", special: formePourchaque}, "POURCHAQUE")

	// operation prenant (liste, gabarit), les deux etant des listes
	oplist := func(name string, fn func(in *Interp, list, tmpl []Datum) (Value, error)) {
		i.register(&primitive{name: name, arity: 2, reporter: true, fn: func(in *Interp, a []Value) (Value, error) {
			if a[0].Kind != KList {
				return Value{}, &badData{a[0].String()}
			}
			if a[1].Kind != KList {
				return Value{}, &badData{a[1].String()}
			}
			return fn(in, a[0].List, a[1].List)
		}}, name)
	}

	// APPLIQUE liste [ expr ] : liste des resultats du gabarit (:? = element)
	oplist("APPLIQUE", func(in *Interp, list, tmpl []Datum) (Value, error) {
		out := make([]Datum, 0, len(list))
		for _, d := range list {
			v, err := datumToValue(d)
			if err != nil {
				return Value{}, err
			}
			in.pushSlots(map[string]Value{"?": v})
			r, err := in.evalExpr(tmpl)
			in.popSlot()
			if err != nil {
				return Value{}, err
			}
			out = append(out, valueToDatum(r))
		}
		return ListValue(out), nil
	})

	// FILTRE liste [ predicat ] : garde les elements ou le gabarit rend VRAI
	oplist("FILTRE", func(in *Interp, list, tmpl []Datum) (Value, error) {
		out := []Datum{}
		for _, d := range list {
			v, err := datumToValue(d)
			if err != nil {
				return Value{}, err
			}
			in.pushSlots(map[string]Value{"?": v})
			r, err := in.evalExpr(tmpl)
			in.popSlot()
			if err != nil {
				return Value{}, err
			}
			if r.IsTrue() {
				out = append(out, d)
			}
		}
		return ListValue(out), nil
	})

	// REDUIS liste [ expr ] : replie la liste, :?1 = accumulateur, :?2 = element
	oplist("REDUIS", func(in *Interp, list, tmpl []Datum) (Value, error) {
		if len(list) == 0 {
			return WordValue(""), nil
		}
		acc, err := datumToValue(list[0])
		if err != nil {
			return Value{}, err
		}
		for _, d := range list[1:] {
			v, err := datumToValue(d)
			if err != nil {
				return Value{}, err
			}
			in.pushSlots(map[string]Value{"?1": acc, "?2": v})
			acc, err = in.evalExpr(tmpl)
			in.popSlot()
			if err != nil {
				return Value{}, err
			}
		}
		return acc, nil
	})
}
