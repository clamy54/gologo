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
		return ListValue(in.knownWords()), nil
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
		src, err := p.sourceText()
		if err != nil {
			return err
		}
		return in.printLine(src)
	}), "IM")

	// IMTS : juste les titres des procedures (ligne POUR ...)
	i.register(cmd(0, func(in *Interp, a []Value) error {
		for _, n := range in.procNamesSorted() {
			if err := in.printLine(procTitle(in.procs[n])); err != nil {
				return err
			}
		}
		return nil
	}), "IMTS")

	// IMNS : les noms et leurs valeurs, forme DONNE (re-executable)
	i.register(cmd(0, func(in *Interp, a []Value) error {
		return in.printNames()
	}), "IMNS")

	// IMTOUT : les procedures completes puis les noms
	i.register(cmd(0, func(in *Interp, a []Value) error {
		for _, n := range in.procNamesSorted() {
			src, err := in.procs[n].sourceText()
			if err != nil {
				return err
			}
			if err := in.printLine(src); err != nil {
				return err
			}
		}
		return in.printNames()
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
// triees), puis les primitives de la langue courante. une procedure y figure en
// nom nu et une variable en "mot : c'est la convention de SAUVE, qui peut ainsi
// relire CONTENU sans confondre une procedure et une variable de meme nom
func (i *Interp) knownWords() []Datum {
	var user []Datum
	for _, n := range i.procNames() {
		user = append(user, Datum{Kind: DSymbol, Text: n})
	}
	for _, n := range i.varNames() {
		user = append(user, Datum{Kind: DWord, Text: n})
	}
	sort.SliceStable(user, func(a, b int) bool {
		if user[a].Text != user[b].Text {
			return user[a].Text < user[b].Text
		}
		return user[a].Kind == DSymbol && user[b].Kind != DSymbol
	})
	for _, n := range i.primWords() {
		user = append(user, Datum{Kind: DWord, Text: n})
	}
	return user
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

// ecrit une ligne sur la console ; une sortie qui refuse l'ecriture (redirection
// cassee) devient une erreur Logo au lieu de passer inapercue
func (i *Interp) printLine(s string) error {
	if _, err := fmt.Fprintln(i.Out, s); err != nil {
		return errEcritureImpossible
	}
	return nil
}

// imprime les variables sous forme DONNE (re-executable), triees
func (i *Interp) printNames() error {
	names := i.varNames()
	sort.Strings(names)
	for _, n := range names {
		line, err := donneSource(n, i.vars[n])
		if err != nil {
			return err
		}
		if err := i.printLine(line); err != nil {
			return err
		}
	}
	return nil
}
