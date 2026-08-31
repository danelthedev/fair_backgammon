package tests

import (
	"fair_backgammon/game"
	"testing"
)

func TestNewGame(t *testing.T) {
	g := game.NewGame()
	if g.Board[23] != 2 || g.Board[0] != -2 {
		t.Fatalf("bad init %v", g.Board)
	}
}

func TestLegalAndApply(t *testing.T) {
	g := game.NewGame()
	g.Roll()
	g.Dice = [2]int{3, 4}
	g.MovesLeft = []int{3, 4}
	g.Turn = game.White
	m := game.Move{From: 23, To: 20, Die: 3}
	if ok, msg := g.IsLegal(m); !ok {
		t.Fatalf("should be legal: %s", msg)
	}
	if err := g.Apply(m); err != nil {
		t.Fatalf("apply fail %v", err)
	}
	g2 := game.NewGame()
	g2.Board = [24]int{}
	g2.Bar[game.White] = 1
	g2.Board[23] = -2
	g2.Dice = [2]int{1, 2}
	g2.MovesLeft = []int{1, 2}
	g2.HasRolled = true
	g2.Turn = game.White
	bad := game.Move{From: game.BarPos, To: 23, Die: 1}
	if ok, _ := g2.IsLegal(bad); ok {
		t.Fatalf("blocked bar entry should be illegal")
	}
}

func TestHitAndBar(t *testing.T) {
	g := game.NewGame()
	g.Board = [24]int{}
	g.Board[10] = 1
	g.Board[7] = -1
	g.Turn = game.White
	g.HasRolled = true
	g.MovesLeft = []int{3}
	if err := g.Apply(game.Move{From: 10, To: 7, Die: 3}); err != nil {
		t.Fatalf("hit fail %v", err)
	}
	if g.Bar[game.Black] != 1 || g.Board[7] != 1 {
		t.Fatalf("hit not recorded bar %v board %v", g.Bar, g.Board)
	}
}

func TestBarEntryBlocked(t *testing.T) {
	g := game.NewGame()
	g.Board = [24]int{}
	g.Bar[game.White] = 1
	g.Board[23] = -2
	g.Turn = game.White
	g.HasRolled = true
	g.MovesLeft = []int{1}
	if ok, _ := g.IsLegal(game.Move{From: game.BarPos, To: 23, Die: 1}); ok {
		t.Fatal("blocked bar entry should be illegal")
	}
	g.MovesLeft = []int{2}
	g.Board[22] = 0
	if ok, _ := g.IsLegal(game.Move{From: game.BarPos, To: 22, Die: 2}); !ok {
		t.Fatal("open bar entry should be legal")
	}
}

func TestBearOff(t *testing.T) {
	g := game.NewGame()
	g.Board = [24]int{}
	g.Board[0] = 1
	g.Board[1] = 2
	g.Turn = game.White
	g.HasRolled = true
	g.MovesLeft = []int{1, 2}
	if ok, _ := g.IsLegal(game.Move{From: 0, To: game.OffPos, Die: 1}); !ok {
		t.Fatal("exact bear off should be legal")
	}
	g.Board[10] = 1
	if ok, _ := g.IsLegal(game.Move{From: 0, To: game.OffPos, Die: 1}); ok {
		t.Fatal("bear off should be illegal when not all home")
	}
	g.Board[10] = 0
	g.MovesLeft = []int{6}
	if ok, _ := g.IsLegal(game.Move{From: 1, To: game.OffPos, Die: 6}); !ok {
		t.Fatal("overshoot bear off should be legal when no checker behind")
	}
	g.Board[3] = 1
	if ok, _ := g.IsLegal(game.Move{From: 1, To: game.OffPos, Die: 6}); ok {
		t.Fatal("overshoot should be illegal when checker on higher point")
	}
}

func TestDoublesAndTurnSwitch(t *testing.T) {
	g := game.NewGame()
	g.Board = [24]int{}
	g.Board[10] = 2
	g.Turn = game.White
	g.HasRolled = true
	g.MovesLeft = []int{3, 3, 3, 3}
	for i := 0; i < 2; i++ {
		if err := g.Apply(game.Move{From: 10, To: 7, Die: 3}); err != nil {
			t.Fatalf("apply fail %v", err)
		}
	}
	if g.Board[10] != 0 || g.Board[7] != 2 {
		t.Fatalf("doubles not applied %v", g.Board)
	}
}

func TestBlockedPoint(t *testing.T) {
	g := game.NewGame()
	g.Board = [24]int{}
	g.Board[10] = 1
	g.Board[7] = -2
	g.Turn = game.White
	g.HasRolled = true
	g.MovesLeft = []int{3}
	if ok, _ := g.IsLegal(game.Move{From: 10, To: 7, Die: 3}); ok {
		t.Fatal("blocked point should be illegal")
	}
}

