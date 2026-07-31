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
		{"saucerLoop", 0.5, generateSaucerLoop(150, 108, 0.5, 0.2)},
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
