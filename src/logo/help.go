package logo

import (
	"fmt"
	"sort"
	"strings"
)

// decrit une primitive pour AIDE, bilingue. cle helpData = nom FR,
// `en` = nom EN, *Aliases = noms courts de chaque langue
type helpEntry struct {
	en         string   // nom canonique anglais
	frAliases  []string // alias francais (ex. AV)
	enAliases  []string // alias anglais (ex. FD)
	params     string   // parametres (en FR, traduits a l'affichage EN)
	kind       string   // C = commande, O = operation, P = predicat
	descFr     string
	descEn     string
	exemples   []string // exemples d'utilisation (commandes FR ; traduites en EN a l'affichage)
	exemplesEn []string // exemples EN explicites quand la traduction auto ne convient pas (ex. JOUE)
	palette    bool     // affiche la legende des couleurs 0-15 (commandes de couleur)
}

// noms des 16 couleurs de la palette (memes codes que le tableau `palette` du
// backend de rendu). affiches dans l'aide des commandes de couleur
var paletteFr = [16]string{
	"noir", "rouge", "vert", "jaune", "bleu", "magenta", "cyan", "blanc",
	"gris", "rouge clair", "vert clair", "jaune clair", "bleu clair", "magenta clair", "cyan clair", "orange",
}
var paletteEn = [16]string{
	"black", "red", "green", "yellow", "blue", "magenta", "cyan", "white",
	"gray", "light red", "light green", "light yellow", "light blue", "light magenta", "light cyan", "orange",
}

// la legende de la palette pour l'aide (4 couleurs par ligne)
func paletteLines(lang string) []string {
	names, title := paletteFr, "Palette des couleurs (codes 0-15) :"
	if lang == "EN" {
		names, title = paletteEn, "Color palette (codes 0-15):"
	}
	out := []string{"", title}
	for i := 0; i < 16; i += 4 {
		row := ""
		for j := i; j < i+4; j++ {
			row += fmt.Sprintf("%2d %-14s", j, names[j])
		}
		out = append(out, "  "+strings.TrimRight(row, " "))
	}
	return out
}

