package logo

import "strings"

// noms XLogo (francophone) en ALIAS de nos primitives quand fonction ET arite
// collent, pour lire des listings XLogo tels quels. cle = nom XLogo, valeur = notre
// nom canonique. on n'ecrase jamais un nom deja pris (cf le test des cibles).
// exclu expres car semantique differente : ALEA (XLogo rend un reel entre 0 et 1,
// notre HASARD rend un entier entre 0 et n-1)
var xlogoAliases = map[string]string{
	// tortue / crayon / ecran : noms longs XLogo -> nos abreviations MO5
	"BAISSECRAYON":      "BC",
	"LEVECRAYON":        "LC",
	"CACHETORTUE":       "CT",
	"MONTRETORTUE":      "MT",
	"VIDEECRAN":         "VE",
	"VIDETEXTE":         "VT",
	"FIXECAP":           "FCAP",
	"FIXEPOSITION":      "FPOS",
	"FIXEXY":            "FXY",
	"FIXECOULEURCRAYON": "FCC",
	"FIXECOULEURFOND":   "FCFG",
	"FIXECOULEURTEXTE":  "FCT",
	"FTC":               "FIXETAILLECRAYON",

	// maths (ABS est deja l'alias anglais de VALABS, inutile d'en remettre)
	"MOD":         "MODULO",
	"ARCTANGENTE": "ARCTAN",

	// espace de travail / editeur / controle
	"EFFACEPROCEDURE": "EFP",
	"EFFACEVARIABLE":  "EFN",
	"EFFACETOUT":      ".EFT",
	"EDITE":           "ED",
	"EXECUTE":         "EXEC",
}

// branche les alias XLogo sur les primitives, sans ecraser un nom deja pris
// a appeler en dernier (les cibles doivent exister)
func (i *Interp) registerXLogoAliases() {
	for alias, target := range xlogoAliases {
		a := strings.ToUpper(alias)
		if i.prims[a] != nil {
			continue // on ne marche pas sur un nom existant
		}
		if p := i.prims[strings.ToUpper(target)]; p != nil {
			i.prims[a] = p
		}
	}
}
