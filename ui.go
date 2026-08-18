package main

import (
	"strings"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	debugGlyphWidth  = 6
	debugGlyphHeight = 16
	uiPadding        = 16
	maxTextCacheSize = 32
)

func scaledDebugTextSize(text string, scale int) (int, int) {
	if text == "" || scale <= 0 {
		return 0, 0
	}

	lines := strings.Split(text, "\n")
	maxRunes := 0
	for _, line := range lines {
		if count := utf8.RuneCountInString(line); count > maxRunes {
			maxRunes = count
		}
	}

	return maxRunes * debugGlyphWidth * scale, len(lines) * debugGlyphHeight * scale
}

func (g *Game) debugTextImage(text string) *ebiten.Image {
	if image := g.textCache[text]; image != nil {
		return image
	}

	width, height := scaledDebugTextSize(text, 1)
	if width == 0 || height == 0 {
		return nil
	}

	image := ebiten.NewImage(width, height)
	// DebugPrintAt offsets glyphs by one pixel internally; -1 keeps the cached
	// image bounds aligned with the measured six-pixel glyph width.
	ebitenutil.DebugPrintAt(image, text, -1, 0)
	if g.textCache == nil {
		g.textCache = make(map[string]*ebiten.Image)
	} else if len(g.textCache) >= maxTextCacheSize {
		clear(g.textCache)
	}
	g.textCache[text] = image
	return image
}

func (g *Game) drawScaledDebugText(screen *ebiten.Image, text string, x, y, scale int) {
	image := g.debugTextImage(text)
	if image == nil {
		return
	}

	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterNearest
	op.GeoM.Scale(float64(scale), float64(scale))
	op.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(image, op)
}

func (g *Game) drawCenteredDebugText(screen *ebiten.Image, text string, y, scale int) {
	width, _ := scaledDebugTextSize(text, scale)
	g.drawScaledDebugText(screen, text, (g.ScreenWidth-width)/2, y, scale)
}
