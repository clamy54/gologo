package logo

import (
	"errors"
	"strings"
)

// signal de LANCE, attrape par un PIEGE de meme etiquette
type throwSignal struct{ tag string }

func (t *throwSignal) Error() string { return "PIEGE MANQUANT POUR " + t.tag }

// erreur attrapable par PIEGE "ERREUR : ni controle de flux, ni interruption/quitte,
// ni un LANCE (celui-la se gere par etiquette exacte)
func catchable(err error) bool {
	if errors.Is(err, ErrInterrompu) || errors.Is(err, ErrQuitter) {
		return false
	}
	if _, ok := err.(*ctrl); ok {
		return false
	}
	if _, ok := err.(*throwSignal); ok {
		return false
	}
	return true
}

func (i *Interp) registerControl2() {
	// LANCE "etiquette : saute jusqu'au PIEGE de meme etiquette
	i.register(&primitive{name: "LANCE", arity: 1, fn: func(in *Interp, a []Value) (Value, error) {
		tag, err := toWord(a[0])
		if err != nil {
			return Value{}, err
		}
		return None, &throwSignal{tag: strings.ToUpper(tag)}
	}}, "LANCE")

	// PIEGE "etiquette [ instructions ] : execute et attrape un LANCE de meme
	// etiquette. avec l'etiquette ERREUR, attrape aussi les erreurs d'execution
	i.register(cmd(2, func(in *Interp, a []Value) error {
		tag, err := toWord(a[0])
		if err != nil {
			return err
		}
		tag = strings.ToUpper(tag)
		if a[1].Kind != KList {
			return &badData{a[1].String()}
		}
		err = in.runSeq(a[1].List)
		if err == nil {
			return nil
		}
		if ts, ok := err.(*throwSignal); ok {
			if ts.tag == tag {
				return nil // attrape
			}
			return err // autre etiquette : on laisse remonter
		}
		if (tag == "ERREUR" || tag == "ERROR") && catchable(err) {
			return nil // attrape une erreur d'execution
		}
		return err
	}), "PIEGE")

	// TESTE pred / SIVRAI [..] / SIFAUX [..] : conditionnelle a memoire
	i.register(cmd(1, func(in *Interp, a []Value) error {
		in.testResult, in.testSet = a[0].IsTrue(), true
		return nil
	}), "TESTE")
	i.register(cmd(1, func(in *Interp, a []Value) error {
		if a[0].Kind != KList {
			return &badData{a[0].String()}
		}
		if in.testSet && in.testResult {
			return in.runSeq(a[0].List)
		}
		return nil
	}), "SIVRAI")
	i.register(cmd(1, func(in *Interp, a []Value) error {
		if a[0].Kind != KList {
			return &badData{a[0].String()}
		}
		if in.testSet && !in.testResult {
			return in.runSeq(a[0].List)
		}
		return nil
	}), "SIFAUX")
}
