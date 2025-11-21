package main

import (
	"math"
	"time"
)

// A simple 2D vector for position and velocity
type Vector2D struct {
	X, Y float64
}

func (v Vector2D) Rotate(angle float64) Vector2D {
	c := math.Cos(angle)
	s := math.Sin(angle)
	return Vector2D{
		X: v.X*c - v.Y*s,
		Y: v.X*s + v.Y*c,
	}
}

func (v Vector2D) Subtract(other Vector2D) Vector2D {
	return Vector2D{X: v.X - other.X, Y: v.Y - other.Y}
}

// Player's ship
type Ship struct {
	Position        Vector2D
	Velocity        Vector2D
	Rotation        float64 // In radians
	IsThrusting     bool
	IsInvincible    bool
	InvincibleTimer float64
}

// An asteroid
type Asteroid struct {
	Position      Vector2D
	Velocity      Vector2D
	Rotation      float64 // In radians
	RotationSpeed float64
	Shape         []Vector2D // Vertices relative to position (e.g., a polygon)
	Size          int        // e.g., 3 = Large, 2 = Medium, 1 = Small
	Radius        float64    // For simple collision detection
}

// A bullet fired by the ship
type Bullet struct {
	Position Vector2D
	Velocity Vector2D
	Lifespan float64 // Time in seconds before it disappears
}

// Main game state struct
type Game struct {
	Ship         *Ship
	Asteroids    []*Asteroid
	Bullets      []*Bullet
	UFOs         []*UFO
	UFOBullets   []*UFOBullet
	Score        int
	Lives        int
	Level        int
	ScreenWidth  int
	ScreenHeight int
	GameState    int // e.g., 0=Playing, 1=GameOver, 2=TitleScreen
	LastShot     time.Time
	LastUFOSpawn time.Time
	NextExtraLifeScore int
}

// A UFO (flying saucer)
type UFO struct {
	Position   Vector2D
	Velocity   Vector2D
	Radius     float64
	Size       int // 0 = Small, 1 = Big
	LastShot   time.Time
	ScoreValue int
}

// A bullet fired by a UFO
type UFOBullet struct {
	Position Vector2D
	Velocity Vector2D
	Lifespan float64 // Time in seconds before it disappears
}

