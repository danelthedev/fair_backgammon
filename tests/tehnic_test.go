package tests

import (
	"fair_backgammon/lobby"
	"testing"
)

func TestTehnicDoublesWhite(t *testing.T) {
	h := lobby.NewHub()
	r := h.Create("alice")
	h.Join(r.Code, "bob")
	r.Game.Board = [24]int{2, 2, 2, 2, 2, 1, 1}
	r.Game.Bar = [2]int{}
	r.Game.Off = [2]int{3, 0}
	r.Game.Turn = 0
	r.Game.HasRolled = true
	r.Game.MovesLeft = []int{1}
	if ok, _ := r.Game.CheckTechnicalWin(0); ok {
		t.Fatal("pattern not complete before final move")
	}
	mc := &mockConn{}
	from, to, die := 6, 5, 1
	r.GameTurn(mc, nil, "", 0, turnMsg{T: "move", From: &from, To: &to, Die: &die})
	if r.Scores[0] != 2 || r.Game.Off[0] != 15 {
		t.Fatalf("tehnic win bad scores=%v off=%v", r.Scores, r.Game.Off)
	}
}

func TestTehnicBlackMirror(t *testing.T) {
	h := lobby.NewHub()
	r := h.Create("alice")
	h.Join(r.Code, "bob")
	r.Game.Board = [24]int{}
	for i := 19; i <= 23; i++ {
		r.Game.Board[i] = -2
	}
	r.Game.Board[18] = -1
	r.Game.Board[17] = -1
	r.Game.Bar = [2]int{}
	r.Game.Off = [2]int{0, 3}
	r.Game.Turn = 1
	r.Game.HasRolled = true
	r.Game.MovesLeft = []int{1}
	mc := &mockConn{}
	from, to, die := 17, 18, 1
	r.GameTurn(mc, nil, "", 1, turnMsg{T: "move", From: &from, To: &to, Die: &die})
	if r.Scores[1] != 2 || r.Game.Off[1] != 15 {
		t.Fatalf("tehnic win bad scores=%v off=%v", r.Scores, r.Game.Off)
	}
}

func TestTehnicSinglesAndCubeAndBar(t *testing.T) {
	h := lobby.NewHub()
	r := h.Create("alice")
	h.Join(r.Code, "bob")
	r.Game.Board = [24]int{1, 1, 1, 1, 1, 0, 1}
	r.Game.Bar = [2]int{}
	r.Game.Off = [2]int{9, 0}
	r.Game.Turn = 0
	r.Game.HasRolled = true
	r.Game.MovesLeft = []int{1}
	r.Cube = 4
	mc := &mockConn{}
	from, to, die := 6, 5, 1
	r.GameTurn(mc, nil, "", 0, turnMsg{T: "move", From: &from, To: &to, Die: &die})
	if r.Scores[0] != 8 {
		t.Fatalf("tehnic with cube 4 should pay 8, got %v", r.Scores)
	}

	r.Game.Bar = [2]int{1, 0}
	if ok, _ := r.Game.CheckTechnicalWin(0); ok {
		t.Fatal("bar checker should block tehnic")
	}
	r.Game.Bar = [2]int{}
	r.Game.Off = [2]int{7, 0}
	r.Game.Board[0] = 3
	r.Game.Board[1] = 1
	if ok, _ := r.Game.CheckTechnicalWin(0); ok {
		t.Fatal("uneven columns should not trigger")
	}
}
func TestTehnicNotMidTurn(t *testing.T) {
	h := lobby.NewHub()
	r := h.Create("alice")
	h.Join(r.Code, "bob")
	// 3 on point 6, 2 elsewhere, 2 already off; roll 6-5 bears off 6 then 5
	r.Game.Board = [24]int{2, 2, 2, 2, 2, 3}
	r.Game.Bar = [2]int{}
	r.Game.Off = [2]int{2, 0}
	r.Game.Turn = 0
	r.Game.HasRolled = true
	r.Game.MovesLeft = []int{6, 5}
	mc := &mockConn{}
	from, to, die := 5, -2, 6
	r.GameTurn(mc, nil, "", 0, turnMsg{T: "move", From: &from, To: &to, Die: &die})
	if r.Scores[0] != 0 || r.Game.Off[0] != 3 {
		t.Fatalf("mid-turn tehnic must not fire: scores=%v off=%v", r.Scores, r.Game.Off)
	}
	from, to, die = 4, -2, 5
	r.GameTurn(mc, nil, "", 0, turnMsg{T: "move", From: &from, To: &to, Die: &die})
	if r.Scores[0] != 0 || r.Game.Off[0] != 4 {
		t.Fatalf("broken pattern must not win: scores=%v off=%v", r.Scores, r.Game.Off)
	}
	if r.Game.Turn != 1 {
		t.Fatalf("turn should pass, got %v", r.Game.Turn)
	}
}
