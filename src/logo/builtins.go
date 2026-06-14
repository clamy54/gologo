package logo

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	"beroot.com/logo/turtle"
)

// primitives de base : tortue, REPETE, DONNE
func (i *Interp) registerBuiltins() {
	// deplacement
	i.register(cmd(1, func(in *Interp, a []Value) error {
		n, err := toNumber(a[0])
		if err != nil {
			return err
		}
		return in.Turtle.Forward(round1(n))
	}), "AV", "AVANCE")

	i.register(cmd(1, func(in *Interp, a []Value) error {
		n, err := toNumber(a[0])
		if err != nil {
			return err
		}
		return in.Turtle.Back(round1(n))
	}), "RE", "RECULE")

	i.register(cmd(1, func(in *Interp, a []Value) error {
		n, err := toNumber(a[0])
		if err != nil {
			return err
		}
		in.Turtle.Right(round1(n))
		return nil
	}), "TD", "TOURNEDROITE")

	i.register(cmd(1, func(in *Interp, a []Value) error {
		n, err := toNumber(a[0])
		if err != nil {
			return err
		}
		in.Turtle.Left(round1(n))
		return nil
	}), "TG", "TOURNEGAUCHE")

	i.register(cmd(1, func(in *Interp, a []Value) error {
		n, err := toNumber(a[0])
		if err != nil {
			return err
		}
		in.Turtle.SetHeading(round1(n))
		return nil
	}), "FCAP")

	// FPOS [x1 y1 x2 y2 ...] : deplace la tortue le long d'une suite de points
	// (cap inchange, trace si le crayon est baisse)
	i.register(cmd(1, func(in *Interp, a []Value) error {
		pts, err := toCoordsList(a[0])
		if err != nil {
			return err
		}
		for _, p := range pts {
			if err := in.Turtle.SetPos(round1(p[0]), round1(p[1])); err != nil {
				return err
			}
		}
		return nil
	}), "FPOS")

	// FXY x y : amene la tortue au point (x, y), trace si crayon baisse
	i.register(cmd(2, func(in *Interp, a []Value) error {
		x, err := toNumber(a[0])
		if err != nil {
			return err
		}
		y, err := toNumber(a[1])
		if err != nil {
			return err
		}
		return in.Turtle.SetPos(round1(x), round1(y))
	}), "FXY")

	// CERCLE r : cercle de rayon r centre sur la tortue (facon XLogo). la tortue
	// reste sur place, position et cap inchanges
	i.register(cmd(1, func(in *Interp, a []Value) error {
		r, err := toNumber(a[0])
		if err != nil {
			return err
		}
		return in.drawArc(r, 0, 360)
	}), "CERCLE")

	// ARC r cap1 cap2 : arc de rayon r centre sur la tortue (facon XLogo), entre les
	// caps boussole cap1 et cap2. la tortue ne bouge pas
	i.register(cmd(3, func(in *Interp, a []Value) error {
		r, err := toNumber(a[0])
		if err != nil {
			return err
		}
		cap1, err := toNumber(a[1])
		if err != nil {
			return err
		}
		cap2, err := toNumber(a[2])
		if err != nil {
			return err
		}
		return in.drawArc(r, cap1, cap2)
	}), "ARC")

	// POINT [x1 y1 x2 y2 ...] : allume une suite de points (couleur du crayon),
	// sans bouger la tortue
	i.register(cmd(1, func(in *Interp, a []Value) error {
		pts, err := toCoordsList(a[0])
		if err != nil {
			return err
		}
		for _, p := range pts {
			x, y := round1(p[0]), round1(p[1])
			if !in.Turtle.CanReach(x, y) {
				return turtle.ErrSortir // sentinelle errors.Is, meme message
			}
			in.Turtle.Point(x, y)
		}
		return nil
	}), "POINT")

	// crayon
	i.register(cmd(0, func(in *Interp, a []Value) error { in.Turtle.PenUp(); return nil }), "LC")
	i.register(cmd(0, func(in *Interp, a []Value) error { in.Turtle.PenDownState(); return nil }), "BC")

	i.register(cmd(1, func(in *Interp, a []Value) error {
		c, err := penColor(a[0])
		if err != nil {
			return err
		}
		in.Turtle.SetPenColor(c)
		return nil
	}), "FCC")

	// FIXETAILLECRAYON n : epaisseur du trait en pixels (mini 1)
	i.register(cmd(1, func(in *Interp, a []Value) error {
		n, err := toNumber(a[0])
		if err != nil {
			return err
		}
		in.Turtle.SetPenSize(int(round1(n)))
		return nil
	}), "FIXETAILLECRAYON")

	// REMPLIS : remplit la zone ou se trouve la tortue. couleur = celle fixee par
	// FCR si elle l'a ete, sinon la couleur du crayon (compat MO5)
	i.register(cmd(0, func(in *Interp, a []Value) error {
		if f, ok := in.Out.(Filler); ok {
			st := in.Turtle.State()
			c := st.Pen
			if fc, set := in.Turtle.FloodColor(); set {
				c = fc // FMSLogo : FILL part de la couleur de remplissage
			}
			f.Fill(st.X, st.Y, c)
		}
		return nil
	}), "REMPLIS")

	// FCR n ou liste : couleur de remplissage de REMPLIS (code 0-15 ou [ r v b ])
	i.register(cmd(1, func(in *Interp, a []Value) error {
		c, err := penColor(a[0])
		if err != nil {
			return err
		}
		in.Turtle.SetFloodColor(c)
		return nil
	}), "FCR")

	// ETIQUETTE obj : ecrit obj dans le champ graphique a la position de la tortue
	i.register(cmd(1, func(in *Interp, a []Value) error {
		if l, ok := in.Out.(Labeler); ok {
			st := in.Turtle.State()
			l.Label(st.X, st.Y, a[0].String(), st.Pen)
		}
		return nil
	}), "ETIQUETTE")

	i.register(cmd(1, func(in *Interp, a []Value) error {
		c, err := penColor(a[0]) // code 0-15 ou [ r v b ]
		if err != nil {
			return err
		}
		in.Turtle.SetBackground(c)
		return nil
	}), "FCFG")

	// tortue / ecran
	i.register(cmd(0, func(in *Interp, a []Value) error { in.Turtle.Show(); return nil }), "MT")
	i.register(cmd(0, func(in *Interp, a []Value) error { in.Turtle.Hide(); return nil }), "CT")
	i.register(cmd(0, func(in *Interp, a []Value) error { in.Turtle.Reset(); return nil }), "VE", "INIT")
	i.register(cmd(0, func(in *Interp, a []Value) error { in.Turtle.Wipe(); return nil }), "NETTOIE")
	i.register(cmd(0, func(in *Interp, a []Value) error { in.Turtle.Home(); return nil }), "ORIGINE")
	i.register(cmd(0, func(in *Interp, a []Value) error { in.Turtle.SetField(turtle.Clos); return nil }), "CLOS")
	i.register(cmd(0, func(in *Interp, a []Value) error { in.Turtle.SetField(turtle.Enroule); return nil }), "ENR")
	i.register(cmd(0, func(in *Interp, a []Value) error { in.Turtle.SetField(turtle.Fenetre); return nil }), "FEN")

	// controle
	i.register(&primitive{name: "REPETE", arity: 2, fn: primRepete}, "REPETE")

	// variables
	i.register(cmd(2, func(in *Interp, a []Value) error {
		name, err := toWord(a[0])
		if err != nil {
			return err
		}
		in.setVar(name, a[1]) // ecrit la locale si elle existe, sinon la globale
		return nil
	}), "DONNE", "FIXE")

	// LOCAL "nom (ou une liste de noms, ou (LOCAL "a "b)) : declare des variables
	// locales a la procedure en cours
	i.register(vcmd(1, func(in *Interp, a []Value) error {
		for _, v := range a {
			if v.Kind == KList {
				for _, d := range v.List {
					dv, err := datumToValue(d)
					if err != nil {
						return err
					}
					w, err := toWord(dv)
					if err != nil {
						return err
					}
					in.declareLocal(w)
				}
				continue
			}
			w, err := toWord(v)
			if err != nil {
				return err
			}
			in.declareLocal(w)
		}
		return nil
	}), "LOCAL")

	// sortie texte
	// forme variable entre parentheses : (ECRIS a b c) imprime les objets separes
	// par une espace, comme PRINT/TYPE en Logo standard
	// ECRIS/TAPE/MONTRE ecrivent sur le flux d'ecriture courant (FIXEECRITURE) s'il
	// y en a un, sinon sur la console (cf in.writeText)
	i.register(vcmd(1, func(in *Interp, a []Value) error {
		return in.writeText(joinValues(a) + "\n")
	}), "ECRIS", "EC")
	i.register(vcmd(1, func(in *Interp, a []Value) error {
		return in.writeText(joinValues(a))
	}), "TAPE")

	// MONTRE : comme ECRIS mais garde les crochets autour des listes (SHOW standard)
	i.register(vcmd(1, func(in *Interp, a []Value) error {
		parts := make([]string, len(a))
		for i, v := range a {
			parts[i] = showValue(v)
		}
		return in.writeText(strings.Join(parts, " ") + "\n")
	}), "MONTRE")
}

