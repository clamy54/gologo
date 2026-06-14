package logo

import "strings"

// remplace dans src chaque nom de primitive connu par son equivalent dans la langue
// cible (toEN = anglais, sinon FR), en gardant la mise en page exacte. ne touche pas
// aux mots quotes, variables :.., nombres, [ ] ( ), commentaires (; ou #) ni aux noms
// inconnus (procs, variables). c'est le Ctrl+T de l'editeur.
// le decoupage se fait aux espaces ET aux [ ] ( ), pour qu'un nom colle a un crochet
// (REPETE 4 [AV 50]) soit quand meme isole et traduit
func (i *Interp) TranslateProgram(src string, toEN bool) string {
	lang := "FR"
	if toEN {
		lang = "EN"
	}
	var b strings.Builder
	n := len(src)
	j := 0
	for j < n {
		c := src[j]
		switch {
		case c == ' ' || c == '\t' || c == '\r' || c == '\n':
			b.WriteByte(c)
			j++
		case c == ';' || c == '#': // commentaire jusqu'a la fin de ligne
			for j < n && src[j] != '\n' {
				b.WriteByte(src[j])
				j++
			}
		case c == '[' || c == ']' || c == '(' || c == ')' || c == '{' || c == '}': // delimiteurs liste/tableau
			b.WriteByte(c)
			j++
		default:
			k := j
			for k < n && !isTokenBreak(src[k]) {
				k++
			}
			b.WriteString(translateToken(src[j:k], lang))
			j = k
		}
	}
	return b.String()
}

// un token s'arrete sur une espace ou un [ ] ( ) { } (jamais inclus dans le mot)
func isTokenBreak(c byte) bool {
	switch c {
	case ' ', '\t', '\r', '\n', '[', ']', '(', ')', '{', '}':
		return true
	}
	return false
}

// equivalent du token dans la langue cible si c'est une primitive connue,
// sinon le token tel quel
func translateToken(tok, lang string) string {
	if tok == "" {
		return tok
	}
	switch tok[0] {
	case '"', ':':
		return tok // mot-donnee ou variable, pas une instruction
	}
	if fr, e, ok := lookupHelp(tok); ok {
		if out := canonical(lang, fr, e); out != "" {
			return out
		}
	}
	return tok
}
