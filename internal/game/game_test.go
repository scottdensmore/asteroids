package game

import "testing"

func TestUFOHasExitedScreen(t *testing.T) {
	tests := []struct {
		name string
		ufo  UFO
		want bool
	}{
		{
			name: "moving right remains until fully clear",
			ufo: UFO{
				Position: Vector2D{X: ScreenWidth + SmallUFORadius},
				Velocity: Vector2D{X: SmallUFOSpeed},
				Radius:   SmallUFORadius,
			},
			want: false,
		},
		{
			name: "moving right exits beyond far edge",
			ufo: UFO{
				Position: Vector2D{X: ScreenWidth + SmallUFORadius + 1},
				Velocity: Vector2D{X: SmallUFOSpeed},
				Radius:   SmallUFORadius,
			},
			want: true,
		},
		{
			name: "moving left remains until fully clear",
			ufo: UFO{
				Position: Vector2D{X: -BigUFORadius},
				Velocity: Vector2D{X: -BigUFOSpeed},
				Radius:   BigUFORadius,
			},
			want: false,
		},
		{
			name: "moving left exits beyond near edge",
			ufo: UFO{
				Position: Vector2D{X: -BigUFORadius - 1},
				Velocity: Vector2D{X: -BigUFOSpeed},
				Radius:   BigUFORadius,
			},
			want: true,
		},
		{
			name: "stationary object does not exit",
			ufo: UFO{
				Position: Vector2D{X: ScreenWidth + SmallUFORadius + 1},
				Radius:   SmallUFORadius,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ufoHasExitedScreen(&tt.ufo, ScreenWidth); got != tt.want {
				t.Fatalf("ufoHasExitedScreen() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestGrantExtraLivesAwardsOnceForOneThreshold(t *testing.T) {
	g := &Game{Lives: 3, Score: ExtraLifeInterval, NextExtraLifeScore: ExtraLifeInterval}

	if got := g.grantExtraLives(); got != 1 {
		t.Fatalf("grantExtraLives() = %d, want 1", got)
	}
	if g.Lives != 4 {
		t.Errorf("Lives = %d, want 4", g.Lives)
	}
	if want := 2 * ExtraLifeInterval; g.NextExtraLifeScore != want {
		t.Errorf("NextExtraLifeScore = %d, want %d", g.NextExtraLifeScore, want)
	}
}

// Destroying a small saucer is worth 1000, and a score can cross more than one
// threshold before the next check, so every crossed threshold must pay out.
func TestGrantExtraLivesAwardsEveryCrossedThreshold(t *testing.T) {
	g := &Game{Lives: 3, Score: 25000, NextExtraLifeScore: ExtraLifeInterval}

	if got := g.grantExtraLives(); got != 2 {
		t.Fatalf("grantExtraLives() = %d, want 2", got)
	}
	if g.Lives != 5 {
		t.Errorf("Lives = %d, want 5", g.Lives)
	}
	if want := 3 * ExtraLifeInterval; g.NextExtraLifeScore != want {
		t.Errorf("NextExtraLifeScore = %d, want %d", g.NextExtraLifeScore, want)
	}
}

func TestGrantExtraLivesBelowThresholdAwardsNothing(t *testing.T) {
	g := &Game{Lives: 3, Score: ExtraLifeInterval - 1, NextExtraLifeScore: ExtraLifeInterval}

	if got := g.grantExtraLives(); got != 0 {
		t.Fatalf("grantExtraLives() = %d, want 0", got)
	}
	if g.Lives != 3 {
		t.Errorf("Lives = %d, want 3 to be unchanged", g.Lives)
	}
	if g.NextExtraLifeScore != ExtraLifeInterval {
		t.Errorf("NextExtraLifeScore = %d, want %d to be unchanged", g.NextExtraLifeScore, ExtraLifeInterval)
	}
}
