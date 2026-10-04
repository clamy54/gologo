package logo

import "strings"

// bilingue (2/2) : on produit les erreurs en FR partout, on traduit a l'affichage si EN
// evite de dupliquer 40+ endroits et garde les erreurs-repere compatibles errors.Is

// texte d'erreur dans la langue courante (FR tel quel, EN traduit)
// appele au moment d'afficher ; en interne tout reste en FR
func (i *Interp) ErrorText(err error) string {
	if err == nil {
		return ""
	}
	if i.Lang() != "EN" {
		return err.Error()
	}
	// erreur rattachee a une procedure : on traduit la partie erreur, "DANS" -> "IN"
	if pe, ok := err.(*procErr); ok {
		return localizeErr(pe.inner.Error()) + " IN " + pe.proc
	}
	return localizeErr(err.Error())
}

// traductions EN des messages d'erreur a texte fixe (sans partie variable)
// ceux qui ont une partie variable passent par localizeErr
var errExact = map[string]string{
	"DIVISION PAR ZERO":                             "DIVISION BY ZERO",
	"CLAVIER INDISPONIBLE":                          "KEYBOARD UNAVAILABLE",
	"PREM/DER N'AIME PAS LA LISTE VIDE":             "FIRST/LAST DOESN'T LIKE THE EMPTY LIST",
	"PREM/DER N'AIME PAS LE MOT VIDE":               "FIRST/LAST DOESN'T LIKE THE EMPTY WORD",
	"SP/SD N'AIME PAS LA LISTE VIDE":                "BUTFIRST/BUTLAST DOESN'T LIKE THE EMPTY LIST",
	"SP/SD N'AIME PAS LE MOT VIDE":                  "BUTFIRST/BUTLAST DOESN'T LIKE THE EMPTY WORD",
	"PAS ASSEZ D'ELEMENTS POUR ITEM":                "NOT ENOUGH ITEMS FOR ITEM",
	"POUR ATTEND UN NOM DE PROCEDURE":               "TO NEEDS A PROCEDURE NAME",
	"FIN N'A DE SENS QU'EN DEFINITION DE PROCEDURE": "END ONLY MAKES SENSE IN A PROCEDURE DEFINITION",
	"OBJET MANQUANT":                                "MISSING OBJECT",
	"OBJET INATTENDU":                               "UNEXPECTED OBJECT",
	"OBJET INATTENDU DANS LA LISTE":                 "UNEXPECTED OBJECT IN LIST",
	"EDITEUR INDISPONIBLE":                          "EDITOR UNAVAILABLE",
	"AIDE INDISPONIBLE":                             "HELP UNAVAILABLE",
	"LA TORTUE VA SORTIR":                           "TURTLE OUT OF BOUNDS",
	"INTERROMPU !":                                  "STOPPED!",
	"ECRITURE IMPOSSIBLE":                           "CANNOT WRITE FILE",
	"LECTURE IMPOSSIBLE":                            "CANNOT READ FILE",
	"PAS DE DOSSIER D'EXEMPLES":                     "NO EXAMPLES DIRECTORY",
	"PLUS DE PLACE":                                 "OUT OF SPACE",
	"FICHIER INTROUVABLE":                           "FILE NOT FOUND",
	"FICHIER DEJA OUVERT":                           "FILE ALREADY OPEN",
	"FICHIER NON OUVERT":                            "FILE NOT OPEN",
	"FICHIER OUVERT":                                "FILE IS OPEN",
	"POSITION INVALIDE":                             "INVALID POSITION",
	"NOM DE FICHIER INVALIDE":                       "INVALID FILE NAME",
	"MAUVAIS MODE D'OUVERTURE":                      "WRONG OPEN MODE",
	"FICHIER BINAIRE":                               "BINARY FILE",
	"IMBRICATION TROP PROFONDE":                     "NESTING TOO DEEP",
	"TABLEAU TROP GRAND":                            "ARRAY TOO LARGE",
	"TABLEAUMD VEUT AU MOINS UNE DIMENSION":         "MDARRAY WANTS AT LEAST ONE DIMENSION",
	"TABLEAUMD VEUT DES TAILLES POSITIVES":          "MDARRAY WANTS POSITIVE SIZES",
	"FIXEITEMMD VEUT AU MOINS UN INDICE":            "MDSETITEM WANTS AT LEAST ONE INDEX",
	"ITEMMD : PAS ASSEZ DE DIMENSIONS":              "MDITEM: NOT ENOUGH DIMENSIONS",
	"FIXEITEMMD : PAS ASSEZ DE DIMENSIONS":          "MDSETITEM: NOT ENOUGH DIMENSIONS",
	"LISTE D'INDICES ATTENDUE":                      "INDEX LIST EXPECTED",
	"DECOUPE VEUT UN SEPARATEUR NON VIDE":           "SPLIT WANTS A NON-EMPTY SEPARATOR",
	`FIXEFINLIGNE VEUT "LF OU "CRLF`:                `SETEOL WANTS "LF OR "CRLF`,
	"CARACTERE NUL DANS LE PROGRAMME":               "NUL CHARACTER IN PROGRAM",
	"FICHIER TROP GROS":                             "FILE TOO LARGE",
	"LIGNE TROP LONGUE":                             "LINE TOO LONG",
	"LE GABARIT NE REND RIEN":                       "THE TEMPLATE OUTPUTS NOTHING",
	"CALCUL TROP GROS":                              "COMPUTATION TOO LARGE",
	"IMPOSSIBLE A ECRIRE EN SOURCE LOGO":            "CANNOT BE WRITTEN AS LOGO SOURCE",

	// calcul litteral
	"APRES ^ IL FAUT UN EXPOSANT ENTIER":                                   "AFTER ^ AN INTEGER EXPONENT IS NEEDED",
	"CETTE EXPRESSION N'EST PAS UN POLYNOME (DIVISION PAR UNE EXPRESSION)": "THIS EXPRESSION IS NOT A POLYNOMIAL (DIVISION BY AN EXPRESSION)",
	"DONNE L'EXPRESSION ENTRE CROCHETS, PAR EXEMPLE [ (X+1)(X-1) ]":        "GIVE THE EXPRESSION IN BRACKETS, FOR EXAMPLE [ (X+1)(X-1) ]",
	"DONNE LES VALEURS ENTRE CROCHETS, PAR EXEMPLE [ X 3 ]":                "GIVE THE VALUES IN BRACKETS, FOR EXAMPLE [ X 3 ]",
	"IL MANQUE UN NOMBRE OU UNE VARIABLE":                                  "A NUMBER OR A VARIABLE IS MISSING",
	"IL MANQUE UNE PARENTHESE FERMANTE":                                    "A CLOSING PARENTHESIS IS MISSING",
	"JE NE COMPRENDS PAS LA SUITE DE L'EXPRESSION":                         "I DON'T UNDERSTAND THE REST OF THE EXPRESSION",
	"JE NE RESOUS QU'UNE EQUATION A UNE SEULE INCONNUE":                    "I ONLY SOLVE EQUATIONS WITH A SINGLE UNKNOWN",
	"JE NE RESOUS QUE LE 1er ET LE 2nd DEGRE":                              "I ONLY SOLVE 1st AND 2nd DEGREE EQUATIONS",
	"L'EXPOSANT DOIT ETRE UN ENTIER POSITIF":                               "THE EXPONENT MUST BE A POSITIVE INTEGER",
	"UNE EQUATION A UN SIGNE = (EXEMPLE [ 2x + 3 = 7 ])":                   "AN EQUATION HAS AN = SIGN (EXAMPLE [ 2x + 3 = 7 ])",
	"UNE EQUATION DOIT AVOIR UN SEUL SIGNE = (EXEMPLE [ 2x + 3 = 7 ])":     "AN EQUATION MUST HAVE A SINGLE = SIGN (EXAMPLE [ 2x + 3 = 7 ])",
}

