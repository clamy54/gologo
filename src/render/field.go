package render

import (
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"

	"beroot.com/logo/turtle"
)

// couleur d'un trace dans le calque du champ. un code negatif est la GOMME : le
// calque redevient transparent a cet endroit et laisse voir le fond, quel qu'il
// soit (couleur libre comprise, et meme si FCFG le change ensuite)
func penRGBA(c turtle.Color) color.RGBA {
	if !c.IsRGB() && int(c) < 0 {
		return color.RGBA{}
	}
	return rgba(c)
}

// compose un champ a partir des seuls traits (fond + segments) dans une image CPU,
// sans fenetre. utilitaire de test ; le rendu complet (remplissages, etiquettes)
// passe par le calque du champ (bakeGfx)
func RenderField(segs []turtle.Segment, bg turtle.Color) *image.RGBA {
	layer := image.NewRGBA(image.Rect(0, 0, fieldW, fieldH))
	for _, seg := range segs {
		x0, y0 := logoToField(seg.X0, seg.Y0)
		x1, y1 := logoToField(seg.X1, seg.Y1)
		drawSeg(layer, x0, y0, x1, y1, penRGBA(seg.Pen), segWidth(seg.Width))
	}
	return flattenField(layer, bg)
}

// rejoue une tranche du journal graphique dans l'ordre, sur le calque du champ
// (coords Logo vers pixels). traits, remplissages et etiquettes partagent le meme
// flux : un REMPLIS ne voit que ce qui le precede, un trait d'apres passe dessus
func bakeGfx(img *image.RGBA, ops []gfxOp) {
	for _, op := range ops {
		switch op.kind {
		case gfxSeg:
			x0, y0 := logoToField(op.seg.X0, op.seg.Y0)
			x1, y1 := logoToField(op.seg.X1, op.seg.Y1)
			drawSeg(img, x0, y0, x1, y1, penRGBA(op.seg.Pen), segWidth(op.seg.Width))
		case gfxFill:
			fx, fy := logoToField(op.x, op.y)
			if fx < 0 || fx >= fieldW || fy < 0 || fy >= fieldH {
				continue // tortue hors de l'ecran (FEN) : rien a remplir
			}
			floodFill(img, int(fx), int(fy), penRGBA(op.col))
		case gfxLabel:
			fx, fy := logoToField(op.x, op.y)
			drawText2x(img, int(fx), int(fy), op.text, penRGBA(op.col))
		}
	}
}

// le champ tel qu'on le voit : le fond, puis le calque des traces par-dessus
func flattenField(layer *image.RGBA, bg turtle.Color) *image.RGBA {
	img := image.NewRGBA(layer.Bounds())
	draw.Draw(img, img.Bounds(), image.NewUniform(rgba(bg)), image.Point{}, draw.Src)
	draw.Draw(img, img.Bounds(), layer, image.Point{}, draw.Over)
	return img
}

// nombre d'operations graphiques en attente au-dela duquel celle qui dessine les
// applique elle-meme au calque, sans attendre la prochaine image (fenetre masquee,
// boucle de dessin tres rapide) : le journal ne grossit jamais sans limite
const maxPendingGfx = 1 << 16

// applique au calque du champ les operations graphiques en attente, puis les
// oublie : une fois dessinees elles ne servent plus a rien. appelable depuis la
// boucle d'evenements comme depuis la tache qui dessine (verrou fieldMu)
func (s *Screen) bakePending() {
	s.fieldMu.Lock()
	defer s.fieldMu.Unlock()
	s.bakeLocked()
}

func (s *Screen) bakeLocked() {
	s.mu.Lock()
	ops, gen := s.gfx, s.clearGen
	s.gfx = s.gfxSpare[:0]
	s.gfxSpare = nil
	s.mu.Unlock()
	if gen != s.bakedGen { // VE/NETTOIE depuis le dernier passage : calque vierge
		clearImage(s.fieldImg)
		s.bakedGen = gen
	}
	bakeGfx(s.fieldImg, ops)
	if cap(ops) <= maxPendingGfx { // on recycle le tampon, sauf s'il a trop grossi
		s.mu.Lock()
		if s.gfxSpare == nil {
			s.gfxSpare = ops[:0]
		}
		s.mu.Unlock()
	}
}

// SAUVEPNG (logo.ImageSaver) : ecrit le champ courant en PNG. c'est le meme calque
// et la meme composition qu'a l'ecran (fond puis traces), donc la meme image : un
// REMPLIS ne peut pas donner un resultat different a l'export
func (s *Screen) SaveFieldPNG(w io.Writer) error {
	s.mu.Lock()
	bg := s.bg
	s.mu.Unlock()
	s.fieldMu.Lock()
	s.bakeLocked()
	img := flattenField(s.fieldImg, bg)
	s.fieldMu.Unlock()
	return png.Encode(w, img)
}

// demande au thread UI une copie de l'ecran (COPIE). l'envoi sur captureCh ne doit
// surtout PAS bloquer : le canal est bufferise (cap 1), donc on depose la requete
// puis schedule() programme une frame (s.invalidate en vrai). draw() draine
// captureCh et renvoie une copie de l'ecran compose, qu'on attend sur resp. sans ce
// buffer, au repos l'envoi bloquerait, schedule() ne serait jamais atteint, et la
// COPIE attendrait une frame qui ne viendra jamais : fige. nil si l'appli ferme
func (s *Screen) captureFrame(schedule func()) *image.RGBA {
	resp := make(chan *image.RGBA, 1)
	s.captureWaiting.Add(1) // leve la mise en sommeil d'invalidate pendant un SCENE gele
	defer s.captureWaiting.Add(-1)
	select {
	case s.captureCh <- resp: // bufferise : non bloquant
	case <-s.closed:
		return nil
	}
	schedule() // programme une frame qui servira la capture
	select {
	case img := <-resp:
		return img
	case <-s.closed: // la fenetre se ferme : plus personne ne repondra
		return nil
	}
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
func (s *Screen) SaveScreenPNG(w io.Writer) error {
	var img *image.RGBA
	if s.win != nil {
		if img = s.captureFrame(s.invalidate); img == nil {
			return errors.New("capture interrompue")
		}
	} else {
		s.bakePending()
		s.composeNow() // headless : pas de boucle d'evenements, on compose nous-memes
		img = cloneRGBA(s.frame)
	}
	return png.Encode(w, img)
}
