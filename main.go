package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	ScreenWidth    = 800
	ScreenHeight   = 600
	ThrustForce    = 0.1
	BulletSpeed    = 4.0
	BulletLifespan = 1.0 // Seconds

	// UFO constants
	UFOSpawnInterval = 10 * time.Second
	BigUFORadius     = 20.0
	SmallUFORadius   = 15.0
	BigUFOSpeed      = 0.5
	SmallUFOSpeed    = 1.0
	BigUFOScore      = 200
	SmallUFOScore    = 1000
)

func NewGame() *Game {
	g := &Game{
		ScreenWidth:  ScreenWidth,
		ScreenHeight: ScreenHeight,
		Lives:        3 + rand.Intn(3), // 3-5 lives
		Level:        1,
		UFOs:         []*UFO{},
		UFOBullets:   []*UFOBullet{},
		LastUFOSpawn: time.Now(),
		NextExtraLifeScore: 10000,
		Particles:    []*Particle{},
	}

	g.Ship = &Ship{
		Position: Vector2D{X: float64(ScreenWidth) / 2, Y: float64(ScreenHeight) / 2},
		Rotation: -math.Pi / 2,
	}

	g.spawnAsteroids(4, 3) // 4 Large asteroids
	return g
}

func (g *Game) spawnAsteroids(count int, size int) {
	for i := 0; i < count; i++ {
		var pos Vector2D
		if rand.Intn(2) == 0 {
			pos.X = rand.Float64() * float64(g.ScreenWidth)
			if rand.Intn(2) == 0 {
				pos.Y = 0
			} else {
				pos.Y = float64(g.ScreenHeight)
			}
		} else {
			pos.Y = rand.Float64() * float64(g.ScreenHeight)
			if rand.Intn(2) == 0 {
				pos.X = 0
			} else {
				pos.X = float64(g.ScreenWidth)
			}
		}

		speed := rand.Float64() + 0.5
		angle := rand.Float64() * 2 * math.Pi
		vel := Vector2D{X: math.Cos(angle) * speed, Y: math.Sin(angle) * speed}

		radius := float64(size * 15)
		points := 8 + rand.Intn(4)
		shape := make([]Vector2D, points)
		for j := 0; j < points; j++ {
			angle := (float64(j) / float64(points)) * 2 * math.Pi
			r := radius * (0.8 + rand.Float64()*0.4)
			shape[j] = Vector2D{
				X: math.Cos(angle) * r,
				Y: math.Sin(angle) * r,
			}
		}

		g.Asteroids = append(g.Asteroids, &Asteroid{
			Position:      pos,
			Velocity:      vel,
			Rotation:      rand.Float64() * 2 * math.Pi,
			RotationSpeed: (rand.Float64() - 0.5) * 0.05,
			Shape:         shape,
			Size:          size,
			Radius:        radius,
		})
	}
}

func (g *Game) splitAsteroid(index int) {
	a := g.Asteroids[index]
	// Remove asteroid
	g.Asteroids = append(g.Asteroids[:index], g.Asteroids[index+1:]...)

	if a.Size > 1 {
		newSize := a.Size - 1
		for i := 0; i < 2; i++ {
			// Diverging velocity
			angle := rand.Float64() * 2 * math.Pi // Random direction for split pieces?
			// Or based on old velocity? Spec says "diverging velocities".
			// Let's just randomizing slightly or rotate old velocity.
			// Simple: random direction, slightly faster.
			speed := math.Hypot(a.Velocity.X, a.Velocity.Y) * 1.5
			vel := Vector2D{X: math.Cos(angle) * speed, Y: math.Sin(angle) * speed}

			radius := float64(newSize * 15)
			points := 8 + rand.Intn(4)
			shape := make([]Vector2D, points)
			for j := 0; j < points; j++ {
				ang := (float64(j) / float64(points)) * 2 * math.Pi
				r := radius * (0.8 + rand.Float64()*0.4)
				shape[j] = Vector2D{
					X: math.Cos(ang) * r,
					Y: math.Sin(ang) * r,
				}
			}

			g.Asteroids = append(g.Asteroids, &Asteroid{
				Position:      a.Position,
				Velocity:      vel,
				Rotation:      rand.Float64() * 2 * math.Pi,
				RotationSpeed: (rand.Float64() - 0.5) * 0.05,
				Shape:         shape,
				Size:          newSize,
				Radius:        radius,
			})
		}
	}
}

