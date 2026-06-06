package logo

import "strings"

// noms FMSLogo (le Logo Windows anglophone, heritier d'UCBLogo) ajoutes en ALIAS
// quand fonction ET arite collent, pour lire des listings FMSLogo tels quels.
// presque tout son vocabulaire recoupe deja nos alias UCBLogo, ou vise des choses
// qu'on ne fait pas (tableaux, GUI, 3D, reseau...). restent les rares synonymes
// vraiment nouveaux ci-dessous. cle = nom FMSLogo, valeur = notre nom canonique.
// on n'ecrase jamais un nom deja pris (cf le test qui verifie les cibles)
var fmslogoAliases = map[string]string{
	"SETSCREENCOLOR": "FCFG", // = SETBACKGROUND (index 0-15 ou [r v b])
	"SETSC":          "FCFG", // abreviation FMSLogo de SETSCREENCOLOR
	"SCREENCOLOR":    "CF",   // = BACKGROUND (lit la couleur de fond)
	"HALT":           "LOGO", // = TOPLEVEL (arret total, retour a l'invite)
	"DRAW":           "MT",   // = SHOWTURTLE : rend la tortue visible
	"NODRAW":         "CT",   // = HIDETURTLE : cache la tortue (le trace continue)
}

// branche les alias FMSLogo sur les primitives, sans ecraser un nom deja pris
// a appeler en dernier (les cibles doivent exister)
func (i *Interp) registerFMSLogoAliases() {
	for alias, target := range fmslogoAliases {
		a := strings.ToUpper(alias)
		if i.prims[a] != nil {
			continue // on ne marche pas sur un nom existant
		}
		if p := i.prims[strings.ToUpper(target)]; p != nil {
			i.prims[a] = p
		}
	}
}
