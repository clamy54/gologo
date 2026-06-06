package logo

import (
	"fmt"
	"strings"

	"beroot.com/logo/turtle"
)

// cote (en pixels) d'une forme DEFSPRITE
const spriteSize = 16

// SPRITE (choix de forme) et DEFSPRITE (formes 16x16 redefinissables, facon
// DEFGR$ du BASIC MO5)
func (i *Interp) registerSprites() {
	// SPRITE n : forme de la tortue active. 0/1/2 = integrees (triangle, tortue,
	// voiture), n >= 3 = forme DEFSPRITE. un numero non defini ou hors plage est ignore
	i.register(cmd(1, func(in *Interp, a []Value) error {
		n, err := turtleIndex(a[0])
		if err != nil {
			return err
		}
		if n >= turtle.SpriteCount {
			if _, ok := in.spriteDefs[n]; !ok {
				return nil // forme non definie : on garde la forme actuelle
			}
		}
		in.Turtle.SetShape(n)
		return nil
	}), "SPRITE")

	// DEFSPRITE n [ ... ] : definit la forme 16x16 numero n (3 a MaxSprites-1),
	// ensuite choisie par SPRITE n. jusqu'a 16 lignes de 16 caracteres : '.' = vide,
	// tout le reste = plein (eviter '#', c'est le marqueur de commentaire). dessinee
	// dans la couleur du crayon, pivote avec le cap
	i.register(cmd(2, func(in *Interp, a []Value) error {
		n, err := turtleIndex(a[0])
		if err != nil {
			return err
		}
		if n < turtle.SpriteCount || n >= turtle.MaxSprites {
			return fmt.Errorf("DEFSPRITE N'AIME PAS %s", a[0].String())
		}
		rows, err := parseSprite(a[1])
		if err != nil {
			return err
		}
		if in.spriteDefs == nil {
			in.spriteDefs = map[int][]string{}
		}
		in.spriteDefs[n] = rows
		if sc, ok := in.screen(); ok {
			sc.DefineSprite(n, rows)
		}
		return nil
	}), "DEFSPRITE")
}

// normalise le corps de DEFSPRITE en spriteSize lignes de spriteSize caracteres
// ('#' = plein, le reste = vide). refuse une grille trop grande
func parseSprite(v Value) ([]string, error) {
	if v.Kind != KList {
		return nil, &badData{v.String()}
	}
	if len(v.List) > spriteSize {
		return nil, fmt.Errorf("SPRITE TROP GRAND (MAX %dx%d)", spriteSize, spriteSize)
	}
	rows := make([]string, spriteSize)
	blank := strings.Repeat(".", spriteSize)
	for i := range rows {
		rows[i] = blank
	}
	for i, d := range v.List {
		src := []rune(d.String())
		if len(src) > spriteSize {
			return nil, fmt.Errorf("SPRITE TROP GRAND (MAX %dx%d)", spriteSize, spriteSize)
		}
		var b strings.Builder
		for _, ch := range src {
			if ch == '.' {
				b.WriteByte('.') // vide
			} else {
				b.WriteByte('#') // plein (codage interne pour drawTurtle)
			}
		}
		for b.Len() < spriteSize {
			b.WriteByte('.')
		}
		rows[i] = b.String()
	}
	return rows, nil
}
