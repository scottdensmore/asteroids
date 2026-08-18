package game

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func (g *Game) Draw(screen *ebiten.Image) {
	if g.GameState == gameStateTitle {
		g.drawCenteredDebugText(screen, "ASTEROIDS", 150, 3)
		g.drawCenteredDebugText(screen, "PRESS ENTER TO START", 250, 2)
		g.drawCenteredDebugText(screen, "LEFT/RIGHT ROTATE  UP THRUST  SPACE FIRE", 330, 2)
		g.drawCenteredDebugText(screen, "VERSION "+displayVersion(), 400, 2)
		return
	}

	if g.GameState == gameStateGameOver {
		g.drawCenteredDebugText(screen, "GAME OVER", 180, 3)
		g.drawCenteredDebugText(screen, "PRESS ENTER FOR NEW GAME", 280, 2)
		g.drawCenteredDebugText(screen, buildInfoMultiline(), 350, 2)
		return
	}

	c := color.White

	// Draw Ship
	if g.Ship != nil {
		// Blink if invincible
		if !g.Ship.IsInvincible || int(g.Ship.InvincibleTimer*10)%2 == 0 {
			x, y := g.Ship.Position.X, g.Ship.Position.Y
			v1 := Vector2D{X: 15, Y: 0}
			v2 := Vector2D{X: -10, Y: -10}
			v3 := Vector2D{X: -10, Y: 10}

			rv1 := v1.Rotate(g.Ship.Rotation)
			rv2 := v2.Rotate(g.Ship.Rotation)
			rv3 := v3.Rotate(g.Ship.Rotation)

			x1, y1 := x+rv1.X, y+rv1.Y
			x2, y2 := x+rv2.X, y+rv2.Y
			x3, y3 := x+rv3.X, y+rv3.Y

			drawLine(screen, x1, y1, x2, y2, c)
			drawLine(screen, x2, y2, x3, y3, c)
			drawLine(screen, x3, y3, x1, y1, c)

			if g.Ship.IsThrusting {
				f1 := Vector2D{X: -10, Y: -5}
				f2 := Vector2D{X: -10, Y: 5}
				f3 := Vector2D{X: -20, Y: 0}
				rf1 := f1.Rotate(g.Ship.Rotation)
				rf2 := f2.Rotate(g.Ship.Rotation)
				rf3 := f3.Rotate(g.Ship.Rotation)
				fx1, fy1 := x+rf1.X, y+rf1.Y
				fx2, fy2 := x+rf2.X, y+rf2.Y
				fx3, fy3 := x+rf3.X, y+rf3.Y
				drawLine(screen, fx1, fy1, fx2, fy2, c)
				drawLine(screen, fx2, fy2, fx3, fy3, c)
				drawLine(screen, fx3, fy3, fx1, fy1, c)
			}
		}
	}

	// Draw Bullets
	for _, b := range g.Bullets {
		drawRect(screen, b.Position.X, b.Position.Y, 2, 2, c)
	}

	// Draw Asteroids
	for _, a := range g.Asteroids {
		for i := 0; i < len(a.Shape); i++ {
			p1 := a.Shape[i]
			p2 := a.Shape[(i+1)%len(a.Shape)]
			rp1 := p1.Rotate(a.Rotation)
			rp2 := p2.Rotate(a.Rotation)
			x1, y1 := a.Position.X+rp1.X, a.Position.Y+rp1.Y
			x2, y2 := a.Position.X+rp2.X, a.Position.Y+rp2.Y
			drawLine(screen, x1, y1, x2, y2, c)
		}
	}

	// UI
	msg := fmt.Sprintf("Score: %d  Lives: %d  Level: %d", g.Score, g.Lives, g.Level)
	g.drawScaledDebugText(screen, msg, uiPadding, uiPadding, 2)

	// Draw UFOs
	for _, ufo := range g.UFOs {
		x, y := ufo.Position.X, ufo.Position.Y
		radius := ufo.Radius

		// Body
		drawLine(screen, x-radius, y, x+radius, y, c)
		drawLine(screen, x-radius*0.7, y-radius*0.5, x+radius*0.7, y-radius*0.5, c)
		drawLine(screen, x-radius*0.7, y+radius*0.5, x+radius*0.7, y+radius*0.5, c)
		drawLine(screen, x-radius, y, x-radius*0.7, y-radius*0.5, c)
		drawLine(screen, x-radius, y, x-radius*0.7, y+radius*0.5, c)
		drawLine(screen, x+radius, y, x+radius*0.7, y-radius*0.5, c)
		drawLine(screen, x+radius, y, x+radius*0.7, y+radius*0.5, c)

		// Cockpit
		drawCircle(screen, x, y-radius*0.5, radius*0.3, c)
	}

	// Draw UFOBullets
	for _, b := range g.UFOBullets {
		drawRect(screen, b.Position.X, b.Position.Y, 2, 2, c)
	}

	// Draw Particles
	for _, p := range g.Particles {
		x1 := p.Position.X - math.Cos(p.Rotation)*p.Length/2
		y1 := p.Position.Y - math.Sin(p.Rotation)*p.Length/2
		x2 := p.Position.X + math.Cos(p.Rotation)*p.Length/2
		y2 := p.Position.Y + math.Sin(p.Rotation)*p.Length/2

		alpha := uint8(255)
		if p.Lifespan < 1.0 {
			alpha = uint8(255 * p.Lifespan)
		}
		col := color.RGBA{255, 255, 255, alpha}
		drawLine(screen, x1, y1, x2, y2, col)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.ScreenWidth, g.ScreenHeight
}

func drawLine(dst *ebiten.Image, x1, y1, x2, y2 float64, clr color.Color) {
	vector.StrokeLine(dst, float32(x1), float32(y1), float32(x2), float32(y2), 1, clr, false)
}

func drawRect(dst *ebiten.Image, x, y, width, height float64, clr color.Color) {
	vector.FillRect(dst, float32(x), float32(y), float32(width), float32(height), clr, false)
}

func drawCircle(dst *ebiten.Image, x, y, radius float64, clr color.Color) {
	vector.FillCircle(dst, float32(x), float32(y), float32(radius), clr, false)
}
