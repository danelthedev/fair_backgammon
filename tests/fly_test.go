package tests

import (
	"fair_backgammon/game"
	"fair_backgammon/lobby"
	"testing"
)

func TestCreateVsFly(t *testing.T) {
	h := lobby.NewHub()
	r := h.CreateVsFly("alice", "Fly")
	if !r.VsFly {
		t.Fatal("VsFly not set")
	}
	seen := map[string]bool{}
	for _, p := range r.Players {
		seen[p] = true
	}
	if !seen["alice"] || !seen["Fly"] {
		t.Fatalf("seats %v", r.Players)
	}
	if _, _, ok := h.Join(r.Code, "eve"); ok {
		t.Fatal("fly room should be full")
	}
	// random color across rooms
	slots := map[int]bool{}
	for i := 0; i < 20; i++ {
		rr := h.CreateVsFly("bob", "Fly")
		for idx, p := range rr.Players {
			if p == "bob" {
				slots[idx] = true
			}
		}
	}
	if !slots[0] || !slots[1] {
		t.Fatalf("human color never varied %v", slots)
	}
}

func TestVsFlyNoDouble(t *testing.T) {
	h := lobby.NewHub()
	r := h.CreateVsFly("alice", "Fly")
	hi := r.PlayerIndex("alice")
	r.Game.Turn = game.Player(hi) // human to move
	mc := &mockConn{}
	dbl := struct {
		T      string  `json:"t"`
		From   *int    `json:"from"`
		To     *int    `json:"to"`
		Die    *int    `json:"die"`
		Action *string `json:"action"`
	}{T: "double"}
	r.GameTurn(mc, nil, "", hi, dbl)
	if mc.lastErr() != "doubling disabled vs fly" {
		t.Fatalf("want double rejection got %s", mc.lastErr())
	}
}
