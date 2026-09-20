// Equivalence with musca: every exported vector must pick the same move.
// Regenerate vectors via musca/scripts/export_go.py after retraining.
package fly

import (
	_ "embed"
	"encoding/json"
	"math/rand"
	"testing"

	"fair_backgammon/game"
)

//go:embed testdata/vectors.json
var rawVectors []byte

func TestChooseMatchesPython(t *testing.T) {
	h, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	var vecs []struct {
		Board [24]int  `json:"board"`
		Bar   [2]int   `json:"bar"`
		Off   [2]int   `json:"off"`
		Turn  int      `json:"turn"`
		Moves [][3]int `json:"moves"`
		Pick  [3]int   `json:"pick"`
	}
	if err := json.Unmarshal(rawVectors, &vecs); err != nil {
		t.Fatalf("vectors: %v", err)
	}
	if len(vecs) == 0 {
		t.Fatal("no vectors")
	}
	w := h.W["trained"]
	rng := rand.New(rand.NewSource(1))
	bad := 0
	for i, v := range vecs {
		moves := make([]game.Move, len(v.Moves))
		for j, m := range v.Moves {
			moves[j] = game.Move{From: m[0], To: m[1], Die: m[2]}
		}
		got := Choose(h, w, v.Board, v.Bar, v.Off, v.Turn, moves, rng)
		if [3]int{got.From, got.To, got.Die} != v.Pick {
			bad++
			if bad <= 3 {
				t.Errorf("vector %d: got %v want %v", i, got, v.Pick)
			}
		}
	}
	if bad > 0 {
		t.Fatalf("%d/%d vectors disagree with python", bad, len(vecs))
	}
}

func TestPipOpening(t *testing.T) {
	var b [24]int
	b[23], b[12], b[7], b[5] = 2, 5, 3, 5
	b[0], b[11], b[16], b[18] = -2, -5, -3, -5
	if Pip(b, [2]int{}, 0) != 167 || Pip(b, [2]int{}, 1) != 167 {
		t.Fatalf("opening pip %d %d", Pip(b, [2]int{}, 0), Pip(b, [2]int{}, 1))
	}
	if len(EncodeState(b, [2]int{}, [2]int{}, 0)) != 52 {
		t.Fatal("encode dim != 52")
	}
}
