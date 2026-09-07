package tests

import (
	"encoding/json"
	"fair_backgammon/lobby"
	"testing"
)

type turnMsg struct {
	T      string  `json:"t"`
	From   *int    `json:"from"`
	To     *int    `json:"to"`
	Die    *int    `json:"die"`
	Action *string `json:"action"`
}

func TestDoubleAcceptScoresStake(t *testing.T) {
	h := lobby.NewHub()
	r := h.Create("alice")
	h.Join(r.Code, "bob")
	mc := &mockConn{}
	r.GameTurn(mc, nil, "", 0, turnMsg{T: "double"})
	if r.DoubleOffer == nil || r.DoubleOffer.By != 0 || r.DoubleOffer.Stake != 2 {
		t.Fatalf("offer bad %+v", r.DoubleOffer)
	}
	// roll blocked while offer pending
	r.GameTurn(mc, nil, "", 0, turnMsg{T: "roll"})
	if mc.lastErr() != "double offer pending" {
		t.Fatalf("want double pending, got %q", mc.lastErr())
	}
	act := "accept"
	r.GameTurn(mc, nil, "", 1, turnMsg{T: "double_response", Action: &act})
	if r.Cube != 2 || r.DoubleOffer != nil {
		t.Fatalf("accept bad cube=%d offer=%+v", r.Cube, r.DoubleOffer)
	}
	// win awards the stake, not 1
	r.Game.Board = [24]int{}
	r.Game.Board[0] = 1
	r.Game.Bar = [2]int{}
	r.Game.Off = [2]int{14, 0}
	r.Game.Turn = 0
	r.Game.HasRolled = true
	r.Game.MovesLeft = []int{1}
	from, to, die := 0, -2, 1
	r.GameTurn(mc, nil, "", 0, turnMsg{T: "move", From: &from, To: &to, Die: &die})
	if r.Scores[0] != 2 || r.Scores[1] != 0 {
		t.Fatalf("scores %v, want [2 0]", r.Scores)
	}
}

func TestDoubleRejectAndRedouble(t *testing.T) {
	h := lobby.NewHub()
	r := h.Create("alice")
	h.Join(r.Code, "bob")
	mc := &mockConn{}
	r.GameTurn(mc, nil, "", 0, turnMsg{T: "double"})
	act := "reject"
	r.GameTurn(mc, nil, "", 1, turnMsg{T: "double_response", Action: &act})
	if r.Scores[0] != 1 || r.Game.Off[0] != 15 {
		t.Fatalf("reject bad scores=%v off=%v", r.Scores, r.Game.Off)
	}

	h2 := lobby.NewHub()
	r2 := h2.Create("alice")
	h2.Join(r2.Code, "bob")
	r2.GameTurn(mc, nil, "", 0, turnMsg{T: "double"})
	act = "redouble"
	r2.GameTurn(mc, nil, "", 1, turnMsg{T: "double_response", Action: &act})
	if r2.Cube != 4 || r2.DoubleOffer != nil {
		t.Fatalf("redouble should auto-accept at 4x, cube=%d offer=%+v", r2.Cube, r2.DoubleOffer)
	}
}
func TestDoubleOncePerTurn(t *testing.T) {
	h := lobby.NewHub()
	r := h.Create("alice")
	h.Join(r.Code, "bob")
	mc := &mockConn{}
	r.GameTurn(mc, nil, "", 0, turnMsg{T: "double"})
	act := "accept"
	r.GameTurn(mc, nil, "", 1, turnMsg{T: "double_response", Action: &act})
	r.GameTurn(mc, nil, "", 0, turnMsg{T: "double"})
	if mc.lastErr() != "wait for opponent to double" {
		t.Fatalf("want wait, got %q", mc.lastErr())
	}
	// turn switch clears the flag: empty board has no legal moves after roll
	r.Game.Board = [24]int{}
	r.Game.Bar = [2]int{}
	r.Game.HasRolled = false
	r.Game.MovesLeft = nil
	r.Game.Turn = 0
	r.GameTurn(mc, nil, "", 0, turnMsg{T: "roll"})
	if r.Game.Turn != 1 || r.DoubledThisTurn {
		t.Fatalf("turn=%d flag=%v, want turn=1 flag=false", r.Game.Turn, r.DoubledThisTurn)
	}
}
func TestDoubleOfferBroadcastToBoth(t *testing.T) {
	h := lobby.NewHub()
	r := h.Create("alice")
	h.Join(r.Code, "bob")
	ch := make(chan []byte, 16)
	r.AddSub(ch, "alice")
	mc := &mockConn{}
	r.GameTurn(mc, nil, "", 0, turnMsg{T: "double"})
	// drain to the latest state message
	var last map[string]any
	for len(ch) > 0 {
		var v map[string]any
		b := <-ch
		if json.Unmarshal(b, &v) != nil {
			continue
		}
		if v["t"] == "state" {
			last = v
		}
	}
	if last == nil {
		t.Fatal("no state broadcast after double")
	}
	off, ok := last["doubleOffer"].(map[string]any)
	if !ok {
		t.Fatalf("state missing doubleOffer: %v", last["doubleOffer"])
	}
	if int(off["by"].(float64)) != 0 || int(off["stake"].(float64)) != 2 {
		t.Fatalf("offer bad %v", off)
	}
	if int(last["cube"].(float64)) != 1 {
		t.Fatalf("cube should still be 1 until accept, got %v", last["cube"])
	}
}

