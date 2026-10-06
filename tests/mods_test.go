package tests

import (
	"testing"

	"fair_backgammon/game"
)

// ponytail: one check for mods — negative die moves backwards, never bears off
func TestModsNegativeBackwards(t *testing.T) {
	g := game.NewGameWithMods(game.Mods{Negative: true})
	g.Board = [24]int{}
	g.Board[10] = 1 // white at 10
	g.Turn = game.White
	g.HasRolled = true
	g.MovesLeft = []int{-3}
	if ok, msg := g.IsLegal(game.Move{From: 10, To: 13, Die: -3}); !ok {
		t.Fatalf("backward move illegal: %s", msg)
	}
	if ok, _ := g.IsLegal(game.Move{From: 10, To: 7, Die: -3}); ok {
		t.Fatal("forward move with negative die should be illegal")
	}
	g.Board = [24]int{}
	g.Board[2] = 1
	g.MovesLeft = []int{-3}
	if ok, _ := g.IsLegal(game.Move{From: 2, To: game.OffPos, Die: -3}); ok {
		t.Fatal("negative die must not bear off")
	}
	if m := g.MaxDie(); m != 6 {
		t.Fatalf("default maxdie=%d, want 6", m)
	}
	g2 := game.NewGameWithMods(game.Mods{MaxDie: 9})
	if m := g2.MaxDie(); m != 9 {
		t.Fatalf("maxdie=%d, want 9", m)
	}
	for i := 0; i < 200; i++ {
		g2.Roll()
		for _, d := range g2.MovesLeft {
			if d < 1 || d > 9 {
				t.Fatalf("die %d out of 1..9", d)
			}
		}
	}
}