func (g *Game) spawnExplosion(pos Vector2D) {
	for i := 0; i < 10; i++ {
		angle := rand.Float64() * 2 * math.Pi
		speed := rand.Float64() * 2.0
		vel := Vector2D{X: math.Cos(angle) * speed, Y: math.Sin(angle) * speed}

		g.Particles = append(g.Particles, &Particle{
			Position:      pos,
			Velocity:      vel,
			Rotation:      rand.Float64() * 2 * math.Pi,
			RotationSpeed: (rand.Float64() - 0.5) * 0.2,
			Lifespan:      1.0 + rand.Float64(),
			Length:        5.0 + rand.Float64()*10.0,
		})
	}
}

func (g *Game) killShip() {
	if g.Ship == nil {
		return
	}
	g.spawnExplosion(g.Ship.Position)
	g.Ship = nil
	g.Lives--
	g.RespawnTimer = 2.0
}

func (g *Game) wrap(pos Vector2D) Vector2D {
	w, h := float64(g.ScreenWidth), float64(g.ScreenHeight)
	if pos.X < 0 {
		pos.X = w
	} else if pos.X > w {
		pos.X = 0
	}
	if pos.Y < 0 {
		pos.Y = h
	} else if pos.Y > h {
		pos.Y = 0
	}
	return pos
}

