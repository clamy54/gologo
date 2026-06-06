package logo

import (
	"fmt"
	"strings"

	"beroot.com/logo/turtle"
)

// animation auto des tortues : ANIME lance un deplacement continu vers une cible,
// STOPANIME l'arrete, CADENCE regle les images/seconde. le mouvement tourne en
// parallele du programme -> jeux et scrollings (cf COLLISION?)
func (i *Interp) registerAnimate() {
	// ANIME id x y vitesse mode : tortue id vers (x,y), `vitesse` pas Logo par image,
	// mode "UNEFOIS / "BOUCLE / "PINGPONG
	i.register(cmd(5, func(in *Interp, a []Value) error {
		id, err := turtleIndex(a[0])
		if err != nil {
			return err
		}
		x, err := toNumber(a[1])
		if err != nil {
			return err
		}
		y, err := toNumber(a[2])
		if err != nil {
			return err
		}
		sp, err := toNumber(a[3])
		if err != nil {
			return err
		}
		if sp <= 0 {
			return fmt.Errorf("ANIME N'AIME PAS %s", a[3].String())
		}
		mode, err := parseAnimMode(a[4])
		if err != nil {
			return err
		}
		return in.Turtle.SetMotion(id, x, y, sp, mode)
	}), "ANIME")

	// STOPANIME id : stoppe l'animation de la tortue id. (STOPANIME) sans argument
	// les arrete toutes
	i.register(vcmd(1, func(in *Interp, a []Value) error {
		if len(a) == 0 {
			in.Turtle.StopAllMotion()
			return nil
		}
		id, err := turtleIndex(a[0])
		if err != nil {
			return err
		}
		in.Turtle.StopMotion(id)
		return nil
	}), "STOPANIME")

	// CADENCE n : images par seconde du moteur d'animation (1 a 240)
	i.register(cmd(1, func(in *Interp, a []Value) error {
		n, err := toNumber(a[0])
		if err != nil {
			return err
		}
		in.Turtle.SetFPS(int(n))
		return nil
	}), "CADENCE")
}

// lit le mot de mode d'ANIME (FR ou EN, insensible a la casse)
func parseAnimMode(v Value) (turtle.AnimMode, error) {
	switch strings.ToUpper(v.String()) {
	case "UNEFOIS", "ONCE":
		return turtle.Once, nil
	case "BOUCLE", "LOOP", "RESTART":
		return turtle.Loop, nil
	case "PINGPONG", "PING-PONG":
		return turtle.PingPong, nil
	}
	return 0, fmt.Errorf("ANIME N'AIME PAS %s", v.String())
}