// messages a partie variable en fin : prefixe FR -> prefixe EN, la suite est gardee
var errPrefix = [][2]string{
	{"PIEGE MANQUANT POUR ", "NO CATCH FOR "},
	{"PARAMETRE EN DOUBLE : ", "DUPLICATE PARAMETER: "},
	{"NOM DE PARAMETRE INVALIDE : ", "INVALID PARAMETER NAME: "},
	{"NUMERO DE TORTUE INVALIDE : ", "INVALID TURTLE NUMBER: "},
	{"TROP DE TORTUES ", "TOO MANY TURTLES "},
	{"SPRITE TROP GRAND ", "SPRITE TOO LARGE "},
	{"CARACTERE INATTENDU : ", "UNEXPECTED CHARACTER: "},
	{"NOMBRE INCORRECT : ", "INVALID NUMBER: "},
	{"EXPOSANT TROP GRAND ", "EXPONENT TOO LARGE "},
	{"ATTENDU UNE VARIABLE (UNE LETTRE), VU ", "EXPECTED A VARIABLE (ONE LETTER), GOT "},
	{"IL MANQUE UNE VALEUR POUR ", "A VALUE IS MISSING FOR "},
}

// traduit un message FR en EN : table exacte, puis prefixe/suffixe reconnus,
// enfin le cas general "X N'AIME PAS Y". inconnu -> rendu tel quel
func localizeErr(msg string) string {
	if en, ok := errExact[msg]; ok {
		return en
	}
	if rest, ok := strings.CutPrefix(msg, "PAS ASSEZ DE DONNEES POUR "); ok {
		return "NOT ENOUGH INPUTS TO " + primNameEN(rest)
	}
	if rest, ok := strings.CutPrefix(msg, "TROP DE DONNEES POUR "); ok {
		return "TOO MANY INPUTS TO " + primNameEN(rest)
	}
	for _, pre := range errPrefix {
		if rest, ok := strings.CutPrefix(msg, pre[0]); ok {
			return pre[1] + rest
		}
	}
	if rest, ok := strings.CutSuffix(msg, " VEUT UNE EXPRESSION, PAS UNE EGALITE"); ok {
		return primNameEN(rest) + " WANTS AN EXPRESSION, NOT AN EQUATION"
	}
	if rest, ok := strings.CutPrefix(msg, "LA VALEUR DE "); ok {
		if name, ok := strings.CutSuffix(rest, " DOIT ETRE UN NOMBRE"); ok {
			return "THE VALUE OF " + name + " MUST BE A NUMBER"
		}
	}
	if rest, ok := strings.CutPrefix(msg, "PAS ASSEZ D'ELEMENTS POUR "); ok {
		return "NOT ENOUGH ITEMS FOR " + primNameEN(rest)
	}
	if rest, ok := strings.CutPrefix(msg, "ERREUR INTERNE : "); ok {
		return "INTERNAL ERROR: " + rest
	}
	if rest, ok := strings.CutPrefix(msg, "ORIGINE DE TABLEAU INVALIDE : "); ok {
		return "INVALID ARRAY ORIGIN: " + rest
	}
	if rest, ok := strings.CutPrefix(msg, "INDICE INVALIDE : "); ok {
		return "INVALID INDEX: " + rest
	}
	if rest, ok := strings.CutPrefix(msg, "POSITION INVALIDE POUR "); ok {
		return "INVALID POSITION FOR " + primNameEN(rest)
	}
	if rest, ok := strings.CutPrefix(msg, "PAS DE CHOSE DONNEE A "); ok {
		return rest + " HAS NO VALUE" // nom de variable : pas de traduction
	}
	if rest, ok := strings.CutPrefix(msg, "COMMENT FAIRE "); ok {
		return "I DON'T KNOW HOW TO " + rest
	}
	if rest, ok := strings.CutPrefix(msg, "JE N'AI PAS D'AIDE POUR "); ok {
		return "NO HELP FOR " + rest
	}
	if rest, ok := strings.CutPrefix(msg, "QUE FAIRE DE "); ok {
		return "YOU DON'T SAY WHAT TO DO WITH " + rest
	}
	if rest, ok := strings.CutPrefix(msg, "FIN MANQUANT POUR "); ok {
		return "END MISSING FOR " + rest // nom de procedure : pas de traduction
	}
	if rest, ok := strings.CutSuffix(msg, " EXISTE DEJA"); ok {
		return rest + " IS ALREADY DEFINED"
	}
	if rest, ok := strings.CutPrefix(msg, "OPERATEUR INCONNU "); ok {
		return "UNKNOWN OPERATOR " + rest
	}
	if rest, ok := strings.CutPrefix(msg, "OPERATEUR INATTENDU "); ok {
		return "UNEXPECTED OPERATOR " + rest
	}
	// erreurs lexicales du reader (en minuscules) : "X manquant" / "X inattendu"
	if rest, ok := strings.CutSuffix(msg, " manquant"); ok {
		return rest + " missing"
	}
	if rest, ok := strings.CutSuffix(msg, " inattendu"); ok {
		return rest + " unexpected"
	}
	if rest, ok := strings.CutSuffix(msg, " REFUSE UN TABLEAU CIRCULAIRE"); ok {
		return primNameEN(rest) + " REFUSES A CIRCULAR ARRAY"
	}
	// "<primitive> N'AIME PAS LA LISTE/LE MOT VIDE" : avant le cas general, sinon la
	// fin "LA LISTE VIDE" resterait en francais
	if rest, ok := strings.CutSuffix(msg, " N'AIME PAS LA LISTE VIDE"); ok {
		return primNameEN(rest) + " DOESN'T LIKE THE EMPTY LIST"
	}
	if rest, ok := strings.CutSuffix(msg, " N'AIME PAS LE MOT VIDE"); ok {
		return primNameEN(rest) + " DOESN'T LIKE THE EMPTY WORD"
	}
	// cas general "<primitive> N'AIME PAS <objet>" : on traduit le nom, on garde l'objet
	if i := strings.Index(msg, " N'AIME PAS "); i >= 0 {
		head, tail := msg[:i], msg[i+len(" N'AIME PAS "):]
		return primNameEN(head) + " DOESN'T LIKE " + tail
	}
	return msg
}

// nom canonique EN d'une primitive (ex. AVANCE -> FORWARD), inconnu rendu tel quel
func primNameEN(name string) string {
	if _, e, ok := lookupHelp(name); ok {
		return e.en
	}
	return name
}