func TestDoubleAlternatesBetweenPlayers(t *testing.T) {
	h := lobby.NewHub()
	r := h.Create("alice")
	h.Join(r.Code, "bob")
	mc := &mockConn{}
	r.GameTurn(mc, nil, "", 0, turnMsg{T: "double"})
	act := "accept"
	r.GameTurn(mc, nil, "", 1, turnMsg{T: "double_response", Action: &act})
	if r.Cube != 2 || r.LastDoubler != 0 {
		t.Fatalf("after accept cube=%d lastDoubler=%d", r.Cube, r.LastDoubler)
	}
	r.GameTurn(mc, nil, "", 0, turnMsg{T: "double"})
	if mc.lastErr() != "wait for opponent to double" {
		t.Fatalf("want wait, got %q", mc.lastErr())
	}
	// redouble auto-accepts: fresh offer from A, B answers "double again"
	h2 := lobby.NewHub()
	r2 := h2.Create("alice")
	h2.Join(r2.Code, "bob")
	r2.GameTurn(mc, nil, "", 0, turnMsg{T: "double"})
	act = "redouble"
	r2.GameTurn(mc, nil, "", 1, turnMsg{T: "double_response", Action: &act})
	if r2.Cube != 4 || r2.DoubleOffer != nil || r2.LastDoubler != 0 {
		t.Fatalf("redouble should auto-accept at 4x, cube=%d offer=%+v lastDoubler=%d", r2.Cube, r2.DoubleOffer, r2.LastDoubler)
	}
	// B doubles on B's turn (turn switch resets per-turn flag, LastDoubler still 0)
	r2.Game.Turn = 1
	r2.Game.HasRolled = false
	r2.DoubledThisTurn = false
	r2.GameTurn(mc, nil, "", 1, turnMsg{T: "double"})
	if r2.DoubleOffer == nil || r2.DoubleOffer.Stake != 8 || r2.LastDoubler != 1 {
		t.Fatalf("B double bad offer=%+v lastDoubler=%d", r2.DoubleOffer, r2.LastDoubler)
	}
	act = "accept"
	r2.GameTurn(mc, nil, "", 0, turnMsg{T: "double_response", Action: &act})
	if r2.Cube != 8 {
		t.Fatalf("cube=%d want 8", r2.Cube)
	}
}
