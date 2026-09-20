// Expectimax equivalence with musca brain/search.py.
package fly

import (
	_ "embed"
	"encoding/json"
	"math/rand"
	"testing"

	"fair_backgammon/game"
)

//go:embed testdata/em_vectors.json
var rawEmVectors []byte

func TestExpectimaxMatchesPython(t *testing.T) {
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
	if err := json.Unmarshal(rawEmVectors, &vecs); err != nil {
		t.Fatalf("vectors: %v", err)
	}
	if len(vecs) == 0 {
		t.Fatal("no vectors")
	}
	w := h.W["expert"]
	rng := rand.New(rand.NewSource(1))
	bad := 0
	for i, v := range vecs {
		moves := make([]game.Move, len(v.Moves))
		for j, m := range v.Moves {
			moves[j] = game.Move{From: m[0], To: m[1], Die: m[2]}
		}
		got := Expectimax(h, w, v.Board, v.Bar, v.Off, v.Turn, moves, rng)
		if [3]int{got.From, got.To, got.Die} != v.Pick {
			bad++
			if bad <= 3 {
				t.Errorf("vector %d: got %v want %v", i, got, v.Pick)
			}
		}
	}
	if bad > 0 {
		t.Fatalf("%d/%d em vectors disagree with python", bad, len(vecs))
	}
}

func TestExpectimaxLegal(t *testing.T) {
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
	}
	if err := json.Unmarshal(rawVectors, &vecs); err != nil {
		t.Fatalf("vectors: %v", err)
	}
	rng := rand.New(rand.NewSource(2))
	for i, v := range vecs {
		for _, name := range []string{"trained", "untrained", "expert"} {
			moves := make([]game.Move, len(v.Moves))
			for j, m := range v.Moves {
				moves[j] = game.Move{From: m[0], To: m[1], Die: m[2]}
			}
			got := Expectimax(h, h.W[name], v.Board, v.Bar, v.Off, v.Turn, moves, rng)
			found := false
			for _, m := range moves {
				if m == got {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("vector %d (%s): pick %v not legal", i, name, got)
			}
		}
	}
}