// colle les objets en une chaine, separes par une espace (ECRIS/TAPE variadiques)
func joinValues(a []Value) string {
	parts := make([]string, len(a))
	for i, v := range a {
		parts[i] = v.String()
	}
	return strings.Join(parts, " ")
}

// objet avec les crochets autour d'une liste (pour MONTRE)
func showValue(v Value) string {
	if v.Kind == KList {
		return "[" + v.String() + "]"
	}
	return v.String()
}

// construit une primitive "commande" (ne rend pas de valeur)
func cmd(arity int, fn func(*Interp, []Value) error) *primitive {
	return &primitive{
		arity: arity,
		fn: func(in *Interp, a []Value) (Value, error) {
			return Value{}, fn(in, a)
		},
	}
}

// commande a nombre d'arguments variable entre parentheses (ex. ECRIS)
func vcmd(arity int, fn func(*Interp, []Value) error) *primitive {
	p := cmd(arity, fn)
	p.variadic = true
	return p
}

// trace un arc de rayon |r| centre sur la tortue (facon XLogo), entre les caps
// boussole cap1 et cap2 (cap 0 = Nord, sens horaire). le point a la boussole cap
// est en center + r*(sin cap, cos cap). on rejoint le depart crayon leve, on trace
// par segments de ~1 degre, puis on remet la tortue exactement comme avant : elle
// ne bouge pas. CERCLE = drawArc(r, 0, 360). interruptible (Ctrl+C)
func (i *Interp) drawArc(r, cap1, cap2 float64) error {
	t := i.Turtle
	st := t.State()
	rr := math.Abs(r)
	pen := st.PenDown
	pointAt := func(bearing float64) (float64, float64) {
		rad := bearing * math.Pi / 180
		return st.X + rr*math.Sin(rad), st.Y + rr*math.Cos(rad)
	}

	defer func() { // on remet tout en place : pour l'observateur, la tortue n'a pas bouge
		t.PenUp()
		_ = t.SetPos(st.X, st.Y) // crayon leve : aucun trace
		t.SetHeading(st.Heading)
		if pen {
			t.PenDownState()
		}
	}()

	delta := cap2 - cap1
	n := int(math.Abs(delta) + 0.5)
	if n == 0 {
		return nil // arc nul
	}
	step := delta / float64(n) // pas signe (~1 degre), dans le sens cap1 -> cap2

	t.PenUp()
	x0, y0 := pointAt(cap1)
	if err := t.SetPos(x0, y0); err != nil { // rejoint le depart sans tracer
		return err
	}
	if pen {
		t.PenDownState()
	}
	for k := 1; k <= n; k++ {
		if i.brk.Load() {
			return ErrInterrompu
		}
		x, y := pointAt(cap1 + step*float64(k))
		if err := t.SetPos(x, y); err != nil {
			return err
		}
	}
	return nil
}

