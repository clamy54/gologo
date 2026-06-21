package logo

// categorie affichee dans la fiche d'aide :
//   - d'origine : dans le manuel SOLI Logo et fonctionnelle
//   - extension : ajout hors manuel (Logo moderne)
//   - compat    : dans le manuel mais sans effet ici (materiel d'epoque)
// par defaut c'est "d'origine", on ne liste donc que les exceptions

// ajouts hors manuel SOLI (Logo moderne)
var categExtension = map[string]bool{
	"AIDE": true, "ANGLAIS": true, "FRANCAIS": true, "LANGUE": true, "QUITTE": true, "RAZ": true,
	"APPLIQUE": true, "FILTRE": true, "REDUIS": true, "POURCHAQUE": true,
	"PIEGE": true, "LANCE": true, "TESTE": true, "SIVRAI": true, "SIFAUX": true,
	"ARC": true, "CERCLE": true, "REMPLIS": true, "ETIQUETTE": true,
	"FIXETAILLECRAYON": true, "DISTANCE": true, "XCOR": true, "YCOR": true,
	"FXY": true, "VERSCAP": true, "SPRITE": true, "SAUVEPNG": true,
	"ARCTAN": true, "EXP": true, "LN": true, "LOG10": true, "MODULO": true,
	"VALABS": true, "ARRONDI": true, "TRONQUE": true, "MOINS": true,
	"PUISSANCE": true, "INVERSE": true, "PI": true, "TANGENTE": true, "OUEX": true,
	"VERSBASE": true, "DEPUISBASE": true, "HEXA": true, "BINAIRE": true,
	"DEVELOPPE": true, "FACTORISE": true, "RESOUS": true, "EVALUE": true,
	"MAJUSCULE": true, "MINUSCULE": true, "TRIE": true, "MONTRE": true,
	"LISMOT": true, "LOCAL": true, "COMPTEUR": true, "ATTENDS": true,
	"TANTQUE": true, "REPETEPOUR": true, "SCENE": true, "EXECRESULTAT": true, "COPIE": true,
	"DONNEPROP": true, "PROP": true, "EFPROP": true, "LISTEPROP": true,
	"DEFINIS": true, "TEXTE": true, "LIS": true,
	"FCR": true, "CR": true, "PIOCHE": true,
	"FIXETORTUE": true, "TORTUE": true, "NBTORTUES": true, "DEMANDE": true, "DISTORTUE": true,
	"DEFSPRITE": true, "COLLISION?": true,
	"ANIME": true, "STOPANIME": true, "CADENCE": true,
	"CATALOGUEEX": true, "RAMENEEX": true, "CHARGEEX": true,
	"MEMBRE": true, "SOUSCHAINE?": true, "AVANT?": true, "DECOUPE": true,
	"ROGNE": true, "ROGNEDEBUT": true, "ROGNEFIN": true, "SUBSTITUE": true,
	"OTETOUT": true, "SANSDOUBLONS": true, "TRANCHE": true,
	"REMPLACE": true, "AJOUTE": true, "OTE": true, "POSITION": true,
	"TABLEAU": true, "TABLEAUMD": true, "LISTEVERSTABLEAU": true, "ORDONNE": true,
	"TABLEAUVERSLISTE": true, "ITEMMD": true, "FIXEITEM": true, "FIXEITEMMD": true,
	"TABLEAU?": true, "COPIETABLEAU": true,
	"EMPILE": true, "DEPILE": true, "ENFILE": true, "DEFILE": true,
	"OUVRELECTURE": true, "OUVREECRITURE": true, "OUVREAJOUT": true, "OUVREMAJ": true,
	"FERME": true, "FERMETOUT": true, "TOUSOUVERTS": true,
	"FIXELECTURE": true, "FIXEECRITURE": true, "FLUXLECTURE": true, "FLUXECRITURE": true,
	"FINFICHIER?": true, "POSLECTURE": true, "POSECRITURE": true,
	"FIXEPOSLECTURE": true, "FIXEPOSECRITURE": true, "LISLIGNE": true, "LISCARS": true,
	"FICHIERVERSTABLEAU": true, "FIXEFINLIGNE": true, "FINLIGNE": true,
}

// des extensions jugees essentielles pour le debutant, montrees quand meme
// dans l'aide F1. leur vraie categorie (extension) ne change pas dans la fiche
var categDebutant = map[string]bool{
	"QUITTE": true, // au cas ou il voudrait fuir : annonce dans la banniere
	// gestion des exemples : un debutant en a besoin pour decouvrir et charger les .GLG
	"CATALOGUEEX": true, "RAMENEEX": true, "CHARGEEX": true,
}

// force le nom affiche dans l'aide debutant FR quand le nom long auto ne convient pas
// (ex. QUITTE, comme la banniere, plutot que l'alias QUITTER)
var aideNomDebutant = map[string]string{
	"QUITTE": "QUITTE",
}

// primitives du manuel sans effet ici (peripheriques d'epoque)
var categCompat = map[string]bool{
	"REGLE": true, "ENTREE": true, "SORTIE": true, "FLI": true,
	".SER": true, ".CHB": true, ".RES": true, ".DEP": true, ".ROUT": true, ".EXA": true,
	"FORMATE": true, "LECTEUR": true, "FLECTEUR": true,
}

// ligne de categorie d'une primitive (cle FR), bilingue
func categorieLine(lang, frKey string) string {
	switch {
	case categCompat[frKey]:
		if lang == "EN" {
			return "Compatibility (no effect)"
		}
		return "Compatibilité (sans effet)"
	case categExtension[frKey]:
		if lang == "EN" {
			return "Extension (modern Logo)"
		}
		return "Extension (Logo moderne)"
	default:
		if lang == "EN" {
			return "Origin: SOLI Logo"
		}
		return "Origine : Logo SOLI"
	}
}