// aide bilingue des primitives (FR d'origine, noms EN derives).
// un test (help_test.go) verifie que ces entrees collent pile aux primitives
var helpData = map[string]helpEntry{
	// tortue : deplacements
	"AVANCE":       {en: "FORWARD", frAliases: []string{"AV"}, enAliases: []string{"FD"}, params: "n", kind: "C", exemples: []string{"AV 100"}, descFr: "Avance la tortue de n pas (max 32767). La 1re décimale est prise en compte.", descEn: "Moves the turtle forward n steps (max 32767). The first decimal is kept."},
	"RECULE":       {en: "BACK", frAliases: []string{"RE"}, enAliases: []string{"BK"}, params: "n", kind: "C", exemples: []string{"RE 50"}, descFr: "Recule la tortue de n pas.", descEn: "Moves the turtle back n steps."},
	"TOURNEDROITE": {en: "RIGHT", frAliases: []string{"TD"}, enAliases: []string{"RT"}, params: "n", kind: "C", exemples: []string{"TD 90"}, descFr: "Tourne à droite (sens horaire) de n degrés. n entre -360 et 360.", descEn: "Turns right (clockwise) by n degrees. n between -360 and 360."},
	"TOURNEGAUCHE": {en: "LEFT", frAliases: []string{"TG"}, enAliases: []string{"LT"}, params: "n", kind: "C", exemples: []string{"TG 45"}, descFr: "Tourne à gauche de n degrés. n entre -360 et 360.", descEn: "Turns left by n degrees. n between -360 and 360."},
	"POS":          {en: "POS", params: "", kind: "O", exemples: []string{"ECRIS POS"}, descFr: "Retourne la position courante [x y].", descEn: "Outputs the current position [x y]."},
	"FPOS":         {en: "SETPOS", params: "liste", kind: "C", exemples: []string{"FPOS [ 50 80 ]"}, descFr: "Déplace la tortue par une suite de points (nombre pair de coordonnées). Ne change pas le cap ; trace si le crayon est baissé.", descEn: "Moves the turtle through a list of points (even number of coordinates). Keeps the heading; draws if the pen is down."},
	"CAP":          {en: "HEADING", params: "", kind: "O", exemples: []string{"ECRIS CAP"}, descFr: "Retourne le cap courant, entre 0 et 359.9 (boussole, Nord = 0 = haut).", descEn: "Outputs the current heading, 0 to 359.9 (compass, North = 0 = up)."},
	"FCAP":         {en: "SETHEADING", enAliases: []string{"SETH"}, params: "n", kind: "C", exemples: []string{"FCAP 90"}, descFr: "Fixe le cap absolu (boussole). n entre -360 et 360.", descEn: "Sets the absolute heading (compass). n between -360 and 360."},
	"VERSCAP":      {en: "TOWARDS", params: "[x y]", kind: "O", exemples: []string{"FCAP VERSCAP [ 100 0 ]"}, descFr: "Retourne le cap (0-360) à prendre pour viser le point [x y] depuis la position de la tortue.", descEn: "Outputs the heading (0-360) to face the point [x y] from the turtle's position."},
	"FXY":          {en: "SETXY", params: "x y", kind: "C", exemples: []string{"FXY 100 50"}, descFr: "Déplace la tortue au point (x, y). Ne change pas le cap ; trace si le crayon est baissé.", descEn: "Moves the turtle to the point (x, y). Keeps the heading; draws if the pen is down."},
	"XCOR":         {en: "XCOR", params: "", kind: "O", exemples: []string{"ECRIS XCOR"}, descFr: "Retourne l'abscisse x de la tortue.", descEn: "Outputs the turtle's x coordinate."},
	"YCOR":         {en: "YCOR", params: "", kind: "O", exemples: []string{"ECRIS YCOR"}, descFr: "Retourne l'ordonnée y de la tortue.", descEn: "Outputs the turtle's y coordinate."},
	"DISTANCE":     {en: "DISTANCE", params: "[x y]", kind: "O", exemples: []string{"ECRIS DISTANCE [ 30 40 ]"}, descFr: "Retourne la distance entre la tortue et le point [x y].", descEn: "Outputs the distance from the turtle to the point [x y]."},
	"CERCLE":       {en: "CIRCLE", params: "r", kind: "C", exemples: []string{"CERCLE 50"}, descFr: "Trace un cercle de rayon r centré sur la tortue : la tortue reste à sa place (au centre), cap inchangé.", descEn: "Draws a circle of radius r centred on the turtle: the turtle stays in place (at the centre), heading unchanged."},
	"ARC":          {en: "ARC", params: "r cap1 cap2", kind: "C", exemples: []string{"ARC 50 0 90"}, descFr: "Trace un arc de cercle de rayon r centré sur la tortue, compris entre les caps boussole cap1 et cap2 (0 = haut, sens horaire). La tortue ne bouge pas.", descEn: "Draws an arc of radius r centred on the turtle, between compass headings cap1 and cap2 (0 = up, clockwise). The turtle does not move."},
	"ORIGINE":      {en: "HOME", params: "", kind: "C", exemples: []string{"ORIGINE"}, descFr: "Ramène la tortue au centre, cap 0, sans changer le crayon.", descEn: "Sends the turtle back to the center, heading 0, without changing the pen."},

	// tortue : le champ
	"CF":         {en: "BACKGROUND", enAliases: []string{"BG"}, params: "", kind: "O", palette: true, exemples: []string{"ECRIS CF"}, descFr: "Retourne le code couleur du fond graphique.", descEn: "Outputs the graphics background color code."},
	"FCFG":       {en: "SETBACKGROUND", enAliases: []string{"SETBG"}, params: "n ou liste", kind: "C", palette: true, exemples: []string{"FCFG 2", "FCFG [ 30 30 60 ]"}, descFr: "Fixe la couleur du fond graphique : code palette 0-15, ou [ rouge vert bleu ] (3 valeurs 0-255).", descEn: "Sets the graphics background color: palette code 0-15, or [ red green blue ] (3 values 0-255)."},
	"FCB":        {en: "SETBORDER", params: "n ou liste", kind: "C", palette: true, exemples: []string{"FCB 7", "FCB [ 80 80 80 ]"}, descFr: "Fixe la couleur du bord de l'écran graphique : code 0-15, ou [ rouge vert bleu ] (0-255).", descEn: "Sets the graphics screen border color: code 0-15, or [ red green blue ] (0-255)."},
	"ECH":        {en: "SCALE", params: "", kind: "O", exemples: []string{"ECRIS ECH"}, descFr: "Retourne les échelles [horizontale verticale] en pourcentage.", descEn: "Outputs the [horizontal vertical] scales, in percent."},
	"FECH":       {en: "SETSCALE", params: "liste", kind: "C", exemples: []string{"FECH [ 50 100 ]"}, descFr: "Fixe les échelles horizontale et verticale (2 nombres de 0 a 200, en %). Étire le tracé de la tortue.", descEn: "Sets the horizontal and vertical scales (2 numbers 0 to 200, in %). Stretches the turtle's drawing."},
	"CT":         {en: "HIDETURTLE", enAliases: []string{"HT"}, params: "", kind: "C", exemples: []string{"CT"}, descFr: "Cache la tortue (le tracé est plus rapide).", descEn: "Hides the turtle (drawing is faster)."},
	"MT":         {en: "SHOWTURTLE", enAliases: []string{"ST"}, params: "", kind: "C", exemples: []string{"MT"}, descFr: "Montre la tortue.", descEn: "Shows the turtle."},
	"VISIBLE?":   {en: "SHOWN?", enAliases: []string{"SHOWNP"}, params: "", kind: "P", exemples: []string{"ECRIS VISIBLE?"}, descFr: "Retourne VRAI si la tortue est visible.", descEn: "Outputs TRUE if the turtle is shown."},
	"SPRITE":     {en: "SPRITE", params: "n", kind: "C", exemples: []string{"SPRITE 1"}, descFr: "Choisit la forme de la tortue active : 0 = triangle (défaut), 1 = tortue détaillée, 2 = voiture, 3 à 255 = formes définies avec DEFSPRITE. Chaque tortue garde sa propre forme.", descEn: "Selects the active turtle's shape: 0 = triangle (default), 1 = detailed turtle, 2 = car, 3 to 255 = shapes defined with DEFSPRITE. Each turtle keeps its own shape."},
	"COLLISION?": {en: "COLLIDE?", enAliases: []string{"COLLIDEP"}, params: "liste", kind: "P", exemples: []string{"ECRIS COLLISION? [ 0 1 ]"}, descFr: "Retourne VRAI si les deux tortues de la liste [ a b ] existent, sont visibles et se chevauchent (boîtes englobantes ajustées à la taille de chaque sprite). Sinon FAUX.", descEn: "Outputs TRUE if the two turtles in the list [ a b ] exist, are visible and overlap (bounding boxes matching each sprite's size). Otherwise FALSE."},
	"DEFSPRITE":  {en: "DEFSPRITE", params: "n liste", kind: "C", exemples: []string{"DEFSPRITE 3 [ .XXXX. XXXXXX XXXXXX .XXXX. ]"}, descFr: "Définit la forme 16×16 numéro n (de 3 à 255), ensuite sélectionnable par SPRITE n. La liste donne jusqu'à 16 lignes de 16 caractères : « . » = vide, tout autre caractère (« X » par exemple) = plein. La forme est dessinée dans la couleur du crayon et pivote avec le cap. Conservée par VE, effacée par RAZ.", descEn: "Defines the 16×16 shape number n (3 to 255), then selectable with SPRITE n. The list gives up to 16 rows of 16 characters: '.' = empty, any other character ('X' for instance) = filled. The shape is drawn in the pen color and rotates with the heading. Kept by CLEARSCREEN, cleared by RESET."},
	"ANIME":      {en: "ANIMATE", params: "id x y vitesse mode", kind: "C", exemples: []string{"ANIME 0 200 100 5 \"PINGPONG"}, descFr: "Déplace automatiquement la tortue id vers le point (x,y), de `vitesse` pas Logo à chaque image, en parallèle du programme. mode est un mot : \"UNEFOIS (va à la cible puis s'arrête), \"BOUCLE (revient au départ et recommence), \"PINGPONG (aller-retour continu). La cadence se règle avec CADENCE. Une commande manuelle (AV, FXY...) sur la tortue arrête son animation.", descEn: "Automatically moves turtle id toward point (x,y), by `speed` Logo steps per frame, in parallel with the program. mode is a word: \"ONCE (go to target then stop), \"LOOP (jump back to start and repeat), \"PINGPONG (continuous back and forth). Frame rate is set with CADENCE. A manual command (FD, SETXY...) on the turtle stops its animation."},
	"STOPANIME":  {en: "STOPANIMATE", params: "id", kind: "C", exemples: []string{"STOPANIME 1", "( STOPANIME )"}, descFr: "Arrête l'animation (ANIME) de la tortue id. Entre parenthèses et sans argument, (STOPANIME) arrête toutes les animations. Ctrl+C arrête aussi les animations.", descEn: "Stops the animation (ANIMATE) of turtle id. In parentheses with no argument, (STOPANIMATE) stops all animations. Ctrl+C also stops animations."},
	"CADENCE":    {en: "SETFPS", params: "n", kind: "C", exemples: []string{"CADENCE 60"}, descFr: "Fixe la cadence du moteur d'animation à n images par seconde (1 à 240, défaut 30). La vitesse d'un ANIME (pas par image) est indépendante de la cadence.", descEn: "Sets the animation engine's frame rate to n frames per second (1 to 240, default 30). An ANIMATE's speed (steps per frame) is independent of the frame rate."},
	"VE":         {en: "CLEARSCREEN", frAliases: []string{"INIT"}, enAliases: []string{"CS", "CLS"}, params: "", kind: "C", exemples: []string{"VE"}, descFr: "Vide l'écran : réinitialise le graphique et l'état de la tortue (centre, cap 0, crayon cyan baissé, fond bleu, champ CLOS) et ne garde qu'une seule tortue, la n°0. INIT est un alias (nom XLogo).", descEn: "Clears the screen: resets the graphics and the turtle state (center, heading 0, pen down cyan, blue background, FENCE field) and keeps a single turtle, number 0. INIT is an alias (XLogo name)."},
	"NETTOIE":    {en: "CLEAN", params: "", kind: "C", exemples: []string{"NETTOIE"}, descFr: "Efface le champ sans changer l'état de la tortue ni du crayon.", descEn: "Clears the field without changing the turtle or pen state."},
	"CLOS":       {en: "FENCE", params: "", kind: "C", exemples: []string{"CLOS"}, descFr: "Champ borné : la tortue refuse de sortir de l'écran (mode par défaut).", descEn: "Bounded field: the turtle refuses to leave the screen (default mode)."},
	"ENR":        {en: "WRAP", params: "", kind: "C", exemples: []string{"ENR"}, descFr: "Champ enroulé : la tortue réapparaît au bord opposé (tore).", descEn: "Wrapped field: the turtle reappears on the opposite edge (torus)."},
	"FEN":        {en: "WINDOW", params: "", kind: "C", exemples: []string{"FEN"}, descFr: "Champ fenêtre : l'espace est étendu au-delà de l'écran visible.", descEn: "Window field: space extends beyond the visible screen."},
	"POINT":      {en: "DOT", params: "liste", kind: "C", exemples: []string{"POINT [ 0 0 50 50 ]"}, descFr: "Allume une suite de points (couleur du crayon) sans bouger la tortue.", descEn: "Lights a list of points (pen color) without moving the turtle."},

	// tortue : plusieurs tortues
	"FIXETORTUE": {en: "SETTURTLE", params: "n", kind: "C", exemples: []string{"FIXETORTUE 1"}, descFr: "Choisit la tortue active n (créée si besoin, numéros de 0 à 1023). Les commandes suivantes (AV, TD, FCC...) agissent sur elle. Le fond, l'échelle et le mode de champ restent communs à toutes les tortues.", descEn: "Selects the active turtle n (created if needed, numbers 0 to 1023). Subsequent commands (FD, RT, SETPC...) act on it. Background, scale and field mode stay shared by all turtles."},
	"TORTUE":     {en: "TURTLE", params: "", kind: "O", exemples: []string{"ECRIS TORTUE"}, descFr: "Retourne le numéro de la tortue active.", descEn: "Outputs the active turtle number."},
	"NBTORTUES":  {en: "TURTLES", params: "", kind: "O", exemples: []string{"ECRIS NBTORTUES"}, descFr: "Retourne le nombre de tortues existantes.", descEn: "Outputs the number of existing turtles."},
	"DEMANDE":    {en: "ASK", params: "n liste", kind: "C", exemples: []string{"DEMANDE 1 [ AV 50 ]", "DEMANDE [ 0 1 ] [ TD 90 ]"}, descFr: "Exécute liste sur la tortue n (ou sur chaque tortue d'une liste de numéros) puis revient à la tortue active précédente.", descEn: "Runs list on turtle n (or on each turtle of a list of numbers) then returns to the previously active turtle."},
	"DISTORTUE":  {en: "CLEARTURTLES", params: "", kind: "C", exemples: []string{"DISTORTUE"}, descFr: "Supprime toutes les tortues sauf la première (n°0) et la sélectionne.", descEn: "Removes all turtles except the first one (#0) and selects it."},

	// tortue : le crayon
	"BC":               {en: "PENDOWN", enAliases: []string{"PD"}, params: "", kind: "C", exemples: []string{"BC"}, descFr: "Baisse le crayon : la tortue trace en se déplaçant.", descEn: "Pen down: the turtle draws as it moves."},
	"LC":               {en: "PENUP", enAliases: []string{"PU"}, params: "", kind: "C", exemples: []string{"LC"}, descFr: "Lève le crayon : la tortue ne trace plus.", descEn: "Pen up: the turtle no longer draws."},
	"BC?":              {en: "PENDOWN?", enAliases: []string{"PENDOWNP"}, params: "", kind: "P", exemples: []string{"ECRIS BC?"}, descFr: "Retourne VRAI si le crayon est baissé.", descEn: "Outputs TRUE if the pen is down."},
	"CC":               {en: "PENCOLOR", enAliases: []string{"PC"}, params: "", kind: "O", palette: true, exemples: []string{"ECRIS CC"}, descFr: "Retourne le code couleur du crayon.", descEn: "Outputs the pen color code."},
	"FCC":              {en: "SETPENCOLOR", enAliases: []string{"SETPC"}, params: "n ou liste", kind: "C", palette: true, exemples: []string{"FCC 1", "FCC [ 255 128 0 ]"}, descFr: "Fixe la couleur du crayon : soit un code palette 0-15 (un code négatif gomme), soit [ rouge vert bleu ] avec 3 valeurs 0-255 pour une couleur libre.", descEn: "Sets the pen color: either a palette code 0-15 (a negative code erases), or [ red green blue ] with 3 values 0-255 for a free color."},
	"FIXETAILLECRAYON": {en: "SETPENSIZE", params: "n", kind: "C", exemples: []string{"FIXETAILLECRAYON 4"}, descFr: "Fixe l'épaisseur du trait en pixels (minimum 1 ; défaut 2).", descEn: "Sets the pen width in pixels (minimum 1; default 2)."},
	"REMPLIS":          {en: "FILL", params: "", kind: "C", exemples: []string{"REMPLIS"}, descFr: "Remplit la zone fermée qui contient la tortue. Couleur : celle fixée par FCR (couleur de remplissage) si elle l'a été, sinon la couleur du crayon.", descEn: "Fills the enclosed area that contains the turtle. Color: the one set by SETFLOODCOLOR if any, otherwise the pen color."},
	"FCR":              {en: "SETFLOODCOLOR", enAliases: []string{"SETFC"}, params: "n ou liste", kind: "C", palette: true, exemples: []string{"FCR 1", "FCR [ 255 128 0 ]"}, descFr: "Fixe la couleur de remplissage utilisée par REMPLIS, distincte de la couleur du crayon : code palette 0-15, ou [ rouge vert bleu ] (3 valeurs 0-255). Tant qu'elle n'est pas fixée, REMPLIS prend la couleur du crayon. (Compat FMSLogo : FILL se base sur cette couleur.)", descEn: "Sets the flood color used by FILL, separate from the pen color: palette code 0-15, or [ red green blue ] (3 values 0-255). Until it is set, FILL uses the pen color. (FMSLogo compatibility: FILL is based on this color.)"},
	"CR":               {en: "FLOODCOLOR", params: "", kind: "O", palette: true, exemples: []string{"ECRIS CR"}, descFr: "Retourne le code de la couleur de remplissage (FCR). Si elle n'a pas été fixée, retourne la couleur du crayon (couleur effective de REMPLIS).", descEn: "Outputs the flood color code (SETFLOODCOLOR). If it has not been set, outputs the pen color (the effective FILL color)."},
	"ETIQUETTE":        {en: "LABEL", params: "obj", kind: "C", exemples: []string{"ETIQUETTE \"BONJOUR"}, descFr: "Écrit obj dans le champ graphique, à la position de la tortue (couleur du crayon).", descEn: "Writes obj in the graphics field at the turtle's position (pen color)."},

	// mots et listes : examiner
	"EGAL?":   {en: "EQUAL?", enAliases: []string{"EQUALP"}, params: "obj1 obj2", kind: "P", exemples: []string{"ECRIS EGAL? 2 2"}, descFr: "Retourne VRAI si obj1 et obj2 sont égaux. Opérateur infixe : =.", descEn: "Outputs TRUE if obj1 and obj2 are equal. Infix operator: =."},
	"VIDE?":   {en: "EMPTY?", enAliases: []string{"EMPTYP"}, params: "obj", kind: "P", exemples: []string{"ECRIS VIDE? [ ]"}, descFr: "Retourne VRAI si obj est le mot vide ou la liste vide.", descEn: "Outputs TRUE if obj is the empty word or empty list."},
	"LISTE?":  {en: "LIST?", enAliases: []string{"LISTP"}, params: "obj", kind: "P", exemples: []string{"ECRIS LISTE? [ A B ]"}, descFr: "Retourne VRAI si obj est une liste.", descEn: "Outputs TRUE if obj is a list."},
	"MOT?":    {en: "WORD?", enAliases: []string{"WORDP"}, params: "obj", kind: "P", exemples: []string{"ECRIS MOT? \"BONJOUR"}, descFr: "Retourne VRAI si obj est un mot.", descEn: "Outputs TRUE if obj is a word."},
	"MEMBRE?": {en: "MEMBER?", enAliases: []string{"MEMBERP"}, params: "obj liste", kind: "P", exemples: []string{"ECRIS MEMBRE? \"A [ A B C ]"}, descFr: "Retourne VRAI si obj est un membre de liste.", descEn: "Outputs TRUE if obj is a member of list."},
	"NOMBRE?": {en: "NUMBER?", enAliases: []string{"NUMBERP"}, params: "obj", kind: "P", exemples: []string{"ECRIS NOMBRE? 42"}, descFr: "Retourne VRAI si obj est un nombre.", descEn: "Outputs TRUE if obj is a number."},
	"ASCII":   {en: "ASCII", params: "mot", kind: "O", exemples: []string{"ECRIS ASCII \"A"}, descFr: "Retourne le code ASCII du 1er caractère de mot (mot vide -> 0).", descEn: "Outputs the ASCII code of the first character of word (empty word -> 0)."},
	"CAR":     {en: "CHAR", params: "n", kind: "O", exemples: []string{"ECRIS CAR 65"}, descFr: "Retourne le caractère de code n (modulo 256 ; 0 -> mot vide).", descEn: "Outputs the character of code n (modulo 256; 0 -> empty word)."},
	"COMPTE":  {en: "COUNT", params: "liste", kind: "O", exemples: []string{"ECRIS COMPTE [ A B C ]"}, descFr: "Retourne le nombre de membres d'une liste (ou de caractères d'un mot).", descEn: "Outputs the number of members of a list (or characters of a word)."},

	// mots et listes : demonter
	"PREM":   {en: "FIRST", frAliases: []string{"PREMIER"}, params: "obj", kind: "O", exemples: []string{"ECRIS PREM [ A B C ]"}, descFr: "Retourne le 1er membre d'une liste (ou le 1er caractère d'un mot).", descEn: "Outputs the first member of a list (or first character of a word)."},
	"SP":     {en: "BUTFIRST", frAliases: []string{"SAUFPREMIER"}, enAliases: []string{"BF"}, params: "obj", kind: "O", exemples: []string{"ECRIS SP [ A B C ]"}, descFr: "Retourne obj sauf son premier élément.", descEn: "Outputs obj without its first element."},
	"DER":    {en: "LAST", frAliases: []string{"DERNIER"}, params: "obj", kind: "O", exemples: []string{"ECRIS DER [ A B C ]"}, descFr: "Retourne le dernier membre d'une liste (ou le dernier caractère d'un mot).", descEn: "Outputs the last member of a list (or last character of a word)."},
	"SD":     {en: "BUTLAST", frAliases: []string{"SAUFDERNIER"}, enAliases: []string{"BL"}, params: "obj", kind: "O", exemples: []string{"ECRIS SD [ A B C ]"}, descFr: "Retourne obj sauf son dernier élément.", descEn: "Outputs obj without its last element."},
	"ITEM":   {en: "ITEM", params: "n liste ou mot", kind: "O", exemples: []string{"ECRIS ITEM 2 [ A B C ]", "ECRIS ITEM 2 \"CHAT"}, descFr: "Retourne le n-ième membre d'une liste, ou le n-ième caractère d'un mot (n entier >= 1).", descEn: "Outputs the nth member of a list, or the nth character of a word (n integer >= 1)."},
	"PIOCHE": {en: "PICK", params: "liste ou mot", kind: "O", exemples: []string{"ECRIS PIOCHE [ PILE FACE ]"}, descFr: "Retourne un membre d'une liste (ou un caractère d'un mot) tiré au hasard. (Compat FMSLogo.)", descEn: "Outputs a randomly chosen member of a list (or character of a word). (FMSLogo compatibility.)"},

	// mots et listes : construire
	"MOT":       {en: "WORD", params: "mot1 mot2", kind: "O", exemples: []string{"ECRIS MOT \"BON \"JOUR"}, descFr: "Concatène mot1 et mot2 en un seul mot.", descEn: "Concatenates word1 and word2 into a single word."},
	"PHRASE":    {en: "SENTENCE", frAliases: []string{"PH"}, enAliases: []string{"SE"}, params: "obj1 obj2", kind: "O", exemples: []string{"ECRIS PHRASE \"A \"B"}, descFr: "Construit une liste à partir de obj1 et obj2 (aplatit les listes).", descEn: "Builds a list from obj1 and obj2 (flattens lists)."},
	"MAJUSCULE": {en: "UPPERCASE", params: "mot", kind: "O", exemples: []string{"ECRIS MAJUSCULE \"bonjour"}, descFr: "Retourne le mot tout en majuscules.", descEn: "Outputs the word in upper case."},
	"MINUSCULE": {en: "LOWERCASE", params: "mot", kind: "O", exemples: []string{"ECRIS MINUSCULE \"BONJOUR"}, descFr: "Retourne le mot tout en minuscules.", descEn: "Outputs the word in lower case."},
	"LISTE":     {en: "LIST", params: "obj1 obj2", kind: "O", exemples: []string{"ECRIS LISTE \"A \"B"}, descFr: "Construit la liste [obj1 obj2].", descEn: "Builds the list [obj1 obj2]."},
	"MP":        {en: "FPUT", frAliases: []string{"METSPREMIER"}, params: "obj liste", kind: "O", exemples: []string{"ECRIS MP \"A [ B C ]"}, descFr: "Ajoute obj en tête de liste.", descEn: "Adds obj at the front of list."},
	"MD":        {en: "LPUT", frAliases: []string{"METSDERNIER"}, params: "obj liste", kind: "O", exemples: []string{"ECRIS MD \"C [ A B ]"}, descFr: "Ajoute obj en queue de liste.", descEn: "Adds obj at the end of list."},

	// valeurs logiques
	"ET":   {en: "AND", params: "pred1 pred2", kind: "P", exemples: []string{"ECRIS ET VRAI FAUX", "ECRIS ( ET VRAI VRAI VRAI )"}, descFr: "ET logique : VRAI si les deux prédicats sont VRAI. Entre parenthèses, accepte plusieurs prédicats : (ET a b c).", descEn: "Logical AND: TRUE if both predicates are TRUE. In parentheses, accepts several: (AND a b c)."},
	"OU":   {en: "OR", params: "pred1 pred2", kind: "P", exemples: []string{"ECRIS OU VRAI FAUX", "ECRIS ( OU FAUX FAUX VRAI )"}, descFr: "OU logique : VRAI si au moins un prédicat est VRAI. Entre parenthèses : (OU a b c).", descEn: "Logical OR: TRUE if at least one predicate is TRUE. In parentheses: (OR a b c)."},
	"NON":  {en: "NOT", params: "pred", kind: "P", exemples: []string{"ECRIS NON VRAI"}, descFr: "Négation logique.", descEn: "Logical negation."},
	"VRAI": {en: "TRUE", params: "", kind: "O", exemples: []string{"ECRIS VRAI"}, descFr: "Retourne le mot VRAI.", descEn: "Outputs the word TRUE."},
	"FAUX": {en: "FALSE", params: "", kind: "O", exemples: []string{"ECRIS FAUX"}, descFr: "Retourne le mot FAUX.", descEn: "Outputs the word FALSE."},

	// procedures
	"POUR":  {en: "TO", params: "mot :d1 :d2 ...", kind: "C", exemples: []string{"POUR CARRE REPETE 4 [ AV 50 TD 90 ] FIN"}, descFr: "Définit une procédure nommée mot, avec d'éventuels paramètres. Terminer par FIN.", descEn: "Defines a procedure named word, with optional parameters. End with END."},
	"FIN":   {en: "END", params: "", kind: "C", exemples: []string{"POUR CARRE AV 50 FIN"}, descFr: "Termine une définition de procédure (POUR ... FIN).", descEn: "Ends a procedure definition (TO ... END)."},
	"PROC?": {en: "PROCEDURE?", enAliases: []string{"PROCEDUREP"}, params: "mot", kind: "P", exemples: []string{"ECRIS PROC? \"CARRE"}, descFr: "Retourne VRAI si mot est le nom d'une procédure utilisateur.", descEn: "Outputs TRUE if word is the name of a user procedure."},
	"PRIM?": {en: "PRIMITIVE?", enAliases: []string{"PRIMITIVEP"}, params: "mot", kind: "P", exemples: []string{"ECRIS PRIM? \"AVANCE"}, descFr: "Retourne VRAI si mot est le nom d'une primitive.", descEn: "Outputs TRUE if word is the name of a primitive."},
	"ED":    {en: "EDIT", params: "[ mot ... ]", kind: "C", exemples: []string{"ED \"CARRE"}, descFr: "Ouvre l'éditeur plein écran (et arrête les animations en cours). ED seul rouvre le contenu ; ED \"NOM ou ED [ A B ] édite des procédures ; ED [ ] part d'un éditeur vide. Ctrl+S sauve, Ctrl+X quitte sans sauver.", descEn: "Opens the full-screen editor (and stops any running animations). EDIT alone reopens the buffer; EDIT \"NAME or EDIT [ A B ] edits procedures; EDIT [ ] starts empty. Ctrl+S saves, Ctrl+X quits without saving."},

	// controle d'execution
	"SI":           {en: "IF", params: "pred liste", kind: "C", exemples: []string{"SI 2 > 1 [ ECRIS \"OUI ]", "SI :N = 0 [ ECRIS \"ZERO ]"}, descFr: "Conditionnelle : exécute liste si pred est VRAI, sinon ne fait rien. Pour un cas « sinon », voir SISINON.", descEn: "Conditional: runs list if pred is TRUE, otherwise does nothing. For an else branch, see IFELSE."},
	"SISINON":      {en: "IFELSE", params: "pred liste1 liste2", kind: "C", exemples: []string{"SISINON :N = 0 [ ECRIS \"ZERO ] [ ECRIS \"AUTRE ]"}, descFr: "Conditionnelle à deux branches : exécute liste1 si pred est VRAI, sinon exécute liste2.", descEn: "Two-branch conditional: runs list1 if pred is TRUE, otherwise runs list2."},
	"REPETE":       {en: "REPEAT", params: "n liste", kind: "C", exemples: []string{"REPETE 4 [ AV 50 TD 90 ]", "REPETE 36 [ AV 20 TD 10 ]"}, descFr: "Répète l'exécution de liste n fois (n entre 0 et 65535).", descEn: "Runs list n times (n between 0 and 65535)."},
	"COMPTEUR":     {en: "REPCOUNT", params: "", kind: "O", exemples: []string{"REPETE 3 [ ECRIS COMPTEUR ]"}, descFr: "Retourne le numéro de l'itération en cours de REPETE (1 à n), ou -1 hors d'une boucle.", descEn: "Outputs the current iteration number of REPEAT (1 to n), or -1 outside a loop."},
	"STOP":         {en: "STOP", params: "", kind: "C", exemples: []string{"SI :N = 0 [ STOP ]"}, descFr: "Termine immédiatement la procédure courante.", descEn: "Immediately ends the current procedure."},
	"RENDS":        {en: "OUTPUT", frAliases: []string{"RETOURNE", "RET"}, enAliases: []string{"OP"}, params: "obj", kind: "C", exemples: []string{"RENDS :N * 2"}, descFr: "Termine la procédure courante en retournant obj (en fait une opération). Alias : RETOURNE, RET.", descEn: "Ends the current procedure, outputting obj (makes it an operation)."},
	"LOGO":         {en: "TOPLEVEL", frAliases: []string{"STOPTOUT"}, enAliases: []string{"STOPALL"}, params: "", kind: "C", exemples: []string{"LOGO"}, descFr: "Arrêt total : retour au niveau supérieur.", descEn: "Full stop: return to top level."},
	"RAZ":          {en: "RESET", params: "", kind: "C", exemples: []string{"RAZ"}, descFr: "Après confirmation (O/N), remet tout à zéro comme au démarrage : efface procédures et variables, réinitialise la tortue et l'écran.", descEn: "After confirmation (Y/N), resets everything to the startup state: erases procedures and variables, resets the turtle and the screen."},
	"EXEC":         {en: "RUN", params: "liste", kind: "C", exemples: []string{"EXEC [ AV 50 ]"}, descFr: "Exécute liste ; retourne son résultat si c'est une opération. Brique des structures de contrôle.", descEn: "Runs list; outputs its result if it is an operation. Building block for control structures."},
	"DEFINIS":      {en: "DEFINE", frAliases: []string{"DEF"}, params: "nom texte", kind: "C", exemples: []string{"DEFINIS \"CARRE [ [ ] [ REPETE 4 [ AV 50 TD 90 ] ] ]", "DEFINIS \"POLY [ NB LG ] [ REPETE :NB [ AV :LG TD 360 / :NB ] ]"}, descFr: "Définit la procédure nom à partir de données (procédures vues comme données, façon Logo adulte). Deux formes acceptées : imbriquée façon UCBLogo, DEFINIS \"nom [ [paramètres] [corps] ] ; ou à 3 arguments façon XLogo, DEFINIS \"nom [paramètres] [corps]. Inverse de TEXTE.", descEn: "Defines procedure name from data (procedures as data, advanced Logo style). Two accepted forms: UCBLogo-style nested, DEFINE \"name [ [inputs] [body] ] ; or XLogo-style 3-argument, DEFINE \"name [inputs] [body]. Inverse of TEXT."},
	"TEXTE":        {en: "TEXT", params: "nom", kind: "O", exemples: []string{"POUR CARRE REPETE 4 [ AV 50 TD 90 ] FIN MONTRE TEXTE \"CARRE"}, descFr: "Rend la définition de la procédure nom sous forme de liste [ [paramètres] [corps] ]. Inverse de DEFINIS.", descEn: "Outputs the definition of procedure name as a list [ [inputs] [body] ]. Inverse of DEFINE."},
	"DONNEPROP":    {en: "PPROP", params: "obj prop valeur", kind: "C", exemples: []string{"DONNEPROP \"TINTIN \"AGE 17"}, descFr: "Donne à l'objet obj la propriété prop = valeur (listes de propriétés, façon Logo adulte). Remplace si la propriété existe déjà.", descEn: "Gives object obj the property prop = value (property lists, advanced Logo style). Replaces it if the property already exists."},
	"PROP":         {en: "GPROP", params: "obj prop", kind: "O", exemples: []string{"DONNEPROP \"TINTIN \"AGE 17 ECRIS PROP \"TINTIN \"AGE"}, descFr: "Rend la valeur de la propriété prop de l'objet obj, ou la liste vide [ ] si elle n'existe pas.", descEn: "Outputs the value of property prop of object obj, or the empty list [ ] if it does not exist."},
	"EFPROP":       {en: "REMPROP", params: "obj prop", kind: "C", exemples: []string{"EFPROP \"TINTIN \"AGE"}, descFr: "Efface la propriété prop de l'objet obj.", descEn: "Removes property prop from object obj."},
	"LISTEPROP":    {en: "PLIST", params: "obj", kind: "O", exemples: []string{"DONNEPROP \"TINTIN \"AGE 17 MONTRE LISTEPROP \"TINTIN"}, descFr: "Rend la liste des propriétés de obj : [ prop1 valeur1 prop2 valeur2 ... ], ou [ ] si obj n'a aucune propriété.", descEn: "Outputs the property list of obj: [ prop1 value1 prop2 value2 ... ], or [ ] if obj has no property."},
	"EXECRESULTAT": {en: "RUNRESULT", params: "liste", kind: "O", exemples: []string{"MONTRE EXECRESULTAT [ SOMME 2 3 ]", "MONTRE EXECRESULTAT [ AV 50 ]"}, descFr: "Exécute liste comme EXEC, mais emballe le résultat : rend [valeur] si c'était une opération, [ ] (liste vide) si c'était une commande. Permet de tester si quelque chose a été rendu.", descEn: "Runs list like RUN, but wraps the result: outputs [value] if it was an operation, [ ] (empty list) if it was a command. Lets you test whether anything was output."},
	"ATTENDS":      {en: "WAIT", params: "n", kind: "C", exemples: []string{"ATTENDS 30"}, descFr: "Fait une pause de n soixantièmes de seconde (ATTENDS 60 = 1 seconde). Interruptible par Ctrl+C.", descEn: "Pauses for n sixtieths of a second (WAIT 60 = 1 second). Interruptible with Ctrl+C."},

	// les noms
	"DONNE": {en: "MAKE", frAliases: []string{"FIXE"}, params: "mot obj", kind: "C", exemples: []string{"DONNE \"AGE 10", "DONNE \"NOM [ JEAN DUPONT ]"}, descFr: "Donne le nom mot à l'objet obj (variable). On récupère sa valeur avec :mot ou CHOSE.", descEn: "Gives the name word to object obj (variable). Read its value with :word or THING."},
	"LOCAL": {en: "LOCAL", params: "mot ou liste", kind: "C", exemples: []string{"LOCAL \"COMPTE", "LOCAL [ X Y ]"}, descFr: "Déclare une ou plusieurs variables locales à la procédure en cours : DONNE n'affecte alors que la copie locale, pas la variable globale du même nom.", descEn: "Declares one or more variables local to the current procedure: MAKE then only affects the local copy, not the global of the same name."},
	"CHOSE": {en: "THING", params: "mot", kind: "O", exemples: []string{"DONNE \"AGE 10 ECRIS CHOSE \"AGE"}, descFr: "Retourne la chose (valeur) désignée par mot. :mot est l'abréviation de CHOSE \"mot.", descEn: "Outputs the thing (value) named by word. :word is short for THING \"word."},
	"NOM?":  {en: "NAME?", enAliases: []string{"NAMEP"}, params: "mot", kind: "P", exemples: []string{"ECRIS NOM? \"AGE"}, descFr: "Retourne VRAI si mot désigne une chose (variable définie).", descEn: "Outputs TRUE if word names a thing (defined variable)."},

	// arithmetique
	"SOMME":  {en: "SUM", params: "n1 n2", kind: "O", exemples: []string{"ECRIS SOMME 3 4"}, descFr: "Somme n1 + n2. Opérateur infixe : +.", descEn: "Sum n1 + n2. Infix operator: +."},
	"DIFF":   {en: "DIFFERENCE", frAliases: []string{"DIF"}, params: "n1 n2", kind: "O", exemples: []string{"ECRIS DIFF 10 4"}, descFr: "Différence n1 - n2. Opérateur infixe : -.", descEn: "Difference n1 - n2. Infix operator: -."},
	"PROD":   {en: "PRODUCT", frAliases: []string{"PRODUIT"}, params: "n1 n2", kind: "O", exemples: []string{"ECRIS PROD 6 7"}, descFr: "Produit n1 * n2. Opérateur infixe : *.", descEn: "Product n1 * n2. Infix operator: *."},
	"DIV":    {en: "QUOTIENT", frAliases: []string{"DIVISE"}, params: "n1 n2", kind: "O", exemples: []string{"ECRIS DIV 20 4"}, descFr: "Quotient n1 / n2 (n2 different de 0). Opérateur infixe : /.", descEn: "Quotient n1 / n2 (n2 not 0). Infix operator: /."},
	"QUOT":   {en: "INTQUOTIENT", params: "n1 n2", kind: "O", exemples: []string{"ECRIS QUOT 17 5"}, descFr: "Quotient entier de n1 par n2 (n2 different de 0).", descEn: "Integer quotient of n1 by n2 (n2 not 0)."},
	"RESTE":  {en: "REMAINDER", params: "n1 n2", kind: "O", exemples: []string{"ECRIS RESTE 17 5"}, descFr: "Reste de la division entière de n1 par n2 (n2 different de 0).", descEn: "Remainder of the integer division of n1 by n2 (n2 not 0)."},
	"MODULO": {en: "MODULO", params: "n1 n2", kind: "O", exemples: []string{"ECRIS MODULO -17 5"}, descFr: "Reste de la division de n1 par n2, du signe de n2 (différent de RESTE pour les négatifs).", descEn: "Remainder of n1 divided by n2, with the sign of n2 (differs from REMAINDER for negatives)."},
	"PLP?":   {en: "LESS?", enAliases: []string{"LESSP"}, params: "n1 n2", kind: "P", exemples: []string{"ECRIS PLP? 3 5"}, descFr: "Retourne VRAI si n1 < n2. Opérateur infixe : <.", descEn: "Outputs TRUE if n1 < n2. Infix operator: <."},
	"PLG?":   {en: "GREATER?", enAliases: []string{"GREATERP"}, params: "n1 n2", kind: "P", exemples: []string{"ECRIS PLG? 5 3"}, descFr: "Retourne VRAI si n1 > n2. Opérateur infixe : >.", descEn: "Outputs TRUE if n1 > n2. Infix operator: >."},
	"ENT":    {en: "INT", params: "n", kind: "O", exemples: []string{"ECRIS ENT 3.7"}, descFr: "Partie entière de n (arrondi vers le bas : ENT -1.5 = -2).", descEn: "Integer part of n (rounded down: INT -1.5 = -2)."},
	"HASARD": {en: "RANDOM", params: "n", kind: "O", exemples: []string{"ECRIS HASARD 6", "ECRIS ( HASARD 1 6 )"}, descFr: "Retourne un entier aléatoire entre 0 et n-1 (n >= 1). Entre parenthèses, (HASARD debut fin) tire entre debut et fin inclus.", descEn: "Outputs a random integer between 0 and n-1 (n >= 1). In parentheses, (RANDOM start end) picks between start and end inclusive."},
	"SIN":    {en: "SIN", frAliases: []string{"SINUS"}, params: "n", kind: "O", exemples: []string{"ECRIS SIN 30"}, descFr: "Sinus de n (n en degrés).", descEn: "Sine of n (n in degrees)."},
	"COS":    {en: "COS", frAliases: []string{"COSINUS"}, params: "n", kind: "O", exemples: []string{"ECRIS COS 60"}, descFr: "Cosinus de n (n en degrés).", descEn: "Cosine of n (n in degrees)."},
	"RC":     {en: "SQRT", frAliases: []string{"RACINE", "RAC"}, params: "n", kind: "O", exemples: []string{"ECRIS RC 16"}, descFr: "Racine carrée de n (n >= 0).", descEn: "Square root of n (n >= 0)."},
	"VALABS": {en: "ABS", params: "n", kind: "O", exemples: []string{"ECRIS VALABS -7"}, descFr: "Valeur absolue de n.", descEn: "Absolute value of n."},
	"ARCTAN": {en: "ARCTAN", frAliases: []string{"ATAN"}, params: "n", kind: "O", exemples: []string{"ECRIS ARCTAN 1"}, descFr: "Arc tangente de n, en degrés.", descEn: "Arctangent of n, in degrees."},
	"EXP":    {en: "EXP", params: "n", kind: "O", exemples: []string{"ECRIS EXP 1"}, descFr: "Exponentielle : e à la puissance n.", descEn: "Exponential: e to the power n."},
	"LN":     {en: "LN", params: "n", kind: "O", exemples: []string{"ECRIS LN 1"}, descFr: "Logarithme naturel de n (n > 0).", descEn: "Natural logarithm of n (n > 0)."},
	"LOG10":  {en: "LOG10", params: "n", kind: "O", exemples: []string{"ECRIS LOG10 100"}, descFr: "Logarithme en base 10 de n (n > 0).", descEn: "Base-10 logarithm of n (n > 0)."},

	// sortie texte
	"ECRIS":  {en: "PRINT", frAliases: []string{"EC"}, enAliases: []string{"PR"}, params: "obj", kind: "C", exemples: []string{"ECRIS 5 + 7", "ECRIS [ BONJOUR LE MONDE ]", "( ECRIS \"VALEUR 42 )"}, descFr: "Écrit obj dans la zone de texte, puis passe à la ligne. Entre parenthèses, écrit plusieurs objets séparés par une espace : (ECRIS a b c).", descEn: "Prints obj in the text area, then starts a new line. In parentheses, prints several objects separated by a space: (PRINT a b c)."},
	"TAPE":   {en: "TYPE", params: "obj", kind: "C", exemples: []string{"TAPE \"BONJOUR", "( TAPE \"A \"B )"}, descFr: "Écrit obj sans passer à la ligne. Entre parenthèses : (TAPE a b c).", descEn: "Prints obj without a new line. In parentheses: (TYPE a b c)."},
	"MONTRE": {en: "SHOW", params: "obj", kind: "C", exemples: []string{"MONTRE [ A B C ]"}, descFr: "Comme ECRIS, mais garde les crochets autour des listes : MONTRE [ A B ] affiche [A B]. Variable entre parenthèses.", descEn: "Like PRINT, but keeps the brackets around lists: SHOW [ A B ] prints [A B]. Variadic in parentheses."},
	"VT":     {en: "CLEARTEXT", params: "", kind: "C", exemples: []string{"VT"}, descFr: "Vide la zone de texte.", descEn: "Clears the text area."},
	"FCT":    {en: "SETTEXTCOLOR", params: "n ou liste", kind: "C", palette: true, exemples: []string{"FCT 3", "FCT [ 255 200 0 ]"}, descFr: "Fixe la couleur du texte : code 0-15, ou [ rouge vert bleu ] (0-255).", descEn: "Sets the text color: code 0-15, or [ red green blue ] (0-255)."},
	"FCFT":   {en: "SETTEXTBACKGROUND", params: "n ou liste", kind: "C", palette: true, exemples: []string{"FCFT 4", "FCFT [ 20 20 20 ]"}, descFr: "Fixe la couleur de fond du texte : code 0-15, ou [ rouge vert bleu ] (0-255).", descEn: "Sets the text background color: code 0-15, or [ red green blue ] (0-255)."},
	"FCURS":  {en: "SETCURSOR", params: "[col lig]", kind: "C", exemples: []string{"FCURS [ 10 5 ]", "VT FCURS [ 0 24 ] TAPE \"BAS"}, descFr: "Place le curseur texte : colonne col (0-39) et ligne lig (0-24). Le prochain ECRIS/TAPE écrit à cet endroit.", descEn: "Places the text cursor: column col (0-39), line lig (0-24). The next PRINT/TYPE writes there."},
	"ME":     {en: "SETTEXTLINES", params: "n", kind: "C", exemples: []string{"ME 25", "ME 5"}, descFr: "Fixe le nombre de lignes de texte visibles (1-25). ME 25 = plein texte (cache le graphique) ; n < 25 = écran mixte (graphique + n lignes de texte).", descEn: "Sets the number of visible text lines (1-25). SETTEXTLINES 25 = full text (hides graphics); n < 25 = mixed screen (graphics + n text lines)."},

	// clavier
	"LISCAR":  {en: "READCHAR", params: "", kind: "O", exemples: []string{"DONNE \"C LISCAR"}, descFr: "Attend une frappe au clavier et retourne le caractère tapé (un mot d'un caractère). Entrée -> retour-chariot.", descEn: "Waits for a keystroke and outputs the typed character (a one-character word). Enter -> carriage return."},
	"LL":      {en: "READLIST", enAliases: []string{"RL"}, params: "", kind: "O", exemples: []string{"DONNE \"L LL"}, descFr: "Attend une ligne au clavier (jusqu'à Entrée) et la retourne sous forme de liste de mots.", descEn: "Waits for a line of input (until Enter) and outputs it as a list of words."},
	"LISMOT":  {en: "READWORD", enAliases: []string{"RW"}, params: "", kind: "O", exemples: []string{"DONNE \"L LISMOT"}, descFr: "Attend une ligne au clavier (jusqu'à Entrée) et la retourne comme un seul mot.", descEn: "Waits for a line of input (until Enter) and outputs it as a single word."},
	"TOUCHE?": {en: "KEY?", enAliases: []string{"KEYP"}, params: "", kind: "P", exemples: []string{"SI TOUCHE? [ ECRIS \"OUI ]"}, descFr: "Retourne VRAI si une touche est disponible au clavier, sans attendre (pour LISCAR).", descEn: "Outputs TRUE if a key is available, without waiting (for READCHAR)."},

	// souris
	"SOURISPOS":     {en: "MOUSEPOS", frAliases: []string{"POSOPT"}, params: "", kind: "O", exemples: []string{"ECRIS SOURISPOS"}, descFr: "Retourne la position de la souris [x y] si le pointeur est dans le champ, sinon la liste vide. (Alias POSOPT : crayon optique d'origine.)", descEn: "Outputs the mouse position [x y] if the pointer is in the field, otherwise the empty list."},
	"SOURISPRESSEE": {en: "MOUSEDOWN?", frAliases: []string{"CONTACT?", "CONTACT"}, params: "", kind: "P", exemples: []string{"SI SOURISPRESSEE [ ECRIS \"CLIC ]"}, descFr: "Retourne VRAI si un bouton de la souris est enfoncé. (Alias CONTACT? : crayon optique d'origine.)", descEn: "Outputs TRUE if a mouse button is pressed."},

	// manettes (emulees au clavier)
	"MANETTE": {en: "JOYSTICK", params: "n", kind: "O", exemples: []string{"SI ( MANETTE 0 ) = 1 [ AV 10 ]"}, descFr: "Retourne la direction de la manette n (0-8) : 0 = repos, 1 = haut, puis dans le sens horaire (3 = droite, 5 = bas, 7 = gauche, et les diagonales). Émulée au clavier par les flèches (manette 0).", descEn: "Outputs the direction of joystick n (0-8): 0 = centered, 1 = up, then clockwise (3 = right, 5 = down, 7 = left, plus diagonals). Emulated with the arrow keys (joystick 0)."},
	"BOUTON?": {en: "BUTTON?", enAliases: []string{"BUTTONP"}, params: "n", kind: "P", exemples: []string{"SI BOUTON? 0 [ ECRIS \"FEU ]"}, descFr: "Retourne VRAI si le bouton de tir de la manette n est enfoncé. Émulé au clavier par la barre d'espace (manette 0).", descEn: "Outputs TRUE if joystick n's fire button is pressed. Emulated with the space bar (joystick 0)."},

	// musique
	"JOUE":   {en: "PLAY", params: "note ou liste", kind: "C", exemples: []string{"JOUE \"DO", "JOUE [ DO RE MI FA SOL LA SI ]", "JOUE [ DO RE# MI ]"}, exemplesEn: []string{"PLAY \"DO", "PLAY [ DO RE MI FA SOL LA SI ]", "PLAY [ DO RE# MI ]"}, descFr: "Joue une note seule ou une liste de notes jouées l'une après l'autre. Notes : DO RE MI FA SO(L) LA SI, et PA pour un silence. Dièse : # (ou <#), bémol : b (ou <b) ; ex. FA# ou MIb. Chaque note utilise OCTAVE/DUREE/TEMPO/TIMBRE en cours. Interruptible par Ctrl+C.", descEn: "Plays a single note or a list of notes one after another. Notes: DO RE MI FA SO(L) LA SI, and PA for a rest. Sharp: # (or <#), flat: b (or <b); e.g. FA# or MIb. Each note uses the current OCTAVE/DURATION/TEMPO/TIMBRE. Interruptible with Ctrl+C."},
	"OCTAVE": {en: "OCTAVE", params: "n", kind: "C", exemples: []string{"OCTAVE 4"}, descFr: "Fixe l'octave des notes jouées par JOUE (1 à 5, défaut 4).", descEn: "Sets the octave of notes played by PLAY (1 to 5, default 4)."},
	"DUREE":  {en: "DURATION", params: "n", kind: "C", exemples: []string{"DUREE 24"}, descFr: "Fixe la durée des notes (1 à 96, défaut 24).", descEn: "Sets the note length (1 to 96, default 24)."},
	"TEMPO":  {en: "TEMPO", params: "n", kind: "C", exemples: []string{"TEMPO 5"}, descFr: "Fixe le tempo (1 à 255, défaut 5) : plus grand = plus rapide.", descEn: "Sets the tempo (1 to 255, default 5): higher = faster."},
	"TIMBRE": {en: "TIMBRE", params: "n", kind: "C", exemples: []string{"TIMBRE 0", "TIMBRE 200"}, descFr: "Fixe le timbre, c'est-à-dire la forme du son (0-63 carré, 64-127 dent de scie, 128-191 triangle, 192-255 sinus).", descEn: "Sets the timbre, i.e. the sound shape (0-63 square, 64-127 sawtooth, 128-191 triangle, 192-255 sine)."},

	// espace de travail
	"CONTENU": {en: "CONTENTS", params: "", kind: "O", exemples: []string{"ECRIS CONTENU"}, descFr: "Retourne la liste de tous les mots connus (procédures et noms).", descEn: "Outputs the list of all known words (procedures and names)."},
	"IM":      {en: "PO", params: "mot", kind: "C", exemples: []string{"IM \"CARRE"}, descFr: "Imprime la définition de la procédure nommée mot.", descEn: "Prints out the definition of the procedure named word."},
	"IMTS":    {en: "POTS", params: "", kind: "C", exemples: []string{"IMTS"}, descFr: "Imprime les titres (ligne POUR) de toutes les procédures.", descEn: "Prints out the titles (TO line) of all procedures."},
	"IMNS":    {en: "PONS", params: "", kind: "C", exemples: []string{"IMNS"}, descFr: "Imprime tous les noms et leurs choses, sous forme DONNE.", descEn: "Prints out all names and their things, as MAKE statements."},
	"IMTOUT":  {en: "POALL", params: "", kind: "C", exemples: []string{"IMTOUT"}, descFr: "Imprime toutes les procédures (définitions complètes) puis tous les noms.", descEn: "Prints out all procedures (full definitions) then all names."},
	"EFP":     {en: "ERASE", enAliases: []string{"ER"}, params: "mot", kind: "C", exemples: []string{"EFP \"CARRE"}, descFr: "Efface la procédure nommée mot.", descEn: "Erases the procedure named word."},
	"EFN":     {en: "ERNAME", enAliases: []string{"ERN"}, params: "mot", kind: "C", exemples: []string{"EFN \"AGE"}, descFr: "Efface le nom (variable) mot.", descEn: "Erases the name (variable) word."},
	".EFT":    {en: "ERALL", params: "", kind: "C", exemples: []string{".EFT"}, descFr: "Efface tout l'espace de travail : toutes les procédures et tous les noms.", descEn: "Erases the whole workspace: all procedures and all names."},
	"PLACE":   {en: "NODES", params: "", kind: "O", exemples: []string{"ECRIS PLACE"}, descFr: "Retourne le nombre de cellules mémoire disponibles.", descEn: "Outputs the number of free memory cells."},
	"RECYCLE": {en: "RECYCLE", params: "", kind: "C", exemples: []string{"RECYCLE"}, descFr: "Récupère la mémoire inutilisée (sans effet : Go gère la mémoire).", descEn: "Reclaims unused memory (no effect: Go manages memory)."},

	// extensions
	"PUISSANCE":  {en: "POWER", params: "n1 n2", kind: "O", exemples: []string{"ECRIS PUISSANCE 2 10"}, descFr: "Retourne n1 élevé à la puissance n2.", descEn: "Outputs n1 raised to the power n2."},
	"ARRONDI":    {en: "ROUND", params: "n", kind: "O", exemples: []string{"ECRIS ARRONDI 3.6"}, descFr: "Retourne l'entier le plus proche de n.", descEn: "Outputs the nearest integer to n."},
	"TRONQUE":    {en: "TRUNCATE", params: "n", kind: "O", exemples: []string{"ECRIS TRONQUE 3.9"}, descFr: "Retourne la partie entière de n (troncature vers zéro).", descEn: "Outputs the integer part of n (truncated toward zero)."},
	"MOINS":      {en: "MINUS", params: "n", kind: "O", exemples: []string{"ECRIS MOINS 5"}, descFr: "Retourne l'opposé de n (-n).", descEn: "Outputs the negation of n (-n)."},
	"TANGENTE":   {en: "TAN", frAliases: []string{"TAN"}, params: "n", kind: "O", exemples: []string{"ECRIS TAN 45"}, descFr: "Tangente de n (n en degrés).", descEn: "Tangent of n (n in degrees)."},
	"INVERSE":    {en: "REVERSE", params: "obj", kind: "O", exemples: []string{"ECRIS INVERSE [ A B C ]"}, descFr: "Retourne la liste (ou le mot) avec ses éléments en ordre inverse.", descEn: "Outputs the list (or word) with its elements in reverse order."},
	"TRIE":       {en: "SORT", params: "liste", kind: "O", exemples: []string{"ECRIS TRIE [ 3 1 2 ]", "ECRIS TRIE [ POMME CERISE BANANE ]"}, descFr: "Retourne la liste triée : les nombres par valeur croissante, sinon par ordre alphabétique. Tri stable.", descEn: "Outputs the list sorted: numbers by increasing value, otherwise alphabetically. Stable sort."},
	"PI":         {en: "PI", params: "", kind: "O", exemples: []string{"ECRIS PI"}, descFr: "Retourne la constante pi (3.14159...).", descEn: "Outputs the constant pi (3.14159...)."},
	"TANTQUE":    {en: "WHILE", params: "[cond] [instr]", kind: "C", exemples: []string{"TANTQUE [ :N > 0 ] [ AV 10 DONNE \"N :N - 1 ]"}, descFr: "Exécute instr tant que l'évaluation de cond rend VRAI (boucle while).", descEn: "Runs instr while cond evaluates to TRUE (while loop)."},
	"REPETEPOUR": {en: "FOR", params: "[var debut fin pas] [instr]", kind: "C", exemples: []string{"REPETEPOUR [ I 1 4 ] [ AV 50 TD 90 ]"}, descFr: "Boucle pour : var va de debut a fin (par pas, defaut 1) ; instr execute a chaque tour.", descEn: "For loop: var goes from start to end (by step, default 1); runs instr each time."},
	"SCENE":      {en: "SCENE", params: "[instr]", kind: "C", exemples: []string{"SCENE [ NETTOIE CARRE ]"}, exemplesEn: []string{"SCENE [ CLEAN SQUARE ]"}, descFr: "Dessine instr dans un tampon caché puis l'affiche d'un seul coup (double tampon). Évite le clignotement des animations qui font NETTOIE puis redessinent à chaque image. À utiliser autour du dessin d'une image dans une boucle.", descEn: "Draws instr in a hidden buffer, then shows it all at once (double buffering). Prevents flicker in animations that CLEAN then redraw every frame. Use it around the drawing of one frame inside a loop."},
	"LIS":        {en: "READ", params: "titre mot", kind: "C", exemples: []string{"LIS [ QUEL AGE AS-TU ] \"AGE"}, exemplesEn: []string{"READ [ HOW OLD ARE YOU ] \"AGE"}, descFr: "Affiche titre (une question) puis lit une saisie au clavier et la range dans la variable mot : un mot si on tape un seul mot, une liste si on en tape plusieurs. (Façon XLogo, ici en mode texte.)", descEn: "Displays title (a question) then reads keyboard input and stores it in variable word: a word for a single word, a list for several. (XLogo style, here in text mode.)"},
	"POURCHAQUE": {en: "FOREACH", params: "liste [instr]", kind: "C", exemples: []string{"POURCHAQUE [ 1 2 3 ] [ ECRIS :? ]", "POURCHAQUE \"I [ 1 2 3 ] [ ECRIS :I ]"}, descFr: "Exécute instr pour chaque élément de liste (ou caractère d'un mot). Deux formes : notre forme POURCHAQUE liste [instr] avec l'élément courant dans :? ; ou la forme XLogo POURCHAQUE \"var liste-ou-mot [instr] avec une variable nommée (:var).", descEn: "Runs instr for each member of a list (or character of a word). Two forms: our form FOREACH list [instr] with the current item in :? ; or the XLogo form FOREACH \"var list-or-word [instr] with a named variable (:var)."},
	"APPLIQUE":   {en: "MAP", params: "liste [expr]", kind: "O", exemples: []string{"ECRIS APPLIQUE [ 1 2 3 ] [ :? * :? ]"}, descFr: "Retourne la liste des résultats du gabarit appliqué à chaque élément (:? = élément).", descEn: "Outputs the list of the template's results applied to each member (:? = member)."},
	"FILTRE":     {en: "FILTER", params: "liste [pred]", kind: "O", exemples: []string{"ECRIS FILTRE [ 1 2 3 4 ] [ :? > 2 ]"}, descFr: "Retourne les éléments de liste pour lesquels le prédicat (:?) est VRAI.", descEn: "Outputs the members of list for which the predicate (:?) is TRUE."},
	"REDUIS":     {en: "REDUCE", params: "liste [expr]", kind: "O", exemples: []string{"ECRIS REDUIS [ 1 2 3 4 ] [ :?1 + :?2 ]"}, descFr: "Replie la liste en une valeur : :?1 = accumulateur, :?2 = élément courant.", descEn: "Folds the list into one value: :?1 = accumulator, :?2 = current member."},
	"PIEGE":      {en: "CATCH", params: "mot [instr]", kind: "C", exemples: []string{"PIEGE \"FIN [ LANCE \"FIN ]"}, descFr: "Exécute instr ; attrape un LANCE de même étiquette. Étiquette ERREUR : attrape aussi les erreurs d'exécution.", descEn: "Runs instr; catches a THROW with the same tag. Tag ERROR also catches runtime errors."},
	"LANCE":      {en: "THROW", params: "mot", kind: "C", exemples: []string{"PIEGE \"FIN [ LANCE \"FIN ]"}, descFr: "Interrompt l'exécution et saute jusqu'au PIEGE de même étiquette.", descEn: "Stops execution and jumps to the CATCH with the same tag."},
	"TESTE":      {en: "TEST", params: "pred", kind: "C", exemples: []string{"TESTE 2 > 1 SIVRAI [ ECRIS \"OUI ]"}, descFr: "Mémorise le résultat d'un prédicat, pour SIVRAI / SIFAUX.", descEn: "Remembers a predicate's result, for IFTRUE / IFFALSE."},
	"SIVRAI":     {en: "IFTRUE", enAliases: []string{"IFT"}, params: "[instr]", kind: "C", exemples: []string{"TESTE 2 > 1 SIVRAI [ ECRIS \"OUI ]"}, descFr: "Exécute instr si le dernier TESTE était VRAI.", descEn: "Runs instr if the last TEST was TRUE."},
	"SIFAUX":     {en: "IFFALSE", enAliases: []string{"IFF"}, params: "[instr]", kind: "C", exemples: []string{"TESTE 1 > 2 SIFAUX [ ECRIS \"NON ]"}, descFr: "Exécute instr si le dernier TESTE était FAUX.", descEn: "Runs instr if the last TEST was FALSE."},

	// fichiers (dossier de travail "Logo", fichiers .GLG)
	"SAUVE":     {en: "SAVE", params: "mot liste", kind: "C", exemples: []string{"SAUVE \"DESSIN CONTENU"}, descFr: "Sauve dans le fichier mot les procédures et variables désignées par liste (noms nus = procédures, \"nom = variables, :nom = liste de noms ; CONTENU = tout).", descEn: "Saves to file word the procedures and variables given by list (bare names = procedures, \"name = variables, :name = list of names; CONTENTS = everything)."},
	"SAUVEPNG":  {en: "SAVEPNG", params: "mot", kind: "C", exemples: []string{"SAUVEPNG \"DESSIN"}, descFr: "Sauve le dessin (champ graphique) dans mot.PNG, dans le dossier de travail (comme SAUVE). Demande confirmation si le fichier existe.", descEn: "Saves the drawing (graphics field) to word.PNG, in the working folder (like SAVE). Asks for confirmation if the file exists."},
	"RAMENE":    {en: "LOAD", params: "mot", kind: "C", exemples: []string{"RAMENE \"DESSIN"}, descFr: "Relit le fichier mot et DEFINIT son contenu directement DANS L'ESPACE DE TRAVAIL : ses procédures et variables redeviennent aussitôt utilisables (chaque procédure affiche \"VOUS VENEZ DE DEFINIR\"). Rien d'autre à faire. --- Différence avec CHARGE : RAMENE définit tout de suite (pour SE SERVIR d'un programme), tandis que CHARGE place seulement le fichier dans l'éditeur sans l'exécuter (pour le RETRAVAILLER).", descEn: "Re-reads file word and DEFINES its contents directly IN THE WORKSPACE: its procedures and variables become usable at once (each procedure prints \"X DEFINED\"). Nothing else to do. --- Difference with EDLOAD: LOAD defines right away (to USE a program), whereas EDLOAD only puts the file in the editor without running it (to REWORK it)."},
	"CHARGE":    {en: "EDLOAD", params: "mot", kind: "C", exemples: []string{"CHARGE \"DESSIN"}, descFr: "Charge le fichier mot DANS L'ÉDITEUR, SANS l'interpréter : son texte devient le contenu de l'éditeur (ED sans argument le rouvre). Il faut ouvrir ED puis valider par Ctrl+S pour qu'il soit exécuté. --- Différence avec RAMENE : CHARGE sert à RELIRE ou MODIFIER un programme avant de le valider, tandis que RAMENE définit directement son contenu dans l'espace de travail (utilisable tout de suite).", descEn: "Loads file word INTO THE EDITOR, WITHOUT running it: its text becomes the editor content (ED with no argument reopens it). You must open ED then validate with Ctrl+S to run it. --- Difference with LOAD: EDLOAD is to READ or EDIT a program before validating it, whereas LOAD defines its contents directly in the workspace (usable right away)."},
	"SAUVED":    {en: "EDSAVE", params: "mot", kind: "C", exemples: []string{"SAUVED \"DESSIN"}, descFr: "Sauve le contenu courant de l'éditeur dans le fichier mot.", descEn: "Saves the current editor content to file word."},
	"CATALOGUE": {en: "CATALOG", enAliases: []string{"CAT"}, params: "", kind: "C", exemples: []string{"CATALOGUE"}, descFr: "Liste les fichiers du dossier de travail (nom et taille).", descEn: "Lists the files in the working directory (name and size)."},
	"DETRUIS":   {en: "ERASEFILE", params: "mot", kind: "C", exemples: []string{"DETRUIS \"DESSIN"}, descFr: "Supprime définitivement le fichier mot.", descEn: "Permanently deletes file word."},

	// exemples livres avec l'install (dossier examples, lecture seule)
	"CATALOGUEEX": {en: "CATALOGEX", enAliases: []string{"CATEX"}, params: "", kind: "C", exemples: []string{"CATALOGUEEX"}, descFr: "Liste les fichiers d'exemples (nom et taille), comme CATALOGUE mais dans le dossier des exemples fourni avec GoLogo. Erreur s'il n'y a pas de dossier d'exemples.", descEn: "Lists the example files (name and size), like CATALOG but in the examples directory shipped with GoLogo. Error if there is no examples directory."},
	"RAMENEEX":    {en: "LOADEX", params: "mot", kind: "C", exemples: []string{"RAMENEEX \"DESSIN"}, descFr: "Comme RAMENE, mais va chercher le fichier dans le dossier des exemples : relit le fichier et DÉFINIT son contenu dans l'espace de travail (utilisable tout de suite). Erreur si le dossier d'exemples ou le fichier est absent ou illisible.", descEn: "Like LOAD, but reads the file from the examples directory: re-reads it and DEFINES its contents in the workspace (usable at once). Error if the examples directory or the file is missing or unreadable."},
	"CHARGEEX":    {en: "EDLOADEX", params: "mot", kind: "C", exemples: []string{"CHARGEEX \"DESSIN"}, descFr: "Comme CHARGE, mais va chercher le fichier dans le dossier des exemples : place son texte dans l'éditeur SANS l'interpréter (ED le rouvre, à valider par Ctrl+S). Erreur si le dossier d'exemples ou le fichier est absent ou illisible.", descEn: "Like EDLOAD, but reads the file from the examples directory: puts its text in the editor WITHOUT running it (EDIT reopens it, validate with Ctrl+S). Error if the examples directory or the file is missing or unreadable."},
	"FORMATE":     {en: "FORMAT", params: "n", kind: "C", exemples: []string{"FORMATE 0"}, descFr: "Formatait une disquette (matériel d'époque). Sans effet ici.", descEn: "Formatted a floppy disk (period hardware). No effect here."},
	"LECTEUR":     {en: "DRIVE", params: "", kind: "O", exemples: []string{"ECRIS LECTEUR"}, descFr: "Retournait le lecteur de disquette courant. Sans objet ici (rend 1).", descEn: "Returned the current disk drive. Not applicable here (outputs 1)."},
	"FLECTEUR":    {en: "SETDRIVE", params: "n", kind: "C", exemples: []string{"FLECTEUR 0"}, descFr: "Choisissait le lecteur de disquette. Sans effet ici.", descEn: "Selected the disk drive. No effect here."},

	// materiel d'origine (compat : sans effet)
	"REGLE":  {en: "REGLE", params: "", kind: "C", exemples: []string{"REGLE"}, descFr: "Crayon optique d'origine. Compatibilité : sans effet.", descEn: "Original light pen. Compatibility: no effect."},
	"ENTREE": {en: "ENTREE", params: "n", kind: "C", exemples: []string{"ENTREE 1"}, descFr: "Choisissait le canal d'entrée. Compatibilité : sans effet.", descEn: "Selected the input channel. Compatibility: no effect."},
	"SORTIE": {en: "SORTIE", params: "n", kind: "C", exemples: []string{"SORTIE 2"}, descFr: "Choisissait le canal de sortie (console, imprimante...). Compatibilité : sans effet.", descEn: "Selected the output channel (console, printer...). Compatibility: no effect."},
	"FLI":    {en: "FLI", params: "n", kind: "C", exemples: []string{"FLI 1"}, descFr: "Fond de ligne d'origine. Compatibilité : sans effet.", descEn: "Original line fill. Compatibility: no effect."},
	"COPIE":  {en: "COPIE", enAliases: []string{"COPY"}, params: "", kind: "C", exemples: []string{"COPIE"}, descFr: "« Imprime » l'écran : enregistre une copie d'écran PNG numérotée (PAGE_1.PNG, PAGE_2.PNG...) dans le sous-dossier PRINTER du dossier de travail.", descEn: "\"Prints\" the screen: saves a numbered PNG screenshot (PAGE_1.PNG, PAGE_2.PNG...) in the PRINTER subfolder of the work directory."},
	".SER":   {en: ".SER", params: "n1 n2", kind: "C", exemples: []string{".SER 1 2"}, descFr: "Port série (inexistant sur MO5). Compatibilité : sans effet.", descEn: "Serial port (absent on the original). Compatibility: no effect."},
	".CHB":   {en: ".CHB", params: "mot n", kind: "C", exemples: []string{".CHB \"PROG 0"}, descFr: "Chargement binaire en mémoire. Compatibilité : sans effet.", descEn: "Binary load into memory. Compatibility: no effect."},
	".RES":   {en: ".RES", params: "a", kind: "C", exemples: []string{".RES 0"}, descFr: "Réservait de la mémoire. Compatibilité : sans effet.", descEn: "Reserved memory. Compatibility: no effect."},
	".DEP":   {en: ".DEP", params: "a v", kind: "C", exemples: []string{".DEP 100 0"}, descFr: "Déposait un octet en mémoire. Compatibilité : sans effet.", descEn: "Stored a byte in memory. Compatibility: no effect."},
	".ROUT":  {en: ".ROUT", params: "a", kind: "C", exemples: []string{".ROUT 100"}, descFr: "Exécutait une routine machine. Compatibilité : sans effet.", descEn: "Ran a machine-code routine. Compatibility: no effect."},
	".EXA":   {en: ".EXA", params: "a", kind: "O", exemples: []string{"ECRIS .EXA 0"}, descFr: "Lisait un octet en mémoire. Compatibilité : rend toujours 0.", descEn: "Read a byte in memory. Compatibility: always outputs 0."},

	// divers
	"QUITTE":   {en: "BYE", frAliases: []string{"QUITTER"}, enAliases: []string{"GOODBYE", "QUIT"}, params: "", kind: "C", exemples: []string{"QUITTE"}, descFr: "Quitte GoLogo.", descEn: "Quits GoLogo."},
	"AIDE":     {en: "HELP", params: "[ commande ]", kind: "C", exemples: []string{"AIDE REPETE"}, descFr: "Sans argument, liste toutes les commandes. Avec un nom (ex. AIDE AVANCE), affiche sa description, ses paramètres et ses alias.", descEn: "Without argument, lists all commands. With a name (e.g. HELP FORWARD), shows its description, parameters and aliases."},
	"FRANCAIS": {en: "FRENCH", frAliases: []string{"FR"}, params: "", kind: "C", exemples: []string{"FRANCAIS"}, descFr: "Bascule l'aide et les messages en français. Alias : FR.", descEn: "Switches help and messages to French. Alias: FR."},
	"ANGLAIS":  {en: "ENGLISH", enAliases: []string{"EN"}, params: "", kind: "C", exemples: []string{"ANGLAIS"}, descFr: "Bascule l'aide et les messages en anglais. Alias anglais : ENGLISH, EN.", descEn: "Switches help and messages to English. English aliases: ENGLISH, EN."},
}

