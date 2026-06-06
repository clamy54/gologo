package logo

import (
	"fmt"
	"sort"
	"strings"
)

// primitives de l'espace de travail (section 8) : inventaire, impression et
// effacement des procedures et des noms
func (i *Interp) registerWorkspace() {
	// CONTENU : tous les mots connus, tries (procs, variables, primitives courantes)
	i.register(&primitive{arity: 0, reporter: true, fn: func(in *Interp, a []Value) (Value, error) {
		names := in.knownWords()
		items := make([]Datum, len(names))
		for k, n := range names {
			items[k] = Datum{Kind: DWord, Text: n}
		}
		return ListValue(items), nil
	}}, "CONTENU")

	// IM mot : imprime la definition d'une procedure
	i.register(cmd(1, func(in *Interp, a []Value) error {
		name, err := toWord(a[0])
		if err != nil {
			return err
		}
		p := in.procs[strings.ToUpper(name)]
		if p == nil {
			return &badData{a[0].String()} // "IM N'AIME PAS ..."
		}
		fmt.Fprintln(in.Out, p.sourceText())
		return nil
	}), "IM")

	// IMTS : juste les titres des procedures (ligne POUR ...)
	i.register(cmd(0, func(in *Interp, a []Value) error {
		for _, n := range in.procNamesSorted() {
			fmt.Fprintln(in.Out, procTitle(in.procs[n]))
		}
		return nil
	}), "IMTS")

	// IMNS : les noms et leurs valeurs, forme DONNE (re-executable)
	i.register(cmd(0, func(in *Interp, a []Value) error {
		in.printNames()
		return nil
	}), "IMNS")

	// IMTOUT : les procedures completes puis les noms
	i.register(cmd(0, func(in *Interp, a []Value) error {
		for _, n := range in.procNamesSorted() {
			fmt.Fprintln(in.Out, in.procs[n].sourceText())
		}
		in.printNames()
		return nil
	}), "IMTOUT")

	// EFP mot : efface une procedure
	i.register(cmd(1, func(in *Interp, a []Value) error {
		name, err := toWord(a[0])
		if err != nil {
			return err
		}
		delete(in.procs, strings.ToUpper(name))
		return nil
	}), "EFP")

	// EFN mot : efface un nom (variable)
	i.register(cmd(1, func(in *Interp, a []Value) error {
		name, err := toWord(a[0])
		if err != nil {
			return err
		}
		delete(in.vars, strings.ToUpper(name))
		return nil
	}), "EFN")

	// .EFT : table rase, on efface tout (procs + noms + proprietes)
	i.register(cmd(0, func(in *Interp, a []Value) error {
		in.procs = map[string]*userProc{}
		in.vars = map[string]Value{}
		in.plists = map[string][]propEntry{}
		in.edBuf = ""
		return nil
	}), ".EFT")

	// PLACE : cellules memoire libres. inutile en Go (GC), on rend une grosse
	// valeur stable pour les programmes qui la regardent
	i.register(&primitive{arity: 0, reporter: true, fn: func(in *Interp, a []Value) (Value, error) {
		return NumberValue(60000), nil
	}}, "PLACE")

	// RECYCLE : passait le ramasse-miettes. en Go le GC s'en charge, donc no-op
	i.register(cmd(0, func(in *Interp, a []Value) error { return nil }), "RECYCLE")
}

// tous les mots connus : d'abord ce que l'utilisateur a defini (procs + variables,
// triees), puis les primitives de la langue courante
func (i *Interp) knownWords() []string {
	user := append(i.procNames(), i.varNames()...)
	sort.Strings(user)
	return append(user, i.primWords()...)
}

// noms de primitives (principaux + alias) dans la langue courante, tries
// et sans doublons (source : helpData)
func (i *Interp) primWords() []string {
	lang := i.Lang()
	set := make(map[string]bool)
	for fr, e := range helpData {
		set[canonical(lang, fr, e)] = true
		aliases := e.frAliases
		if lang == "EN" {
			aliases = e.enAliases
		}
		for _, a := range aliases {
			set[a] = true
		}
	}
	out := make([]string, 0, len(set))
	for n := range set {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// noms de procedures, ordre indefini
func (i *Interp) procNames() []string {
	names := make([]string, 0, len(i.procs))
	for n := range i.procs {
		names = append(names, n)
	}
	return names
}

func (i *Interp) procNamesSorted() []string {
	names := i.procNames()
	sort.Strings(names)
	return names
}

// noms de variables, ordre indefini
func (i *Interp) varNames() []string {
	names := make([]string, 0, len(i.vars))
	for n := range i.vars {
		names = append(names, n)
	}
	return names
}

// imprime les variables sous forme DONNE (re-executable), triees
func (i *Interp) printNames() {
	names := i.varNames()
	sort.Strings(names)
	for _, n := range names {
		fmt.Fprintf(i.Out, "DONNE \"%s %s\n", n, valueSource(i.vars[n]))
	}
}

// la 1re ligne "POUR nom :p1 ..." d'une procedure
func procTitle(p *userProc) string {
	t := "POUR " + p.name
	for _, par := range p.params {
		t += " :" + par
	}
	return t
}

// une valeur sous forme relisible (listes entre crochets)
func valueSource(v Value) string {
	if v.Kind == KList {
		return "[" + v.String() + "]"
	}
	return v.String()
}
