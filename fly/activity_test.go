// LogActivity writes one JSON line with the reservoir vector for viz.
package fly

import (
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"fair_backgammon/game"
)

func TestLogActivityWritesReservoir(t *testing.T) {
	heads, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	h := heads["retarded"]
	if h == nil {
		t.Fatal("no retarded variant")
	}
	w := h.W["retarded"]
	var vecs []struct {
		Board [24]int  `json:"board"`
		Bar   [2]int   `json:"bar"`
		Off   [2]int   `json:"off"`
		Turn  int      `json:"turn"`
		Moves [][3]int `json:"moves"`
	}
	if err := json.Unmarshal(rawVectors, &vecs); err != nil || len(vecs) == 0 {
		t.Fatalf("vectors: %v", err)
	}
	v := vecs[0]
	moves := make([]game.Move, len(v.Moves))
	for j, m := range v.Moves {
		moves[j] = game.Move{From: m[0], To: m[1], Die: m[2]}
	}
	rng := rand.New(rand.NewSource(1))
	pick := Choose(h, w, v.Board, v.Bar, v.Off, v.Turn, moves, rng)
	path := filepath.Join(t.TempDir(), "act.jsonl")
	t.Setenv("FLY_ACTIVITY_LOG", path)
	LogActivity(h, w, "trained", v.Board, v.Bar, v.Off, v.Turn, pick)
	buf, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	var rec struct {
		Variant string         `json:"variant"`
		X       []float64      `json:"x"`
		V       float64        `json:"v"`
		Move    map[string]int `json:"move"`
	}
	if err := json.Unmarshal(buf, &rec); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if rec.Variant != "trained" || len(rec.X) != h.N {
		t.Fatalf("bad record variant=%s xdim=%d want %d", rec.Variant, len(rec.X), h.N)
	}
	if rec.Move["from"] != pick.From || rec.Move["to"] != pick.To {
		t.Fatalf("move mismatch %+v vs %+v", rec.Move, pick)
	}
}