// REPETE n [ ... ]
func primRepete(in *Interp, a []Value) (Value, error) {
	// entier exact (comme ITEM/HASARD) : refuse 3.6, l'infini et les compteurs
	// gigantesques (qui partiraient en boucle quasi sans fin) plutot que de tronquer
	n, err := intArg(a[0])
	if err != nil {
		return Value{}, err
	}
	if a[1].Kind != KList {
		return Value{}, &badData{a[1].String()} // -> "REPETE N'AIME PAS ..."
	}
	// compteur d'iteration empile pour COMPTEUR/REPCOUNT (depile meme si ca plante)
	in.repStack = append(in.repStack, 0)
	defer func() { in.repStack = in.repStack[:len(in.repStack)-1] }()
	for k := 0; k < n; k++ {
		if in.brk.Load() { // interruptible meme avec un corps vide (REPETE 1e18 [])
			return Value{}, ErrInterrompu
		}
		in.repStack[len(in.repStack)-1] = k + 1
		if err := in.runSeq(a[1].List); err != nil {
			return Value{}, err
		}
	}
	return Value{}, nil
}

// conversions

func toNumber(v Value) (float64, error) {
	switch v.Kind {
	case KNumber:
		return v.Num, nil
	case KInt:
		// approche float64 de l'entier exact (perd de la precision au-dela de
		// 2^53, mais c'est le contrat de toNumber : trigo, indices, couleurs...)
		f, _ := new(big.Float).SetInt(v.Int).Float64()
		return f, nil
	case KWord:
		if n, err := strconv.ParseFloat(strings.Replace(v.Word, ",", ".", 1), 64); err == nil {
			return n, nil
		}
	}
	return 0, &badData{v.String()}
}