// libelle long d'un type de primitive, selon la langue
func kindLabel(lang, k string) string {
	en := map[string]string{"C": "command", "O": "operation", "P": "predicate"}
	fr := map[string]string{"C": "commande", "O": "opération", "P": "prédicat"}
	if lang == "EN" {
		return en[k]
	}
	return fr[k]
}

// nom canonique d'une entree dans la langue donnee
func canonical(lang, frKey string, e helpEntry) string {
	if lang == "EN" {
		return e.en
	}
	return frKey
}

// nom affiche dans le navigateur d'aide. en vue debutant FR on prefere le nom long
// et parlant (CACHETORTUE plutot que CT, RACINE plutot que RC) : le plus long parmi
// la cle, ses alias FR et les noms longs XLogo qui visent cette commande. sinon
// (vue complete ou EN) on garde le nom canonique
func aideDisplay(lang, frKey string, e helpEntry, extended bool) string {
	if extended || lang == "EN" {
		return canonical(lang, frKey, e)
	}
	if n, ok := aideNomDebutant[frKey]; ok {
		return n
	}
	best := frKey
	for _, a := range e.frAliases {
		if len(a) > len(best) {
			best = a
		}
	}
	for long, target := range xlogoAliases {
		if target == frKey && len(long) > len(best) {
			best = long
		}
	}
	return best
}

