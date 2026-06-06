package logo

import "strings"

// bascules de langue (FRANCAIS/ANGLAIS)
// les alias FRENCH/ENGLISH viennent de registerEnglishAliases
func (i *Interp) registerLang() {
	i.register(cmd(0, func(in *Interp, a []Value) error { in.setLang("FR"); return nil }), "FRANCAIS", "FR")
	i.register(cmd(0, func(in *Interp, a []Value) error { in.setLang("EN"); return nil }), "ANGLAIS")
}

// ajoute les noms anglais (en + enAliases) de chaque primitive
// a appeler en dernier, une fois tous les noms FR poses
func (i *Interp) registerEnglishAliases() {
	for fr, e := range helpData {
		p := i.prims[fr]
		if p == nil {
			continue
		}
		for _, n := range append([]string{e.en}, e.enAliases...) {
			if n != "" {
				i.prims[strings.ToUpper(n)] = p
			}
		}
	}
}
