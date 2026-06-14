package render

import (
	"image"
	"image/draw"
	"image/png"
	"os"

	"beroot.com/logo/turtle"
)

// compose un champ a partir des seuls traits (fond + segments) dans une image CPU,
// sans fenetre. utilitaire de test ; le rendu complet (remplissages, etiquettes)
// passe par renderTimeline
func RenderField(segs []turtle.Segment, bg turtle.Color) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, fieldW, fieldH))
	draw.Draw(img, img.Bounds(), image.NewUniform(rgba(bg)), image.Point{}, draw.Src)
	for _, seg := range segs {
		x0, y0 := logoToField(seg.X0, seg.Y0)
		x1, y1 := logoToField(seg.X1, seg.Y1)
		drawSeg(img, x0, y0, x1, y1, rgba(seg.Pen), segWidth(seg.Width))
	}
	return img
}

// rejoue une tranche du journal graphique dans l'ordre, sur une image du champ
// (coords Logo vers pixels). traits, remplissages et etiquettes partagent le meme
// flux : un REMPLIS ne voit que ce qui le precede, un trait d'apres passe dessus.
// utilise par compose et SAUVEPNG
func bakeGfx(img *image.RGBA, ops []gfxOp) {
	for _, op := range ops {
		switch op.kind {
		case gfxSeg:
			x0, y0 := logoToField(op.seg.X0, op.seg.Y0)
			x1, y1 := logoToField(op.seg.X1, op.seg.Y1)
			drawSeg(img, x0, y0, x1, y1, rgba(op.seg.Pen), segWidth(op.seg.Width))
		case gfxFill:
			fx, fy := logoToField(op.x, op.y)
			floodFill(img, int(fx), int(fy), rgba(op.col))
		case gfxLabel:
			fx, fy := logoToField(op.x, op.y)
			drawText2x(img, int(fx), int(fy), op.text, rgba(op.col))
		}
	}
}

// compose tout le journal graphique (fond + traits + remplissages + etiquettes,
// dans l'ordre reel) en une image CPU. pour SAUVEPNG
func renderTimeline(gfx []gfxOp, bg turtle.Color) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, fieldW, fieldH))
	draw.Draw(img, img.Bounds(), image.NewUniform(rgba(bg)), image.Point{}, draw.Src)
	bakeGfx(img, gfx)
	return img
}

// SAUVEPNG (logo.ImageSaver) : ecrit le champ courant en PNG, fond + traits +
// remplissages + etiquettes dans l'ordre
func (s *Screen) SaveFieldPNG(path string) error {
	s.mu.Lock()
	gfx := append([]gfxOp(nil), s.gfx...)
	bg := s.bg
	s.mu.Unlock()
	img := renderTimeline(gfx, bg)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// demande au thread UI une copie de la frame courante (COPIE). l'envoi sur
// captureCh ne doit surtout PAS bloquer : le canal est bufferise (cap 1), donc on
// depose la requete puis schedule() programme une frame (s.invalidate en vrai).
// draw() draine captureCh apres compose() et renvoie une copie de s.frame, qu'on
// attend sur resp. sans ce buffer, au repos l'envoi bloquerait, schedule() ne
// serait jamais atteint, et la COPIE attendrait une frame qui ne viendra jamais : fige
func (s *Screen) captureFrame(schedule func()) *image.RGBA {
	resp := make(chan *image.RGBA, 1)
	s.captureWaiting.Add(1) // leve la mise en sommeil d'invalidate pendant un SCENE gele
	defer s.captureWaiting.Add(-1)
	s.captureCh <- resp // bufferise : non bloquant
	schedule()          // programme une frame qui servira la capture
	return <-resp
}

// copie une image RGBA (pixels compris) pour la filer a un autre thread
func cloneRGBA(src *image.RGBA) *image.RGBA {
	dst := image.NewRGBA(src.Bounds())
	copy(dst.Pix, src.Pix)
	return dst
}

// COPIE (logo.ScreenSaver) : sauve l'ecran entier tel qu'affiche (champ + texte +
// tortue), comme la vieille copie d'ecran MO5. la frame appartient au thread UI ;
// on en demande une copie via captureCh (servie dans draw), puis on l'encode
// ailleurs. sans fenetre (tests headless), on compose la frame courante directement
func (s *Screen) SaveScreenPNG(path string) error {
	var img *image.RGBA
	if s.win != nil {
		img = s.captureFrame(s.invalidate)
	} else {
		s.compose() // headless : pas de boucle d'evenements, on compose nous-memes
		img = cloneRGBA(s.frame)
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
