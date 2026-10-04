package render

import (
	"bytes"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"beroot.com/logo/turtle"
)

// tests de non-regression du rendu, sans fenetre : on dessine dans l'ecran en
// memoire et on regarde les pixels des PNG produits (SAUVEPNG / COPIE)

// couleur du champ exporte au point Logo (x,y)
func fieldPixel(t *testing.T, s *Screen, x, y float64) color.RGBA {
	t.Helper()
	var buf bytes.Buffer
	if err := s.SaveFieldPNG(&buf); err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(&buf)
	if err != nil {
		t.Fatal(err)
	}
	fx, fy := logoToField(x, y)
	r, g, b, a := img.At(int(fx), int(fy)).RGBA()
	return color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
}

func seg(x0, y0, x1, y1 float64, pen turtle.Color, w int) turtle.Segment {
	return turtle.Segment{X0: x0, Y0: y0, X1: x1, Y1: y1, Pen: pen, Width: w}
}

// l'export voit les memes frontieres que l'ecran : un contour de la couleur du fond
// arrete bien le remplissage
func TestRemplissageExporte(t *testing.T) {
	s := New()
	for _, c := range [][4]float64{{-100, -100, 100, -100}, {100, -100, 100, 100}, {100, 100, -100, 100}, {-100, 100, -100, -100}} {
		s.DrawSegment(seg(c[0], c[1], c[2], c[3], turtle.Bleu, 2))
	}
	s.Fill(0, 0, turtle.Rouge)
	if got := fieldPixel(t, s, 0, 0); got != rgba(turtle.Rouge) {
		t.Errorf("interieur : %v", got)
	}
	if got := fieldPixel(t, s, 300, 300); got != rgba(turtle.Bleu) {
		t.Errorf("exterieur envahi par le remplissage : %v", got)
	}
}

// un code de couleur negatif gomme : le fond reapparait, meme change apres coup
func TestGomme(t *testing.T) {
	s := New()
	s.DrawSegment(seg(-100, 0, 100, 0, turtle.Rouge, 20))
	s.DrawSegment(seg(-50, 0, 50, 0, turtle.Color(-1), 6))
	s.SetBackground(turtle.Vert)
	if got := fieldPixel(t, s, 0, 0); got != rgba(turtle.Vert) {
		t.Errorf("gomme : %v", got)
	}
	if got := fieldPixel(t, s, 80, 0); got != rgba(turtle.Rouge) {
		t.Errorf("hors gomme : %v", got)
	}
}

// en mode FEN les coordonnees peuvent etre gigantesques : le segment qui traverse
// l'ecran est quand meme trace
func TestSegmentsGeants(t *testing.T) {
	s := New()
	s.DrawSegment(seg(-1e300, 0, 1e300, 0, turtle.Blanc, 2))
	s.DrawSegment(seg(-1e15, -1e15, 1e15, 1e15, turtle.Jaune, 2))
	if got := fieldPixel(t, s, 200, 0); got != rgba(turtle.Blanc) {
		t.Errorf("horizontal : %v", got)
	}
	if got := fieldPixel(t, s, 100, 100); got != rgba(turtle.Jaune) {
		t.Errorf("diagonal : %v", got)
	}
}

// le journal graphique ne grossit pas sans limite, et une etiquette interminable
// ne garde que sa partie visible
func TestJournalBorne(t *testing.T) {
	s := New()
	s.Label(-900, 0, strings.Repeat("ABCDEFGH", 1<<20), turtle.Blanc)
	s.Label(900, 0, "INVISIBLE", turtle.Blanc)
	if len(s.gfx) != 1 || len(s.gfx[0].text) > 80 {
		t.Fatalf("etiquettes : %d operations", len(s.gfx))
	}
	for k := 0; k < maxPendingGfx+10; k++ {
		s.DrawSegment(seg(0, 0, 10, 10, turtle.Blanc, 2))
	}
	if len(s.gfx) > 20 {
		t.Errorf("journal : %d operations en attente", len(s.gfx))
	}
}

// pendant un SCENE l'ecran garde l'image d'avant, mais une COPIE voit le dessin
// en cours
func TestCopieDansScene(t *testing.T) {
	s := New()
	s.ShowGraphics()
	s.bakePending()
	s.compose()
	s.BeginFrame()
	s.DrawSegment(seg(-200, 0, 200, 0, turtle.Rouge, 4))
	s.bakePending()
	s.compose()
	fx, fy := logoToField(0, 0)
	px, py := fieldX+int(fx), fieldY+int(fy)
	if s.frame.RGBAAt(px, py) == rgba(turtle.Rouge) {
		t.Error("SCENE : image partielle affichee")
	}
	var buf bytes.Buffer
	if err := s.SaveScreenPNG(&buf); err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(&buf)
	if err != nil {
		t.Fatal(err)
	}
	r, g, b, _ := img.At(px, py).RGBA()
	if (color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), 255}) != rgba(turtle.Rouge) {
		t.Error("COPIE dans SCENE : ancienne image")
	}
	s.EndFrame()
}
