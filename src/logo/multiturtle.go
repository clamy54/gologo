package logo

// multi-tortue (compat MSWLogo/FMSLogo et XLogo) : plusieurs tortues sur le meme
// ecran, une seule active a la fois qui recoit les commandes normales (AV, TD...).
// FIXETORTUE choisit l'active, DEMANDE execute un bloc sur d'autres puis revient.
// fond, echelle et mode de champ restent globaux. numerotation des 0, et la tortue
// 0 existe toujours, donc un programme mono-tortue (MO5) ne voit aucune difference

// primitives multi-tortue (noms FR ; les noms anglais arrivent via registerEnglishAliases)
func (i *Interp) registerMultiTurtle() {
	// FIXETORTUE n : choisit (et cree au besoin) la tortue active
	i.register(cmd(1, func(in *Interp, a []Value) error {
		n, err := turtleIndex(a[0])
		if err != nil {
			return err
		}
		return in.Turtle.SetTurtle(n)
	}), "FIXETORTUE")

	// TORTUE : numero de la tortue active
	i.register(&primitive{arity: 0, reporter: true, fn: func(in *Interp, a []Value) (Value, error) {
		return NumberValue(float64(in.Turtle.CurrentTurtle())), nil
	}}, "TORTUE")

	// NBTORTUES : combien de tortues existent
	i.register(&primitive{arity: 0, reporter: true, fn: func(in *Interp, a []Value) (Value, error) {
		return NumberValue(float64(in.Turtle.TurtleCount())), nil
	}}, "NBTORTUES")

	// DEMANDE n [ ... ] : joue le bloc sur la tortue n (ou chaque numero d'une liste)
	// puis revient a l'active precedente
	i.register(&primitive{arity: 2, fn: primDemande}, "DEMANDE")

	// DISTORTUE : zigouille toutes les tortues sauf la 0, et selectionne la 0
	i.register(cmd(0, func(in *Interp, a []Value) error {
		in.Turtle.KillExtraTurtles()
		return nil
	}), "DISTORTUE")

	// COLLISION? [ a b ] : VRAI si a et b existent, sont visibles et se chevauchent
	// (boites englobantes), FAUX sinon
	i.register(&primitive{arity: 1, reporter: true, fn: func(in *Interp, a []Value) (Value, error) {
		if a[0].Kind != KList || len(a[0].List) != 2 {
			return Value{}, &badData{a[0].String()}
		}
		na, oka := datumInt(a[0].List[0])
		nb, okb := datumInt(a[0].List[1])
		if !oka || !okb {
			return Value{}, &badData{a[0].String()}
		}
		return BoolValue(in.Turtle.Collide(na, nb)), nil
	}}, "COLLISION?")
}

// convertit un argument en numero de tortue. au contraire d'un simple int(n) il
// refuse les non-entiers (FIXETORTUE 1.9 est une erreur, pas un round vers 1) et
// les valeurs qui ne tiennent pas dans un entier
func turtleIndex(v Value) (int, error) { return intArg(v) }

// entier exact porte par un element de liste (numero de tortue)
func datumInt(d Datum) (int, bool) {
	v, err := datumToValue(d)
	if err != nil {
		return 0, false
	}
	n, err := intArg(v)
	return n, err == nil
}

// DEMANDE / ASK : n est un numero de tortue ou une liste de numeros. on joue le
// bloc pour chacune puis on restaure l'active d'origine, meme si ca a plante
func primDemande(in *Interp, a []Value) (Value, error) {
	if a[1].Kind != KList {
		return Value{}, &badData{a[1].String()} // -> "DEMANDE N'AIME PAS ..."
	}
	body := a[1].List

	var nums []int
	switch a[0].Kind {
	case KList:
		for _, d := range a[0].List {
			n, ok := datumInt(d)
			if !ok {
				return Value{}, &badData{a[0].String()}
			}
			nums = append(nums, n)
		}
	default:
		n, err := turtleIndex(a[0])
		if err != nil {
			return Value{}, err
		}
		nums = append(nums, n)
	}

	save := in.Turtle.CurrentTurtle()
	defer func() { // on revient a la tortue de depart, si elle existe encore : apres
		// un VE ou un DISTORTUE dans le bloc, la reselectionner la recreerait
		if save < in.Turtle.TurtleCount() {
			in.Turtle.SetTurtle(save)
		}
	}()
	for _, n := range nums {
		if err := in.Turtle.SetTurtle(n); err != nil {
			return Value{}, err
		}
		if err := in.runSeq(body); err != nil {
			return Value{}, err
		}
	}
	return Value{}, nil
}
