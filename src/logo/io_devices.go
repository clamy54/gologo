package logo

import (
	"fmt"
	"strings"
)

// ligne -> mot si un seul token, liste sinon (semantique de LIS, XLogo). vide => mot vide
func lineToWordOrList(line string) Value {
	fields := strings.Fields(line)
	switch len(fields) {
	case 0:
		return WordValue("")
	case 1:
		return WordValue(fields[0])
	default:
		ds := make([]Datum, len(fields))
		for i, f := range fields {
			ds[i] = Datum{Kind: DWord, Text: f}
		}
		return ListValue(ds)
	}
}

// primitives du "monde exterieur" cote backend : clavier, souris, manettes
// a part de registerOperations juste pour respirer. le son (JOUE...) vit dans music.go
func (i *Interp) registerIODevices() {
	op := func(arity int, fn func(*Interp, []Value) (Value, error), names ...string) {
		i.register(&primitive{arity: arity, reporter: true, fn: fn}, names...)
	}

	// clavier (manuel p.72-73). si un flux de lecture courant est en place
	// (FIXELECTURE), ces recepteurs lisent dedans au lieu du clavier ; [ ] en
	// fin de fichier
	op(0, func(in *Interp, a []Value) (Value, error) {
		if of := in.curReadStream(); of != nil {
			r, eof, err := of.readRune()
			if err != nil {
				return Value{}, errLectureImpossible
			}
			if eof {
				return ListValue(nil), nil
			}
			return WordValue(string(r)), nil
		}
		if in.keyb == nil {
			// pas de clavier, pas de dialogue : la machine reste muette
			return Value{}, fmt.Errorf("CLAVIER INDISPONIBLE")
		}
		r, ok := in.keyb.ReadChar()
		if !ok {
			return Value{}, ErrInterrompu
		}
		return WordValue(string(r)), nil
	}, "LISCAR")
	op(0, func(in *Interp, a []Value) (Value, error) {
		if of := in.curReadStream(); of != nil {
			line, eof, err := of.readLine()
			if err != nil {
				return Value{}, errLectureImpossible
			}
			if eof {
				return ListValue(nil), nil
			}
			return lineToList(line), nil
		}
		if in.keyb == nil {
			return Value{}, fmt.Errorf("CLAVIER INDISPONIBLE")
		}
		line, ok := in.keyb.ReadLine()
		if !ok {
			return Value{}, ErrInterrompu
		}
		return lineToList(line), nil
	}, "LL")
	op(0, func(in *Interp, a []Value) (Value, error) {
		if of := in.curReadStream(); of != nil {
			line, eof, err := of.readLine()
			if err != nil {
				return Value{}, errLectureImpossible
			}
			if eof {
				return ListValue(nil), nil
			}
			return WordValue(line), nil
		}
		if in.keyb == nil {
			return Value{}, fmt.Errorf("CLAVIER INDISPONIBLE")
		}
		line, ok := in.keyb.ReadLine()
		if !ok {
			return Value{}, ErrInterrompu
		}
		return WordValue(line), nil // toute la ligne d'un bloc
	}, "LISMOT")
	op(0, func(in *Interp, a []Value) (Value, error) {
		return BoolValue(in.keyb != nil && in.keyb.KeyPending()), nil
	}, "TOUCHE?")

	// souris (alias POSOPT/CONTACT? = l'antique crayon optique, manuel p.77)
	op(0, func(in *Interp, a []Value) (Value, error) {
		if in.mouse == nil {
			return ListValue(nil), nil
		}
		x, y, inField := in.mouse.MousePos()
		if !inField { // hors champ -> liste vide
			return ListValue(nil), nil
		}
		return ListValue([]Datum{
			{Kind: DNumber, Num: round1(x)}, {Kind: DNumber, Num: round1(y)},
		}), nil
	}, "SOURISPOS", "POSOPT")
	op(0, func(in *Interp, a []Value) (Value, error) {
		return BoolValue(in.mouse != nil && in.mouse.MouseDown()), nil
	}, "SOURISPRESSEE", "CONTACT?", "CONTACT")

	// LIS liste mot (XLogo) : affiche le titre, lit une saisie, la range dans la variable
	// (mot si un seul token, liste sinon). XLogo ouvre une boite ; nous, texte inline
	i.register(cmd(2, func(in *Interp, a []Value) error {
		name, err := toWord(a[1])
		if err != nil {
			return err
		}
		if in.keyb == nil {
			return fmt.Errorf("CLAVIER INDISPONIBLE")
		}
		fmt.Fprint(in.Out, a[0].String()+" ") // le titre sert d'invite
		line, ok := in.keyb.ReadLine()
		if !ok {
			return ErrInterrompu
		}
		in.setVar(strings.ToUpper(name), lineToWordOrList(line))
		return nil
	}), "LIS")

	// manettes (emulees au clavier : fleches + barre d'espace)
	// MANETTE n : direction 0-8 de la manette n (0 = repos)
	op(1, func(in *Interp, a []Value) (Value, error) {
		n, err := toNumber(a[0])
		if err != nil {
			return Value{}, err
		}
		if in.joy == nil {
			return numResult(0), nil
		}
		return numResult(float64(in.joy.JoyDir(int(n)))), nil
	}, "MANETTE")
	// BOUTON? n : bouton de tir de la manette n enfonce ?
	op(1, func(in *Interp, a []Value) (Value, error) {
		n, err := toNumber(a[0])
		if err != nil {
			return Value{}, err
		}
		return BoolValue(in.joy != nil && in.joy.JoyButton(int(n))), nil
	}, "BOUTON?")
}