func (g *Game) Update() error {
	if g.GameState == 1 {
		// Game Over
		if ebiten.IsKeyPressed(ebiten.KeyEnter) {
			// Restart
			*g = *NewGame()
		}
		return nil
	}

	// Update Particles
	for i := len(g.Particles) - 1; i >= 0; i-- {
		p := g.Particles[i]
		p.Position.X += p.Velocity.X
		p.Position.Y += p.Velocity.Y
		p.Rotation += p.RotationSpeed
		p.Lifespan -= 1.0 / 60.0
		if p.Lifespan <= 0 {
			g.Particles = append(g.Particles[:i], g.Particles[i+1:]...)
		}
	}

	// Respawn Timer
	if g.RespawnTimer > 0 {
		g.RespawnTimer -= 1.0 / 60.0
		if g.RespawnTimer <= 0 {
			if g.Lives > 0 {
				// Respawn Ship
				g.Ship = &Ship{
					Position:        Vector2D{X: float64(ScreenWidth) / 2, Y: float64(ScreenHeight) / 2},
					Rotation:        -math.Pi / 2,
					IsInvincible:    true,
					InvincibleTimer: 3.0,
				}
			} else {
				g.GameState = 1 // Game Over
			}
		}
	}

	// 1. Update Ship
	if g.Ship != nil {
		if g.Ship.IsInvincible {
			g.Ship.InvincibleTimer -= 1.0 / 60.0
			if g.Ship.InvincibleTimer <= 0 {
				g.Ship.IsInvincible = false
			}
		}

		if ebiten.IsKeyPressed(ebiten.KeyLeft) {
			g.Ship.Rotation -= 0.05
		}
		if ebiten.IsKeyPressed(ebiten.KeyRight) {
			g.Ship.Rotation += 0.05
		}

		g.Ship.IsThrusting = false
		if ebiten.IsKeyPressed(ebiten.KeyUp) {
			g.Ship.IsThrusting = true
			accelX := math.Cos(g.Ship.Rotation) * ThrustForce
			accelY := math.Sin(g.Ship.Rotation) * ThrustForce
			g.Ship.Velocity.X += accelX
			g.Ship.Velocity.Y += accelY
		}

		g.Ship.Position.X += g.Ship.Velocity.X
		g.Ship.Position.Y += g.Ship.Velocity.Y
		g.Ship.Position = g.wrap(g.Ship.Position)

		if ebiten.IsKeyPressed(ebiten.KeySpace) {
			if time.Since(g.LastShot) > 200*time.Millisecond {
				noseOffset := Vector2D{X: 15, Y: 0}.Rotate(g.Ship.Rotation)
				spawnPos := Vector2D{
					X: g.Ship.Position.X + noseOffset.X,
					Y: g.Ship.Position.Y + noseOffset.Y,
				}
				dir := Vector2D{X: 1, Y: 0}.Rotate(g.Ship.Rotation)
				bulletVel := Vector2D{
					X: g.Ship.Velocity.X + dir.X*BulletSpeed,
					Y: g.Ship.Velocity.Y + dir.Y*BulletSpeed,
				}
				bullet := &Bullet{
					Position: spawnPos,
					Velocity: bulletVel,
					Lifespan: BulletLifespan,
				}
				g.Bullets = append(g.Bullets, bullet)
				g.LastShot = time.Now()
			}
		}
	}

	// 2. Update Bullets and Collision
	for i := len(g.Bullets) - 1; i >= 0; i-- {
		b := g.Bullets[i]
		b.Position.X += b.Velocity.X
		b.Position.Y += b.Velocity.Y
		b.Position = g.wrap(b.Position)

		b.Lifespan -= 1.0 / 60.0

		// Collision with Asteroids
		hit := false
		for j := len(g.Asteroids) - 1; j >= 0; j-- {
			a := g.Asteroids[j]
			dist := math.Hypot(b.Position.X-a.Position.X, b.Position.Y-a.Position.Y)
			if dist < a.Radius {
				g.splitAsteroid(j)
				g.Score += 100
				hit = true
				break
			}
		}

		// Collision with UFOs
		if !hit { // Only check UFOs if not already hit an asteroid
			for j := len(g.UFOs) - 1; j >= 0; j-- {
				ufo := g.UFOs[j]
				dist := math.Hypot(b.Position.X-ufo.Position.X, b.Position.Y-ufo.Position.Y)
				if dist < ufo.Radius {
					g.Score += ufo.ScoreValue // Add UFO score
					g.UFOs = append(g.UFOs[:j], g.UFOs[j+1:]...)
					hit = true
					break
				}
			}
		}

		if hit || b.Lifespan <= 0 {
			g.Bullets = append(g.Bullets[:i], g.Bullets[i+1:]...)
		}
	}

	// 3. Update Asteroids and Ship Collision
	for i := len(g.Asteroids) - 1; i >= 0; i-- {
		a := g.Asteroids[i]
		a.Position.X += a.Velocity.X
		a.Position.Y += a.Velocity.Y
		a.Position = g.wrap(a.Position)
		a.Rotation += a.RotationSpeed

		// Ship Collision
		if g.Ship != nil && !g.Ship.IsInvincible && g.GameState == 0 {
			dist := math.Hypot(g.Ship.Position.X-a.Position.X, g.Ship.Position.Y-a.Position.Y)
			if dist < a.Radius+10 { // Ship radius approx 10
				g.killShip()
			}
		}
	}

	// Grant extra life
	if g.Score >= g.NextExtraLifeScore {
		g.Lives++
		g.NextExtraLifeScore += 10000 // Increment for the next 10,000 points
	}

	// 4. Update UFOs and UFOBullets
	if time.Since(g.LastUFOSpawn) > UFOSpawnInterval && len(g.UFOs) == 0 {
		var ufoSize int
		var ufoRadius float64
		var ufoSpeed float64
		var ufoScore int

		if g.Score >= 40000 {
			ufoSize = 0 // Small UFO
			ufoRadius = SmallUFORadius
			ufoSpeed = SmallUFOSpeed
			ufoScore = SmallUFOScore
		} else {
			if rand.Intn(2) == 0 {
				ufoSize = 1 // Big UFO
				ufoRadius = BigUFORadius
				ufoSpeed = BigUFOSpeed
				ufoScore = BigUFOScore
			} else {
				ufoSize = 0 // Small UFO
				ufoRadius = SmallUFORadius
				ufoSpeed = SmallUFOSpeed
				ufoScore = SmallUFOScore
			}
		}

		// Spawn UFO from left or right edge
		startX := 0.0
		velX := ufoSpeed
		if rand.Intn(2) == 0 {
			startX = float64(g.ScreenWidth)
			velX = -ufoSpeed
		}
		startY := rand.Float64() * float64(g.ScreenHeight)

		g.UFOs = append(g.UFOs, &UFO{
			Position:   Vector2D{X: startX, Y: startY},
			Velocity:   Vector2D{X: velX, Y: 0},
			Radius:     ufoRadius,
			Size:       ufoSize,
			LastShot:   time.Now(),
			ScoreValue: ufoScore,
		})
		g.LastUFOSpawn = time.Now()
	}

	for i := len(g.UFOs) - 1; i >= 0; i-- {
		ufo := g.UFOs[i]
		ufo.Position.X += ufo.Velocity.X
		ufo.Position.Y += ufo.Velocity.Y
		ufo.Position = g.wrap(ufo.Position)

		// Remove UFO if it goes off screen
		if (ufo.Velocity.X > 0 && ufo.Position.X > float64(g.ScreenWidth)+ufo.Radius) ||
			(ufo.Velocity.X < 0 && ufo.Position.X < -ufo.Radius) {
			g.UFOs = append(g.UFOs[:i], g.UFOs[i+1:]...)
			continue
		}

		// Ship Collision with UFO
		if g.Ship != nil && !g.Ship.IsInvincible && g.GameState == 0 {
			dist := math.Hypot(g.Ship.Position.X-ufo.Position.X, g.Ship.Position.Y-ufo.Position.Y)
			if dist < ufo.Radius+10 { // Ship radius approx 10
				g.killShip()
			}
		}

		// UFO Firing
		var fireRate time.Duration
		if ufo.Size == 1 { // Big UFO
			fireRate = 2 * time.Second
		} else { // Small UFO
			fireRate = 1 * time.Second
		}

		if time.Since(ufo.LastShot) > fireRate {
			// Determine bullet direction
			var bulletVel Vector2D
			if ufo.Size == 1 { // Big UFO shoots randomly
				angle := rand.Float64() * 2 * math.Pi
				bulletVel = Vector2D{X: math.Cos(angle) * BulletSpeed, Y: math.Sin(angle) * BulletSpeed}
			} else { // Small UFO shoots at ship
				if g.Ship != nil {
					diff := g.Ship.Position.Subtract(ufo.Position)
					angle := math.Atan2(diff.Y, diff.X)

					// Small saucer accuracy increases with score
					// Angle range reduces from +/- Pi/4 to 0 as score goes from 0 to ~60000
					accuracy := 0.0 // 0 means perfect aim
					if g.Score < 60000 {
						accuracy = math.Pi / 4 * (1 - float64(g.Score)/60000)
					}
					angle += (rand.Float64()*2 - 1) * accuracy // Add random deviation within accuracy range

					bulletVel = Vector2D{X: math.Cos(angle) * BulletSpeed, Y: math.Sin(angle) * BulletSpeed}
				}
			}

			if (bulletVel != Vector2D{}) { // Only fire if a target was set (e.g. ship exists)
				g.UFOBullets = append(g.UFOBullets, &UFOBullet{
					Position: ufo.Position,
					Velocity: bulletVel,
					Lifespan: BulletLifespan,
				})
				ufo.LastShot = time.Now()
			}
		}
	}

	// Update UFOBullets
	for i := len(g.UFOBullets) - 1; i >= 0; i-- {
		b := g.UFOBullets[i]
		b.Position.X += b.Velocity.X
		b.Position.Y += b.Velocity.Y
		b.Position = g.wrap(b.Position)

		b.Lifespan -= 1.0 / 60.0

		// Collision with Ship
		if g.Ship != nil && !g.Ship.IsInvincible && g.GameState == 0 {
			dist := math.Hypot(b.Position.X-g.Ship.Position.X, b.Position.Y-g.Ship.Position.Y)
			if dist < 10 { // Ship radius approx 10
				g.killShip()
				g.UFOBullets = append(g.UFOBullets[:i], g.UFOBullets[i+1:]...) // Remove bullet on hit
				continue
			}
		}

		if b.Lifespan <= 0 {
			g.UFOBullets = append(g.UFOBullets[:i], g.UFOBullets[i+1:]...)
		}
	}


	// 5. Level Progression
	if len(g.Asteroids) == 0 && len(g.UFOs) == 0 {
		g.Level++
		// Clear all UFO bullets from the screen on new level
		g.UFOBullets = []*UFOBullet{}

		numAsteroids := g.Level + 3
		if g.Score >= 40000 && g.Score < 60000 {
			numAsteroids = 7 // Capped at 7 between 40k and 60k
		} else if g.Score >= 60000 {
			numAsteroids = 8 // Capped at 8 after 60k
		}
		g.spawnAsteroids(numAsteroids, 3)
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.GameState == 1 {
		ebitenutil.DebugPrint(screen, "GAME OVER\nPress Enter to Restart")
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

			ebitenutil.DrawLine(screen, x1, y1, x2, y2, c)
			ebitenutil.DrawLine(screen, x2, y2, x3, y3, c)
			ebitenutil.DrawLine(screen, x3, y3, x1, y1, c)

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
				ebitenutil.DrawLine(screen, fx1, fy1, fx2, fy2, c)
				ebitenutil.DrawLine(screen, fx2, fy2, fx3, fy3, c)
				ebitenutil.DrawLine(screen, fx3, fy3, fx1, fy1, c)
			}
		}
	}

	// Draw Bullets
	for _, b := range g.Bullets {
		ebitenutil.DrawRect(screen, b.Position.X, b.Position.Y, 2, 2, c)
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
			ebitenutil.DrawLine(screen, x1, y1, x2, y2, c)
		}
	}

	// UI
	msg := fmt.Sprintf("Score: %d  Lives: %d  Level: %d", g.Score, g.Lives, g.Level)
	ebitenutil.DebugPrint(screen, msg)

	// Draw UFOs
	for _, ufo := range g.UFOs {
		x, y := ufo.Position.X, ufo.Position.Y
		radius := ufo.Radius

		// Body
		ebitenutil.DrawLine(screen, x-radius, y, x+radius, y, c)
		ebitenutil.DrawLine(screen, x-radius*0.7, y-radius*0.5, x+radius*0.7, y-radius*0.5, c)
		ebitenutil.DrawLine(screen, x-radius*0.7, y+radius*0.5, x+radius*0.7, y+radius*0.5, c)
		ebitenutil.DrawLine(screen, x-radius, y, x-radius*0.7, y-radius*0.5, c)
		ebitenutil.DrawLine(screen, x-radius, y, x-radius*0.7, y+radius*0.5, c)
		ebitenutil.DrawLine(screen, x+radius, y, x+radius*0.7, y-radius*0.5, c)
		ebitenutil.DrawLine(screen, x+radius, y, x+radius*0.7, y+radius*0.5, c)

		// Cockpit
		ebitenutil.DrawCircle(screen, x, y-radius*0.5, radius*0.3, c)
	}

	// Draw UFOBullets
	for _, b := range g.UFOBullets {
		ebitenutil.DrawRect(screen, b.Position.X, b.Position.Y, 2, 2, c)
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
		ebitenutil.DrawLine(screen, x1, y1, x2, y2, col)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.ScreenWidth, g.ScreenHeight
}

func main() {
	rand.Seed(time.Now().UnixNano())
	ebiten.SetWindowSize(ScreenWidth, ScreenHeight)
	ebiten.SetWindowTitle("Asteroids Clone")

	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}
