package logo

import (
	"sort"
	"strings"
)

// noms de commandes (primitives + procedures) commencant par prefix, tries
// insensible a la casse. sert a l'autocompletion du REPL
func (i *Interp) Completions(prefix string) []string {
	prefix = strings.ToUpper(prefix)
	set := map[string]bool{}
	for n := range i.prims {
		if strings.HasPrefix(n, prefix) {
			set[n] = true
		}
	}
	for n := range i.procs {
		if strings.HasPrefix(n, prefix) {
			set[n] = true
		}
	}
	out := make([]string, 0, len(set))
	for n := range set {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}
