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

// Ebitengine plays linear PCM as 16-bit little endian, 2 channel stereo. The
// generators below are mono, so every sample is written to both channels; a
// buffer that skips this plays an octave high and half as long.
const bytesPerFrame = 4

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
	extraLife []byte

	thrustPlayer *audio.Player
	ufoPlayer    *audio.Player
	oneShots     []*audio.Player
	ufoLoopSize  int
	nextBeat     time.Time
	quietUntil   time.Time
	beatFlip     bool
}

func NewSoundManager() *SoundManager {
	ctx := audio.CurrentContext()
	if ctx == nil {
		ctx = audio.NewContext(sampleRate)
	}

	sm := &SoundManager{
		ctx:         ctx,
		ufoLoopSize: -1,
	}

	sm.fire = generateSweep(1250, 620, 0.09, 0.28, waveSquare, 18.0)
	sm.thrust = generateThrust()
	sm.beat1 = generateTone(110, 0.09, 0.28, waveSquare, 0.03)
	sm.beat2 = generateTone(82, 0.09, 0.30, waveSquare, 0.03)
	sm.bangSmall = generateExplosion(0.14, 0.34, 190, 90)
	sm.bangMed = generateExplosion(0.22, 0.42, 130, 55)
	sm.bangLarge = generateExplosion(0.32, 0.48, 90, 32)
	sm.ufoBig = generateSaucerLoop(150, 108, 0.86, 0.16)
	sm.ufoSmall = generateSaucerLoop(315, 235, 0.66, 0.14)
	sm.ufoFire = generateSweep(980, 420, 0.08, 0.24, waveSquare, 16.0)
	sm.shipBoom = generateShipExplosion()
	sm.extraLife = generateExtraLife()
	sm.nextBeat = time.Now().Add(700 * time.Millisecond)

	return sm
}