// liste triee des noms affiches dans la langue donnee. extended=false : seulement
// les commandes d'origine (SOLI fonctionnel), avec leur nom long pour le debutant ;
// extended=true : toutes les primitives (origine + extensions + compat), nom canonique
func aideNames(lang string, extended bool) []string {
	names := make([]string, 0, len(helpData))
	for fr, e := range helpData {
		if !extended && (categExtension[fr] || categCompat[fr]) && !categDebutant[fr] {
			continue
		}
		names = append(names, aideDisplay(lang, fr, e, extended))
	}
	sort.Strings(names)
	return names
}

// trouve la cle FR et l'entree d'un nom (FR ou EN, principal ou alias, alias XLogo
// inclus qui renvoient vers la fiche de leur cible)
func lookupHelp(name string) (string, helpEntry, bool) {
	name = strings.ToUpper(name)
	for fr, e := range helpData {
		if name == fr || name == e.en {
			return fr, e, true
		}
		for _, a := range e.frAliases {
			if a == name {
				return fr, e, true
			}
		}
		for _, a := range e.enAliases {
			if a == name {
				return fr, e, true
			}
		}
	}
	// alias XLogo (BAISSECRAYON, FIXECAP...) : on renvoie la fiche de la cible (BC, FCAP...)
	if target, ok := xlogoAliases[name]; ok {
		return lookupHelp(target)
	}
	// alias FMSLogo (SCREENCOLOR, SETSC, HALT...) : pareil
	if target, ok := fmslogoAliases[name]; ok {
		return lookupHelp(target)
	}
	return "", helpEntry{}, false
}

