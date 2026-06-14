package turtle

// Canvas bidon qui note tout au lieu de dessiner. pour les tests (golden tests
// de segments) et comme reference headless
type Recorder struct {
	Segments      []Segment
	Background    Color
	Cleared       int   // nombre d'appels a Clear
	GraphicsShown bool  // ShowGraphics appele ?
	LastTurtle    State // dernier etat notifie
}

func NewRecorder() *Recorder { return &Recorder{} }

// plafond anti-fuite : le journal sert aux golden tests et au mode headless, pas a
// stocker une animation infinie. au-dela on cesse d'enregistrer (la tete suffit).
const maxRecorderSegments = 1 << 20

func (r *Recorder) DrawSegment(seg Segment) {
	if len(r.Segments) >= maxRecorderSegments {
		return
	}
	r.Segments = append(r.Segments, seg)
}

func (r *Recorder) Clear() {
	r.Cleared++
	r.Segments = nil
}

func (r *Recorder) SetBackground(c Color) { r.Background = c }

func (r *Recorder) ShowGraphics() { r.GraphicsShown = true }

func (r *Recorder) TurtleMoved(s State) { r.LastTurtle = s }
