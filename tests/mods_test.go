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

// ponytail: one check for power-ups — reroll/skip/protect + shield expiry
func TestPowerUps(t *testing.T) {
	cfg := game.PowerConfig{Reroll: 1, Skip: 1, Protect: 1}
	g := game.NewGameWithMods(game.Mods{Powers: cfg})
	if g.PowerLeft[0] != cfg || g.PowerLeft[1] != cfg {
		t.Fatalf("banks not init: %+v", g.PowerLeft)
	}
	// protect: white blot blocks black
	g.Board = [24]int{}
	g.Board[10] = 1
	g.Turn = game.White
	if err := g.UseProtect(game.White); err != nil {
		t.Fatalf("protect: %v", err)
	}
	g.Turn = game.Black
	g.Board[7] = -1
	g.HasRolled = true
	g.MovesLeft = []int{3}
	if ok, _ := g.IsLegal(game.Move{From: 7, To: 10, Die: 3}); ok {
		t.Fatal("protected blot must be blocked")
	}
	// shield expires when turn returns to white
	g.SetTurn(game.White)
	if g.Shield[game.White] {
		t.Fatal("shield must expire on owner's next turn")
	}
	// reroll: once per roll, before spending
	g.Turn = game.White
	g.HasRolled = true
	g.Dice = [2]int{1, 2}
	g.MovesLeft = []int{1, 2}
	g.Dealt = 2
	if err := g.UseReroll(game.White); err != nil {
		t.Fatalf("reroll: %v", err)
	}
	if !g.Rerolled || g.PowerLeft[game.White].Reroll != 0 {
		t.Fatal("reroll must flag + consume")
	}
	if err := g.UseReroll(game.White); err == nil {
		t.Fatal("second reroll same roll must fail")
	}
	// skip passes turn even unrolled
	g2 := game.NewGameWithMods(game.Mods{Powers: cfg})
	g2.Turn = game.Black
	if err := g2.UseSkip(game.Black); err != nil {
		t.Fatalf("skip: %v", err)
	}
	if g2.Turn != game.White || g2.PowerLeft[game.Black].Skip != 0 {
		t.Fatal("skip must pass turn + consume")
	}
}
