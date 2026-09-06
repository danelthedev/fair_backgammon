package tests

import (
	"sync"
	"testing"

	"fair_backgammon/lobby"
)

func TestConcurrentWriteNoPanic(t *testing.T) {
	h := lobby.NewHub()
	r := h.Create("alice")
	h.Join(r.Code, "bob")
	// create per-conn channel as in wsHandler
	ch := make(chan []byte, 16)
	r.AddSub(ch, "alice")
	// start writer that would have been conn.WriteMessage
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range ch {
		}
	}()
	// setup game state once with lock
	r.Game.HasRolled = true
	r.Game.MovesLeft = []int{3}
	r.Game.Turn = 0
	r.Game.Board[12] = 1
	// concurrent GameTurn calls (simulates two messages at same time)
	var wg2 sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg2.Add(1)
		go func() {
			defer wg2.Done()
			from, to, die := 12, 9, 3
			fakeConn := &fakeConn{}
			r.GameTurn(fakeConn, nil, "", 0, struct {
				T    string `json:"t"`
				From *int   `json:"from"`
				To   *int   `json:"to"`
				Die  *int   `json:"die"`
			}{T: "move", From: &from, To: &to, Die: &die}, ch)
		}()
	}
	wg2.Wait()
	close(ch)
	wg.Wait()
}

type fakeConn struct{}

func (f *fakeConn) WriteMessage(int, []byte) error { return nil }
