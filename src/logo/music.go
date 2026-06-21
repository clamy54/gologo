package logo

import (
	"fmt"
	"math"
	"strings"
)

// valeurs par defaut des parametres musicaux
const (
	octaveDefaut = 4
	dureeDefaut  = 24
	tempoDefaut  = 5
	timbreDefaut = 0
	volumeDefaut = 100 // a fond : on garde l'ancien niveau
)

// demi-tons de chaque note a partir de DO
var notesDemiton = map[string]int{
	"DO": 0, "RE": 2, "MI": 4, "FA": 5, "SO": 7, "SOL": 7, "LA": 9, "SI": 11,
}

// frequence (Hz) d'une note : nom (DO RE MI FA SO/SOL LA SI) + alteration option.
// diese = suffixe # ou <# ; bemol = B ou <b (formes collees ok, le lecteur ne coupe
// pas "<#"/"<b"). "PA" = pause (freq 0). gamme temperee, DO octave 4 = do central
// (~261.6 Hz), approx. ok=false si nom inconnu
func noteFreq(word string, octave int) (float64, bool) {
	w := strings.ToUpper(word)
	if w == "PA" {
		return 0, true
	}
	alt := 0
	switch {
	case strings.HasSuffix(w, "<#"):
		alt, w = 1, w[:len(w)-2]
	case strings.HasSuffix(w, "<B"):
		alt, w = -1, w[:len(w)-2]
	case strings.HasSuffix(w, "#"):
		alt, w = 1, w[:len(w)-1]
	case strings.HasSuffix(w, "B"): // aucun nom de note ne finit par B
		alt, w = -1, w[:len(w)-1]
	}
	semi, ok := notesDemiton[w]
	if !ok {
		return 0, false
	}
	midi := 12*(octave+1) + semi + alt
	return 440 * math.Pow(2, float64(midi-69)/12), true
}

// suite de notes d'un argument de JOUE : une liste [ DO RE MI ] ou une note seule
func joueNotes(v Value) []string {
	if v.Kind == KList {
		out := make([]string, len(v.List))
		for i, d := range v.List {
			out[i] = d.String()
		}
		return out
	}
	return []string{v.String()}
}

// duree d'une note en ms : plus le tempo est grand, plus la note est courte
// approx (defaut 24/5 -> ~288 ms)
func noteDurationMs(duree, tempo int) int {
	ms := int(math.Round(float64(duree) * 60.0 / float64(tempo)))
	if ms < 1 {
		ms = 1
	}
	return ms
}

// OCTAVE/DUREE/TEMPO/TIMBRE (les parametres) et JOUE
func (i *Interp) registerMusic() {
	param := func(name string, lo, hi int, set func(*Interp, int)) {
		i.register(cmd(1, func(in *Interp, a []Value) error {
			n, err := toNumber(a[0])
			if err != nil {
				return err
			}
			k := int(n)
			if k < lo || k > hi {
				return fmt.Errorf("%s N'AIME PAS %s", name, a[0].String())
			}
			set(in, k)
			return nil
		}), name)
	}
	param("OCTAVE", 1, 5, func(in *Interp, n int) { in.musOctave = n })
	param("DUREE", 1, 96, func(in *Interp, n int) { in.musDuree = n })
	param("TEMPO", 1, 255, func(in *Interp, n int) { in.musTempo = n })
	param("TIMBRE", 0, 255, func(in *Interp, n int) { in.musTimbre = n })
	param("VOLUME", 0, 100, func(in *Interp, n int) { in.musVolume = n })

	i.register(cmd(1, func(in *Interp, a []Value) error {
		for _, w := range joueNotes(a[0]) { // une note seule ou une liste de notes
			if in.brk.Load() { // interruptible (Ctrl+C) entre les notes
				return ErrInterrompu
			}
			freq, ok := noteFreq(w, in.musOctave)
			if !ok {
				return fmt.Errorf("JOUE N'AIME PAS %s", w)
			}
			if in.sound != nil {
				in.sound.Tone(freq, noteDurationMs(in.musDuree, in.musTempo), in.musTimbre, in.musVolume)
			}
		}
		return nil
	}), "JOUE")
}