func TestNoLegalPass(t *testing.T) {
	g := game.NewGame()
	g.Board = [24]int{}
	g.Board[23] = 1
	g.Board[20] = -2
	g.Board[19] = -2
	g.Turn = game.White
	g.HasRolled = true
	g.MovesLeft = []int{3, 4}
	if g.HasAnyLegal() {
		t.Fatal("should have no legal moves")
	}
}

func legalList(g *game.Game) []game.Move {
	var out []game.Move
	for _, d := range g.MovesLeft {
		for _, from := range append([]int{game.BarPos}, func() []int { a := make([]int, 24); for i := range a { a[i] = i }; return a }()...) {
			for to := -2; to < 24; to++ {
				if to == -1 {
					continue
				}
				m := game.Move{From: from, To: to, Die: d}
				if ok, _ := g.IsLegal(m); ok {
					out = append(out, m)
				}
			}
		}
	}
	return out
}

func checkInvariants(t *testing.T, g *game.Game) {
	t.Helper()
	sum := g.Bar[0] + g.Bar[1] + g.Off[0] + g.Off[1]
	for _, v := range g.Board {
		if v > 0 {
			sum += v
		} else {
			sum -= v
		}
	}
	if sum != 30 {
		t.Fatalf("invariant sum !=30 got %d board %v bar %v off %v", sum, g.Board, g.Bar, g.Off)
	}
	for i, v := range g.Board {
		if v > 15 || v < -15 {
			t.Fatalf("point %d overflow %d", i, v)
		}
	}
	if g.Bar[0] < 0 || g.Bar[1] < 0 || g.Off[0] < 0 || g.Off[1] < 0 {
		t.Fatalf("bar/off negative %v %v", g.Bar, g.Off)
	}
	if g.Off[0] > 15 || g.Off[1] > 15 {
		t.Fatalf("off >15 %v", g.Off)
	}
}

func FuzzLegalMoves(f *testing.F) {
	f.Add([]byte{2, 5, 3, 5}, 3, 4)
	f.Fuzz(func(t *testing.T, boardSeed []byte, d1, d2 int) {
		g := game.NewGame()
		for i := range g.Board {
			if i < len(boardSeed) {
				g.Board[i] = int(int8(boardSeed[i])) % 6
			}
		}
		if d1 < 1 {
			d1 = 1
		}
		if d1 > 6 {
			d1 = 6
		}
		if d2 < 1 {
			d2 = 1
		}
		if d2 > 6 {
			d2 = 6
		}
		g.Dice = [2]int{d1, d2}
		if d1 == d2 {
			g.MovesLeft = []int{d1, d1, d1, d1}
		} else {
			g.MovesLeft = []int{d1, d2}
		}
		g.HasRolled = true
		_ = g.HasAnyLegal()
		for _, d := range g.MovesLeft {
			for _, from := range []int{game.BarPos, 0, 5, 12, 23} {
				for _, to := range []int{game.OffPos, 0, 5, 20} {
					m := game.Move{From: from, To: to, Die: d}
					ok, _ := g.IsLegal(m)
					if ok {
						c := g.Clone()
						if err := c.Apply(m); err != nil {
							t.Fatalf("legal but apply failed %v move %+v", err, m)
						}
					}
				}
			}
		}
	})
}

func FuzzFullGame(f *testing.F) {
	f.Add([]byte{1, 2, 3, 4, 5, 6, 7, 8})
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) == 0 {
			return
		}
		g := game.NewGame()
		checkInvariants(t, g)
		pos := 0
		nextByte := func() byte {
			b := data[pos%len(data)]
			pos++
			return b
		}
		for ply := 0; ply < 500; ply++ {
			if win, _ := g.CheckWin(); win {
				break
			}
			if !g.HasRolled {
				d1 := int(nextByte()%6) + 1
				d2 := int(nextByte()%6) + 1
				g.Dice = [2]int{d1, d2}
				if d1 == d2 {
					g.MovesLeft = []int{d1, d1, d1, d1}
				} else {
					g.MovesLeft = []int{d1, d2}
				}
				g.HasRolled = true
				if !g.HasAnyLegal() {
					g.HasRolled = false
					g.MovesLeft = nil
					g.Turn = 1 - g.Turn
					continue
				}
			}
			moves := legalList(g)
			if len(moves) == 0 {
				g.HasRolled = false
				g.MovesLeft = nil
				g.Turn = 1 - g.Turn
				continue
			}
			m := moves[int(nextByte())%len(moves)]
			if err := g.Apply(m); err != nil {
				t.Fatalf("apply legal failed %v move %+v state %+v", err, m, g)
			}
			checkInvariants(t, g)
			if len(g.MovesLeft) == 0 || !g.HasAnyLegal() {
				g.MovesLeft = nil
				g.HasRolled = false
				g.Turn = 1 - g.Turn
			}
			if win, w := g.CheckWin(); win {
				if g.Off[w] < 15 {
					t.Fatalf("win but off %d", g.Off[w])
				}
				break
			}
		}
	})
}
