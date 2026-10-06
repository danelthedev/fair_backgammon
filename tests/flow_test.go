package tests

import (
	"encoding/json"
	"testing"

	"fair_backgammon/game"
	"fair_backgammon/lobby"
)

func collect(ch chan []byte) []map[string]any {
	var out []map[string]any
	for {
		select {
		case b := <-ch:
			var m map[string]any
			json.Unmarshal(b, &m)
			out = append(out, m)
		default:
			return out
		}
	}
}

func TestCustomEndToEnd(t *testing.T) {
	h := lobby.NewHub()
	r := h.Create("alice")
	h.Join(r.Code, "bob")
	lay := game.Layout{Board: [24]int{}, Turn: 0}
	lay.Board[0] = 2
	lay.Board[23] = -2
	if err := r.Game.ApplyLayout(lay); err != nil {
		t.Fatal(err)
	}
	r.Layout = &lay
	mc := &mockConn{}
	ach := make(chan []byte, 64)
	r.AddSub(ach, "alice")
	// white bears off both: needs all-in-home; board[0]=2 white home. roll 1s manually
	r.Game.Turn = 0
	r.Game.HasRolled = true
	r.Game.MovesLeft = []int{1, 1}
	r.Game.Dealt = 2
	from, to, die := 0, -2, 1
	r.GameTurn(mc, nil, "", 0, turnMsg{T: "move", From: &from, To: &to, Die: &die}, ach)
	r.GameTurn(mc, nil, "", 0, turnMsg{T: "move", From: &from, To: &to, Die: &die}, ach)
	msgs := collect(ach)
	var sawWin, sawErr bool
	for _, m := range msgs {
		if m["t"] == "win" {
			sawWin = true
			t.Logf("win: winner=%v scores=%v reason=%v", m["winner"], m["scores"], m["reason"])
		}
		if m["t"] == "error" {
			sawErr = true
			t.Logf("ERROR: %v", m["msg"])
		}
	}
	if !sawWin {
		t.Fatal("no win broadcast")
	}
	if sawErr {
		t.Fatal("unexpected error during winning moves")
	}
	if win, _ := r.Game.CheckWin(); !win {
		t.Fatal("CheckWin false after bearing all off")
	}
	// rematch both
	r.GameTurn(mc, nil, "", 0, turnMsg{T: "rematch"}, ach)
	r.GameTurn(mc, nil, "", 1, turnMsg{T: "rematch"}, ach)
	if r.Game.Off != ([2]int{}) {
		t.Fatalf("fresh game off=%v", r.Game.Off)
	}
	if r.Game.Board != lay.Board {
		t.Fatal("layout not re-applied on rematch")
	}
	for _, m := range collect(ach) {
		if m["t"] == "error" {
			t.Fatalf("rematch ERROR: %v", m["msg"])
		}
	}
}

// ponytail: hotseat seats both sides, moves play for the turn seat
func TestHotseatFlow(t *testing.T) {
	h := lobby.NewHub()
	r := h.CreateHotseat("solo")
	if r.Players != ([2]string{"solo", "solo"}) || !r.Hotseat {
		t.Fatalf("hotseat seats %+v hotseat=%v", r.Players, r.Hotseat)
	}
	if _, _, ok := h.Join(r.Code, "intruder"); ok {
		t.Fatal("hotseat room must be full")
	}
	// white opens 3-1... craft: white blot at 10, roll-less move via GameTurn
	r.Game.Board = [24]int{}
	r.Game.Board[10] = 1
	r.Game.Board[20] = -1
	r.Game.Turn = 0
	r.Game.HasRolled = true
	r.Game.MovesLeft = []int{3}
	r.Game.Dealt = 1
	mc := &mockConn{}
	ach := make(chan []byte, 64)
	r.AddSub(ach, "solo")
	from, to, die := 10, 7, 3
	// idx would be 0 for solo anyway; force turn-seat override path by playing black next
	r.GameTurn(mc, nil, "", 0, turnMsg{T: "move", From: &from, To: &to, Die: &die}, ach)
	if r.Game.Board[7] != 1 {
		t.Fatal("white move not applied")
	}
	// now black to move (turn auto-passed? movesLeft empty -> turn=1)
	if r.Game.Turn != 1 {
		t.Fatalf("turn=%v, want black", r.Game.Turn)
	}
	// black answers through the same hotseat seat
	r.Game.Board[20] = -1
	r.Game.HasRolled = true
	r.Game.MovesLeft = []int{3}
	r.Game.Dealt = 1
	from, to, die = 20, 23, 3
	r.GameTurn(mc, nil, "", 1, turnMsg{T: "move", From: &from, To: &to, Die: &die}, ach)
	if r.Game.Board[23] != -1 {
		t.Fatal("black move not applied")
	}
	for _, m := range collect(ach) {
		if m["t"] == "error" {
			t.Fatalf("hotseat ERROR: %v", m["msg"])
		}
	}
}
