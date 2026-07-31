package main

import (
	"encoding/binary"
	"math"
	"testing"
)

func frameCount(buf []byte) int {
	return len(buf) / bytesPerFrame
}

// channels decodes frame i into its left and right samples.
func channels(t *testing.T, buf []byte, i int) (int16, int16) {
	t.Helper()

	offset := i * bytesPerFrame
	if offset+bytesPerFrame > len(buf) {
		t.Fatalf("frame %d is out of range for a %d byte buffer", i, len(buf))
	}

	left := int16(binary.LittleEndian.Uint16(buf[offset : offset+2]))
	right := int16(binary.LittleEndian.Uint16(buf[offset+2 : offset+4]))
	return left, right
}

func generatedSounds(t *testing.T) map[string][]byte {
	t.Helper()

	return map[string][]byte{
		"tone":          generateTone(440, 0.25, 0.5, waveSquare, 0),
		"sweep":         generateSweep(900, 400, 0.25, 0.5, waveSquare, 4),
		"thrust":        generateThrust(),
		"saucerLoop":    generateSaucerLoop(150, 108, 0.5, 0.2),
		"explosion":     generateExplosion(0.25, 0.4, 120, 50),
		"shipExplosion": generateShipExplosion(),
	}
}

func TestGeneratedSoundsUseWholeStereoFrames(t *testing.T) {
	for name, buf := range generatedSounds(t) {
		if len(buf) == 0 {
			t.Errorf("%s: generated an empty buffer", name)
			continue
		}
		if len(buf)%bytesPerFrame != 0 {
			t.Errorf("%s: length %d is not a whole number of %d byte stereo frames", name, len(buf), bytesPerFrame)
		}
	}
}

func TestGeneratedSoundsFillBothChannels(t *testing.T) {
	for name, buf := range generatedSounds(t) {
		if frameCount(buf) == 0 {
			t.Errorf("%s: generated no frames", name)
			continue
		}

		for _, i := range []int{0, frameCount(buf) / 2, frameCount(buf) - 1} {
			left, right := channels(t, buf, i)
			if left != right {
				t.Errorf("%s: frame %d has left %d and right %d, want both channels equal", name, i, left, right)
			}
		}
	}
}

// A mono buffer read as stereo plays at double pitch and half duration, so the
// frame count is what pins each sound to its requested length.
func TestGeneratedSoundsHoldRequestedDuration(t *testing.T) {
	tests := []struct {
		name    string
		seconds float64
		buf     []byte
	}{
		{"tone", 0.25, generateTone(440, 0.25, 0.5, waveSquare, 0)},
		{"sweep", 0.3, generateSweep(900, 400, 0.3, 0.5, waveSquare, 4)},
		{"explosion", 0.2, generateExplosion(0.2, 0.4, 120, 50)},
	}

	for _, tt := range tests {
		want := int(tt.seconds * sampleRate)
		if got := frameCount(tt.buf); got != want {
			t.Errorf("%s: %d frames, want %d (%.2fs at %d Hz)", tt.name, got, want, tt.seconds, sampleRate)
		}
	}
}

func TestGeneratedToneStaysWithinRequestedVolume(t *testing.T) {
	volume := 0.3
	buf := generateTone(440, 0.1, volume, waveSquare, 0)

	limit := int16(volume*math.MaxInt16) + 1
	for i := 0; i < frameCount(buf); i++ {
		left, _ := channels(t, buf, i)
		if left > limit || left < -limit {
			t.Fatalf("frame %d sample %d exceeds requested volume limit %d", i, left, limit)
		}
	}
}

// The thrust and saucer beds play under the game continuously, so their
// buffers are looped. A loop whose length is not a whole number of wave cycles
// jumps in phase at the wrap and clicks once per repeat.

func bufferSeconds(buf []byte) float64 {
	return float64(frameCount(buf)) / sampleRate
}

func wholeCycles(t *testing.T, label string, freq float64, seconds float64) {
	t.Helper()

	cycles := freq * seconds
	if math.Abs(cycles-math.Round(cycles)) > 1e-6 {
		t.Errorf("%s: %.4f Hz spans %.4f cycles in %.4fs, want a whole number", label, freq, cycles, seconds)
	}
}

func TestSnapFreqCompletesWholeCyclesInWindow(t *testing.T) {
	windows := []struct {
		freq   float64
		window float64
	}{
		{150, 0.09},
		{108, 0.09},
		{315, 0.09},
		{235, 0.09},
		{78, 0.5},
		{22, 0.5},
	}

	for _, w := range windows {
		got := snapFreq(w.freq, w.window)
		wholeCycles(t, "snapFreq", got, w.window)
	}
}

func TestSnapFreqStaysNearRequestedPitch(t *testing.T) {
	for _, freq := range []float64{150, 108, 315, 235} {
		got := snapFreq(freq, 0.09)
		if drift := math.Abs(got-freq) / freq; drift > 0.1 {
			t.Errorf("snapFreq(%v, 0.09) = %v, a %.1f%% shift; snapping must not retune the sound", freq, got, drift*100)
		}
	}
}

func TestSnapFreqKeepsAtLeastOneCycle(t *testing.T) {
	// 4 Hz spans 0.36 cycles in the window and would otherwise round to silence.
	got := snapFreq(4, 0.09)
	if cycles := got * 0.09; cycles < 1-1e-9 {
		t.Fatalf("snapFreq(4, 0.09) = %v spanning %v cycles, want at least one", got, cycles)
	}
}

func TestWholePeriodsLandsOnPeriodBoundary(t *testing.T) {
	for _, seconds := range []float64{0.86, 0.66, 0.05} {
		got := wholePeriods(seconds, saucerPulseSeconds)
		periods := got / saucerPulseSeconds
		if math.Abs(periods-math.Round(periods)) > 1e-6 {
			t.Errorf("wholePeriods(%v) = %v spanning %v periods, want a whole number", seconds, got, periods)
		}
		if periods < 1 {
			t.Errorf("wholePeriods(%v) = %v, want at least one period", seconds, got)
		}
	}
}

func TestSaucerLoopSpansWholePulses(t *testing.T) {
	for _, buf := range [][]byte{
		generateSaucerLoop(150, 108, 0.86, 0.16),
		generateSaucerLoop(315, 235, 0.66, 0.14),
	} {
		pulses := bufferSeconds(buf) / saucerPulseSeconds
		if math.Abs(pulses-math.Round(pulses)) > 1e-6 {
			t.Errorf("saucer loop spans %.4f pulses, want a whole number", pulses)
		}
	}
}

func TestThrustLoopSpansWholeCycles(t *testing.T) {
	seconds := bufferSeconds(generateThrust())
	wholeCycles(t, "thrust carrier", snapFreq(thrustCarrierHz, seconds), seconds)
	wholeCycles(t, "thrust flutter", snapFreq(thrustFlutterHz, seconds), seconds)
}
