package main

import "testing"

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
