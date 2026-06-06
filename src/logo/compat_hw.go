package logo

// primitives materielles d'origine en mode "sans effet" : un vieux programme
// qui les appelle ne plante pas, il parle juste dans le vide
func (i *Interp) registerHardwareCompat() {
	noop := func(arity int, names ...string) {
		i.register(cmd(arity, func(*Interp, []Value) error { return nil }), names...)
	}
	noop(0, "REGLE")  // crayon optique
	noop(1, "ENTREE") // canal d'entree
	noop(1, "SORTIE") // canal de sortie
	noop(1, "FLI")    // fond de ligne
	noop(2, ".SER")   // port serie (inexistant sur MO5)
	noop(2, ".CHB")   // chargement binaire
	noop(1, ".RES")   // reserve memoire
	noop(2, ".DEP")   // depose un octet en memoire
	noop(1, ".ROUT")  // execute une routine machine
	// .EXA a : lecture memoire, rend toujours 0 (la memoire est fictive)
	i.register(&primitive{name: ".EXA", arity: 1, reporter: true, fn: func(*Interp, []Value) (Value, error) {
		return NumberValue(0), nil
	}}, ".EXA")
}
