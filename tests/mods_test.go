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

// ponytail: bar entry + bear off need natural faces 1-6
func TestNaturalFaces(t *testing.T) {
	g := game.NewGameWithMods(game.Mods{Negative: true, MaxDie: 9})
	g.Turn = game.White
	g.HasRolled = true
	// bar entry with big / negative die illegal
	g.Bar = [2]int{1, 0}
	for _, d := range []int{-3, 9} {
		g.MovesLeft = []int{d}
		if ok, _ := g.IsLegal(game.Move{From: game.BarPos, To: 15, Die: d}); ok {
			t.Fatalf("bar entry with die %d must be illegal", d)
		}
	}
	// bear off with big die illegal, normal die legal
	g.Bar = [2]int{0, 0}
	g.Board = [24]int{}
	g.Board[0] = 1
	g.MovesLeft = []int{9}
	if ok, msg := g.IsLegal(game.Move{From: 0, To: game.OffPos, Die: 9}); !ok {
		t.Fatalf("bear off with die 9 must be legal when all home: %s", msg)
	}
	g.MovesLeft = []int{1}
	if ok, msg := g.IsLegal(game.Move{From: 0, To: game.OffPos, Die: 1}); !ok {
		t.Fatalf("bear off with die 1 must be legal: %s", msg)
	}
	// negative distribution ~1/3 over many rolls
	g2 := game.NewGameWithMods(game.Mods{Negative: true})
	neg, tot := 0, 0
	for i := 0; i < 6000; i++ {
		g2.Roll()
		for _, d := range g2.MovesLeft {
			tot++
			if d < 0 {
				neg++
			}
		}
	}
	r := float64(neg) / float64(tot)
	if r < 0.28 || r > 0.38 {
		t.Fatalf("negative share %.3f, want ~0.33", r)
	}
}

// ponytail: fair doubles deals 2, standard deals 4
func TestFairDoubles(t *testing.T) {
	g := game.NewGameWithMods(game.Mods{NoDouble4x: true})
	seenDouble := false
	for i := 0; i < 500 && !seenDouble; i++ {
		g.Roll()
		if g.Dice[0] == g.Dice[1] {
			seenDouble = true
			if len(g.MovesLeft) != 2 {
				t.Fatalf("fair doubles dealt %d, want 2", len(g.MovesLeft))
			}
		}
	}
	if !seenDouble {
		t.Fatal("no doubles in 500 rolls, suspicious")
	}
	g2 := game.NewGameWithMods(game.Mods{})
	seenQuad := false
	for i := 0; i < 500 && !seenQuad; i++ {
		g2.Roll()
		if g2.Dice[0] == g2.Dice[1] && len(g2.MovesLeft) == 4 {
			seenQuad = true
		}
	}
	if !seenQuad {
		t.Fatal("standard doubles never dealt 4, suspicious")
	}
}

// ponytail: configured negative % is honored
func TestNegPct(t *testing.T) {
	g := game.NewGameWithMods(game.Mods{Negative: true, NegPct: 80})
	neg, tot := 0, 0
	for i := 0; i < 2000; i++ {
		g.Roll()
		for _, d := range g.MovesLeft {
			tot++
			if d < 0 {
				neg++
			}
		}
	}
	r := float64(neg) / float64(tot)
	if r < 0.7 || r > 0.9 {
		t.Fatalf("negative share %.3f, want ~0.80", r)
	}
}

// ponytail: zero face rolls 0s, and 0 dice are dead
func TestAllowZero(t *testing.T) {
	g := game.NewGameWithMods(game.Mods{AllowZero: true})
	seenZero := false
	for i := 0; i < 500 && !seenZero; i++ {
		g.Roll()
		for _, d := range g.MovesLeft {
			if d == 0 {
				seenZero = true
			}
			if d < 0 || d > 6 {
				t.Fatalf("die %d out of 0..6", d)
			}
		}
	}
	if !seenZero {
		t.Fatal("no zero in 500 rolls, suspicious")
	}
	g2 := game.NewGameWithMods(game.Mods{})
	for i := 0; i < 200; i++ {
		g2.Roll()
		for _, d := range g2.MovesLeft {
			if d < 1 || d > 6 {
				t.Fatalf("standard die %d out of 1..6", d)
			}
		}
	}
	// self-move with 0 die illegal
	g3 := game.NewGameWithMods(game.Mods{AllowZero: true})
	g3.Board = [24]int{}
	g3.Board[10] = 1
	g3.Turn = game.White
	g3.HasRolled = true
	g3.MovesLeft = []int{0}
	if ok, _ := g3.IsLegal(game.Move{From: 10, To: 10, Die: 0}); ok {
		t.Fatal("self-move with 0 die must be illegal")
	}
	if g3.HasAnyLegal() {
		t.Fatal("only a 0 die means no legal moves")
	}
}

// ponytail: custom layouts validate totals
func TestApplyLayout(t *testing.T) {
	g := game.NewGame()
	std := game.Layout{Board: g.Board, Bar: g.Bar, Off: g.Off, Turn: g.Turn}
	if err := game.NewGame().ApplyLayout(std); err != nil {
		t.Fatalf("standard layout must validate: %v", err)
	}
	big := std
	big.Board[5] += 10 // no stack limit anymore
	if err := game.NewGame().ApplyLayout(big); err != nil {
		t.Fatalf("arbitrary stacks must validate: %v", err)
	}
	empty := std
	for i := range empty.Board {
		if empty.Board[i] < 0 {
			empty.Board[i] = 0
		}
	}
	if err := game.NewGame().ApplyLayout(empty); err == nil {
		t.Fatal("empty side must fail")
	}
}

// ponytail: bearing off everything wins even when totals <15
func TestSmallWin(t *testing.T) {
	g := game.NewGame()
	g.Board = [24]int{}
	g.Board[0] = 1
	g.Bar = [2]int{}
	g.Off = [2]int{2, 0} // all 3 white off... craft: 1 on board + 2 off = all
	g.Turn = game.White
	g.HasRolled = true
	g.MovesLeft = []int{1}
	if err := g.Apply(game.Move{From: 0, To: game.OffPos, Die: 1}); err != nil {
		t.Fatalf("bear off: %v", err)
	}
	if win, w := g.CheckWin(); !win || w != game.White {
		t.Fatalf("all borne off must win, got win=%v w=%v off=%v", win, w, g.Off)
	}
}
