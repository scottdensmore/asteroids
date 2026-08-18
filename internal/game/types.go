package game

import (
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// GameState identifies the screen and update mode currently in use.
type GameState uint8

const (
	gameStatePlaying GameState = iota
	gameStateGameOver
	gameStateTitle
)

// Vector2D represents a position, direction, or velocity in game space.
type Vector2D struct {
	X, Y float64
}

// Rotate returns v rotated counterclockwise around the origin by angle radians.
func (v Vector2D) Rotate(angle float64) Vector2D {
	c := math.Cos(angle)
	s := math.Sin(angle)
	return Vector2D{
		X: v.X*c - v.Y*s,
		Y: v.X*s + v.Y*c,
	}
}

// Subtract returns the vector from other to v.
func (v Vector2D) Subtract(other Vector2D) Vector2D {
	return Vector2D{X: v.X - other.X, Y: v.Y - other.Y}
}

// Ship is the player-controlled craft.
type Ship struct {
	Position        Vector2D
	Velocity        Vector2D
	Rotation        float64 // In radians
	IsThrusting     bool
	IsInvincible    bool
	InvincibleTimer float64
}

// Asteroid is a rotating, destructible obstacle.
type Asteroid struct {
	Position      Vector2D
	Velocity      Vector2D
	Rotation      float64 // In radians
	RotationSpeed float64
	Shape         []Vector2D // Vertices relative to position (e.g., a polygon)
	Size          int        // e.g., 3 = Large, 2 = Medium, 1 = Small
	Radius        float64    // For simple collision detection
}

// Bullet is a projectile fired by the ship.
type Bullet struct {
	Position Vector2D
	Velocity Vector2D
	Lifespan float64 // Time in seconds before it disappears
}

// Game owns the complete mutable state for one game session.
type Game struct {
	Ship               *Ship
	Asteroids          []*Asteroid
	Bullets            []*Bullet
	UFOs               []*UFO
	UFOBullets         []*UFOBullet
	Score              int
	Lives              int
	Level              int
	ScreenWidth        int
	ScreenHeight       int
	GameState          GameState
	LastShot           time.Time
	LastUFOSpawn       time.Time
	NextExtraLifeScore int
	Particles          []*Particle
	RespawnTimer       float64
	Sound              *SoundManager
	textCache          map[string]*ebiten.Image
}

// Particle is one fading line segment in a ship explosion.
type Particle struct {
	Position      Vector2D
	Velocity      Vector2D
	Rotation      float64
	RotationSpeed float64
	Lifespan      float64
	Length        float64
}

// UFO is an enemy saucer crossing the playfield.
type UFO struct {
	Position   Vector2D
	Velocity   Vector2D
	Radius     float64
	Size       int // 0 = Small, 1 = Big
	LastShot   time.Time
	ScoreValue int
}

// UFOBullet is a projectile fired by a saucer.
type UFOBullet struct {
	Position Vector2D
	Velocity Vector2D
	Lifespan float64 // Time in seconds before it disappears
}