func (sm *SoundManager) Update(g *Game) {
	if sm == nil || g == nil {
		return
	}
	sm.cleanupOneShots()

	now := time.Now()
	if g.GameState != 0 {
		sm.stopThrust()
		sm.stopUFOLoop()
		return
	}

	if now.Before(sm.quietUntil) {
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

func (sm *SoundManager) PlayExtraLife() {
	sm.play(sm.extraLife)
}

func (sm *SoundManager) PlayShipExplosion() {
	sm.quietUntil = time.Now().Add(650 * time.Millisecond)
	sm.stopThrust()
	sm.stopUFOLoop()
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
	player := sm.newLoopingPlayer(sm.thrust)
	if player == nil {
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
	player := sm.newLoopingPlayer(sound)
	if player == nil {
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

func (sm *SoundManager) Close() {
	if sm == nil {
		return
	}
	sm.stopThrust()
	sm.stopUFOLoop()
	for _, p := range sm.oneShots {
		p.Pause()
		_ = p.Close()
	}
	sm.oneShots = nil
}

func (sm *SoundManager) play(sound []byte) {
	if sm == nil || len(sound) == 0 {
		return
	}
	p := sm.ctx.NewPlayerFromBytes(sound)
	sm.oneShots = append(sm.oneShots, p)
	p.Play()
}

// newLoopingPlayer builds a player that repeats sound for as long as it plays,
// so the thrust and saucer beds sustain instead of cutting out after one buffer.
func (sm *SoundManager) newLoopingPlayer(sound []byte) *audio.Player {
	if sm == nil || len(sound) == 0 {
		return nil
	}

	loop := audio.NewInfiniteLoop(bytes.NewReader(sound), int64(len(sound)))
	player, err := sm.ctx.NewPlayer(loop)
	if err != nil {
		return nil
	}
	return player
}

func (sm *SoundManager) cleanupOneShots() {
	active := sm.oneShots[:0]
	for _, p := range sm.oneShots {
		if p.IsPlaying() {
			active = append(active, p)
			continue
		}
		_ = p.Close()
	}
	sm.oneShots = active
}

const (
	waveSquare = iota
	waveSaw
)

// Tuning for the two beds that play continuously under the game.
const (
	thrustCarrierHz   = 78.0
	thrustFlutterHz   = 22.0
	thrustLoopSeconds = 0.5

	saucerPulseSeconds = 0.18
	saucerGateSeconds  = 0.12
)

// frameCountFor rounds a duration to whole frames. Truncating instead would
// drop the last frame whenever a computed duration lands a hair below its exact
// value, which is enough to break a loop's phase alignment.
func frameCountFor(seconds float64) int {
	return int(math.Round(seconds * sampleRate))
}

// snapFreq returns the frequency nearest freq that completes a whole number of
// cycles in window. A looped buffer built from whole windows then wraps at zero
// phase instead of jumping, which is the difference between a steady bed and a
// click once per repeat. Frequencies below one cycle per window are held at one.
func snapFreq(freq float64, window float64) float64 {
	cycles := math.Round(freq * window)
	if cycles < 1 {
		cycles = 1
	}
	return cycles / window
}

// wholePeriods returns the multiple of period nearest seconds, never shorter
// than a single period, so a loop ends exactly on a period boundary.
func wholePeriods(seconds float64, period float64) float64 {
	periods := math.Round(seconds / period)
	if periods < 1 {
		periods = 1
	}
	return periods * period
}

func generateTone(freq float64, seconds float64, volume float64, wave int, decay float64) []byte {
	count := frameCountFor(seconds)
	buf := bytes.NewBuffer(make([]byte, 0, count*bytesPerFrame))

	for i := 0; i < count; i++ {
		t := float64(i) / sampleRate
		amp := 1.0
		if decay > 0 {
			amp = math.Exp(-decay * t)
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

		writeSample(buf, v*volume*amp)
	}

	return buf.Bytes()
}

func generateSweep(startFreq float64, endFreq float64, seconds float64, volume float64, wave int, decay float64) []byte {
	count := frameCountFor(seconds)
	buf := bytes.NewBuffer(make([]byte, 0, count*bytesPerFrame))

	phase := 0.0
	for i := 0; i < count; i++ {
		progress := float64(i) / float64(count)
		t := float64(i) / sampleRate
		freq := startFreq + (endFreq-startFreq)*progress
		phase += 2 * math.Pi * freq / sampleRate
		amp := math.Exp(-decay * t)

		v := sampleWave(phase, wave)
		writeSample(buf, v*volume*amp)
	}

	return buf.Bytes()
}

func generateThrust() []byte {
	carrier := snapFreq(thrustCarrierHz, thrustLoopSeconds)
	flutterHz := snapFreq(thrustFlutterHz, thrustLoopSeconds)
	count := frameCountFor(thrustLoopSeconds)
	buf := bytes.NewBuffer(make([]byte, 0, count*bytesPerFrame))

	phase := 0.0
	for i := 0; i < count; i++ {
		t := float64(i) / sampleRate
		phase += 2 * math.Pi * carrier / sampleRate
		flutter := 0.75 + 0.25*math.Sin(2*math.Pi*flutterHz*t)
		noise := (mrand.Float64()*2 - 1) * 0.18
		rumble := sampleWave(phase, waveSaw) * 0.30
		writeSample(buf, (rumble+noise)*flutter)
	}

	return buf.Bytes()
}

func generateSaucerLoop(highFreq float64, lowFreq float64, seconds float64, volume float64) []byte {
	// Snap each tone to whole cycles per half pulse and the buffer to whole
	// pulses, so the loop repeats without a phase jump at either boundary.
	half := saucerPulseSeconds / 2
	high := snapFreq(highFreq, half)
	low := snapFreq(lowFreq, half)
	seconds = wholePeriods(seconds, saucerPulseSeconds)

	count := frameCountFor(seconds)
	buf := bytes.NewBuffer(make([]byte, 0, count*bytesPerFrame))

	phase := 0.0
	for i := 0; i < count; i++ {
		t := float64(i) / sampleRate
		pulse := math.Mod(t, saucerPulseSeconds)
		freq := low
		if pulse < half {
			freq = high
		}
		phase += 2 * math.Pi * freq / sampleRate
		gate := 0.35
		if pulse < saucerGateSeconds {
			gate = 1.0
		}
		writeSample(buf, sampleWave(phase, waveSquare)*volume*gate)
	}

	return buf.Bytes()
}

func generateExplosion(seconds float64, volume float64, startRumble float64, endRumble float64) []byte {
	count := frameCountFor(seconds)
	buf := bytes.NewBuffer(make([]byte, 0, count*bytesPerFrame))

	phase := 0.0
	for i := 0; i < count; i++ {
		progress := float64(i) / float64(count)
		t := float64(i) / sampleRate
		freq := startRumble + (endRumble-startRumble)*progress
		phase += 2 * math.Pi * freq / sampleRate

		envelope := math.Exp(-5.4 * t / seconds)
		crackle := (mrand.Float64()*2 - 1) * envelope
		rumble := sampleWave(phase, waveSquare) * envelope * 0.42
		writeSample(buf, (crackle+rumble)*volume)
	}

	return buf.Bytes()
}

// generateExtraLife is the free ship award: a rising two-step chime, pitched
// clear of the fire blip and the beat so it reads over whatever else is playing.
func generateExtraLife() []byte {
	low := generateTone(660, 0.10, 0.26, waveSquare, 1.2)
	high := generateTone(990, 0.14, 0.26, waveSquare, 1.6)
	return append(low, high...)
}

func generateShipExplosion() []byte {
	seconds := 0.62
	count := frameCountFor(seconds)
	buf := bytes.NewBuffer(make([]byte, 0, count*bytesPerFrame))

	rumblePhase := 0.0
	whinePhase := 0.0
	for i := 0; i < count; i++ {
		progress := float64(i) / float64(count)
		t := float64(i) / sampleRate

		rumbleFreq := 88 - 54*progress
		whineFreq := 520 * math.Pow(0.20, progress)
		rumblePhase += 2 * math.Pi * rumbleFreq / sampleRate
		whinePhase += 2 * math.Pi * whineFreq / sampleRate

		crackEnvelope := math.Exp(-12.0 * t)
		bodyEnvelope := math.Exp(-3.6 * t)
		tailEnvelope := math.Exp(-5.2 * t)

		crack := (mrand.Float64()*2 - 1) * 0.88 * crackEnvelope
		body := sampleWave(rumblePhase, waveSquare) * 0.44 * bodyEnvelope
		fallingWhine := sampleWave(whinePhase, waveSaw) * 0.34 * tailEnvelope
		dust := (mrand.Float64()*2 - 1) * 0.22 * bodyEnvelope

		writeSample(buf, crack+body+fallingWhine+dust)
	}

	return buf.Bytes()
}

func sampleWave(phase float64, wave int) float64 {
	switch wave {
	case waveSaw:
		cycle := math.Mod(phase/(2*math.Pi), 1)
		return 2 * (cycle - math.Floor(0.5+cycle))
	default:
		if math.Sin(phase) >= 0 {
			return 1
		}
		return -1
	}
}

func writeSample(buf *bytes.Buffer, value float64) {
	if value > 1 {
		value = 1
	} else if value < -1 {
		value = -1
	}

	sample := int16(value * math.MaxInt16)
	_ = binary.Write(buf, binary.LittleEndian, sample)
	_ = binary.Write(buf, binary.LittleEndian, sample)
}
