// Command asteroids starts the desktop game.
package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/scottdensmore/asteroids/internal/game"
)

func run() error {
	g := game.New()
	defer g.Close()

	return ebiten.RunGame(g)
}

func main() {
	ebiten.SetWindowSize(game.ScreenWidth, game.ScreenHeight)
	ebiten.SetWindowTitle("Asteroids Clone " + game.Version())

	if err := run(); err != nil {
		log.Fatal(err)
	}
}
