// Package audio joue les notes de JOUE via oto (portable : ALSA sous Linux,
// WinMM sous Windows sans DLL, CoreAudio via purego sous macOS)
package audio

import (
	"bytes"
	"encoding/binary"
	"math"
	"time"

	"github.com/ebitengine/oto/v3"
)

const sampleRate = 44100

// implemente logo.Sound. si l'audio ne demarre pas (pas de peripherique), ctx reste
// nil et Tone se contente d'attendre la duree de la note pour garder le rythme
type Player struct {
	ctx *oto.Context
}

// ouvre le contexte audio. si ca echoue, rend un Player muet
func New() *Player {
	ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:   sampleRate,
		ChannelCount: 1,
		Format:       oto.FormatSignedInt16LE,
	})
	if err != nil {
		return &Player{}
	}
	<-ready
	return &Player{ctx: ctx}
}

// joue une note de freq Hz pendant ms et bloque jusqu'a la fin. la forme d'onde
// depend du timbre, le niveau du volume (0 a 100). freq <= 0 (ou pas d'audio) : on
// attend juste la duree, en silence
func (p *Player) Tone(freq float64, ms, timbre, volume int) {
	if ms <= 0 {
		return
	}
	if p.ctx == nil || freq <= 0 {
		time.Sleep(time.Duration(ms) * time.Millisecond)
		return
	}
	pl := p.ctx.NewPlayer(bytes.NewReader(wave(freq, ms, timbre, volume)))
	pl.Play()
	for pl.IsPlaying() {
		time.Sleep(2 * time.Millisecond)
	}
	pl.Close()
}

// genere un signal 16 bits mono, avec une courte attaque/extinction pour eviter
// les claquements. la forme depend du timbre (cf sample), l'amplitude du volume
func wave(freq float64, ms, timbre, volume int) []byte {
	n := sampleRate * ms / 1000
	period := float64(sampleRate) / freq
	fade := sampleRate / 200 // ~5 ms
	// volume 0 a 100 : a fond (100) on retrouve l'ancien niveau
	if volume < 0 {
		volume = 0
	} else if volume > 100 {
		volume = 100
	}
	amp := 6000 * float64(volume) / 100
	buf := bytes.NewBuffer(make([]byte, 0, n*2))
	for i := 0; i < n; i++ {
		phase := math.Mod(float64(i), period) / period // position 0..1 dans la periode
		env := 1.0
		if i < fade {
			env = float64(i) / float64(fade)
		}
		if k := n - 1 - i; k < fade {
			env = math.Min(env, float64(k)/float64(fade))
		}
		binary.Write(buf, binary.LittleEndian, int16(sample(phase, timbre)*amp*env))
	}
	return buf.Bytes()
}

// un echantillon (-1..1) selon le timbre : 4 familles d'ondes sur la plage 0-255
// (0 = carre, le bon vieux son retro par defaut)
func sample(phase float64, timbre int) float64 {
	switch {
	case timbre < 64: // carre
		if phase < 0.5 {
			return 1
		}
		return -1
	case timbre < 128: // dent de scie
		return 2*phase - 1
	case timbre < 192: // triangle
		if phase < 0.5 {
			return 4*phase - 1
		}
		return 3 - 4*phase
	default: // sinus
		return math.Sin(2 * math.Pi * phase)
	}
}