// traduit les mots des parametres pour l'anglais
func translateParams(lang, p string) string {
	if lang != "EN" {
		return p
	}
	r := strings.NewReplacer("liste", "list", "mot", "word", "commande", "command", " ou ", " or ")
	return r.Replace(p)
}

// les lignes d'aide d'une commande dans la langue donnee
func aideDetail(lang, name string) ([]string, bool) {
	frKey, e, ok := lookupHelp(name)
	if !ok {
		return nil, false
	}
	canon := frKey
	if lang == "EN" {
		canon = e.en
	}
	header := canon
	if e.params != "" {
		header += " " + translateParams(lang, e.params)
	}
	header += "   (" + kindLabel(lang, e.kind) + ")"
	lines := []string{header}
	lines = append(lines, categorieLine(lang, frKey)) // d'origine / extension / compat
	lines = append(lines, "")
	desc := e.descFr
	if lang == "EN" {
		desc = e.descEn
	}
	out := append(lines, wrapText(desc, 70)...)
	ex := e.exemples
	if lang == "EN" {
		ex = enExamples(e)
	}
	out = append(out, exampleLines(lang, ex)...)
	if e.palette {
		out = append(out, paletteLines(lang)...)
	}
	// section Alias (tous les autres noms, XLogo inclus), juste avant Voir aussi
	out = append(out, aliasLines(lang, frKey, e)...)
	return append(out, seeAlsoLines(lang, voirAussi[frKey])...), true
}

