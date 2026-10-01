package tests

import (
	"encoding/json"
	"testing"
	"time"

	"fair_backgammon/lobby"
)

func TestGifBroadcast(t *testing.T) {
	h := lobby.NewHub()
	r := h.Create("alice")
	h.Join(r.Code, "bob")
	chA := make(chan []byte, 4)
	chB := make(chan []byte, 4)
	r.AddSub(chA, "alice")
	r.AddSub(chB, "bob")
	mc := &mockConn{}
	url := "https://media.klipy.com/x/abc.gif"
	r.GameTurn(mc, nil, "", 0, turnMsg{T: "gif", Url: &url}, chA)
	for name, ch := range map[string]chan []byte{"alice": chA, "bob": chB} {
		select {
		case b := <-ch:
			var m map[string]any
			if err := json.Unmarshal(b, &m); err != nil {
				t.Fatalf("%s bad json: %v", name, err)
			}
			if m["t"] != "gif" || m["url"] != url || m["from"] != float64(0) {
				t.Fatalf("%s got %v", name, m)
			}
		case <-time.After(time.Second):
			t.Fatalf("%s no gif broadcast", name)
		}
	}
}

func TestGifValidation(t *testing.T) {
	h := lobby.NewHub()
	r := h.Create("alice")
	h.Join(r.Code, "bob")
	mc := &mockConn{}
	bad := "http://evil.example/x.gif"
	r.GameTurn(mc, nil, "", 0, turnMsg{T: "gif", Url: &bad}, nil)
	if mc.lastErr() != "bad gif url" {
		t.Fatalf("want bad gif url got %s", mc.lastErr())
	}
	mc = &mockConn{}
	good := "https://media.klipy.com/x/ok.gif"
	r.GameTurn(mc, nil, "", 0, turnMsg{T: "gif", Url: &good}, nil)
	if mc.lastErr() != "" {
		t.Fatalf("want ok got %s", mc.lastErr())
	}
	mc = &mockConn{}
	r.GameTurn(mc, nil, "", 0, turnMsg{T: "gif", Url: &good}, nil)
	if mc.lastErr() != "gif too fast" {
		t.Fatalf("want gif too fast got %s", mc.lastErr())
	}
}