func toWord(v Value) (string, error) {
	switch v.Kind {
	case KWord:
		return v.Word, nil
	case KNumber:
		return formatNumber(v.Num), nil
	case KInt:
		return v.Int.String(), nil
	}
	return "", &badData{v.String()}
}

func toColor(v Value) (turtle.Color, error) {
	n, err := toNumber(v)
	if err != nil {
		return 0, err
	}
	return turtle.Color(int(n)), nil
}

// lit une couleur facon FCC : code palette (nombre) ou [ r v b ] (3 valeurs 0-255)
func penColor(v Value) (turtle.Color, error) {
	if v.Kind != KList {
		return toColor(v)
	}
	if len(v.List) != 3 {
		return 0, &badData{v.String()}
	}
	var rgb [3]int
	for i, d := range v.List {
		dv, err := datumToValue(d)
		if err != nil {
			return 0, err
		}
		n, err := toNumber(dv)
		if err != nil {
			return 0, err
		}
		if n < 0 || n > 255 {
			return 0, &badData{v.String()}
		}
		rgb[i] = int(n)
	}
	return turtle.RGB(rgb[0], rgb[1], rgb[2]), nil
}

// toutes les paires de coordonnees d'une liste [x1 y1 x2 y2 ...] (nombre pair
// d'elements, au moins une paire). pour FPOS/POINT
func toCoordsList(v Value) ([][2]float64, error) {
	if v.Kind != KList || len(v.List) < 2 || len(v.List)%2 != 0 {
		// le nom de la primitive (FPOS ou POINT) est ajoute par invoke
		return nil, &badData{v.String()}
	}
	pts := make([][2]float64, 0, len(v.List)/2)
	for i := 0; i < len(v.List); i += 2 {
		xv, err := datumToValue(v.List[i])
		if err != nil {
			return nil, err
		}
		yv, err := datumToValue(v.List[i+1])
		if err != nil {
			return nil, err
		}
		x, err := toNumber(xv)
		if err != nil {
			return nil, err
		}
		y, err := toNumber(yv)
		if err != nil {
			return nil, err
		}
		pts = append(pts, [2]float64{x, y})
	}
	return pts, nil
}

