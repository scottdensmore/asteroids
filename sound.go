package main

import (
	"bytes"
	"encoding/binary"
	"math"
	mrand "math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

const sampleRate = 44100

type SoundManager struct {
	ctx *audio.Context

	fire      []byte
	thrust    []byte
	beat1     []byte
	beat2     []byte
	bangSmall []byte
	bangMed   []byte
	bangLarge []byte
	ufoBig    []byte
	ufoSmall  []byte
	ufoFire   []byte
	shipBoom  []byte

	thrustPlayer *audio.Player
	ufoPlayer    *audio.Player
	ufoLoopSize  int
	nextBeat     time.Time
	beatFlip     bool
}

func NewSoundManager() *SoundManager {
	sm := &SoundManager{
		ctx:         audio.NewContext(sampleRate),
		ufoLoopSize: -1,
	}

	sm.fire = generateTone(880, 0.08, 0.25, waveSquare, 0.06)
	sm.thrust = generateTone(95, 0.15, 0.15, waveSaw, 0.00)
	sm.beat1 = generateTone(110, 0.09, 0.28, waveSquare, 0.03)
	sm.beat2 = generateTone(82, 0.09, 0.30, waveSquare, 0.03)
	sm.bangSmall = generateNoise(0.12, 0.24)
	sm.bangMed = generateNoise(0.18, 0.30)
	sm.bangLarge = generateNoise(0.25, 0.35)
	sm.ufoBig = generateTone(170, 0.22, 0.14, waveSquare, 0.00)
	sm.ufoSmall = generateTone(285, 0.18, 0.12, waveSquare, 0.00)
	sm.ufoFire = generateTone(760, 0.07, 0.20, waveSquare, 0.05)
	sm.shipBoom = generateNoise(0.30, 0.40)
	sm.nextBeat = time.Now().Add(700 * time.Millisecond)

	return sm
}

func (sm *SoundManager) Update(g *Game) {
	if sm == nil || g == nil {
		return
	}

	if g.GameState != 0 {
		sm.stopThrust()
		sm.stopUFOLoop()
		return
	}

	if g.Ship != nil && g.Ship.IsThrusting {
		sm.startThrust()
	} else {
		sm.stopThrust()
	}

	hasSmall := false
	hasBig := false
	for _, u := range g.UFOs {
		if u.Size == 0 {
			hasSmall = true
		} else {
			hasBig = true
		}
	}
	if hasSmall {
		sm.startUFOLoop(0)
	} else if hasBig {
		sm.startUFOLoop(1)
	} else {
		sm.stopUFOLoop()
	}

	now := time.Now()
	if !now.Before(sm.nextBeat) {
		if sm.beatFlip {
			sm.play(sm.beat2)
		} else {
			sm.play(sm.beat1)
		}
		sm.beatFlip = !sm.beatFlip

		remaining := len(g.Asteroids)
		interval := 250 + remaining*70
		if interval > 900 {
			interval = 900
		}
		sm.nextBeat = now.Add(time.Duration(interval) * time.Millisecond)
	}
}

func (sm *SoundManager) PlayShipFire() {
	sm.play(sm.fire)
}

func (sm *SoundManager) PlayUFOFire() {
	sm.play(sm.ufoFire)
}

func (sm *SoundManager) PlayShipExplosion() {
	sm.play(sm.shipBoom)
}

func (sm *SoundManager) PlayAsteroidHit(size int) {
	switch size {
	case 1:
		sm.play(sm.bangSmall)
	case 2:
		sm.play(sm.bangMed)
	default:
		sm.play(sm.bangLarge)
	}
}

func (sm *SoundManager) PlayUFOHit(size int) {
	if size == 0 {
		sm.play(sm.bangSmall)
		return
	}
	sm.play(sm.bangMed)
}

func (sm *SoundManager) startThrust() {
	if sm.thrustPlayer != nil && sm.thrustPlayer.IsPlaying() {
		return
	}
	if sm.thrustPlayer != nil {
		_ = sm.thrustPlayer.Close()
		sm.thrustPlayer = nil
	}
	player, err := sm.ctx.NewPlayerFromBytes(sm.thrust)
	if err != nil {
		return
	}
	sm.thrustPlayer = player
	player.Play()
}

func (sm *SoundManager) stopThrust() {
	if sm.thrustPlayer == nil {
		return
	}
	sm.thrustPlayer.Pause()
	_ = sm.thrustPlayer.Close()
	sm.thrustPlayer = nil
}

func (sm *SoundManager) startUFOLoop(size int) {
	if sm.ufoPlayer != nil && sm.ufoPlayer.IsPlaying() && sm.ufoLoopSize == size {
		return
	}
	sm.stopUFOLoop()

	var sound []byte
	if size == 0 {
		sound = sm.ufoSmall
	} else {
		sound = sm.ufoBig
	}
	player, err := sm.ctx.NewPlayerFromBytes(sound)
	if err != nil {
		return
	}
	sm.ufoPlayer = player
	sm.ufoLoopSize = size
	player.Play()
}

func (sm *SoundManager) stopUFOLoop() {
	if sm.ufoPlayer == nil {
		sm.ufoLoopSize = -1
		return
	}
	sm.ufoPlayer.Pause()
	_ = sm.ufoPlayer.Close()
	sm.ufoPlayer = nil
	sm.ufoLoopSize = -1
}

func (sm *SoundManager) play(sound []byte) {
	if sm == nil || len(sound) == 0 {
		return
	}
	p, err := sm.ctx.NewPlayerFromBytes(sound)
	if err != nil {
		return
	}
	p.Play()
}

const (
	waveSquare = iota
	waveSaw
)

func generateTone(freq float64, seconds float64, volume float64, wave int, decay float64) []byte {
	count := int(float64(sampleRate) * seconds)
	buf := bytes.NewBuffer(make([]byte, 0, count*2))

	for i := 0; i < count; i++ {
		t := float64(i) / sampleRate
		amp := 1.0
		if decay > 0 {
			amp = math.Exp(-decay * float64(i))
		}

		phase := 2 * math.Pi * freq * t
		var v float64
		switch wave {
		case waveSaw:
			saw := 2 * (t*freq - math.Floor(0.5+t*freq))
			v = saw
		default:
			if math.Sin(phase) >= 0 {
				v = 1
			} else {
				v = -1
			}
		}

		sample := int16(v * volume * amp * math.MaxInt16)
		_ = binary.Write(buf, binary.LittleEndian, sample)
	}

	return buf.Bytes()
}

func generateNoise(seconds float64, volume float64) []byte {
	count := int(float64(sampleRate) * seconds)
	buf := bytes.NewBuffer(make([]byte, 0, count*2))

	for i := 0; i < count; i++ {
		envelope := math.Exp(-6.5 * float64(i) / float64(count))
		noise := (mrand.Float64()*2 - 1) * volume * envelope
		sample := int16(noise * math.MaxInt16)
		_ = binary.Write(buf, binary.LittleEndian, sample)
	}

	return buf.Bytes()
}
