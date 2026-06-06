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

func (r *Recorder) DrawSegment(seg Segment) { r.Segments = append(r.Segments, seg) }

func (r *Recorder) Clear() {
	r.Cleared++
	r.Segments = nil
}

func (r *Recorder) SetBackground(c Color) { r.Background = c }

func (r *Recorder) ShowGraphics() { r.GraphicsShown = true }

func (r *Recorder) TurtleMoved(s State) { r.LastTurtle = s }
