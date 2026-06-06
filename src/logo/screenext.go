package logo

// fonctions d'ecran cherchees sur i.Out a l'appel. si i.Out ne les propose pas
// (mode sans ecran), les primitives concernees ne font rien
type screenControl interface {
	SetBorder(code int)     // FCB : couleur du bord
	SetTextColor(code int)  // FCT : couleur du texte
	SetTextBg(code int)     // FCFT : couleur de fond du texte
	ClearText()             // VT : vide la zone de texte
	SetCursor(col, lig int) // FCURS : place le curseur texte (0-39, 0-24)
	SetTextLines(n int)     // ME : nb de lignes texte visibles (1-25)

	DefineSprite(n int, rows []string) // DEFSPRITE : forme 16x16 ('#' = plein)
}

func (i *Interp) screen() (screenControl, bool) {
	sc, ok := i.Out.(screenControl)
	return sc, ok
}

// double tampon optionnel (primitive SCENE) cherche sur i.Out. BeginFrame gele
// l'affichage, EndFrame le bascule d'un coup. absent en mode texte
type frameBuffer interface {
	BeginFrame()
	EndFrame()
}

func (i *Interp) frameBuffer() (frameBuffer, bool) {
	fb, ok := i.Out.(frameBuffer)
	return fb, ok
}

// champ etendu (echelles) et couleurs/bord d'ecran (sections 1 et 9)
func (i *Interp) registerScreenExt() {
	// FCB n : couleur du bord (code 0-15 ou [ r v b ])
	i.register(cmd(1, func(in *Interp, a []Value) error {
		c, err := penColor(a[0])
		if err != nil {
			return err
		}
		if sc, ok := in.screen(); ok {
			sc.SetBorder(int(c))
		}
		return nil
	}), "FCB")

	// FCT n : couleur du texte (code 0-15 ou [ r v b ])
	i.register(cmd(1, func(in *Interp, a []Value) error {
		c, err := penColor(a[0])
		if err != nil {
			return err
		}
		if sc, ok := in.screen(); ok {
			sc.SetTextColor(int(c))
		}
		return nil
	}), "FCT")

	// FCFT n : couleur de fond du texte (code 0-15 ou [ r v b ])
	i.register(cmd(1, func(in *Interp, a []Value) error {
		c, err := penColor(a[0])
		if err != nil {
			return err
		}
		if sc, ok := in.screen(); ok {
			sc.SetTextBg(int(c))
		}
		return nil
	}), "FCFT")

	// VT : vide la zone de texte
	i.register(cmd(0, func(in *Interp, a []Value) error {
		if sc, ok := in.screen(); ok {
			sc.ClearText()
		}
		return nil
	}), "VT")

	// FCURS [col lig] : place le curseur texte (col 0-39, lig 0-24)
	i.register(cmd(1, func(in *Interp, a []Value) error {
		col, lig, err := toCoords(a[0])
		if err != nil {
			return err
		}
		if sc, ok := in.screen(); ok {
			sc.SetCursor(int(round1(col)), int(round1(lig)))
		}
		return nil
	}), "FCURS")

	// ME n : nb de lignes texte visibles (1-25, 25 = plein texte)
	i.register(cmd(1, func(in *Interp, a []Value) error {
		n, err := toNumber(a[0])
		if err != nil {
			return err
		}
		if sc, ok := in.screen(); ok {
			sc.SetTextLines(int(round1(n)))
		}
		return nil
	}), "ME")

	// ECH : rend les echelles [h v] en %
	i.register(&primitive{arity: 0, reporter: true, fn: func(in *Interp, a []Value) (Value, error) {
		h, v := in.Turtle.Scale()
		return ListValue([]Datum{
			{Kind: DNumber, Num: h, Text: formatNumber(h)},
			{Kind: DNumber, Num: v, Text: formatNumber(v)},
		}), nil
	}}, "ECH")

	// FECH liste : fixe les echelles (2 nombres dans [0,200])
	i.register(cmd(1, func(in *Interp, a []Value) error {
		h, v, err := toCoords(a[0])
		if err != nil {
			return err
		}
		in.Turtle.SetScale(clampScale(h), clampScale(v))
		return nil
	}), "FECH")
}

// borne une echelle dans [0,200]
func clampScale(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 200 {
		return 200
	}
	return x
}
