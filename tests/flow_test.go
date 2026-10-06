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
