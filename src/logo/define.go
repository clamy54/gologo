package logo

import (
	"fmt"
	"strings"
)

// DEFINIS / TEXTE : manipuler des procedures comme des donnees (Logo adulte)
// DEFINIS (alias DEF, anglais DEFINE) a deux signatures, devinees sur la 1re liste :
//   imbriquee UCBLogo : DEFINIS "nom [ [params] [ligne1] [ligne2]... ]
//   plate XLogo       : DEFINIS "nom [params] [corps]   (le corps est l'arg suivant)
// ecart assume : on garde le corps en une seule "ligne logique" a plat, comme XLogo.
// du coup pour la forme multi-lignes UCBLogo, TEXTE recrache tout sur une ligne

// noms de parametres a partir d'une liste de Datum, ':' initial tolere, mis en MAJ
func datumParamNames(items []Datum) ([]string, error) {
	params := make([]string, 0, len(items))
	for _, d := range items {
		name := strings.TrimPrefix(d.Text, ":")
		if name == "" {
			return nil, &badData{"[" + d.String() + "]"}
		}
		params = append(params, strings.ToUpper(name))
	}
	return params, nil
}

// forme speciale DEFINIS : lit ses arguments elle-meme, l'arite depend de la
// signature (2 args en UCBLogo imbrique, 3 en XLogo)
func formeDefinis(e *eval) (Value, error) {
	args, err := e.readArgs("DEFINIS", 2) // nom + 1re liste
	if err != nil {
		return Value{}, err
	}
	name, err := toWord(args[0])
	if err != nil {
		return Value{}, err
	}
	nameU := strings.ToUpper(name)
	if args[1].Kind != KList {
		return Value{}, &badData{args[1].String()}
	}
	first := args[1].List

	var params []string
	var body []Datum
	if len(first) > 0 && first[0].Kind == DList {
		// forme UCBLogo imbriquee : [ [params] [ligne1] [ligne2]... ]
		if params, err = datumParamNames(first[0].List); err != nil {
			return Value{}, err
		}
		for _, line := range first[1:] {
			if !isDatumList(line) {
				return Value{}, &badData{args[1].String()}
			}
			body = append(body, line.List...)
		}
	} else {
		// forme XLogo : nom [params] [corps] -> corps = argument suivant
		if params, err = datumParamNames(first); err != nil {
			return Value{}, err
		}
		more, err := e.readArgs("DEFINIS", 1)
		if err != nil {
			return Value{}, err
		}
		if more[0].Kind != KList {
			return Value{}, &badData{more[0].String()}
		}
		body = append(body, more[0].List...)
	}

	if e.i.prims[nameU] != nil {
		return None, fmt.Errorf("%s EXISTE DEJA", nameU)
	}
	e.i.procs[nameU] = &userProc{name: nameU, params: params, body: body}
	return None, nil
}

func (i *Interp) registerDefine() {
	i.register(&primitive{name: "DEFINIS", special: formeDefinis}, "DEFINIS", "DEF")

	// TEXTE nom (TEXT) : rend la def de la procedure en [ [params] [corps] ]
	// (corps sur une ligne, cf note plus haut). erreur si nom n'est pas une procedure
	i.register(&primitive{name: "TEXTE", arity: 1, reporter: true, fn: func(in *Interp, a []Value) (Value, error) {
		name, err := toWord(a[0])
		if err != nil {
			return Value{}, err
		}
		p := in.procs[strings.ToUpper(name)]
		if p == nil {
			return Value{}, &badData{a[0].String()}
		}
		params := make([]Datum, len(p.params))
		for k, par := range p.params {
			params[k] = Datum{Kind: DWord, Text: par}
		}
		items := []Datum{
			{Kind: DList, List: params},
			{Kind: DList, List: append([]Datum(nil), p.body...)},
		}
		return ListValue(items), nil
	}}, "TEXTE")
}

// le Datum est-il une liste (ou un groupe '( )') ?
func isDatumList(d Datum) bool { return d.Kind == DList || d.Kind == DGroup }
