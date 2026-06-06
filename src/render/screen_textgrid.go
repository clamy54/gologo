package render

import (
	"image"
	"image/color"
	"strings"
)

// remonte toute la grille d'une ligne ; la ligne du haut part aux oubliettes, celle du bas nait vide
func gridScrollUp(g *[textRows][textCols]rune, fg *[textRows][textCols]color.RGBA) {
	for r := 1; r < textRows; r++ {
		g[r-1], fg[r-1] = g[r], fg[r]
	}
	for c := 0; c < textCols; c++ {
		g[textRows-1][c], fg[textRows-1][c] = ' ', textDefault
	}
}

// va au debut de la ligne suivante, en faisant defiler si on est deja en bas
func gridNewline(g *[textRows][textCols]rune, fg *[textRows][textCols]color.RGBA, cr, cc *int) {
	*cc = 0
	if *cr < textRows-1 {
		*cr++
	} else {
		gridScrollUp(g, fg)
	}
}

// ramene le curseur sur une cellule valide : arrive au bord droit (col 40), on
// passe au debut de la ligne suivante
func gridNormalize(g *[textRows][textCols]rune, fg *[textRows][textCols]color.RGBA, cr, cc *int) {
	if *cc >= textCols {
		gridNewline(g, fg, cr, cc)
	}
}

// ecrit un caractere imprimable au curseur et avance (retour a la ligne en bout)
func gridPutRune(g *[textRows][textCols]rune, fg *[textRows][textCols]color.RGBA, cr, cc *int, r rune, col color.RGBA) {
	gridNormalize(g, fg, cr, cc)
	g[*cr][*cc], fg[*cr][*cc] = r, col
	*cc++
}

// ecrit une chaine dans la grille (gere \n et \r)
func gridWrite(g *[textRows][textCols]rune, fg *[textRows][textCols]color.RGBA, cr, cc *int, s string, col color.RGBA) {
	for _, r := range s {
		switch r {
		case '\n':
			gridNewline(g, fg, cr, cc)
		case '\r':
			*cc = 0
		default:
			gridPutRune(g, fg, cr, cc, r, col)
		}
	}
}

// remplit la grille d'espaces, curseur en haut a gauche
func gridClear(g *[textRows][textCols]rune, fg *[textRows][textCols]color.RGBA, cr, cc *int) {
	for r := 0; r < textRows; r++ {
		for c := 0; c < textCols; c++ {
			g[r][c], fg[r][c] = ' ', textDefault
		}
	}
	*cr, *cc = 0, 0
}

// FCURS (screenControl) : place le curseur texte (col 0-39, lig 0-24)
func (s *Screen) SetCursor(col, lig int) {
	s.mu.Lock()
	s.curCol = clampInt(col, 0, textCols-1)
	s.curRow = clampInt(lig, 0, textRows-1)
	s.mu.Unlock()
	s.invalidate()
}

// ME (screenControl) : nombre de lignes texte visibles (1-25). 25 = plein texte,
// le champ graphique disparait
func (s *Screen) SetTextLines(n int) {
	s.mu.Lock()
	s.meLines = clampInt(n, 1, textRows)
	s.meTextOnly = s.meLines >= textRows
	s.mu.Unlock()
	s.invalidate()
}

// dessine la grille texte 40x25 (la fenetre visible depend de ME et de la place) ;
// l'invite REPL (ou la saisie LL) est ecrite au curseur dans une copie
func (s *Screen) drawGrid(top int, grid [textRows][textCols]rune, fg [textRows][textCols]color.RGBA, curRow, curCol, meLines int, col color.RGBA) {
	// invite : au repos (REPL) ou pendant une lecture de ligne LL (montre la saisie)
	prompt := !s.running.Load()
	in, inCol := s.input, s.inputCol
	if s.kbActive.Load() && s.kbWantLine {
		prompt, in, inCol = true, s.kbInput, s.kbInputCol
	}
	// ecrit l'invite "?"+saisie dans la copie depuis le curseur, et note la cellule
	// du curseur de saisie (position lineaire row*40+col, -1 si pas d'invite)
	curCell := -1
	if prompt {
		cr, cc := curRow, curCol
		for i, r := range []rune("?" + in) {
			gridNormalize(&grid, &fg, &cr, &cc) // position reelle du prochain caractere (apres retour ligne)
			if i == 1+inCol {
				curCell = cr*textCols + cc
			}
			gridPutRune(&grid, &fg, &cr, &cc, r, col)
		}
		if curCell < 0 { // curseur en fin de saisie : meme normalisation
			gridNormalize(&grid, &fg, &cr, &cc)
			curCell = cr*textCols + cc
		}
	}
	// lignes physiquement affichables, plafonnees par ME
	visRows := (ScreenH - top - margin) / lineH
	if visRows < 1 {
		visRows = 1
	}
	if meLines < visRows {
		visRows = meLines
	}
	if visRows > textRows {
		visRows = textRows
	}
	// fenetre verticale : on garde visible la ligne du curseur
	focus := curRow
	if curCell >= 0 {
		focus = curCell / textCols
	}
	scroll := 0
	if focus >= visRows {
		scroll = focus - visRows + 1
	}
	if maxScroll := textRows - visRows; scroll > maxScroll {
		scroll = maxScroll
	}
	for vr := 0; vr < visRows && scroll+vr < textRows; vr++ {
		drawGridRow(s.frame, margin, top+vr*lineH, grid[scroll+vr][:], fg[scroll+vr][:])
	}
	// curseur de saisie : un bloc plein, le caractere dessous passe en video inverse
	if curCell >= 0 {
		cr, cc := curCell/textCols, curCell%textCols
		if cr >= scroll && cr < scroll+visRows {
			x := margin + cc*charW
			y := top + (cr-scroll)*lineH
			fillRect(s.frame, x, y-2, charW, 13*textScale+2, col)
			if ch := grid[cr][cc]; ch != ' ' && ch != 0 {
				drawText2x(s.frame, x, y, string(ch), color.RGBA{0, 0, 0, 255})
			}
		}
	}
}

// dessine une ligne de la grille en groupant les caracteres consecutifs de meme
// couleur (les espaces, eux, ne sont pas dessines)
func drawGridRow(dst *image.RGBA, x, y int, cells []rune, fg []color.RGBA) {
	for i := 0; i < len(cells); {
		if cells[i] == ' ' || cells[i] == 0 {
			i++
			continue
		}
		var b strings.Builder
		j := i
		for j < len(cells) && cells[j] != ' ' && cells[j] != 0 && fg[j] == fg[i] {
			b.WriteRune(cells[j])
			j++
		}
		drawText2x(dst, x+i*charW, y, b.String(), fg[i])
		i = j
	}
}
