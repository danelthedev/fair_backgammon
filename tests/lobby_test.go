package tests

import (
	"encoding/json"
	"fair_backgammon/game"
	"fair_backgammon/lobby"
	"testing"
)

func TestHubCreateJoin(t *testing.T) {
	h := lobby.NewHub()
	r := h.Create("alice")
	if r.Players[0] != "alice" || r.Code == "" {
		t.Fatalf("create bad %v", r)
	}
	_, idx, ok := h.Join(r.Code, "alice")
	if !ok || idx != 0 {
		t.Fatalf("alice rejoin fail")
	}
	_, idx, ok = h.Join(r.Code, "bob")
	if !ok || idx != 1 {
		t.Fatalf("bob join fail %d %v", idx, ok)
	}
	_, _, ok = h.Join(r.Code, "eve")
	if ok {
		t.Fatalf("should be full")
	}
	_, _, ok = h.Join("XXXX", "x")
	if ok {
		t.Fatalf("bad code should fail")
	}
	if h.Get("XXXX") != nil {
		t.Fatal("get bad should nil")
	}
}

func TestHubGenCodeUnique(t *testing.T) {
	h := lobby.NewHub()
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		r := h.Create("u")
		if seen[r.Code] {
			t.Fatalf("dup code %s", r.Code)
		}
		seen[r.Code] = true
	}
}

type mockConn struct{ msgs [][]byte }

func (m *mockConn) WriteMessage(_ int, b []byte) error { m.msgs = append(m.msgs, b); return nil }
func (m *mockConn) lastErr() string {
	for i := len(m.msgs) - 1; i >= 0; i-- {
		var v map[string]string
		if json.Unmarshal(m.msgs[i], &v) == nil && v["t"] == "error" {
			return v["msg"]
		}
	}
	return ""
}

func TestTurnRollAndMove(t *testing.T) {
	h := lobby.NewHub()
	r := h.Create("alice")
	h.Join(r.Code, "bob")
	mc := &mockConn{}
	bob := &mockConn{}
	bobMsg := struct {
		T      string  `json:"t"`
		From   *int    `json:"from"`
		To     *int    `json:"to"`
		Die    *int    `json:"die"`
		Action *string `json:"action"`
	}{T: "roll"}
	r.GameTurn(bob, nil, "", 1, bobMsg)
	if bob.lastErr() != "not your turn" {
		t.Fatalf("want not your turn got %s", bob.lastErr())
	}
	r.GameTurn(mc, nil, "", 0, bobMsg)
	if !r.Game.HasRolled || len(r.Game.MovesLeft) == 0 {
		t.Fatalf("roll failed %v", r.Game)
	}
	die := 7
	from, to := 23, 20
	bad := struct {
		T      string  `json:"t"`
		From   *int    `json:"from"`
		To     *int    `json:"to"`
		Die    *int    `json:"die"`
		Action *string `json:"action"`
	}{T: "move", From: &from, To: &to, Die: &die}
	mc2 := &mockConn{}
	r.GameTurn(mc2, nil, "", 0, bad)
	if mc2.lastErr() == "" {
		t.Fatalf("want error for bad die")
	}
	d := r.Game.MovesLeft[0]
	var found *game.Move
	for _, f := range []int{23, 12, 7, 5} {
		for to := 0; to < 24; to++ {
			m := game.Move{From: f, To: to, Die: d}
			if ok, _ := r.Game.IsLegal(m); ok {
				found = &m
				break
			}
		}
		if found != nil {
			break
		}
	}
	if found == nil {
		t.Fatalf("no legal move for die %d", d)
	}
	mc3 := &mockConn{}
	r.GameTurn(mc3, nil, "", 0, struct {
		T      string  `json:"t"`
		From   *int    `json:"from"`
		To     *int    `json:"to"`
		Die    *int    `json:"die"`
		Action *string `json:"action"`
	}{T: "move", From: &found.From, To: &found.To, Die: &found.Die})
	if mc3.lastErr() != "" {
		t.Fatalf("legal move got err %s", mc3.lastErr())
	}
}

func TestWinBroadcast(t *testing.T) {
	h := lobby.NewHub()
	r := h.Create("alice")
	r.Game.Board = [24]int{}
	r.Game.Bar = [2]int{}
	r.Game.Off = [2]int{14, 0}
	r.Game.Board[0] = 1
	r.Game.Turn = 0
	r.Game.HasRolled = true
	r.Game.MovesLeft = []int{1}
	r.Game.Dice = [2]int{1, 2}
	ch := make(chan []byte, 4)
	r.AddSub(ch, "alice")
	mc := &mockConn{}
	from, to, die := 0, -2, 1
	r.GameTurn(mc, nil, "", 0, struct {
		T      string  `json:"t"`
		From   *int    `json:"from"`
		To     *int    `json:"to"`
		Die    *int    `json:"die"`
		Action *string `json:"action"`
	}{T: "move", From: &from, To: &to, Die: &die})
	foundWin := false
	for len(ch) > 0 {
		b := <-ch
		var v map[string]any
		json.Unmarshal(b, &v)
		if v["t"] == "win" {
			foundWin = true
		}
	}
	if !foundWin {
		t.Fatalf("win not broadcast")
	}
	if r.Game.Off[0] != 15 {
		t.Fatalf("off not 15 %v", r.Game.Off)
	}
}