// tous les AUTRES noms de la commande (alias) dans la langue lang : alias FR, nom EN
// + alias EN, et les alias XLogo qui pointent dessus, en excluant le nom canonique
// deja affiche dans l'en-tete
func aliasNames(lang, frKey string, e helpEntry) []string {
	set := map[string]bool{frKey: true}
	for _, a := range e.frAliases {
		set[a] = true
	}
	if e.en != "" {
		set[e.en] = true
	}
	for _, a := range e.enAliases {
		set[a] = true
	}
	for x, target := range xlogoAliases {
		if set[strings.ToUpper(target)] {
			set[strings.ToUpper(x)] = true
		}
	}
	for x, target := range fmslogoAliases {
		if set[strings.ToUpper(target)] {
			set[strings.ToUpper(x)] = true
		}
	}
	canon := frKey
	if lang == "EN" {
		canon = e.en
	}
	var out []string
	for n := range set {
		if n != canon {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}

// la section "Alias" (les autres noms, en ligne, separes par des espaces), ou nil
// si la commande n'a pas d'alias
func aliasLines(lang, frKey string, e helpEntry) []string {
	al := aliasNames(lang, frKey, e)
	if len(al) == 0 {
		return nil
	}
	label := "Alias : "
	if lang == "EN" {
		label = "Aliases: "
	}
	return []string{"", label + strings.Join(al, " ")}
}

// la section "Voir aussi" / "See also" : les primitives liees, avec leur nom dans
// la langue courante
func seeAlsoLines(lang string, names []string) []string {
	var out []string
	for _, n := range names {
		if frKey, e, ok := lookupHelp(n); ok {
			out = append(out, canonical(lang, frKey, e))
		}
	}
	if len(out) == 0 {
		return nil
	}
	label := "Voir aussi : "
	if lang == "EN" {
		label = "See also: "
	}
	return []string{"", label + strings.Join(out, " ")}
}

// les exemples en anglais : l'override exemplesEn s'il existe (quand la traduction
// auto se plante, ex. les notes de JOUE), sinon la traduction auto de chaque exemple FR
func enExamples(e helpEntry) []string {
	if len(e.exemplesEn) > 0 {
		return e.exemplesEn
	}
	out := make([]string, len(e.exemples))
	for i, s := range e.exemples {
		out[i] = translateExample(s)
	}
	return out
}

// formate les lignes "Exemple(s)" (deja dans la bonne langue) : un seul sur la meme
// ligne, plusieurs en liste sous un titre
func exampleLines(lang string, ex []string) []string {
	if len(ex) == 0 {
		return nil
	}
	if len(ex) == 1 {
		label := "Exemple : "
		if lang == "EN" {
			label = "Example: "
		}
		return []string{"", label + ex[0]}
	}
	label := "Exemples :"
	if lang == "EN" {
		label = "Examples:"
	}
	out := []string{"", label}
	for _, e := range ex {
		out = append(out, "  "+e)
	}
	return out
}

// traduit les noms de commandes d'un exemple FR vers l'EN : complet -> complet
// (ECRIS->PRINT), court -> court (AV->FD). le reste (nombres, mots, crochets,
// variables) passe tel quel
func translateExample(ex string) string {
	toks := strings.Fields(ex)
	for i, t := range toks {
		up := strings.ToUpper(t)
		frKey, e, ok := lookupHelp(up)
		if !ok {
			continue
		}
		switch {
		case up == frKey:
			toks[i] = e.en
		case len(e.enAliases) > 0:
			toks[i] = e.enAliases[0]
		default:
			toks[i] = e.en
		}
	}
	return strings.Join(toks, " ")
}

// coupe un texte en lignes d'au plus width caracteres (sur les espaces)
func wrapText(s string, width int) []string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	cur := words[0]
	for _, w := range words[1:] {
		if len(cur)+1+len(w) > width {
			lines = append(lines, cur)
			cur = w
		} else {
			cur += " " + w
		}
	}
	return append(lines, cur)
}

// resout un mot (primitive ou alias, FR/EN) vers le nom de fiche d'aide tel
// qu'affiche dans la langue courante, ou ("", false) si aucune fiche. sert au F1
// contextuel de l'editeur (aide directe sur le mot sous le curseur)
func (i *Interp) HelpName(word string) (string, bool) {
	if frKey, entry, ok := lookupHelp(word); ok {
		return canonical(i.Lang(), frKey, entry), true
	}
	return "", false
}

// prepare les donnees du navigateur d'aide. extended=false : aide debutant
// (commandes d'origine SOLI) ; extended=true : aide complete (toutes les
// primitives). le mode survit a la bascule de langue (Ctrl+L)
func (i *Interp) HelpOpen(extended bool) (names []string, details map[string][]string, lang string, switchLang HelpSwitch) {
	build := func(lang string) ([]string, map[string][]string) {
		names := aideNames(lang, extended)
		details := make(map[string][]string, len(names))
		for _, n := range names {
			lines, _ := aideDetail(lang, n)
			details[n] = lines
		}
		return names, details
	}
	switchLang = func(lang, current string) ([]string, map[string][]string, string) {
		i.setLang(lang)
		names, details := build(lang)
		newCurrent := ""
		if frKey, entry, ok := lookupHelp(current); ok {
			newCurrent = aideDisplay(lang, frKey, entry, extended)
		}
		return names, details, newCurrent
	}
	lang = i.Lang()
	names, details = build(lang)
	return names, details, lang, switchLang
}

func formeAide(e *eval) (Value, error) {
	in := e.i
	if in.help == nil {
		return Value{}, fmt.Errorf("AIDE INDISPONIBLE")
	}
	names, details, lang, switchLang := in.HelpOpen(true) // AIDE ouvre l'aide complete
	start := ""
	if !e.atEnd() {
		if d := e.data[e.pos]; d.Kind == DSymbol || d.Kind == DWord {
			e.pos++
			frKey, entry, ok := lookupHelp(d.Text)
			if !ok {
				// erreur produite en FR, ErrorText la traduira si on est en EN
				return Value{}, fmt.Errorf("JE N'AI PAS D'AIDE POUR %s", strings.ToUpper(d.Text))
			}
			start = canonical(lang, frKey, entry)
		}
	}
	in.help(names, details, start, lang, switchLang)
	return None, nil
}