// les deux premieres coordonnees d'une liste [x y] (pour plusieurs points, voir toCoordsList)
func toCoords(v Value) (x, y float64, err error) {
	if v.Kind != KList || len(v.List) < 2 {
		// le nom de la primitive est ajoute par invoke
		return 0, 0, &badData{v.String()}
	}
	xv, err := datumToValue(v.List[0])
	if err != nil {
		return 0, 0, err
	}
	yv, err := datumToValue(v.List[1])
	if err != nil {
		return 0, 0, err
	}
	if x, err = toNumber(xv); err != nil {
		return 0, 0, err
	}
	if y, err = toNumber(yv); err != nil {
		return 0, 0, err
	}
	return x, y, nil
}

// applique un operateur infixe a deux valeurs
func applyOp(op string, l, r Value) (Value, error) {
	// l'egalite accepte mots et nombres
	if op == "=" || op == "<>" {
		eq := valuesEqual(l, r)
		if op == "<>" {
			eq = !eq
		}
		return BoolValue(eq), nil
	}
	// chemin entier exact : + - * sur deux entiers restent exacts (au-dela de 2^53),
	// les comparaisons aussi. petits entiers -> int64 sans allocation ; tout le
	// reste (fractions) retombe sur le float64.
	switch op {
	case "+":
		if v, ok := addInt(l, r); ok {
			return v, nil
		}
	case "-":
		if v, ok := subInt(l, r); ok {
			return v, nil
		}
	case "*":
		if v, ok := mulInt(l, r); ok {
			return v, nil
		}
	case "<", ">", "<=", ">=":
		if x, ok := asSmallInt(l); ok {
			if y, ok2 := asSmallInt(r); ok2 {
				return BoolValue(cmpHolds(op, cmpInt64(x, y))), nil
			}
		}
		if c, ok := intCmp(l, r); ok {
			return BoolValue(cmpHolds(op, c)), nil
		}
	}
	ln, err := toNumber(l)
	if err != nil {
		return Value{}, err
	}
	rn, err := toNumber(r)
	if err != nil {
		return Value{}, err
	}
	switch op {
	case "+":
		return numResult(ln + rn), nil
	case "-":
		return numResult(ln - rn), nil
	case "*":
		return numResult(ln * rn), nil
	case "/":
		if rn == 0 {
			return Value{}, fmt.Errorf("DIVISION PAR ZERO")
		}
		return numResult(ln / rn), nil
	case "<":
		return BoolValue(ln < rn), nil
	case ">":
		return BoolValue(ln > rn), nil
	case "<=":
		return BoolValue(ln <= rn), nil
	case ">=":
		return BoolValue(ln >= rn), nil
	}
	return Value{}, fmt.Errorf("OPERATEUR INCONNU %s", op)
}

func valuesEqual(l, r Value) bool {
	// au moins un entier exact : compare les valeurs entieres si l'autre en est
	// un aussi (sinon on laisse filer vers la comparaison de chaines plus bas,
	// pour ne pas changer le sens de "007 = "7 entre deux mots)
	if l.Kind == KInt || r.Kind == KInt {
		if c, ok := intCmp(l, r); ok {
			return c == 0
		}
	}
	if l.Kind == KNumber && r.Kind == KNumber {
		return l.Num == r.Num
	}
	// tableaux : egalite par identite (le MEME tableau), pas par contenu - deux
	// tableaux distincts au contenu identique ne sont pas egaux (facon FMSLogo)
	if l.Kind == KArray || r.Kind == KArray {
		return l.Kind == KArray && r.Kind == KArray && l.Arr == r.Arr
	}
	return strings.EqualFold(l.String(), r.String())
}
